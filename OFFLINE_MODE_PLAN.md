# Implementation Plan: Offline ls-refs Support

## Overview
Enable Goblet to serve ls-refs requests from cache when the upstream server is unavailable, making the proxy resilient to upstream failures.

## Current Limitation
From `README.md:28-31`:
> Note that Goblet forwards the ls-refs traffic to the upstream server. If the upstream server is down, Goblet is effectively down. Technically, we can modify Goblet to serve even if the upstream is down, but the current implementation doesn't do such thing.

## Goals
1. ✅ Cache ls-refs responses for offline serving
2. ✅ Serve from cache when upstream is unavailable
3. ✅ Add configuration to enable/disable upstream (for testing)
4. ✅ Maintain backward compatibility
5. ✅ Provide clear metrics and health status

---

## Architecture Changes

### 1. Configuration Extension (`ServerConfig`)

**File**: `server_config.go` or inline in relevant files

Add new configuration options:

```go
type ServerConfig struct {
    // ... existing fields ...

    // Offline mode configuration
    EnableOfflineMode      bool          // Enable ls-refs cache fallback
    UpstreamEnabled        bool          // For testing: disable upstream completely
    LsRefsCacheTTL         time.Duration // How long to trust cached ls-refs (default: 5m)
    LsRefsCachePath        string        // Path to persist ls-refs cache (optional)
}
```

**Default values**:
- `EnableOfflineMode`: `true` (enable resilience)
- `UpstreamEnabled`: `true` (production default)
- `LsRefsCacheTTL`: `5 * time.Minute`
- `LsRefsCachePath`: `{LocalDiskCacheRoot}/.ls-refs-cache`

### 2. ls-refs Cache Structure

**File**: `ls_refs_cache.go` (new file)

```go
type LsRefsCache struct {
    mu       sync.RWMutex
    entries  map[string]*LsRefsCacheEntry
    diskPath string
}

type LsRefsCacheEntry struct {
    RepoPath     string                 // Repository identifier
    Refs         map[string]string      // ref name -> commit hash
    SymRefs      map[string]string      // symbolic refs (HEAD -> refs/heads/main)
    Timestamp    time.Time              // When cached
    RawResponse  []byte                 // Original protocol response
    UpstreamURL  string                 // Source upstream
}
```

**Operations**:
- `Get(repoPath string) (*LsRefsCacheEntry, bool)`
- `Set(repoPath string, entry *LsRefsCacheEntry) error`
- `IsStale(entry *LsRefsCacheEntry, ttl time.Duration) bool`
- `LoadFromDisk() error`
- `SaveToDisk() error`
- `Invalidate(repoPath string)`

### 3. Modified Request Flow

**File**: `git_protocol_v2_handler.go`

Current flow:
```
ls-refs request
    ↓
lsRefsUpstream() ──[error]──> return error to client
    ↓
return upstream response
```

New flow:
```
ls-refs request
    ↓
Check if UpstreamEnabled == false (test mode)
    ↓ [false]
    Serve from cache or error

    ↓ [true]
Try lsRefsUpstream()
    ↓
    ├─ [success] ──> Cache response ──> Return to client
    │
    └─ [error]
        ↓
        Check EnableOfflineMode
        ↓
        ├─ [false] ──> Return error (current behavior)
        │
        └─ [true]
            ↓
            Check cache for valid entry
            ↓
            ├─ [found & fresh] ──> Serve from cache (with warning header)
            ├─ [found & stale]  ──> Serve from cache (with staleness warning)
            └─ [not found]      ──> Return error (no cached data)
```

---

## Implementation Steps

### Phase 1: Configuration and Cache Infrastructure

#### 1.1 Add Configuration Options
**File**: `server_config.go` or where `ServerConfig` is defined

```go
type ServerConfig struct {
    // ... existing fields ...

    // Offline mode support
    EnableOfflineMode bool
    UpstreamEnabled   bool
    LsRefsCacheTTL    time.Duration
    LsRefsCachePath   string
}
```

#### 1.2 Create ls-refs Cache Manager
**File**: `ls_refs_cache.go` (new)

Implement:
- In-memory cache with mutex protection
- Disk persistence (JSON or protobuf format)
- TTL checking
- Atomic updates

