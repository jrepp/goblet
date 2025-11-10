# Offline Mode and Resilience

Goblet provides automatic fallback to cached data when upstream Git servers are unavailable, enabling continued operations during outages.

## Overview

Goblet automatically serves ls-refs requests from the local cache when the upstream server is unavailable, providing resilience during upstream outages.

## Features

- **Automatic fallback**: When upstream is down or unreachable, Goblet automatically serves cached ref listings from the local git repository
- **Graceful degradation**: Git operations continue to work with cached data during upstream outages
- **Thread-safe configuration**: Uses atomic operations for concurrent read/write access to configuration
- **Staleness tracking**: Logs warnings when serving refs older than 5 minutes, helping identify stale cache scenarios
- **Testing support**: Upstream connectivity can be disabled entirely for integration testing
- **Zero configuration**: Works out of the box - automatic fallback requires no configuration changes

## How It Works

### Normal Operation (Upstream Available)

1. Goblet forwards ls-refs requests to upstream
2. Caches the response locally
3. Serves subsequent fetch requests from cache when possible

### Upstream Failure (Network Down)

1. Goblet detects upstream failure on ls-refs request
2. Automatically reads refs from local git repository cache
3. Logs fallback event for monitoring
4. Serves refs to client from cache

### Upstream Recovery

1. Next ls-refs request attempts upstream again
2. On success, cache is updated with latest refs
3. System returns to normal operation

## Configuration

### Production Mode (Default)

By default, Goblet operates with automatic fallback enabled. No configuration needed:

```go
config := &goblet.ServerConfig{
    LocalDiskCacheRoot: "/path/to/cache",
    URLCanonializer:    canonicalizer,
    TokenSource:        tokenSource,
    RequestAuthorizer:  authorizer,
    // UpstreamEnabled defaults to true with automatic fallback
}

server := goblet.NewServer(config)
```

### Testing Mode (Disable Upstream)

For integration testing where you want to disable upstream connectivity entirely:

```go
falseValue := false
config := &goblet.ServerConfig{
    LocalDiskCacheRoot: "/path/to/cache",
    URLCanonializer:    canonicalizer,
    TokenSource:        tokenSource,
    RequestAuthorizer:  authorizer,
}
config.SetUpstreamEnabled(&falseValue)  // Thread-safe: disable all upstream calls
```

Or during server initialization:

```go
falseValue := false
ts := NewTestServer(&TestServerConfig{
    LocalDiskCacheRoot: t.TempDir(),
    UpstreamEnabled:    &falseValue,  // Start with upstream disabled
})
```

## Monitoring

### Log Events

Goblet logs important offline mode events:

```
# Fallback to local cache
Upstream ls-refs failed (connection refused), attempting local fallback for /cache/path

# Stale cache warning (>5 minutes old)
Warning: serving stale ls-refs for /cache/path (last update: 10m ago)
```

### Metrics

Monitor these Prometheus metrics:

```
# Upstream request success/failure
goblet_upstream_requests_total{status="success|failure"}

# Cache hit rate
goblet_cache_hits_total / goblet_requests_total

# Fallback events
goblet_offline_fallbacks_total
```

### Alerting

Set up alerts for:

1. **Extended offline periods** - Cache older than threshold (e.g., 1 hour)
2. **High failure rate** - Upstream failures exceeding threshold
3. **Zero cache hits** - Indicates cold cache or configuration issue

Example Prometheus alert:

```yaml
- alert: GobletExtendedOffline
  expr: time() - goblet_cache_last_update > 3600
  labels:
    severity: warning
  annotations:
    summary: "Goblet cache stale for {{ $value }} seconds"
```

## Use Cases

### CI/CD Pipelines

**Scenario:** GitHub is down during deployment window

**Behavior:**
- First fetch attempt fails upstream
- Goblet serves ls-refs from cache
- Pipeline continues with cached refs
- Fetch succeeds if objects are in cache
- Pipeline completes or degrades gracefully

### Infrastructure as Code

**Scenario:** Terraform/Ansible runs during upstream maintenance

**Benefit:**
- Module fetches continue from cache
- No blocking on upstream availability
- Automated deployments remain reliable

### Security Scanning

**Scenario:** Continuous scanning of repositories

**Benefit:**
- Scans continue during upstream issues
- Reduced dependency on external services
- Consistent scanning schedule maintained

## Limitations

### Cache Misses Still Fail

While Goblet can serve ls-refs from cache during upstream outages, **fetch operations for objects not already in the cache will still fail** if the upstream is unavailable. This is expected behavior as Goblet cannot serve content it doesn't have cached.

### Cold Cache Requirement

**Important:** The local cache must be populated before offline mode can serve requests. A cold cache (no prior fetches) will result in appropriate errors when upstream is unavailable.

**Mitigation:** Pre-populate cache by:

1. Running initial fetch during setup
2. Using warm cache from backup/restore
3. Scheduling periodic cache warming

### Stale Data

Cached refs may become stale if:

- Upstream is down for extended period
- Force pushes occur upstream
- Branches are deleted upstream

**Mitigation:**
- Monitor cache age via logs/metrics
- Set alerts for staleness thresholds
- Document acceptable staleness for your use case

## Testing

### Unit Tests

Test offline functionality:

```bash
# Run all offline-related tests
go test ./testing -v -run "Offline|Upstream|LsRefsLocal"

# Test with race detector (verifies thread safety)
go test -race ./testing -run "Offline"

# Test specific scenarios
go test ./testing -v -run TestOfflineModeWithWarmCache
go test ./testing -v -run TestUpstreamFailureFallback
go test ./testing -v -run TestConcurrentOfflineRequests
```

### Integration Tests

Test end-to-end offline scenarios:

```bash
# Run full integration test suite
task test-integration

# Run specific offline integration tests
go test ./testing -v -run TestOfflineIntegration
```

### Load Testing

Test offline behavior under load:

```bash
cd loadtest
python3 loadtest.py \
  --workers 50 \
  --requests 1000 \
  --offline-mode true
```

## Best Practices

### 1. Pre-populate Cache

Ensure cache is warm before relying on offline mode:

```bash
# Run initial fetch for critical repos
for repo in $CRITICAL_REPOS; do
  git clone --mirror http://goblet:8080/$repo
done
```

### 2. Monitor Cache Age

Track when cache was last updated:

```go
// Log cache age on each request
lastUpdate := time.Since(repo.lastUpdate)
if lastUpdate > 5*time.Minute {
    log.Warnf("Cache stale: %v old", lastUpdate)
}
```

### 3. Set Staleness Thresholds

Define acceptable cache age for your use case:

- **CI/CD:** 5-15 minutes acceptable
- **Security scanning:** 1 hour acceptable
- **Development:** Cache age less critical

### 4. Plan for Recovery

Document procedures for:

- Cache restoration from backup
- Manual cache warming
- Verifying cache integrity

### 5. Test Regularly

Include offline scenarios in testing:

```go
// Test with upstream disabled
func TestWithUpstreamDown(t *testing.T) {
    falseValue := false
    config.SetUpstreamEnabled(&falseValue)

    // Verify operations still work
    // ...
}
```

## See Also

- [Architecture Design Decisions](../architecture/design-decisions.md) - Offline mode architecture
- [Testing Guide](../../testing/TEST_COVERAGE.md) - Offline mode test coverage
- [Deployment Patterns](deployment-patterns.md) - HA and resilience patterns
- [Monitoring](monitoring.md) - Metrics and alerting
