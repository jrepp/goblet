# Release Checklist

## Pre-Release
- [ ] All tests passing on main branch
- [ ] Documentation updated
- [ ] CHANGELOG.md reviewed
- [ ] Breaking changes documented

## Release Process

### Automatic Semantic Release (Recommended)

Releases are created automatically when commits are merged to `main` using [Conventional Commits](https://www.conventionalcommits.org/):

**Commit Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types that trigger releases:**
- `feat:` - New feature (triggers **minor** version bump)
- `fix:` - Bug fix (triggers **patch** version bump)
- `perf:` - Performance improvement (triggers **patch** version bump)
- `BREAKING CHANGE:` - Breaking change (triggers **major** version bump)

**Examples:**
```bash
# New feature (0.1.0 -> 0.2.0)
git commit -m "feat(auth): add OIDC authentication support"

# Bug fix (0.2.0 -> 0.2.1)
git commit -m "fix(storage): resolve S3 connection timeout"

# Breaking change (0.2.1 -> 1.0.0)
git commit -m "feat(api)!: redesign REST API

BREAKING CHANGE: API endpoints have changed. See migration guide."
```

### Manual Release

If you need to create a manual release:

1. **Create a tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Or use GitHub UI:**
   - Go to Releases → Draft a new release
   - Create tag: `v1.0.0`
   - Publish release

The release workflow will automatically:
- Build binaries for all platforms
- Build Docker images for all variants
- Create GitHub release with artifacts
- Publish images to GitHub Container Registry

## Post-Release
- [ ] Verify Docker images published
- [ ] Test binary downloads
- [ ] Announce release (if major version)
- [ ] Update deployment documentation

## Image Variants

Three variants are built for each release:

| Variant | Base Image | Size | Use Case |
|---------|-----------|------|----------|
| `alpine` | Alpine Linux | ~15MB | Development, debugging |
| `distroless` | Google Distroless | ~8MB | Production (recommended) |
| `scratch` | Scratch | ~5MB | Minimal production |

## Supported Platforms

All images and binaries support:
- `linux/amd64` - Intel/AMD 64-bit
- `linux/arm64` - ARM 64-bit (Apple Silicon, AWS Graviton)
- `darwin/amd64` - macOS Intel
- `darwin/arm64` - macOS Apple Silicon
- `windows/amd64` - Windows 64-bit

## Conventional Commits Reference

See: https://www.conventionalcommits.org/

**Common Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only changes
- `style` - Code style changes (formatting, etc.)
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `test` - Adding or updating tests
- `build` - Build system changes
- `ci` - CI/CD changes
- `chore` - Other changes (dependencies, etc.)
- `revert` - Revert a previous commit

**Scopes (examples):**
- `auth` - Authentication
- `storage` - Storage layer
- `api` - API changes
- `docker` - Docker/container changes
- `ci` - CI/CD changes
