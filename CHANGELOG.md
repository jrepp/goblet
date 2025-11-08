# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### 🔒 Security - CRITICAL

**Multi-Tenant Security Vulnerability Identified and Mitigated**

- **Identified:** Cross-tenant private repository data leakage in default configuration
- **Impact:** Critical for multi-tenant deployments with private repositories
- **Severity:** CVSS 8.1 (High)
- **Mitigation:** Multiple isolation strategies provided (sidecar pattern deployable today)

### Added

#### Security Infrastructure
- Complete security documentation suite (`docs/security/`)
- Tenant isolation framework (`isolation.go`) with 4 isolation modes
- Secure deployment manifests (`examples/kubernetes-sidecar-secure.yaml`)
- Security testing infrastructure
- NetworkPolicy and SecurityContext templates

#### Load Testing & Deployment
- Docker Compose multi-instance test environment
- Python and k6 load testing harnesses (`loadtest/`)
- HAProxy configuration with consistent hashing
- Prometheus + Grafana monitoring stack
- Comprehensive deployment pattern guide

#### Storage Optimization
- Tiered storage strategies for AWS, GCP, and Azure
- Cost optimization guide (60-95% potential savings)
- Terraform configurations for cloud storage
- Automated lifecycle management examples

#### Documentation
- Restructured documentation in `docs/` (10,000+ lines)
- Getting started guide
- Security guides (3 documents)
- Operations guides (4 documents)
- Architecture documentation (3 documents)
- Configuration examples for isolation modes

#### CI/CD & Release
- GitHub Actions automated release pipeline
- Multi-platform binary builds (Linux, macOS, Windows)
- Automated release notes generation
- SHA256 checksums for all release binaries
- Docker multi-arch image builds and publishing
- Comprehensive offline mode documentation with testing guides

### Changed
- Root README with prominent security warnings
- Documentation organization (`docs/` structure)
- Enhanced README with offline mode configuration, monitoring, and testing sections

### Security
- **Action Required for Multi-Tenant Deployments:** Review `docs/security/README.md`
- Sidecar pattern provides immediate security (no code changes)
- Namespace isolation for enterprise compliance
- Application-level isolation framework (requires integration)

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