**File format** (JSON example):
```json
{
  "github.com/user/repo": {
    "timestamp": "2025-11-06T10:30:00Z",
    "upstream_url": "https://github.com/user/repo",
    "refs": {
      "refs/heads/main": "abc123...",
      "refs/heads/feature": "def456...",
      "refs/tags/v1.0.0": "789abc..."
    },
    "symrefs": {
      "HEAD": "refs/heads/main"
    },
    "raw_response": "base64-encoded-protocol-response"
  }
}
```

#### 1.3 Initialize Cache on Server Start
**File**: `http_proxy_server.go`

In `StartServer()` or similar:
```go
lsRefsCache, err := NewLsRefsCache(config.LsRefsCachePath)
if err != nil {
    return fmt.Errorf("failed to initialize ls-refs cache: %w", err)
}
if err := lsRefsCache.LoadFromDisk(); err != nil {
    log.Printf("Warning: could not load ls-refs cache: %v", err)
}
```

### Phase 2: Upstream Interaction Changes

#### 2.1 Modify `lsRefsUpstream`
**File**: `managed_repository.go:129-170`

Add caching after successful upstream response:

```go
func (repo *managedRepository) lsRefsUpstream(command *gitprotocolio.ProtocolV2Command) (...) {
    // Check if upstream is disabled (test mode)
    if !repo.config.UpstreamEnabled {
        return nil, status.Error(codes.Unavailable, "upstream disabled for testing")
    }

    // ... existing upstream call ...

    // On success, cache the response
    if repo.config.EnableOfflineMode {
        entry := &LsRefsCacheEntry{
            RepoPath:    repo.localDiskPath,
            Refs:        refs,  // parsed from response
            SymRefs:     symrefs,
            Timestamp:   time.Now(),
            RawResponse: rawResponse,
            UpstreamURL: repo.upstreamURL.String(),
        }
        if err := lsRefsCache.Set(repo.localDiskPath, entry); err != nil {
            log.Printf("Warning: failed to cache ls-refs: %v", err)
        }
    }

    return refs, rawResponse, nil
}
```

#### 2.2 Add Fallback Method
**File**: `managed_repository.go` (new method)

```go
func (repo *managedRepository) lsRefsFromCache() (map[string]string, []byte, error) {
    if !repo.config.EnableOfflineMode {
        return nil, nil, status.Error(codes.Unavailable, "offline mode disabled")
    }

    entry, found := lsRefsCache.Get(repo.localDiskPath)
    if !found {
        return nil, nil, status.Error(codes.NotFound, "no cached ls-refs available")
    }

    // Check staleness
    isStale := lsRefsCache.IsStale(entry, repo.config.LsRefsCacheTTL)

    // Optionally add warning to response
    if isStale {
        log.Printf("Warning: serving stale ls-refs for %s (age: %v)",
            repo.localDiskPath, time.Since(entry.Timestamp))
    }

    return entry.Refs, entry.RawResponse, nil
}
```

#### 2.3 Update ls-refs Handler
**File**: `git_protocol_v2_handler.go:54-83`

Modify the ls-refs handling:

```go
case "ls-refs":
    var refs map[string]string
    var rawResponse []byte
    var err error

    // Try upstream first
    refs, rawResponse, err = repo.lsRefsUpstream(command)

    // If upstream fails, try cache fallback
    if err != nil && repo.config.EnableOfflineMode {
        log.Printf("Upstream ls-refs failed, attempting cache fallback: %v", err)
        refs, rawResponse, err = repo.lsRefsFromCache()
        if err == nil {
            // Successfully served from cache
            repo.config.RequestLogger(req, "ls-refs", "cache-fallback", ...)
        }
    }

    if err != nil {
        return err // No fallback available
    }

    // ... rest of existing logic ...
```

### Phase 3: Metrics and Observability

#### 3.1 Add Metrics
**File**: `reporting.go` or new `metrics.go`

Add counters/gauges:
```go
var (
    lsRefsCacheHits   = /* counter */
    lsRefsCacheMisses = /* counter */
    lsRefsServedStale = /* counter */
    upstreamAvailable = /* gauge: 0 or 1 */
)
```

