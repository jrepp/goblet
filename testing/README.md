# Goblet Integration Tests

This directory contains comprehensive integration tests for the Goblet Git caching proxy server.

## Test Structure

The integration tests are organized into several files, each testing specific functionality:

### Test Files

1. **`integration_test.go`** - Core test infrastructure
   - Docker Compose management
   - Minio setup and teardown
   - Test environment configuration

2. **`healthcheck_integration_test.go`** - Health check tests
   - `/healthz` endpoint testing
   - Server readiness checks
   - Minio connectivity verification

3. **`fetch_integration_test.go`** - Git fetch operations
   - Basic fetch operations
   - Multiple sequential fetches
   - Protocol v2 verification
   - Fetch after upstream updates
   - Performance testing

4. **`cache_integration_test.go`** - Cache behavior
   - Cache hit/miss testing
   - Concurrent fetch consistency
   - Cache invalidation on updates
   - Multi-repository isolation

5. **`auth_integration_test.go`** - Authentication
   - Valid/invalid token handling
   - Header format validation
   - Concurrent authenticated requests
   - Unauthorized access prevention

6. **`storage_integration_test.go`** - Storage backend (S3/Minio)
   - Minio connectivity
   - Storage provider initialization
   - Bundle backup and restore
   - Upload/download operations
   - Storage health checks

## Running Tests

### Quick Tests (Unit-style, no Docker)

Run fast tests that don't require Docker:

```bash
go test -v -short ./testing
```

### Full Integration Tests (with Docker)

Run all tests including those that require Minio:

```bash
# Start Minio first
docker-compose -f docker-compose.test.yml up -d

# Run all tests
go test -v ./testing

# Clean up
docker-compose -f docker-compose.test.yml down -v
```

### Run Specific Tests

```bash
# Run only health check tests
go test -v -short ./testing -run TestHealthCheck

# Run only authentication tests
go test -v -short ./testing -run TestAuth

# Run only fetch tests
go test -v -short ./testing -run TestFetch

# Run only cache tests
go test -v -short ./testing -run TestCache

# Run storage tests (requires Docker)
go test -v ./testing -run TestStorage
go test -v ./testing -run TestMinio
```

## Test Coverage

The integration tests cover:

### ✅ Implemented Features

- **Basic Git Operations**
  - Clone/fetch through proxy
  - Git protocol v2 support
  - Multiple fetch operations
  - Upstream updates

- **Caching**
  - Cache hit behavior
  - Cache consistency with concurrent requests
  - Cache invalidation on upstream changes
  - Multi-repository isolation

- **Authentication**
  - Bearer token validation
  - Request authorization
  - Header format validation
  - Unauthorized access prevention

- **Health Checks**
  - `/healthz` endpoint
  - Server readiness
  - Minio connectivity (with Docker)

- **Storage (S3/Minio)**
  - Provider initialization
  - Upload/download operations
  - Bundle management
  - Connectivity testing

## Test Results

All short tests pass:

```
PASS: TestAuthenticationRequired
PASS: TestValidAuthentication
PASS: TestInvalidAuthentication
PASS: TestAuthenticationHeaderFormat
PASS: TestConcurrentAuthenticatedRequests
PASS: TestUnauthorizedEndpointAccess
PASS: TestCacheHitBehavior
PASS: TestCacheConsistency
PASS: TestCacheInvalidationOnUpdate
PASS: TestCacheWithDifferentRepositories
PASS: TestBasicFetchOperation
PASS: TestMultipleFetchOperations
PASS: TestFetchWithProtocolV2
PASS: TestFetchAfterUpstreamUpdate
PASS: TestHealthCheckEndpoint
PASS: TestServerReadiness
```

## Docker Compose Configuration

Two Docker Compose files are provided:

1. **`docker-compose.dev.yml`** - Development environment with full Goblet server
2. **`docker-compose.test.yml`** - Minimal test environment with just Minio

The test suite uses `docker-compose.test.yml` which provides:
- Minio S3-compatible storage
- Automatic bucket creation
- Network isolation
- Easy cleanup

## Environment Variables

The integration tests use these defaults:

- **Minio Endpoint**: `localhost:9000`
- **Minio Access Key**: `minioadmin`
- **Minio Secret Key**: `minioadmin`
- **Test Bucket**: `goblet-test`

## CI/CD Integration

To integrate with CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run Integration Tests
  run: |
    docker-compose -f docker-compose.test.yml up -d
    sleep 10  # Wait for Minio to be ready
    go test -v ./testing
    docker-compose -f docker-compose.test.yml down -v
```

## Troubleshooting

### Tests Timeout

If tests timeout, increase the timeout:

```bash
go test -v -timeout 5m ./testing
```

### Port Already in Use

If port 9000 is already in use, modify `docker-compose.test.yml` to use different ports.

### Docker Not Available

If Docker is not available, tests will automatically skip with:

```
SKIP: Skipping integration test in short mode
```

Run with `-short` flag to skip Docker-dependent tests:

```bash
go test -v -short ./testing
```

## Contributing

When adding new integration tests:

1. Add test to appropriate file based on functionality
2. Use `testing.Short()` to skip Docker-dependent tests
3. Always clean up resources (use `defer`)
4. Add clear logging with `t.Logf()` for debugging
5. Update this README with new test coverage
