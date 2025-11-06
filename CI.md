# Continuous Integration Guide

This project uses **GitHub Actions** for CI/CD and **Task** for local development. You can run the exact same checks locally that will run in CI.

---

## 🚀 Quick Start - Run CI Locally

### Option 1: Quick Check (Fast - 30 seconds)
Perfect for rapid feedback before committing:

```bash
task ci-quick
```

Runs:
- ✓ Format checking
- ✓ Linting
- ✓ Unit tests

---

### Option 2: Standard CI (2-3 minutes)
Same as what runs on GitHub Actions for PRs:

```bash
task ci
```

Runs:
- ✓ Format checking
- ✓ Go mod tidiness
- ✓ Linting (golangci-lint + staticcheck)
- ✓ Unit tests
- ✓ Build for current platform

---

### Option 3: Full CI Pipeline (5-8 minutes)
Complete validation including integration tests:

```bash
task ci-full
```

Runs:
- ✓ Format checking
- ✓ Go mod tidiness
- ✓ Linting
- ✓ Unit tests
- ✓ Multi-platform builds (all architectures)
- ✓ Integration tests with Docker
- ✓ End-to-end tests

---

### Option 4: Complete Local CI (10 minutes)
Exactly matches GitHub Actions workflow:

```bash
task ci-local
```

Runs:
- ✓ Tool installation
- ✓ Dependency download
- ✓ Full CI pipeline
- ✓ Everything that GitHub Actions will run

---

## 📋 Available CI Tasks

| Task | Duration | Use Case |
|------|----------|----------|
| `task ci-quick` | ~30s | Fast feedback loop |
| `task ci` | 2-3min | Standard pre-commit check |
| `task ci-full` | 5-8min | Complete validation |
| `task ci-local` | ~10min | Exact GitHub Actions simulation |
| `task pre-commit` | ~1min | Auto-fix + test before commit |

---

## 🔧 GitHub Actions Workflow

The CI pipeline runs on:
- Every push to `main`
- Every pull request to `main`

### Jobs

#### 1. **Test Job**
- Runs unit tests
- Checks formatting
- Verifies linting
- Uploads coverage to Codecov

#### 2. **Integration Test Job**
- Starts Docker services (Minio)
- Runs integration tests
- Tests with real S3-compatible storage
- Uploads integration coverage

#### 3. **Build Job** (Matrix)
- Builds for all platforms:
  - linux/amd64
  - linux/arm64
  - darwin/amd64
  - darwin/arm64
- Uploads build artifacts

#### 4. **Lint Job**
- Checks code formatting
- Verifies go.mod tidiness
- Runs golangci-lint
- Runs staticcheck

---

## 🛠️ Development Workflow

### Before Committing

```bash
# Quick check
task ci-quick

# Or auto-fix issues
task pre-commit
```

### Before Creating PR

```bash
# Run full validation
task ci-full
```

### Debugging CI Failures

If CI fails on GitHub but passes locally:

```bash
# Run exact CI environment
task ci-local

# Check specific job
task test-integration  # Integration tests
task lint              # Linting
task build-all         # Multi-platform builds
```

---

## 📊 Coverage Requirements

- **Unit tests:** Minimum 35% coverage (current: 37.4%)
- **Integration tests:** 100% pass rate (current: 24/24 ✓)
- **No flaky tests:** Zero tolerance

Coverage reports are uploaded to Codecov on every CI run.

---

## 🔍 Linting Tools

The project uses:

1. **golangci-lint** - Comprehensive linter suite
   - Configuration: `.golangci.yml`
   - Runs: ~20 linters in parallel

2. **staticcheck** - Advanced static analysis
   - Detects: bugs, performance issues, style violations

3. **gofmt** - Standard Go formatting
   - Enforced: No unformatted code accepted

4. **goimports** - Import organization
   - Auto-fixes: Import grouping and ordering

---

## 🚨 Common CI Failures and Fixes

### 1. Format Check Fails

