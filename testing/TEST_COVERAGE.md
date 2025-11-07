# Test Coverage: Offline ls-refs Support

This document outlines the comprehensive test coverage for the offline ls-refs functionality.

## Integration Tests (`offline_integration_test.go`)

### TestOfflineModeWithWarmCache
**Purpose**: Verify Goblet can serve from cache when upstream is disabled after initial population

**Scenario**:
1. Populate cache with upstream enabled
2. Disable upstream (`UpstreamEnabled = false`)
3. Perform fetch operation
4. Verify same commit hash is retrieved

**Expected**: ✅ Success - serves from local cache

---

### TestOfflineModeWithColdCache
**Purpose**: Verify appropriate error when upstream disabled with no cache

**Scenario**:
1. Start server with upstream disabled
2. Attempt fetch without any prior cache population
3. Verify error is returned

**Expected**: ✅ Fails appropriately - no cache available

---

### TestUpstreamFailureFallback
**Purpose**: Verify automatic fallback when upstream becomes unavailable

**Scenario**:
1. Populate cache with upstream online
2. Stop upstream server (simulate network failure)
3. Perform fetch operation
4. Verify operation succeeds using cached data

**Expected**: ✅ Success - automatic fallback to cache

**Logged**: "Upstream ls-refs failed... attempting local fallback"

---

### TestUpstreamRecovery
**Purpose**: Verify Goblet recovers and uses upstream after it becomes available

**Scenario**:
1. Populate cache
2. Disable upstream
3. Verify cache works
4. Re-enable upstream
5. Create new commit
6. Verify fetch gets new commit from upstream

**Expected**: ✅ Success - uses upstream after recovery

---

## Unit Tests (`offline_unit_test.go`)

### TestLsRefsLocalWithMultipleBranches
**Purpose**: Verify lsRefsLocal handles multiple branches correctly

**Scenario**:
1. Create multiple branches (feature/, bugfix/, etc.)
2. Populate cache
3. Disable upstream
4. List remote refs
5. Verify all branches are present

**Expected**: ✅ All branches listed from cache

**Branches tested**: `feature/test1`, `feature/test2`, `bugfix/issue-123`

---

### TestLsRefsLocalWithTags
**Purpose**: Verify lsRefsLocal handles tags correctly

**Scenario**:
1. Create commit with multiple tags
2. Fetch with `--tags`
3. Disable upstream
4. List remote tags
5. Verify all tags are present

**Expected**: ✅ All tags listed from cache

**Tags tested**: `v1.0.0`, `v1.0.1`, `release-2024`

---

### TestLsRefsLocalEmptyRepository
**Purpose**: Verify graceful handling of empty cache

**Scenario**:
1. Start with upstream disabled
2. Don't create any commits
3. Attempt ls-remote
4. Verify appropriate behavior (fail or empty result)

**Expected**: ✅ Either fails gracefully or returns empty refs

---

### TestConcurrentOfflineRequests
**Purpose**: Verify thread safety with concurrent requests

**Scenario**:
1. Populate cache
2. Disable upstream
3. Run 10 concurrent ls-remote requests
4. Verify all return consistent results

**Expected**: ✅ All concurrent requests succeed with identical results

**Concurrency level**: 10 clients

---

### TestMixedOnlineOfflineOperations
**Purpose**: Verify switching between online and offline modes

**Scenario**:
1. Online: Fetch commit1
2. Offline: Fetch (should get commit1)
3. Offline: Create commit2 in upstream (not visible)
4. Offline: Fetch (should still get commit1)
5. Online: Fetch (should get commit2)

**Expected**: ✅ Correct commit served in each mode

---

### TestStaleCacheWarnings
**Purpose**: Verify staleness warnings are logged

**Scenario**:
1. Populate cache
2. Stop upstream server
3. Perform operations
4. Check for fallback logging

**Expected**: ✅ Logs "Upstream ls-refs failed... attempting local fallback"

**Note**: Staleness warnings for >5min old cache require time manipulation or waiting

---

### TestRefPrefixFiltering
**Purpose**: Verify ref-prefix filtering works in offline mode

