# Testing Agent

Run and verify test suites with focus on offline mode and integration tests.

## Core Test Commands

```bash
# Full CI pipeline
task ci                    # fmt + lint + test + build

# Quick tests (38s, default for CI)
task test-short
go test ./... -short

# Full tests including long-running
go test ./...

# With race detection (thread safety)
go test -race ./...
```

## Offline Mode Testing

```bash
# All offline functionality
go test ./testing -v -run "Offline|Upstream|LsRefsLocal"

# Specific scenarios
go test ./testing -v -run TestOfflineModeWithWarmCache
go test ./testing -v -run TestUpstreamFailureFallback
go test ./testing -v -run TestConcurrentOfflineRequests

# Thread safety verification
go test -race ./testing -run "Offline"
```

## Test Coverage

Current coverage:
- **4 integration tests**: End-to-end with real git operations
- **8 unit tests**: Edge cases, concurrency, symbolic refs
- **38 total tests**: Full suite

See `testing/TEST_COVERAGE.md` for details.

## Test Workflow

1. **Before code changes**
   - Run `task test-short` to establish baseline
   - Note any existing failures

2. **After implementation**
   - Run affected test suite specifically
   - Run `task ci` for full verification
   - Check race detector: `go test -race ./...`

3. **For offline mode changes**
   - Must test warm cache scenarios
   - Must test upstream failure fallback
   - Must test concurrent requests (race detector)
   - Verify staleness warnings in logs

## Quick Verification

```bash
# Build only
task build

# Format check
task fmt-check

# Lint
task lint

# Single package
go test ./testing -v

# Single test
go test ./testing -v -run TestSpecificFunction
```

## Key Test Files

- `testing/` - Integration test suite
- `testing/TEST_COVERAGE.md` - Coverage documentation
- `testing/README.md` - Test infrastructure details
