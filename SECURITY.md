# Security Policy

## ⚠️ Critical Security Notice

**Goblet's default configuration is unsafe for multi-tenant deployments with private repositories.**

### The Vulnerability

Default cache keys include only repository URL, not user identity. This allows authenticated users to access cached private repositories belonging to other users.

**Severity:** Critical (CVSS 8.1)
**Impact:** Private repository data leakage between users/tenants

### Quick Assessment

**✅ Your deployment is SAFE if:**
- Single user or service account per Goblet instance
- Only public repositories accessed
- Using sidecar pattern (one instance per workload)

**🚨 Your deployment is AT RISK if:**
- Multiple users share a Goblet instance
- Users access private repositories with different permissions
- Multi-tenant SaaS, automated IaC/CI tools, or security scanning scenarios

## Immediate Actions

### Safe Today: Sidecar Pattern

Deploy one Goblet instance per workload. No code changes required:

```bash
kubectl apply -f examples/kubernetes-sidecar-secure.yaml
```

**Complete guide:** [docs/security/multi-tenant-deployment.md](docs/security/multi-tenant-deployment.md)

### For Detailed Information

- **Security Overview:** [docs/security/README.md](docs/security/README.md)
- **Isolation Strategies:** [docs/security/isolation-strategies.md](docs/security/isolation-strategies.md)
- **Deployment Guide:** [docs/security/multi-tenant-deployment.md](docs/security/multi-tenant-deployment.md)

## Reporting Security Issues

**Do not** open public GitHub issues for security vulnerabilities.

**Email:** security@example.com

Include:
- Description of vulnerability
- Steps to reproduce
- Affected versions
- Suggested remediation (optional)

We follow a 90-day coordinated disclosure policy.

## Security Updates

Security updates are published in:
- [CHANGELOG.md](CHANGELOG.md)
- [GitHub Security Advisories](https://github.com/google/goblet/security/advisories)
- Security mailing list (subscribe at security@example.com)

## Supported Versions

| Version | Security Support |
|---------|------------------|
| 2.x | ✅ Full support |
| 1.x | ⚠️  Critical fixes only |
| < 1.0 | ❌ Not supported |

## Security Best Practices

1. **Never** share Goblet instances across tenants without isolation
2. **Always** use TLS for production deployments
3. **Enable** audit logging for compliance requirements
4. **Review** security documentation before deploying
5. **Monitor** for unauthorized access attempts

## Additional Resources

- [Complete Security Guide](docs/security/README.md)
- [Deployment Patterns](docs/operations/deployment-patterns.md)
- [Getting Started](docs/getting-started.md)

---

**Last Updated:** 2025-11-07
