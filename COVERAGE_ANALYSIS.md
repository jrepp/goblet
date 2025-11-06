# Code Coverage Analysis Report

**Date:** November 6, 2025
**Project:** Goblet Git Caching Proxy
**Current Coverage:** 84% (testing package only), 0% (core packages)

## Executive Summary

The current test suite has excellent coverage of the **testing infrastructure** (84%), but the **core application code** has 0% coverage when running with the `-short` flag. This is expected behavior as our integration tests exercise the core code, but we need additional **unit tests** to improve coverage and catch regressions early.

---

## Coverage by Package

| Package | Current Coverage | Lines | Priority | Impact |
|---------|------------------|-------|----------|--------|
| `testing` | 84.0% | ~500 | Low | Integration tests |
| `goblet` (core) | 0.0% | ~800 | **HIGH** | Core functionality |
| `storage` | 0.0% | ~300 | **HIGH** | Storage backends |
| `google` | 0.0% | ~400 | Medium | Google Cloud integration |
| `goblet-server` | 0.0% | ~200 | Low | Main entry point |

---

## Top 10 Areas for Coverage Improvement

Ranked by **probability of coverage increase** and **testing ROI**:

### 1. **Health Check System** (Highest Priority)
**File:** `health.go`
**Lines:** ~155
**Current Coverage:** 0%
**Potential Coverage:** 90%+

**Functions to Test:**
- `NewHealthChecker()` - Constructor
- `Check()` - Main health check logic
- `checkStorage()` - Storage connectivity check
- `checkCache()` - Cache health check
- `ServeHTTP()` - HTTP handler for /healthz

**Why High Priority:**
- **New code** just created, not yet tested
- **Critical for production** monitoring
- **Easy to test** - minimal dependencies
- **High ROI** - complete coverage achievable
- **Low complexity** - straightforward logic

**Testing Strategy:**
- Unit tests with mock storage provider
- Test all health states (healthy, degraded, unhealthy)
- Test timeout scenarios
- Test both simple and detailed endpoints

---

### 2. **HTTP Proxy Server Core**
**File:** `http_proxy_server.go`
**Lines:** ~150
**Current Coverage:** 0%
**Potential Coverage:** 75%+

**Functions to Test:**
- `ServeHTTP()` - Main HTTP handler
- `infoRefsHandler()` - Git info/refs endpoint
- `uploadPackHandler()` - Git upload-pack endpoint
- `parseAllCommands()` - Command parsing

**Why High Priority:**
- **Core functionality** - all requests go through here
- **Well-defined** - HTTP handlers are testable
- **Catches regressions** - protocol compliance
- **Medium complexity** - requires mock setup

**Testing Strategy:**
- Unit tests with httptest.ResponseRecorder
- Test all HTTP paths (/info/refs, /git-upload-pack, /git-receive-pack)
- Test error conditions (auth failures, protocol errors)
- Test gzip decompression

---

### 3. **Storage Provider System**
**File:** `storage/storage.go`, `storage/s3.go`, `storage/gcs.go`
**Lines:** ~300
**Current Coverage:** 0%
**Potential Coverage:** 80%+

**Functions to Test:**
- `NewProvider()` - Provider factory
- `Writer()` / `Reader()` - I/O operations
- `List()` - Object listing
- `Delete()` - Object deletion
- S3-specific: Connection handling, error cases
- GCS-specific: Authentication, bucket operations

**Why High Priority:**
- **Critical for backups** - data persistence
- **External dependencies** - needs mocking
- **Error-prone** - network, auth, timeouts
- **High value** - prevents data loss

**Testing Strategy:**
- Unit tests with mock storage
- Integration tests with Minio (already have some)
- Test error conditions (network failures, auth errors)
- Test edge cases (large files, timeouts)

---

### 4. **Managed Repository Operations**
**File:** `managed_repository.go`
**Lines:** ~350
**Current Coverage:** 0%
**Potential Coverage:** 60%+

**Functions to Test:**
- `openManagedRepository()` - Repository initialization
- `getManagedRepo()` - Repository retrieval
- `lsRefsUpstream()` - Ref listing
- `fetchUpstream()` - Upstream fetching
- `serveFetchLocal()` - Local serving
- `hasAnyUpdate()` / `hasAllWants()` - Cache logic

**Why High Priority:**
- **Core caching logic** - most complex code
- **Concurrency** - sync.Map operations
- **Git operations** - subprocess handling
- **Moderate complexity** - needs git binary

**Testing Strategy:**
- Unit tests with mock git operations
- Test repository lifecycle
- Test concurrent access
- Test cache hit/miss scenarios

**Challenges:**
- Requires git binary
- Complex state management
- Subprocess execution

---

### 5. **Git Protocol V2 Handler**
**File:** `git_protocol_v2_handler.go`
**Lines:** ~180
**Current Coverage:** 0%
**Potential Coverage:** 70%+

**Functions to Test:**
- `handleV2Command()` - Command dispatcher
- `parseLsRefsResponse()` - Response parsing
- `parseFetchWants()` - Want parsing

**Why High Priority:**
- **Protocol compliance** - Git interoperability
- **Well-defined** - Git protocol spec
- **Parser logic** - bug-prone
- **Moderate complexity** - binary protocol

**Testing Strategy:**
- Unit tests with sample protocol data
- Test valid/invalid protocol sequences
- Test all command types (ls-refs, fetch)
- Test error handling

---

### 6. **IO Operations**
**File:** `io.go`
**Lines:** ~80
**Current Coverage:** 0%
**Potential Coverage:** 95%+

