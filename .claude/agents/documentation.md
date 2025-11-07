# Documentation Agent

Maintain comprehensive documentation following project standards.

## Documentation Structure

```
├── README.md                    # Main project overview, usage
├── RELEASING.md                 # Release process (GoReleaser)
├── CHANGELOG.md                 # Keep a Changelog format
├── CONTRIBUTING.md              # Contribution guidelines
├── TESTING.md                   # Test infrastructure overview
├── testing/TEST_COVERAGE.md     # Detailed test coverage
├── testing/README.md            # Test details
└── .claude/agents/*.md          # Agent workflows
```

## Documentation Standards

**Format**: GitHub-flavored Markdown
**Style**:
- Clear, concise, actionable
- Code blocks with language hints
- Examples before explanations
- Commands before concepts

**Structure**:
1. Quick start / TL;DR section
2. Detailed explanations
3. Examples and code snippets
4. Troubleshooting (if applicable)
5. Related resources

## Conventional Commits for Docs

```bash
# Documentation changes
git commit -m "docs: update offline mode configuration examples"
git commit -m "docs: add troubleshooting section to RELEASING.md"
git commit -m "docs: clarify semantic versioning guidelines"

# Documentation fixes
git commit -m "fix(docs): correct Docker image registry URL"
```

## Key Documentation Areas

### 1. README.md
- Project overview and purpose
- Quick start usage
- Offline mode features
- Testing instructions
- Basic configuration

### 2. RELEASING.md
- Complete release process
- Conventional commit guidelines
- GoReleaser workflow
- Troubleshooting releases
- Release checklist

### 3. CHANGELOG.md
- Follow [Keep a Changelog](https://keepachangelog.com/)
- Semantic versioning links
- Group by: Added, Changed, Fixed, Deprecated, Removed, Security
- Update on each release (automated by GoReleaser)

### 4. Testing Documentation
- Test coverage details (`testing/TEST_COVERAGE.md`)
- Test infrastructure (`testing/README.md`)
- Quick test commands in README

## Documentation Workflow

1. **When adding features**
   - Update README.md with usage
   - Add examples and configuration
   - Update test documentation if applicable
   - Consider agent workflow updates

2. **When fixing bugs**
   - Add troubleshooting section if useful
   - Update examples if bug was in docs

3. **When releasing**
   - Verify CHANGELOG.md updated (automatic)
   - Check version references
   - Verify all new features documented

4. **For breaking changes**
   - Add migration guide
   - Update UPGRADING.md
   - Clear examples of before/after
   - Use `BREAKING CHANGE:` in commit

## Documentation Verification

```bash
# Check markdown formatting
markdownlint **/*.md

# Verify links
markdown-link-check README.md

# Test code examples
# Extract and run code blocks to verify accuracy
```

## Writing Guidelines

**Code Examples**:
- Include full commands, not fragments
- Show expected output
- Use realistic paths and values
- Test before documenting

**Structure**:
- Use headers (##, ###) for organization
- Bullet points for lists
- Numbered lists for sequences
- Code fences with language hints

**Cross-references**:
- Link to related docs
- Reference specific files with full paths
- Use anchor links for long documents

## Agent Workflow Documentation

Agent files (`.claude/agents/*.md`) should:
- Be concise (1-2 pages max)
- Lead with core workflow
- Include quick command reference
- Link to detailed docs
- Focus on actionable steps

## Related Resources

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Flavored Markdown](https://github.github.com/gfm/)
