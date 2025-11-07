# Offline Mode Verification Agent

Verify offline mode functionality and resilience features.

## Core Capabilities

Goblet automatically falls back to local cache when upstream is unavailable:
- ls-refs requests served from local git repository
- Graceful degradation during outages
- Thread-safe configuration with atomic operations
- Staleness warnings (>5 minutes old refs)

## Verification Workflow

1. **Test automatic fallback**
   ```bash
   # Integration test with warm cache
   go test ./testing -v -run TestOfflineModeWithWarmCache

   # Upstream failure scenarios
   go test ./testing -v -run TestUpstreamFailureFallback
   ```

2. **Test thread safety**
   ```bash
   # Race detection for concurrent access
   go test -race ./testing -run TestConcurrentOfflineRequests
   go test -race ./testing -run "Offline"
   ```

3. **Verify configuration**
   - Default: `UpstreamEnabled` = true with auto-fallback
   - Testing: Use `SetUpstreamEnabled(&false)` to disable upstream entirely
   - Thread-safe: Uses atomic operations for concurrent access

## Expected Behaviors

**Normal operation** (upstream available):
- Forwards ls-refs to upstream
- Caches response locally
- Serves fetches from cache when possible

**Upstream failure** (network down):
- Detects failure on ls-refs
- Reads refs from local git repo
- Logs fallback event
- Serves cached refs to client

**Staleness warnings**:
- Logs warning if refs >5 minutes old
- Format: "Warning: serving stale ls-refs for /path (last update: Xm ago)"

## Monitoring Log Patterns

```bash
# Fallback events
"Upstream ls-refs failed (connection refused), attempting local fallback"

# Stale cache warnings
"Warning: serving stale ls-refs for /cache/path (last update: 10m ago)"
```

## Limitations

- Fetch operations for uncached objects still fail (expected)
- Cold cache (no prior fetches) will error when upstream down (expected)
- Only ls-refs can be served offline, not fetch with new objects

## Key Configuration

```go
// Production (default) - auto-fallback enabled
config := &goblet.ServerConfig{
    LocalDiskCacheRoot: "/path/to/cache",
    // UpstreamEnabled defaults to true
}

// Testing - disable upstream entirely
falseValue := false
config.SetUpstreamEnabled(&falseValue)
```

## Test Coverage

- 4 integration tests: End-to-end offline scenarios
- 8 unit tests: Edge cases, concurrency, filtering
- Thread safety verified with race detector
- See `testing/TEST_COVERAGE.md` for details

## Related Documentation

- `README.md` - Offline mode features and configuration
- `testing/TEST_COVERAGE.md` - Test details
- `testing/README.md` - Test infrastructure