```bash
# Fix automatically
task fmt

# Verify
task fmt-check
```

### 2. Lint Errors

```bash
# Run linters
task lint

# If issues found, fix code and rerun
```

### 3. Tests Fail

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestHealthChecker ./...

# Check test coverage
task coverage
```

### 4. Build Fails

```bash
# Try building locally
task build

# Check for missing dependencies
task deps
task tidy
```

### 5. Integration Tests Fail

```bash
# Ensure Docker is running
docker ps

# Restart test environment
task docker-test-down
task docker-test-up

# Run integration tests
task test-integration
```

---

## ⚡ Performance Tips

### Speed Up Local CI

1. **Use ci-quick for iteration**
   ```bash
   task ci-quick  # 30s instead of 5min
   ```

2. **Run only changed tests**
   ```bash
   go test -short ./path/to/changed/package
   ```

3. **Skip integration tests**
   ```bash
   task ci  # Skips Docker-based tests
   ```

4. **Parallel test execution**
   ```bash
   go test -parallel 8 ./...
   ```

---

## 🎯 CI Best Practices

### Do's ✅
- Run `task ci-quick` before every commit
- Run `task ci-full` before pushing
- Fix linting issues immediately
- Keep tests fast (<5s per test file)
- Write tests for new code
- Update coverage when adding features

### Don'ts ❌
- Don't push without running CI locally
- Don't ignore linter warnings
- Don't commit failing tests
- Don't skip test coverage checks
- Don't push unformatted code

---

## 📈 CI Metrics

Current project metrics:

| Metric | Value | Target |
|--------|-------|--------|
| Unit Test Coverage | 37.4% | 60% |
| Integration Test Pass Rate | 100% | 100% |
| Build Time (CI) | ~5min | <10min |
| Flaky Tests | 0 | 0 |
| Lint Issues | 0 | 0 |

---

## 🔗 Related Documentation

- [Testing Guide](testing/README.md) - Comprehensive test documentation
- [Integration Tests](INTEGRATION_TEST_REPORT.md) - Integration test details
- [Coverage Analysis](COVERAGE_ANALYSIS.md) - Coverage breakdown
- [Taskfile](Taskfile.yml) - All available tasks

---

## 🆘 Getting Help

### CI Pipeline Issues

1. Check GitHub Actions logs
2. Run `task ci-local` to reproduce locally
3. Review error messages in detail
4. Check [Taskfile.yml](Taskfile.yml) for task definitions

### Test Failures

1. Run tests locally: `task test-short`
2. Run with verbose output: `go test -v ./...`
3. Check test logs for details
4. Verify Docker services: `task docker-test-up`

### Linting Issues

1. Auto-fix: `task fmt`
2. Check specific issues: `task lint`
3. Review `.golangci.yml` for rules
4. Fix issues manually if needed

---

## 📝 Example CI Run

```bash
$ task ci-local
==> Running complete local CI (simulates GitHub Actions)...
task: [install-tools] Installing required tools...
✓ golangci-lint installed
✓ staticcheck installed
task: [deps] Downloading dependencies...
✓ Dependencies downloaded
task: [fmt-check] Checking code formatting...
✓ All files formatted correctly
task: [tidy-check] Checking go.mod tidiness...
✓ go.mod is tidy
task: [lint] Running linters...
✓ golangci-lint passed
✓ staticcheck passed
✓ go vet passed
task: [test-short] Running unit tests...
✓ All tests passed (0.8s)
task: [build-all] Building for all platforms...
✓ linux-amd64 built
✓ linux-arm64 built
✓ darwin-amd64 built
✓ darwin-arm64 built
task: [int] Running integration tests...
==> Starting Docker services...
✓ Services healthy
✓ Integration tests passed (3m15s)
==> ✓ Local CI complete - ready to push!
```

---

**Last Updated:** November 6, 2025
**CI Configuration:** `.github/workflows/ci.yml`
**Task Configuration:** `Taskfile.yml`
