# GitHub Actions Workflows

## CI Workflow (`.github/workflows/ci.yml`)

The CI workflow parallelizes the `task ci` command into separate jobs for optimal performance.

### Workflow Structure

```
Pull Request / Push to main
│
├─── format-check ────┐
├─── tidy-check ──────┤
├─── lint ────────────┤──> ci-complete (status check)
├─── test-unit ───────┤
├─── build ───────────┘
│
├─── build-multi (matrix: 4 platforms)
│
└─── integration-test (conditional: main branch or label)
```

### Job Details

#### Parallel CI Jobs (No Docker Required)

| Job | Task Equivalent | Duration | Description |
|-----|----------------|----------|-------------|
| `format-check` | `task fmt-check` | ~10s | Validates code formatting with gofmt and goimports |
| `tidy-check` | `task tidy-check` | ~15s | Checks go.mod and go.sum are tidy |
| `lint` | `task lint` | ~45s | Runs golangci-lint, staticcheck, and go vet |
| `test-unit` | `task test-unit` | ~30s | Unit tests with race detector, uploads coverage |
| `build` | `task build` | ~20s | Builds for current platform, uploads artifact |

#### Matrix Build Job

| Job | Platforms | Description |
|-----|-----------|-------------|
| `build-multi` | linux-amd64, linux-arm64, darwin-amd64, darwin-arm64 | Cross-platform builds in parallel |

#### Status Check Job

| Job | Dependencies | Description |
|-----|-------------|-------------|
| `ci-complete` | All parallel jobs | Provides single PR status check |

#### Integration Test Job

| Job | When | Docker | Description |
|-----|------|--------|-------------|
| `integration-test` | main branch or label | ✅ Yes | Runs `task test-integration-go` |

### Triggering Integration Tests on PRs

To run integration tests on a pull request, add the `run-integration-tests` label:

```bash
# Via GitHub CLI
gh pr edit <PR_NUMBER> --add-label "run-integration-tests"

# Via GitHub UI
Add label: run-integration-tests
```

### Local Testing

Run the same checks locally:

```bash
# Fast CI checks (no Docker)
task ci

# Quick feedback (no race detector)
task ci-quick

# Full CI with integration tests (requires Docker)
task ci-full

# Individual checks
task fmt-check
task tidy-check
task lint
task test-unit
task build
```

### Performance Comparison

| Approach | Duration | Parallelization |
|----------|----------|----------------|
| Sequential (`task ci`) | ~2min | ❌ No |
| GitHub Actions (parallel) | ~45s | ✅ Yes (5 jobs) |

### Workflow Features

✓ **Parallel Execution** - All CI checks run simultaneously
✓ **Fast Feedback** - Get results in ~45s instead of ~2min
✓ **Granular Status** - See which specific check failed
✓ **Artifact Uploads** - Build artifacts and coverage reports saved
✓ **Conditional Integration Tests** - Only run when needed
✓ **Go Caching** - Dependencies cached between runs
✓ **Multi-platform Builds** - Cross-compile for 4 platforms in parallel

### Codecov Integration

The workflow uploads coverage reports to Codecov:

- **Unit tests**: `coverage-unit.out` → flag: `unittests`
- **Integration tests**: `coverage-integration.out` → flag: `integration`

**Note:** Requires `CODECOV_TOKEN` secret to be configured in repository settings.

### Customization

#### Change Go Version

Edit the environment variable in `.github/workflows/ci.yml`:

```yaml
env:
  GO_VERSION: '1.25.3'  # Change this
```

#### Skip Integration Tests

Integration tests are automatically skipped on PRs unless:
- The PR has the `run-integration-tests` label
- The push is to main/master branch

#### Adjust Parallel Jobs

To add/remove jobs from the `ci-complete` dependency list:

```yaml
ci-complete:
  needs:
    - format-check
    - tidy-check
    - lint
    - test-unit
    - build
    # Add new jobs here
```

## Workflow Best Practices

1. **All CI checks must pass** - The `ci-complete` job provides a single status check
2. **Integration tests optional on PRs** - Use label to run when needed
3. **Coverage uploaded automatically** - View reports on Codecov
4. **Artifacts retained for 7 days** - Download builds from GitHub Actions UI
5. **Test locally first** - Run `task ci` before pushing

## Troubleshooting

### Job Fails: "goimports not found"

The `format-check` job installs goimports automatically. If it fails, the Go tools cache may be corrupted.

**Solution:** Re-run the job or clear the cache.

### Job Fails: "golangci-lint not found"

The `lint` job runs `task install-tools` to install linters. If it fails:

**Solution:** Check that `task install-tools` works locally.

### Integration Tests Skipped on PR

Integration tests only run when:
- On main/master branch, OR
- PR has `run-integration-tests` label

**Solution:** Add the label to your PR.

### Coverage Upload Fails

Requires `CODECOV_TOKEN` secret.

**Solution:** Add the token in repository settings:
1. Go to repository Settings → Secrets and variables → Actions
2. Add new secret: `CODECOV_TOKEN`
3. Get token from https://codecov.io

### All Jobs Pending

GitHub Actions may be queueing jobs.

**Solution:** Wait for runners to become available, or check GitHub Actions status page.