**Functions to Test:**
- `writePacket()` - Packet writing
- `writeResp()` / `writeError()` - Response writing
- `copyRequestChunk()` / `copyResponseChunk()` - Chunk copying

**Why Medium Priority:**
- **Simple logic** - straightforward I/O
- **High testability** - pure functions
- **Low complexity** - minimal dependencies
- **Quick wins** - fast to test

**Testing Strategy:**
- Unit tests with buffers
- Test all packet types
- Test error conditions
- Test data integrity

---

### 7. **Reporting & Metrics**
**File:** `reporting.go`
**Lines:** ~120
**Current Coverage:** 0%
**Potential Coverage:** 80%+

**Functions to Test:**
- `logHTTPRequest()` - Request logging
- `httpErrorReporter` - Error reporting
- Metrics recording

**Why Medium Priority:**
- **Observability** - debugging aid
- **Well-isolated** - minimal coupling
- **Moderate value** - not critical path
- **Easy to test** - straightforward logic

**Testing Strategy:**
- Unit tests with mock loggers
- Test all error types
- Test metrics recording
- Test HTTP status code mapping

---

### 8. **Backup System**
**File:** `google/backup.go`
**Lines:** ~280
**Current Coverage:** 0%
**Potential Coverage:** 50%+

**Functions to Test:**
- `RunBackupProcess()` - Main backup loop
- `backupManagedRepo()` - Repository backup
- `recoverFromBackup()` - Restore logic
- `gcBundle()` - Garbage collection

**Why Lower Priority:**
- **Google Cloud specific** - not always used
- **Complex setup** - requires storage
- **Long-running** - background process
- **Already tested** - via integration

**Testing Strategy:**
- Unit tests with mocks
- Test backup/restore cycle
- Test error recovery
- Integration tests with storage

---

### 9. **Google Cloud Hooks**
**File:** `google/hooks.go`
**Lines:** ~180
**Current Coverage:** 0%
**Potential Coverage:** 60%+

**Functions to Test:**
- `NewRequestAuthorizer()` - Auth initialization
- `CanonicalizeURL()` - URL canonicalization
- Authorization methods (cookie, token, header)

**Why Lower Priority:**
- **Google Cloud specific** - not always used
- **Complex dependencies** - OAuth, GCP
- **Alternative implementations** - custom auth possible
- **Moderate value** - specific use case

**Testing Strategy:**
- Unit tests with mock OAuth
- Test URL canonicalization
- Test auth header parsing
- Test error conditions

---

### 10. **Main Server Startup**
**File:** `goblet-server/main.go`
**Lines:** ~210
**Current Coverage:** 0%
**Potential Coverage:** 30%+

**Functions to Test:**
- Configuration parsing
- Flag validation
- Component initialization
- Signal handling

**Why Lowest Priority:**
- **Entry point** - hard to unit test
- **Integration tested** - via docker-compose
- **Complex dependencies** - full stack
- **Low ROI** - better as E2E tests

**Testing Strategy:**
- Integration tests (already have)
- Configuration validation tests
- Smoke tests

---

## Testing Strategy Recommendations

### Quick Wins (High ROI, Low Effort)

1. **Health Check Tests** - 2-3 hours
2. **IO Operations Tests** - 1-2 hours
3. **Storage Provider Unit Tests** - 3-4 hours

**Expected Coverage Increase:** +20-25%

### Core Functionality (High ROI, Medium Effort)

4. **HTTP Proxy Server Tests** - 4-6 hours
5. **Git Protocol Handler Tests** - 3-4 hours

**Expected Coverage Increase:** +15-20%

### Advanced Coverage (Medium ROI, High Effort)

6. **Managed Repository Tests** - 6-8 hours
7. **Backup System Tests** - 4-6 hours

**Expected Coverage Increase:** +10-15%

---

## Current Test Distribution

```
Integration Tests (24 tests):
├── Health checks: 3 tests ✅
├── Git operations: 6 tests ✅
├── Cache behavior: 4 tests ✅
├── Authentication: 6 tests ✅
└── Storage: 5 tests ✅

Unit Tests (0 tests):
├── Core packages: 0 tests ❌
├── Storage: 0 tests ❌
└── Google: 0 tests ❌
```

---

## Coverage Goals

| Timeframe | Target | Focus Areas |
|-----------|--------|-------------|
| **Phase 1** (1 day) | 40% | Health, IO, HTTP handlers |
| **Phase 2** (3 days) | 60% | Storage, Git protocol, Reporting |
| **Phase 3** (1 week) | 75% | Managed repos, Backup, Advanced |

---

## Key Insights

1. **Integration tests work well** - 84% coverage of test infrastructure
2. **Core code untested** - 0% in production packages
3. **Easy wins available** - Health checks, IO operations
4. **Mock strategy needed** - Storage, Git operations require mocking
5. **Balance needed** - Unit + integration tests together

---

## Recommended Next Steps

1. ✅ **Create health check unit tests** (Top Priority #1)
2. ✅ **Create HTTP handler unit tests** (Top Priority #2)
3. ✅ **Create storage provider unit tests** (Top Priority #3)
4. Create IO operation unit tests
5. Create Git protocol handler tests
6. Add mock utilities for testing
7. Set up coverage gates in CI (minimum 60%)
8. Add coverage badge to README

---

## Appendix: Running Coverage Analysis

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage by function
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# View coverage for specific package
go test -coverprofile=coverage.out ./storage
go tool cover -func=coverage.out

# Run with coverage and tests
task test-short  # Fast unit tests
go tool cover -html=coverage.out
```

---

**Report End**

*Next Action: Implement tests for Top 3 priority areas*
