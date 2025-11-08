# Contributing to Goblet

Thank you for your interest in contributing to Goblet! This document provides guidelines for contributing to the project.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **Environment details** (OS, Go version, Goblet version)
- **Logs and error messages**

### Suggesting Enhancements

Enhancement suggestions are welcome! Please include:

- **Clear use case**: Why is this enhancement needed?
- **Proposed solution**: How would you like it to work?
- **Alternatives considered**: What other approaches did you consider?
- **Impact**: Who benefits from this enhancement?

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes**:
   - Write clear, concise commit messages
   - Follow the existing code style
   - Add tests for new functionality
   - Update documentation as needed
3. **Test your changes**:
   ```bash
   make test
   make test-integration
   ```
4. **Ensure code quality**:
   ```bash
   make lint
   make fmt
   ```
5. **Submit the pull request**:
   - Link any related issues
   - Describe what the PR does
   - Note any breaking changes

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Docker (for integration tests)
- Make

### Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/goblet.git
cd goblet

# Add upstream remote
git remote add upstream https://github.com/google/goblet.git

# Install dependencies
go mod download

# Build
make build

# Run tests
make test
```

### Project Structure

```
github-cache-daemon/
├── cmd/                    # Command-line tools
├── pkg/                    # Public libraries
├── internal/               # Private libraries
├── docs/                   # Documentation
├── examples/               # Configuration examples
├── loadtest/              # Load testing infrastructure
├── scripts/               # Utility scripts
└── testing/               # Test infrastructure
```

## Development Guidelines

### Code Style

- **Follow Go best practices**: See [Effective Go](https://golang.org/doc/effective_go.html)
- **Format code**: Use `gofmt` and `goimports`
- **Lint code**: Use `golangci-lint`
- **Write tests**: Aim for 80%+ coverage
- **Document exported symbols**: Use Go doc comments

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): subject

body

footer
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions or changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build process or auxiliary tool changes

**Examples:**
```
feat(cache): add LRU eviction policy

Implements a configurable LRU cache eviction policy to
prevent unbounded cache growth.

Closes #123
```

```
fix(auth): handle OAuth2 token refresh

Fixes an issue where expired tokens were not properly
refreshed, causing authentication failures.

Fixes #456
```

### Testing

**Unit Tests:**
```bash
make test
```

**Integration Tests:**
```bash
make test-integration
```

**Load Tests:**
```bash
cd loadtest && make start && make loadtest-python
```

**Test Coverage:**
```bash
make coverage
open coverage.html
```

### Documentation

- **Update docs/** when adding features
- **Update README.md** for major changes
- **Add examples/** for new configurations
- **Update CHANGELOG.md** for releases

**Validate Documentation Links:**

All documentation links are automatically validated in CI. Before submitting a PR, run:

```bash
# Validate all markdown links
./scripts/validate-links.py
```

The CI pipeline will fail if any broken links are detected. This ensures:
- All relative file links point to existing files
- All anchor links point to existing headers
- Documentation stays consistent and navigable

## Security

### Reporting Security Issues

**DO NOT** create public issues for security vulnerabilities.

Instead, email security@example.com with:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Security Guidelines

- Never commit credentials or secrets
- Follow the [Security Guide](docs/security/README.md)
- Test security-sensitive changes thoroughly
- Consider multi-tenant implications

## Release Process

Releases are handled by project maintainers:

1. Update CHANGELOG.md
2. Update version in code
3. Create git tag: `git tag -a v1.2.3 -m "Release v1.2.3"`
4. Push tag: `git push origin v1.2.3`
5. GitHub Actions builds and publishes release

See [Releasing Guide](docs/operations/releasing.md) for details.

## Getting Help

- **Documentation**: [docs/index.md](docs/index.md)
- **Questions**: [GitHub Discussions](https://github.com/google/goblet/discussions)
- **Issues**: [GitHub Issues](https://github.com/google/goblet/issues)

## Recognition

Contributors are recognized in:
- CHANGELOG.md (for significant contributions)
- GitHub contributors list
- Release notes

Thank you for contributing to Goblet! 🎉
