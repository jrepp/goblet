# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions automated release pipeline
- Multi-platform binary builds (Linux, macOS, Windows)
- Automated release notes generation
- SHA256 checksums for all release binaries
- Docker multi-arch image builds and publishing
- Comprehensive offline mode documentation with testing guides

### Changed
- Enhanced README with offline mode configuration, monitoring, and testing sections

## Template for New Releases

When creating a new release, copy the following template and fill in the details:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features and capabilities

### Changed
- Changes to existing functionality

### Deprecated
- Features that will be removed in future releases

### Removed
- Features that have been removed

### Fixed
- Bug fixes

### Security
- Security-related changes and fixes
```

[Unreleased]: https://github.com/jrepp/github-cache-daemon/compare/main...HEAD