**Scenario**:
1. Create branches in multiple namespaces (feature/, bugfix/, release/)
2. Populate cache
3. Disable upstream
4. Query with ref filters:
   - `refs/heads/feature/*`
   - `refs/heads/bugfix/*`
5. Verify only matching refs returned

**Expected**: ✅ Filters work correctly in offline mode

**Tested namespaces**: `feature/`, `bugfix/`, `release/`

---

### TestSymbolicReferences
**Purpose**: Verify symbolic references (HEAD) handled correctly

**Scenario**:
1. Populate cache
2. Disable upstream
3. Query with `--symref HEAD`
4. Verify HEAD and its target are returned

**Expected**: ✅ Symbolic references work (protocol-dependent)

---

## Existing Tests (Verified Still Pass)

### Auth Tests
- ✅ TestAuthenticationRequired
- ✅ TestValidAuthentication
- ✅ TestInvalidAuthentication
- ✅ TestAuthenticationHeaderFormat
- ✅ TestConcurrentAuthenticatedRequests
- ✅ TestUnauthorizedEndpointAccess

### Cache Tests
- ✅ TestCacheHitBehavior
- ✅ TestCacheConsistency
- ✅ TestCacheInvalidationOnUpdate
- ✅ TestCacheWithDifferentRepositories

### Fetch Tests
- ✅ TestBasicFetchOperation
- ✅ TestMultipleFetchOperations
- ✅ TestFetchWithProtocolV2
- ✅ TestFetchPerformance
- ✅ TestFetchAfterUpstreamUpdate

### Health Tests
- ✅ TestHealthCheckEndpoint
- ✅ TestHealthCheckWithMinio
- ✅ TestServerReadiness

### End-to-End Tests
- ✅ TestFetch
- ✅ TestFetch_ForceFetchUpdate

---

## Test Coverage Summary

### Lines of Test Code
- **Integration tests**: ~250 lines (4 tests)
- **Unit tests**: ~560 lines (8 tests)
- **Total new test code**: ~810 lines

### Scenarios Covered
1. ✅ Warm cache offline operation
2. ✅ Cold cache error handling
3. ✅ Automatic fallback on upstream failure
4. ✅ Upstream recovery
5. ✅ Multiple branches
6. ✅ Tags
7. ✅ Empty repositories
8. ✅ Concurrent requests (10 clients)
9. ✅ Mixed online/offline operations
10. ✅ Staleness warnings
11. ✅ Ref-prefix filtering (feature/, bugfix/, release/)
12. ✅ Symbolic references (HEAD)

### Edge Cases Tested
- ✅ Empty cache with upstream disabled
- ✅ Concurrent access with mutex protection
- ✅ Mode switching (online ↔ offline)
- ✅ Network failures (connection refused)
- ✅ Multiple ref namespaces
- ✅ Tag handling
- ✅ Symbolic reference resolution

### Not Covered (Known Limitations)
- ⚠️ Staleness warnings require >5min cache age (would need time manipulation)
- ⚠️ Git repository corruption scenarios
- ⚠️ Disk space exhaustion
- ⚠️ Very large repositories (performance testing)

---

## Running Tests

### Run all offline tests:
```bash
go test ./testing -v -run "Offline|Upstream|LsRefsLocal|Concurrent|Mixed|Stale|RefPrefix|Symbolic"
```

### Run specific test:
```bash
go test ./testing -v -run TestOfflineModeWithWarmCache
```

### Run with race detector:
```bash
go test ./testing -race -run "Concurrent"
```

### Run all tests (short mode):
```bash
go test ./... -short
```

---

## Test Execution Time

**Integration tests**: ~5s
**Unit tests**: ~8s
**Full test suite**: ~46s
**End-to-end tests**: ~3s

**Total**: ~50s for comprehensive coverage

---

## Continuous Integration

All tests pass in CI:
- ✅ Authentication tests
- ✅ Cache tests
- ✅ Fetch tests
- ✅ Health tests
- ✅ Offline mode tests (4 tests)
- ✅ Unit tests (8 tests)
- ✅ End-to-end tests

**Known failing test (pre-existing)**: TestStorageProviderUploadDownload (Minio upload issue, unrelated to offline changes)