#### 3.2 Update Health Check
**File**: `health_check.go` (if exists) or `http_proxy_server.go`

Add to health check response:
```json
{
  "status": "healthy",
  "upstream_status": "unavailable",
  "offline_mode": "active",
  "cached_repos": 42,
  "cache_stats": {
    "hits": 150,
    "misses": 3,
    "stale_serves": 12
  }
}
```

### Phase 4: Integration Testing

#### 4.1 Test Helper: Disable Upstream
**File**: `testing/test_helpers.go` or similar

```go
func NewTestServerWithoutUpstream(t *testing.T) *httpProxyServer {
    config := &ServerConfig{
        // ... standard test config ...
        EnableOfflineMode: true,
        UpstreamEnabled:   false,  // Key: disable upstream
        LsRefsCacheTTL:    5 * time.Minute,
    }
    return newServer(config)
}
```

#### 4.2 Test: Offline Mode with Warm Cache
**File**: `testing/offline_integration_test.go` (new)

```go
func TestLsRefsOfflineWithCache(t *testing.T) {
    server := NewTestServer(t)

    // Step 1: Populate cache with real upstream
    client := git.NewClient(server.URL)
    refs1, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)

    // Step 2: Disable upstream
    server.config.UpstreamEnabled = false

    // Step 3: Verify cache serves refs
    refs2, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)
    assert.Equal(t, refs1, refs2, "cached refs should match")
}
```

#### 4.3 Test: Offline Mode with Cold Cache
**File**: `testing/offline_integration_test.go`

```go
func TestLsRefsOfflineWithoutCache(t *testing.T) {
    server := NewTestServerWithoutUpstream(t)

    client := git.NewClient(server.URL)
    _, err := client.LsRefs("github.com/user/repo")

    // Should fail: no cache, no upstream
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "no cached ls-refs available")
}
```

#### 4.4 Test: Stale Cache Serving
**File**: `testing/offline_integration_test.go`

```go
func TestLsRefsStaleCache(t *testing.T) {
    server := NewTestServer(t)
    server.config.LsRefsCacheTTL = 1 * time.Second

    // Populate cache
    client := git.NewClient(server.URL)
    _, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)

    // Wait for cache to become stale
    time.Sleep(2 * time.Second)

    // Disable upstream
    server.config.UpstreamEnabled = false

    // Should still serve from stale cache
    _, err = client.LsRefs("github.com/user/repo")
    require.NoError(t, err)

    // Verify metrics show stale serve
    assert.Equal(t, 1, server.metrics.LsRefsServedStale)
}
```

#### 4.5 Test: Upstream Recovery
**File**: `testing/offline_integration_test.go`

```go
func TestLsRefsUpstreamRecovery(t *testing.T) {
    server := NewTestServer(t)

    // Populate cache
    client := git.NewClient(server.URL)
    refs1, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)

    // Simulate upstream failure
    server.config.UpstreamEnabled = false
    refs2, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)
    assert.Equal(t, refs1, refs2)

    // Simulate upstream recovery
    server.config.UpstreamEnabled = true
    updateUpstreamRefs(t, "github.com/user/repo", "new-commit")

    // Should fetch fresh refs
    refs3, err := client.LsRefs("github.com/user/repo")
    require.NoError(t, err)
    assert.NotEqual(t, refs2, refs3, "refs should be updated")
}
```

### Phase 5: Documentation

#### 5.1 Update README.md
**File**: `README.md:28-31`

Replace limitation note with:

```markdown
### Offline Mode and Resilience

Goblet can now serve ls-refs requests from cache when the upstream server is unavailable:

- **Automatic fallback**: When upstream is down, Goblet serves cached ref listings
- **Configurable TTL**: Control cache freshness (default: 5 minutes)
- **Testing support**: Disable upstream connectivity for integration tests
- **Metrics**: Track cache hits, misses, and stale serves

Configure offline mode:
```go
config := &ServerConfig{
    EnableOfflineMode: true,           // Enable cache fallback
    LsRefsCacheTTL:    5 * time.Minute, // Cache freshness
    LsRefsCachePath:   "/path/to/cache",
}
```

For testing without upstream:
```go
config.UpstreamEnabled = false  // Disable all upstream calls
```
```

