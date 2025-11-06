# Release Documentation

## Overview

This project uses automated semantic versioning and releases. Releases are triggered automatically when commits are merged to `main` following the [Conventional Commits](https://www.conventionalcommits.org/) specification.

## Quick Start

### Using Docker (Recommended)

Pull the latest image:
```bash
# Alpine variant (recommended for development)
docker pull ghcr.io/jrepp/github-cache-daemon:latest

# Distroless variant (recommended for production)
docker pull ghcr.io/jrepp/github-cache-daemon:latest-distroless

# Scratch variant (minimal)
docker pull ghcr.io/jrepp/github-cache-daemon:latest-scratch
```

### Using Pre-built Binaries

Download from [Releases](https://github.com/jrepp/github-cache-daemon/releases):

**Linux (amd64):**
```bash
wget https://github.com/jrepp/github-cache-daemon/releases/latest/download/goblet-server-linux-amd64
chmod +x goblet-server-linux-amd64
./goblet-server-linux-amd64 --help
```

**macOS (Apple Silicon):**
```bash
wget https://github.com/jrepp/github-cache-daemon/releases/latest/download/goblet-server-darwin-arm64
chmod +x goblet-server-darwin-arm64
./goblet-server-darwin-arm64 --help
```

## Release Process

### Automatic Releases (Conventional Commits)

The project uses [semantic-release](https://semantic-release.gitbook.io/) with Conventional Commits to automatically:
1. Determine the next version number
2. Generate release notes
3. Create a Git tag
4. Build and publish artifacts
5. Update CHANGELOG.md

### Commit Message Format

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

### Types and Version Bumps

| Type | Version Bump | Example |
|------|-------------|---------|
| `feat` | **Minor** (0.1.0 → 0.2.0) | `feat(auth): add OAuth2 support` |
| `fix` | **Patch** (0.2.0 → 0.2.1) | `fix(storage): resolve timeout issue` |
| `perf` | **Patch** (0.2.1 → 0.2.2) | `perf(cache): optimize lookup performance` |
| `BREAKING CHANGE` | **Major** (0.2.2 → 1.0.0) | `feat!: redesign API` |

### Non-Release Commits

These types do NOT trigger releases:
- `docs` - Documentation changes
- `style` - Code formatting
- `refactor` - Code refactoring (no behavior change)
- `test` - Test updates
- `chore` - Maintenance tasks
- `ci` - CI/CD changes

### Examples

**New Feature (Minor Release):**
```bash
git commit -m "feat(oidc): add OIDC authentication provider

Implements OpenID Connect authentication using Dex as IdP.
Includes token verification and request authorization."
```

**Bug Fix (Patch Release):**
```bash
git commit -m "fix(auth): prevent empty tokens from being sent upstream

Conditionally set Authorization headers only when token is non-empty.
This prevents 401 errors from GitHub on public repositories.

Fixes #123"
```

**Breaking Change (Major Release):**
```bash
git commit -m "feat(api)!: redesign REST API endpoints

BREAKING CHANGE: API endpoints have been reorganized:
- /v1/repos -> /api/v2/repositories
- /v1/health -> /api/v2/health

Migration guide: docs/migration-v2.md"
```

## Docker Images

### Image Variants

Three variants are built for each release, optimized for different use cases:

#### Alpine (Development & Debugging)
```bash
docker pull ghcr.io/jrepp/github-cache-daemon:1.0.0-alpine
```

**Features:**
- Full Alpine Linux base
- Package manager (apk)
- Shell access
- ca-certificates and git included
- ~15MB compressed

**Best for:**
- Development
- Debugging
- Interactive troubleshooting

#### Distroless (Production - Recommended)
```bash
docker pull ghcr.io/jrepp/github-cache-daemon:1.0.0-distroless
```

**Features:**
- Google Distroless base
- No shell, no package manager
- CA certificates included
- Runs as non-root user
- ~8MB compressed

**Best for:**
- Production deployments
- Security-conscious environments
- Minimal attack surface

#### Scratch (Minimal)
```bash
docker pull ghcr.io/jrepp/github-cache-daemon:1.0.0-scratch
```

**Features:**
- Minimal scratch base
- Only binary + CA certs
- ~5MB compressed

**Best for:**
- Extremely size-constrained environments
- Custom base image requirements

### Multi-Architecture Support

All images support multiple architectures:
- `linux/amd64` - Intel/AMD 64-bit
- `linux/arm64` - ARM 64-bit (Apple Silicon, AWS Graviton, Raspberry Pi)

Docker automatically pulls the correct architecture for your platform.

### Tag Conventions

| Tag Pattern | Description | Example |
|-------------|-------------|---------|
| `latest` | Latest stable (alpine) | `ghcr.io/jrepp/github-cache-daemon:latest` |
| `latest-{variant}` | Latest stable variant | `ghcr.io/jrepp/github-cache-daemon:latest-distroless` |
| `{version}` | Specific version (alpine) | `ghcr.io/jrepp/github-cache-daemon:1.2.3` |
| `{version}-{variant}` | Specific version variant | `ghcr.io/jrepp/github-cache-daemon:1.2.3-distroless` |
| `{major}.{minor}` | Latest patch | `ghcr.io/jrepp/github-cache-daemon:1.2` |
| `{major}` | Latest minor | `ghcr.io/jrepp/github-cache-daemon:1` |

### Usage Examples

**Development (Alpine with shell):**
```bash
docker run -it --rm \
  -p 8080:8080 \
  ghcr.io/jrepp/github-cache-daemon:latest-alpine \
  -cache_root=/cache
```

**Production (Distroless):**
```bash
docker run -d \
  -p 8080:8080 \
  -v /data/cache:/cache \
  ghcr.io/jrepp/github-cache-daemon:latest-distroless \
  -cache_root=/cache \
  -storage_provider=s3
```

## Binary Artifacts

### Supported Platforms

Pre-built binaries are available for:

| OS | Architecture | Binary Name |
|----|--------------|-------------|
| Linux | amd64 | `goblet-server-linux-amd64` |
| Linux | arm64 | `goblet-server-linux-arm64` |
| Linux | arm | `goblet-server-linux-arm` |
| macOS | amd64 (Intel) | `goblet-server-darwin-amd64` |
| macOS | arm64 (M1/M2/M3) | `goblet-server-darwin-arm64` |
| Windows | amd64 | `goblet-server-windows-amd64.exe` |

### Verification

Each binary includes a SHA256 checksum file:

```bash
# Download binary and checksum
wget https://github.com/jrepp/github-cache-daemon/releases/latest/download/goblet-server-linux-amd64
wget https://github.com/jrepp/github-cache-daemon/releases/latest/download/goblet-server-linux-amd64.sha256

# Verify checksum
sha256sum -c goblet-server-linux-amd64.sha256
```

## Manual Release

For emergency releases or testing, you can manually trigger a release:

### Via Git Tag

```bash
git tag v1.0.0
git push origin v1.0.0
```

### Via GitHub UI

1. Go to [Releases](https://github.com/jrepp/github-cache-daemon/releases)
2. Click "Draft a new release"
3. Choose or create tag: `v1.0.0`
4. Click "Publish release"

### Via GitHub Actions

1. Go to [Actions](https://github.com/jrepp/github-cache-daemon/actions)
2. Select "Release" workflow
3. Click "Run workflow"
4. Enter tag (e.g., `v1.0.0`)
5. Click "Run workflow"

## Release Artifacts

Each release includes:

### Docker Images
- 3 image variants (alpine, distroless, scratch)
- 2 architectures per variant (amd64, arm64)
- Published to GitHub Container Registry

### Binary Artifacts
- 6 platform binaries
- SHA256 checksums for each binary
- Attached to GitHub release

### Release Notes
- Automatically generated changelog
- Commit history since last release
- Breaking changes highlighted
- Installation instructions

## Troubleshooting

### Release Not Triggered

**Check commit message format:**
```bash
# View recent commits
git log --oneline -5

# Check if commits follow conventional format
git log --format="%s" -1
```

**Ensure you're on main branch:**
```bash
git branch --show-current
```

### Docker Image Not Found

**Check registry URL:**
```bash
# Correct
docker pull ghcr.io/jrepp/github-cache-daemon:latest

# Incorrect (missing ghcr.io)
docker pull jrepp/github-cache-daemon:latest
```

**Authenticate to GHCR:**
```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

### Binary Checksums Don't Match

**Re-download artifacts:**
```bash
rm goblet-server-*
wget https://github.com/jrepp/github-cache-daemon/releases/latest/download/goblet-server-linux-amd64
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Pull latest image
        run: docker pull ghcr.io/jrepp/github-cache-daemon:latest-distroless

      - name: Deploy
        run: |
          docker run -d \
            --name goblet \
            -p 8080:8080 \
            ghcr.io/jrepp/github-cache-daemon:latest-distroless
```

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goblet-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: goblet-server
  template:
    metadata:
      labels:
        app: goblet-server
    spec:
      containers:
      - name: goblet
        image: ghcr.io/jrepp/github-cache-daemon:latest-distroless
        ports:
        - containerPort: 8080
        args:
        - -cache_root=/cache
        - -storage_provider=s3
```

## Version History

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.

## References

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [semantic-release](https://semantic-release.gitbook.io/)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
