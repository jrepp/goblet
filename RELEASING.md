# Release Process

This document describes how to create a new release of Goblet.

## Overview

Goblet uses **[GoReleaser](https://goreleaser.com/)** for automated, standardized releases. GoReleaser is the industry-standard tool for Go project releases and provides:

- ✅ **Automatic semantic versioning** from git tags
- ✅ **Multi-platform binary builds** (Linux, macOS, Windows)
- ✅ **Automatic changelog generation** from git commits
- ✅ **SHA256 checksum generation**
- ✅ **GitHub release creation** with all artifacts
- ✅ **Multi-arch Docker images** (amd64, arm64)
- ✅ **Archive generation** (tar.gz, zip)

## Prerequisites

- Write access to the GitHub repository
- Clean working directory on the `main` branch
- All CI checks passing on `main`
- Follow [Conventional Commits](https://www.conventionalcommits.org/) for automatic changelog generation

## Release Workflow Overview

When you push a version tag, GoReleaser automatically:

1. Builds binaries for all supported platforms
2. Generates SHA256 checksums for verification
3. Creates archives (tar.gz for Unix, zip for Windows)
4. Generates changelog from git history using conventional commits
5. Creates a GitHub release with all binaries attached
6. Builds and pushes multi-arch Docker images to GitHub Container Registry (GHCR)

## Supported Platforms

The release pipeline builds binaries for:

- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64

## Conventional Commits for Automatic Changelogs

GoReleaser generates changelogs automatically from git commit messages. Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Common Types

- `feat`: New features (appears in changelog under "Features")
- `fix`: Bug fixes (appears in changelog under "Bug Fixes")
- `perf`: Performance improvements
- `docs`: Documentation changes
- `test`: Test additions or changes
- `build`: Build system changes
- `ci`: CI/CD changes
- `chore`: Other changes (excluded from changelog)

### Examples

```bash
# Feature
git commit -m "feat: add offline ls-refs support with local cache fallback"

# Bug fix
git commit -m "fix: resolve data race in UpstreamEnabled configuration"

# Breaking change
git commit -m "feat!: change config API to use atomic operations

BREAKING CHANGE: UpstreamEnabled now requires SetUpstreamEnabled() method"

# With scope
git commit -m "fix(auth): handle expired tokens correctly"
```

### Breaking Changes

Indicate breaking changes with `!` or `BREAKING CHANGE:` in the footer:

```bash
git commit -m "feat!: require Go 1.21 or higher"
# or
git commit -m "feat: require Go 1.21 or higher

BREAKING CHANGE: Go 1.20 is no longer supported"
```

## Step-by-Step Release Process

### 1. Ensure Clean Commit History

Make sure recent commits follow conventional commit format:

```bash
# View recent commits
git log --oneline -10

# Good examples:
# feat: add new cache backend support
# fix: resolve memory leak in repository manager
# docs: update installation instructions

# If needed, update commit messages before release
git rebase -i HEAD~5  # Interactive rebase to edit messages
```

### 2. Verify CI Status

Ensure all CI checks are passing on main:

```bash
# Check latest CI status
gh run list --branch main --limit 1

# Or visit GitHub Actions
# https://github.com/jrepp/github-cache-daemon/actions
```

### 3. Create and Push the Release Tag

Create a version tag following semantic versioning:

```bash
# For a new major version (breaking changes)
git tag -a v1.0.0 -m "Release v1.0.0"

# For a new minor version (new features, backwards compatible)
git tag -a v1.1.0 -m "Release v1.1.0"

# For a patch version (bug fixes)
git tag -a v1.0.1 -m "Release v1.0.1"

# Push the tag to trigger the release pipeline
git push origin v1.0.0
```

**Important**: Use the `v` prefix (e.g., `v1.0.0`) to match the workflow trigger pattern.

### 4. Monitor the Release Pipeline

Watch the release workflow progress:

```bash
# Watch the release workflow in real-time
gh run watch

# Or view release workflow runs
gh run list --workflow=release.yml --limit 5

# Or check GitHub Actions UI
# https://github.com/jrepp/github-cache-daemon/actions/workflows/release.yml
```

GoReleaser will:
- ✅ Build binaries for all platforms (~3-5 minutes)
- ✅ Generate SHA256 checksums
- ✅ Create archives (tar.gz/zip)
- ✅ Generate changelog from commits
- ✅ Create GitHub release with all artifacts
- ✅ Build and push multi-arch Docker images to GHCR

### 5. Verify the Release

Once the pipeline completes:

1. **Check the GitHub Release page**:
   ```bash
   gh release view v1.0.0
   # Or visit: https://github.com/jrepp/github-cache-daemon/releases
   ```

2. **Verify all archives and checksums are attached**:
   - `goblet_1.0.0_linux_amd64.tar.gz`
   - `goblet_1.0.0_linux_arm64.tar.gz`
   - `goblet_1.0.0_darwin_amd64.tar.gz`
   - `goblet_1.0.0_darwin_arm64.tar.gz`
   - `goblet_1.0.0_windows_amd64.zip`
   - `checksums.txt` (contains all SHA256 checksums)

3. **Test a binary download and verification**:
   ```bash
   # Download archive and checksums
   gh release download v1.0.0 -p "goblet_1.0.0_linux_amd64.tar.gz"
   gh release download v1.0.0 -p "checksums.txt"

   # Verify checksum
   sha256sum -c --ignore-missing checksums.txt

   # Extract and test
   tar -xzf goblet_1.0.0_linux_amd64.tar.gz
   ./goblet-server --version
   ```

4. **Verify Docker images on GitHub Container Registry**:
   ```bash
   # Pull version-specific tag
   docker pull ghcr.io/jrepp/goblet-server:1.0.0

   # Pull latest tag
   docker pull ghcr.io/jrepp/goblet-server:latest

   # Verify multi-arch support
   docker inspect ghcr.io/jrepp/goblet-server:1.0.0 | grep Architecture
   ```

### 6. Announce the Release

After verification:

1. Update any documentation referencing version numbers
2. Announce on relevant channels (if applicable)
3. Update any dependent projects

## Pre-releases and Release Candidates

To create a pre-release:

```bash
# Alpha release
git tag -a v1.0.0-alpha.1 -m "Release v1.0.0-alpha.1"
git push origin v1.0.0-alpha.1

# Beta release
git tag -a v1.0.0-beta.1 -m "Release v1.0.0-beta.1"
git push origin v1.0.0-beta.1

# Release candidate
git tag -a v1.0.0-rc.1 -m "Release v1.0.0-rc.1"
git push origin v1.0.0-rc.1
```

Pre-releases are automatically marked as "pre-release" on GitHub (any tag containing a hyphen).

## Semantic Versioning Guidelines

Follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version (`v2.0.0`): Breaking changes, incompatible API changes
- **MINOR** version (`v1.1.0`): New features, backwards compatible
- **PATCH** version (`v1.0.1`): Bug fixes, backwards compatible

Examples:
- Adding offline mode feature: `v1.1.0` (new feature, backwards compatible)
- Fixing race condition: `v1.0.1` (bug fix)
- Changing config API: `v2.0.0` (breaking change)

## GitHub Container Registry (GHCR)

Docker images are automatically pushed to GitHub Container Registry (GHCR) during releases. No additional configuration is required - the workflow uses the built-in `GITHUB_TOKEN` with `packages: write` permission.

Images are available at:
- `ghcr.io/jrepp/goblet-server:latest`
- `ghcr.io/jrepp/goblet-server:1.0.0`
- `ghcr.io/jrepp/goblet-server:1.0`
- `ghcr.io/jrepp/goblet-server:1`

### Making Images Public

By default, GHCR images are private. To make them public:

1. Go to https://github.com/users/jrepp/packages/container/goblet-server/settings
2. Scroll to "Danger Zone"
3. Click "Change visibility" → "Public"

## Local Testing with GoReleaser

Test the release process locally before creating a tag:

### Install GoReleaser

```bash
# macOS
brew install goreleaser

# Linux
go install github.com/goreleaser/goreleaser@latest

# Or download from: https://github.com/goreleaser/goreleaser/releases
```

### Test Build Without Publishing

```bash
# Build for all platforms (no publishing)
goreleaser build --snapshot --clean

# Check dist/ directory for binaries
ls -la dist/

# Test a specific binary
./dist/goblet-server_linux_amd64_v1/goblet-server --version
```

### Test Full Release (Snapshot Mode)

```bash
# Run complete release process without publishing
goreleaser release --snapshot --clean --skip=publish

# This will:
# - Build all binaries
# - Create archives
# - Generate checksums
# - Create changelog
# - Skip: GitHub release creation, Docker push
```

### Validate Configuration

```bash
# Check .goreleaser.yml for errors
goreleaser check

# View current configuration
goreleaser --help
```

## Troubleshooting

### Pipeline Fails at Build Step

Check the build logs for compilation errors:
```bash
gh run view --log
```

Common issues:
- Go module issues: Ensure `go.mod` is up to date
- Build errors: Run `task ci` locally before tagging

### Release Already Exists

If you need to recreate a release:

```bash
# Delete the GitHub release
gh release delete v1.0.0

# Delete the tag locally and remotely
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0

# Recreate and push
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### Missing Binaries in Release

If some binaries are missing, check:
- Build matrix configuration in `.github/workflows/release.yml`
- Platform-specific build errors in workflow logs

### Docker Push Fails

If Docker image push fails:
- Verify `DOCKER_HUB_USERNAME` and `DOCKER_HUB_TOKEN` are configured
- Check Docker Hub access token permissions
- Verify repository name in workflow matches Docker Hub repository

## Testing the Release Pipeline

To test the release pipeline without creating an official release:

1. Create a test tag in a feature branch:
   ```bash
   git checkout -b test-release
   git tag -a v0.0.0-test.1 -m "Test release"
   git push origin v0.0.0-test.1
   ```

2. Monitor the workflow

3. Clean up:
   ```bash
   gh release delete v0.0.0-test.1
   git push origin :refs/tags/v0.0.0-test.1
   git tag -d v0.0.0-test.1
   ```

## Version Numbering Strategy

Current development follows this strategy:

- **Main branch**: Unreleased development (`main`)
- **Stable releases**: `v1.0.0`, `v1.1.0`, `v1.0.1`
- **Pre-releases**: `v1.0.0-alpha.1`, `v1.0.0-beta.1`, `v1.0.0-rc.1`
- **Feature branches**: No tags (merge to main first)

## Release Checklist

Use this checklist when creating a release:

**Pre-Release:**
- [ ] All CI checks passing on `main`
- [ ] Recent commits follow conventional commit format
- [ ] Version number determined (follows semantic versioning)
- [ ] GoReleaser config validated locally (`goreleaser check`)
- [ ] Local snapshot build tested (`goreleaser build --snapshot --clean`)

**Release:**
- [ ] Tag created with `v` prefix (e.g., `v1.0.0`)
- [ ] Tag pushed to GitHub
- [ ] GitHub Actions workflow triggered

**Post-Release Verification:**
- [ ] Pipeline completed successfully
- [ ] All archives present in GitHub release
- [ ] Checksums file (`checksums.txt`) included
- [ ] Changelog automatically generated and accurate
- [ ] Docker images pushed to GHCR
- [ ] Downloaded binary tested and verified
- [ ] Release notes reviewed
- [ ] Documentation updated (if needed)
- [ ] Release announced (if needed)

## Getting Help

If you encounter issues with the release process:

1. Check GitHub Actions logs: https://github.com/jrepp/github-cache-daemon/actions
2. Review workflow file: `.github/workflows/release.yml`
3. Open an issue: https://github.com/jrepp/github-cache-daemon/issues