#### 5.2 Add Configuration Guide
**File**: `docs/CONFIGURATION.md` (if exists) or add section to README

Document all new configuration options with examples.

---

## Testing Strategy

### Unit Tests
- `ls_refs_cache_test.go`: Cache operations (Get, Set, TTL, persistence)
- `managed_repository_test.go`: Cache fallback logic
- Mock upstream responses

### Integration Tests
1. ✅ **Warm cache offline**: Upstream populated cache, then disabled
2. ✅ **Cold cache offline**: No cache, upstream disabled (should fail)
3. ✅ **Stale cache serving**: Expired cache still serves when upstream down
4. ✅ **Upstream recovery**: Cache updates when upstream comes back
5. ✅ **Concurrent access**: Multiple clients with cache fallback
6. ✅ **Cache persistence**: Server restart preserves cache

### Manual Testing
- Deploy with upstream Github down
- Verify git clone/fetch works from cache
- Monitor metrics and logs
- Test cache invalidation

---

## Rollout Strategy

### Phase 1: Feature Flag (Week 1)
- Deploy with `EnableOfflineMode: false` (disabled)
- Monitor cache population
- No behavior change

### Phase 2: Canary (Week 2)
- Enable for 10% of traffic
- Monitor error rates, cache hit ratios
- Compare latency: cache vs upstream

### Phase 3: Full Rollout (Week 3+)
- Enable for all traffic
- Update documentation
- Announce feature

---

## Risks and Mitigations

### Risk 1: Stale Cache Serving Wrong Refs
**Impact**: Clients fetch outdated commits

**Mitigation**:
- Conservative default TTL (5 minutes)
- Log warnings for stale serves
- Metric tracking for monitoring

### Risk 2: Cache Size Growth
**Impact**: Disk space exhaustion

**Mitigation**:
- LRU eviction policy
- Configurable max cache size
- Periodic cleanup job

### Risk 3: Upstream Never Recovers
**Impact**: Perpetually stale cache

**Mitigation**:
- Health check reports upstream status
- Alert on prolonged upstream unavailability
- Manual cache invalidation API

### Risk 4: Race Conditions
**Impact**: Concurrent requests corrupt cache

**Mitigation**:
- RWMutex protection for all cache operations
- Atomic file writes for disk persistence
- Integration tests for concurrency

---

## Success Metrics

1. **Availability**: Proxy remains operational during upstream outages
2. **Cache Hit Ratio**: >80% of ls-refs served from cache (eventually)
3. **Latency**: Cache-served ls-refs <10ms (vs ~100ms upstream)
4. **Error Rate**: Zero increase in client errors during upstream outages
5. **Test Coverage**: >90% for new code

---

## Future Enhancements

1. **Smart Cache Invalidation**: Webhook-based cache updates
2. **Multi-Tier Caching**: Redis/Memcached for distributed deployments
3. **Partial Offline Mode**: Serve cached refs, but fail fetch if objects missing
4. **Circuit Breaker**: Automatically detect upstream failure patterns
5. **Admin API**: Manual cache inspection and invalidation endpoints

---

## Files to Modify/Create

### New Files
- `ls_refs_cache.go`: Cache manager implementation
- `ls_refs_cache_test.go`: Unit tests
- `testing/offline_integration_test.go`: Integration tests
- `OFFLINE_MODE_PLAN.md`: This document

### Modified Files
- `server_config.go`: Add configuration options
- `managed_repository.go`: Add cache fallback methods
- `git_protocol_v2_handler.go`: Update ls-refs handling
- `http_proxy_server.go`: Initialize cache on startup
- `health_check.go`: Add cache status
- `reporting.go`: Add offline mode metrics
- `README.md`: Update documentation

---

## Timeline Estimate

- **Phase 1** (Config + Cache Infrastructure): 2-3 days
- **Phase 2** (Upstream Integration): 2-3 days
- **Phase 3** (Metrics + Observability): 1-2 days
- **Phase 4** (Integration Testing): 2-3 days
- **Phase 5** (Documentation): 1 day

**Total**: ~8-12 days for full implementation and testing
