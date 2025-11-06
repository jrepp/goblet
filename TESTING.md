# Testing Guide

This document describes the testing strategy and how to run different types of tests for the Goblet Git cache proxy.

## Test Organization

Tests are organized into two categories:

### 1. Unit Tests (No Docker Required)
Unit tests run with the `-short` flag and skip any tests requiring Docker containers. These are safe to run in CI environments without Docker.

**Command:**
```bash
task test-unit
```

**What gets tested:**
- Pure Go unit tests
- Logic and algorithm tests
- Tests that don't require external services

**Coverage:** `coverage-unit.out`

### 2. Integration Tests (Require Docker)
Integration tests require Docker containers to be running and test the full system end-to-end.

#### Go Integration Tests
Tests in `./testing/...` that require Docker Compose test environment.

**Command:**
```bash
task test-integration-go
```

**What gets tested:**
- Git fetch operations
- Cache functionality
- Storage backend integration
- Authentication flows
- Health checks

**Coverage:** `coverage-integration.out`

#### OIDC Integration Tests
End-to-end tests for OIDC authentication using Dex IdP.

**Command:**
```bash
task test-integration-oidc
# or
task test-oidc
```

**What gets tested:**
- Service health
- Token generation and retrieval
- Authentication flows (401, 400, 200 responses)
- Git operations (ls-remote, clone)
- Metrics collection
- Server logs

**Details:** 12 integration tests covering full OIDC workflow

#### All Integration Tests
Run both Go and OIDC integration tests.

**Command:**
```bash
task test-integration-all
# or
task test-integration
```

## Quick Reference

### Development Workflow

```bash
# Quick feedback loop (no Docker)
task test-unit              # Run unit tests
task test-short             # Run unit tests (fast, no race detector)

# Pre-commit checks (no Docker)
task pre-commit             # fmt + tidy + lint + unit tests

# Full local testing (requires Docker)
task test-integration       # All integration tests
task int                    # Full integration cycle (clean + build + test)
```

### CI/CD Workflows

```bash
# Fast CI (no Docker - use in pull request checks)
task ci                     # fmt-check + lint + unit tests + build

# Quick checks (no Docker)
task ci-quick               # fmt-check + lint + unit tests (fastest)

# Full CI (requires Docker - use in post-merge or nightly)
task ci-full                # unit tests + build-all + integration tests

# Complete local CI (simulates GitHub Actions)
task ci-local               # install-tools + deps + ci-full
```

### Specific Test Types

```bash
# Unit tests only (no Docker)
task test-unit              # With race detector
task test-short             # Without race detector (faster)
task test                   # Alias for test-unit

# Integration tests (require Docker)
task test-integration-go    # Go integration tests
task test-integration-oidc  # OIDC integration tests
task test-integration-all   # All integration tests
task test-integration       # Alias for test-integration-all

# Parallel testing (require Docker)
task test-parallel          # Run Go integration tests in parallel

# OIDC-specific
task test-oidc             # Run OIDC integration tests
task validate-token        # Validate token mount
task get-token             # Get bearer token
```

## Test Categories Matrix

| Task | Docker Required? | CI Safe? | Coverage File | Duration |
|------|-----------------|----------|---------------|----------|
| `test-unit` | ❌ No | ✅ Yes | `coverage-unit.out` | ~5s |
| `test-short` | ❌ No | ✅ Yes | None | ~3s |
| `test-integration-go` | ✅ Yes | ❌ No | `coverage-integration.out` | ~30s |
| `test-integration-oidc` | ✅ Yes | ❌ No | None | ~15s |
| `test-integration-all` | ✅ Yes | ❌ No | Mixed | ~45s |
| `test-parallel` | ✅ Yes | ❌ No | None | ~20s |

## CI/CD Integration

### GitHub Actions Workflow

The project uses a parallelized GitHub Actions workflow (`.github/workflows/ci.yml`) that extracts each `task ci` step into separate jobs:

**Parallel CI Jobs (No Docker):**
- `format-check` - Code formatting validation with goimports
- `tidy-check` - Go module tidiness check
- `lint` - Static analysis with golangci-lint and staticcheck
- `test-unit` - Unit tests with race detector and coverage
- `build` - Build for current platform
- `build-multi` - Multi-platform builds (matrix strategy)

**Status Check:**
- `ci-complete` - Depends on all parallel jobs, provides single PR status

**Integration Tests (Docker Required):**
- `integration-test` - Only runs on main branch or with `run-integration-tests` label

**Local Equivalent:**
```bash
# Run same checks locally (sequential)
task ci

# Run full CI with integration tests
task ci-full
```

### GitLab CI Example

```yaml
test:unit:
  stage: test
  script:
    - task test-unit

test:integration:
  stage: test
  services:
    - docker:dind
  script:
    - task test-integration
```

## Coverage Reports

Generate and view coverage:

```bash
# Generate coverage HTML report
task coverage

# View unit test coverage only
go tool cover -html=coverage-unit.out

# View integration test coverage only
go tool cover -html=coverage-integration.out
```

## Test Environment Setup

### For Unit Tests
No setup required - unit tests run without external dependencies.

### For Integration Tests

#### Docker Compose Test Environment
```bash
# Start test environment
task docker-test-up

# Run tests
task test-integration-go

# Stop test environment
task docker-test-down

# View logs
task docker-test-logs
```

#### Docker Compose Dev Environment (for OIDC tests)
```bash
# Start dev environment
task up

# Run OIDC tests
task test-oidc

# Stop dev environment
task down

# View logs
task docker-logs
```

## Troubleshooting

### Unit Tests Failing
```bash
# Run with verbose output
go test -short -v ./...

# Run specific test
go test -short -v ./... -run TestName
```

### Integration Tests Failing
```bash
# Check Docker containers are running
docker ps

# View service logs
task docker-test-logs

# Clean and restart
task docker-test-down
task docker-test-up
```

### OIDC Tests Failing
```bash
# Validate token is accessible
task validate-token

# Check dev services
docker-compose -f docker-compose.dev.yml ps

# View server logs
docker logs goblet-server-dev
```

## Writing New Tests

### Unit Tests
Mark tests that require Docker with build tags or skip in short mode:

```go
func TestSomething(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // Test code that requires Docker
}
```

### Integration Tests
Place integration tests in `./testing/...` directory:

```go
// testing/my_integration_test.go
package testing

func TestIntegration(t *testing.T) {
    // Full integration test with Docker
}
```

### OIDC Tests
Add new tests to `Taskfile.yml` under `test-oidc-*` tasks following the existing pattern.

## Best Practices

1. **Always run unit tests before committing:**
   ```bash
   task pre-commit
   ```

2. **Run integration tests before pushing:**
   ```bash
   task test-integration
   ```

3. **Use parallel testing for faster feedback:**
   ```bash
   task test-parallel
   ```

4. **Keep unit tests fast** (< 1s per test)

5. **Mark integration tests clearly** with `-short` skip or build tags

6. **Use table-driven tests** for multiple scenarios

7. **Clean up test resources** in defer statements

## Summary

- **No Docker?** Use `task test-unit` or `task ci`
- **Have Docker?** Use `task test-integration` or `task ci-full`
- **Quick check?** Use `task ci-quick`
- **Pre-commit?** Use `task pre-commit`
- **OIDC testing?** Use `task test-oidc`
