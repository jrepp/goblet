# Agent Workflows

Token-efficient AI agent workflows for Goblet development tasks.

## Available Agents

| Agent | Purpose | Key Tasks |
|-------|---------|-----------|
| **release.md** | Manage releases | Version determination, GoReleaser workflow, artifact verification |
| **testing.md** | Run test suites | CI pipeline, offline mode tests, race detection |
| **offline-mode.md** | Verify offline features | Fallback testing, thread safety, monitoring |
| **documentation.md** | Maintain docs | Standards, structure, conventional commits |

## Usage

Each agent provides:
- **Core Workflow**: Step-by-step process for the task
- **Quick Commands**: Copy-paste command reference
- **Key Files**: Related source files and documentation
- **Verification Steps**: How to validate success

## Design Principles

1. **Token Efficient**: Concise, actionable content
2. **Self-Contained**: Each agent is independently usable
3. **Standards-Based**: Uses project conventions (GoReleaser, conventional commits)
4. **Practical**: Real commands, not abstract concepts

## When to Use

**Release Agent**: Before creating a version tag
**Testing Agent**: Before committing code changes
**Offline Mode Agent**: When modifying cache/offline features
**Documentation Agent**: When updating project docs

## Integration with Development

These agents complement:
- `Taskfile.yml` - Development task automation
- `.github/workflows/` - CI/CD pipelines
- `RELEASING.md` - Full release documentation
- `testing/TEST_COVERAGE.md` - Test details

## Contributing Agent Workflows

When adding new agents:
1. Keep under 2 pages (200-300 lines)
2. Lead with core workflow
3. Include quick command reference
4. Link to detailed documentation
5. Focus on actionable steps
6. Use existing agents as templates

## Example Usage

```bash
# Before creating a release
cat .claude/agents/release.md
# Follow the workflow: verify CI, check commits, create tag

# When writing offline mode tests
cat .claude/agents/offline-mode.md
# Use the test commands and verification steps

# For test-driven development
cat .claude/agents/testing.md
# Run baseline tests, implement, verify with CI
```

## Project Context

**Repository**: github-cache-daemon (Goblet)
**Purpose**: Git caching proxy server
**Key Technologies**: Go, GoReleaser, GitHub Actions
**Main Features**: Offline mode, multi-platform releases, comprehensive testing
