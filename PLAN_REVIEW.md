# Staff Engineer Review: Offline ls-refs Implementation Plan

## Executive Summary
**Recommendation**: Simplify the implementation significantly. We're over-engineering the solution.

**Key insight**: We already have a local git repository on disk that IS the cache. We don't need a separate ls-refs cache layer.

---

## Critical Issues with Current Plan

### 1. Over-Engineering: Unnecessary Cache Layer ❌

**Problem**: The plan introduces a new cache layer (`LsRefsCache`) with:
- In-memory storage (`map[string]*LsRefsCacheEntry`)
- Disk persistence (JSON files)
- TTL management
- Cache invalidation logic
- ~300+ lines of new code

**Why this is wrong**: We already have the refs cached in the local git repository at `{LocalDiskCacheRoot}/{host}/{path}`. The local git repo already maintains refs in `.git/refs/` and `.git/packed-refs`.

**Evidence**:
- `managed_repository.go:251-268` already reads refs from local repo using `go-git` library
- `hasAnyUpdate()` uses `git.PlainOpen()` and `g.Reference()` to read refs
- Local repo is kept up-to-date by `fetchUpstream()` (already exists)

### 2. Testing Complexity ❌

**Current plan requires**:
- Mock cache state
- Manage TTL expiration
- Test cache persistence/loading
- Handle cache corruption
- Test race conditions in cache access

**This is 5x more test surface area than needed.**

### 3. Configuration Bloat ❌

Four new config options:
```go
EnableOfflineMode  bool          // Do we need this?
UpstreamEnabled    bool          // OK for testing
LsRefsCacheTTL     time.Duration // Unnecessary if using local repo
LsRefsCachePath    string        // Unnecessary
```

**We only need one**: `UpstreamEnabled` for testing.

---

## Simplified Architecture

### Core Insight
**The local git repository IS the cache.** We just need to read from it when upstream is unavailable.

### Implementation (3 simple changes)

#### Change 1: Add `lsRefsLocal()` method
**File**: `managed_repository.go` (new method, ~30 lines)

```go
func (r *managedRepository) lsRefsLocal(command *gitprotocolio.ProtocolV2Command) (map[string]plumbing.Hash, []byte, error) {
    // Open local git repo
    g, err := git.PlainOpen(r.localDiskPath)
    if err != nil {
        return nil, nil, status.Errorf(codes.Unavailable, "local repo not available: %v", err)
    }

    // List all refs
    refs, err := g.References()
    if err != nil {
        return nil, nil, status.Errorf(codes.Internal, "failed to read refs: %v", err)
    }

    // Convert to map and protocol response
    refMap := make(map[string]plumbing.Hash)
    var buf bytes.Buffer

    refs.ForEach(func(ref *plumbing.Reference) error {
        // Apply ls-refs filters from command (ref-prefix, etc.)
        if shouldIncludeRef(ref, command) {
            refMap[ref.Name().String()] = ref.Hash()
            fmt.Fprintf(&buf, "%s %s\n", ref.Hash(), ref.Name())
        }
        return nil
    })

    // Add symrefs (HEAD -> refs/heads/main)
    head, _ := g.Head()
    if head != nil {
        fmt.Fprintf(&buf, "symref-target:%s %s\n", head.Name(), "HEAD")
    }

    buf.WriteString("0000") // Protocol delimiter
    return refMap, buf.Bytes(), nil
}
```

#### Change 2: Update `handleV2Command` for ls-refs
**File**: `git_protocol_v2_handler.go:54-83` (modify existing)

```go
case "ls-refs":
    var refs map[string]plumbing.Hash
    var rawResponse []byte
    var err error
    var source string

    // Try upstream first (if enabled)
    if repo.config.UpstreamEnabled {
        refs, rawResponse, err = repo.lsRefsUpstream(command)
        source = "upstream"

        if err != nil {
            // Upstream failed, try local fallback
            log.Printf("Upstream ls-refs failed (%v), falling back to local", err)
            refs, rawResponse, err = repo.lsRefsLocal(command)
            source = "local-fallback"
        }
    } else {
        // Testing mode: serve from local only
        refs, rawResponse, err = repo.lsRefsLocal(command)
        source = "local"
    }

    if err != nil {
        return err
    }

    // Log staleness warning if serving from local
    if source != "upstream" && time.Since(repo.lastUpdate) > 5*time.Minute {
        log.Printf("Warning: serving stale ls-refs for %s (last update: %v ago)",
            repo.localDiskPath, time.Since(repo.lastUpdate))
    }

    // ... rest of existing logic (hasAnyUpdate check, etc.)
    repo.config.RequestLogger(req, "ls-refs", source, ...)
```

#### Change 3: Add single config option
**File**: `server_config.go` or inline

```go
type ServerConfig struct {
    // ... existing fields ...

    // Testing: set false to disable all upstream calls
    UpstreamEnabled bool  // default: true
}
```

**That's it.** Three changes, ~60 lines of code total.

---

## Why This is Better

### 1. Simplicity ✅
- **No new data structures**: Uses existing local git repo
- **No cache management**: Git handles ref storage
- **No TTL logic**: Just check `lastUpdate` timestamp (already exists)
- **No persistence code**: Git already persists refs to disk

### 2. Testability ✅

**Unit tests** (simple mocks):
```go
func TestLsRefsLocal(t *testing.T) {
    // Create test git repo
    repo := createTestRepo(t)

    // Write some refs
    writeRef(repo, "refs/heads/main", "abc123")
    writeRef(repo, "refs/tags/v1.0", "def456")

    // Read via lsRefsLocal
    mr := &managedRepository{localDiskPath: repo.Path()}
    refs, _, err := mr.lsRefsLocal(nil)

    require.NoError(t, err)
    assert.Equal(t, "abc123", refs["refs/heads/main"])
    assert.Equal(t, "def456", refs["refs/tags/v1.0"])
}
```

