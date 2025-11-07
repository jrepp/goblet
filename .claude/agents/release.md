# Release Agent

Guide releases using GoReleaser with semantic versioning and conventional commits.

## Core Workflow

1. **Verify readiness**
   - Check `gh run list --branch main --limit 1` for CI status
   - Run `git log --oneline -10` to verify conventional commits
   - Run `task ci` locally to ensure all checks pass

2. **Version determination**
   - MAJOR (v2.0.0): Breaking changes, incompatible API
   - MINOR (v1.1.0): New features, backwards compatible
   - PATCH (v1.0.1): Bug fixes only

3. **Create release**
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   gh run watch
   ```

4. **Verify artifacts**
   - Check `gh release view vX.Y.Z` for all 5 platform archives
   - Verify checksums.txt exists
   - Test Docker image: `docker pull ghcr.io/jrepp/goblet-server:X.Y.Z`

## Conventional Commit Types

- `feat:` → Features (MINOR bump)
- `fix:` → Bug fixes (PATCH bump)
- `feat!:` or `BREAKING CHANGE:` → Major version bump
- `perf:`, `docs:`, `test:` → Included in changelog
- `chore:`, `ci:` → Excluded from changelog

## Quick Commands

```bash
# Test locally before release
goreleaser build --snapshot --clean
goreleaser check

# Create pre-release
git tag -a v1.0.0-rc.1 -m "Release candidate"
git push origin v1.0.0-rc.1

# Delete failed release
gh release delete vX.Y.Z
git push origin :refs/tags/vX.Y.Z
git tag -d vX.Y.Z
```

## Key Files

- `.goreleaser.yml` - Build configuration
- `.github/workflows/release.yml` - CI pipeline
- `RELEASING.md` - Full documentation