**Integration tests** (no mocking needed):
```go
func TestLsRefsOfflineMode(t *testing.T) {
    // Step 1: Normal operation (populate local cache)
    server := NewTestServer(t)
    client := NewGitClient(server.URL)

    refs1, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)

    // Step 2: Disable upstream
    server.config.UpstreamEnabled = false

    // Step 3: Should still work (serves from local)
    refs2, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)
    assert.Equal(t, refs1, refs2)
}

func TestLsRefsNoLocalCache(t *testing.T) {
    // Start server with upstream disabled
    server := NewTestServer(t)
    server.config.UpstreamEnabled = false

    client := NewGitClient(server.URL)

    // Should fail: no local cache exists
    _, err := client.LsRefs("github.com/never/cached")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "local repo not available")
}
```

### 3. Maintenance ✅
- **Fewer bugs**: Less code = fewer bugs
- **No cache invalidation bugs**: Git handles consistency
- **No cache corruption**: Git is battle-tested
- **No synchronization bugs**: We already lock `managedRepository`

### 4. Performance ✅
- **Fast**: Reading from local git repo is ~1-2ms
- **No extra memory**: No in-memory cache needed
- **No extra I/O**: No separate cache file writes

---

## Comparison: Lines of Code

| Component | Original Plan | Simplified |
|-----------|---------------|------------|
| Cache manager | ~150 lines | 0 |
| Cache persistence | ~80 lines | 0 |
| TTL management | ~40 lines | 0 |
| Configuration | ~20 lines | ~5 lines |
| Core logic change | ~50 lines | ~35 lines |
| Unit tests | ~200 lines | ~50 lines |
| Integration tests | ~150 lines | ~50 lines |
| **Total** | **~690 lines** | **~140 lines** |

**5x reduction in code and complexity.**

---

## What We Still Get

✅ **Offline resilience**: Serves ls-refs when upstream is down
✅ **Testing support**: `UpstreamEnabled = false` for tests
✅ **Staleness tracking**: Use existing `lastUpdate` timestamp
✅ **Zero config**: Works out of the box, no tuning needed
✅ **Observability**: Log source (upstream/local-fallback/local)

---

## What We Lose (Intentionally)

❌ **Separate cache file**: Don't need it, git repo is the cache
❌ **Configurable TTL**: Use `lastUpdate`, warn if > 5min
❌ **Cache warming**: Happens naturally via `fetchUpstream()`
❌ **Circuit breaker**: Can add later if needed (YAGNI)

None of these are necessary for the core requirement.

---

## Implementation Plan (Simplified)

### Phase 1: Core Implementation (1 day)
1. Add `lsRefsLocal()` method to `managed_repository.go`
2. Modify `handleV2Command` to try local on upstream failure
3. Add `UpstreamEnabled` config option

### Phase 2: Testing (1 day)
1. Unit test `lsRefsLocal()` with various ref scenarios
2. Integration test: offline mode with warm cache
3. Integration test: offline mode with cold cache
4. Integration test: stale cache warning

### Phase 3: Documentation (0.5 days)
1. Update README.md limitation note
2. Add example test usage

**Total: 2.5 days** (vs 8-12 days in original plan)

---

## Recommended Changes to Plan

### Remove These Sections
- ❌ Section 2.2: "ls-refs Cache Structure" - unnecessary
- ❌ Section 2.3: "Modified Request Flow" - over-complicated
- ❌ Phase 1.2: "Create ls-refs Cache Manager" - don't need it
- ❌ Phase 1.3: "Initialize Cache on Server Start" - nothing to initialize
- ❌ Phase 2.1: Caching in `lsRefsUpstream` - just rely on `fetchUpstream`
- ❌ Section 3.1: Complex metrics - simple counters are enough
- ❌ "Risks and Mitigations" section - most risks gone with simpler design

### Keep These (Simplified)
- ✅ `UpstreamEnabled` config option
- ✅ Basic integration tests
- ✅ README update
- ✅ Request logging with source indicator

---

## Questions to Answer

### Q: "What if the local repo is corrupted?"
**A**: Same as today - the repo is already critical infrastructure. Git corruption is extremely rare and already a failure mode for fetch operations.

### Q: "What about cache staleness?"
**A**: We already track `lastUpdate` timestamp. Just log warnings if serving refs older than 5 minutes. No TTL needed.

### Q: "What if refs are deleted upstream?"
**A**: Next `fetchUpstream()` will sync. Until then, serving stale refs is better than being completely down. This is acceptable for a cache.

### Q: "How do we force cache refresh?"
**A**: Already exists: `fetchUpstream()` is called when `hasAnyUpdate()` detects changes. No new code needed.

---

## Summary

**Original plan**: 690 lines, 8-12 days, complex cache layer
**Simplified plan**: 140 lines, 2.5 days, leverage existing git repo

**Staff engineer principle**: Use existing infrastructure. The local git repository is already a perfect cache for refs. Adding another cache layer is textbook over-engineering.

**Recommendation**:
1. Implement the 3-change simplified version
2. Ship it and gather metrics
3. Only add complexity if data shows it's needed (it won't be)

---

## Next Steps

If you agree with this review:
1. Archive `OFFLINE_MODE_PLAN.md` as reference
2. Create `OFFLINE_MODE_PLAN_V2.md` with simplified approach
3. Start implementation with Phase 1 (core logic)
4. Write tests as we go (TDD)

**Estimated delivery**: 2-3 days vs 2-3 weeks
