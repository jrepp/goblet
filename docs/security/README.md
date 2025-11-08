# Security Guide

This guide covers security considerations for deploying Goblet, particularly for multi-tenant environments with private repositories.

## Overview

Goblet's default configuration is designed for single-tenant deployments. Multi-tenant scenarios with private repositories require additional security measures to prevent data leakage between users or organizations.

## Threat Model

### Default Configuration Security Boundary

In the default configuration, Goblet provides:

✅ **Authentication** - Per-request authentication via OAuth2/OIDC
✅ **TLS Support** - Encrypted communication with upstream servers
✅ **Authorization** - Validates user identity on each request

❌ **Tenant Isolation** - No separation of cached data by user/tenant
❌ **Encryption at Rest** - Repository data stored unencrypted
❌ **Audit Logging** - Limited access tracking

### Vulnerability: Cross-Tenant Data Access

**Scenario:**
```
1. User Alice (authorized) fetches github.com/company/secrets
   → Cached at /cache/github.com/company/secrets

2. User Bob (unauthorized) requests same repository
   → Bob is authenticated as a valid user
   → Cache serves Bob the repository WITHOUT checking his permissions
   → Bob gains access to Alice's private repository
```

**Root Cause:** Cache keys include only repository URL, not user identity.

**Severity:** Critical for multi-tenant deployments with private repositories

**CVSS Score:** 8.1 (High)

## Determining Your Risk Level

### ✅ Low Risk (No Action Required)

Your deployment is safe if ANY of these apply:
- Single user or service account per Goblet instance
- All users have identical repository access permissions
- Only public repositories are accessed
- Sidecar pattern (one Goblet instance per workload)

### ⚠️ Medium Risk (Review Required)

Review security measures if:
- Multiple users share a Goblet instance
- Users access different sets of private repositories
- Operating in a development or staging environment

### 🚨 High Risk (Immediate Action Required)

Take immediate action if:
- Production multi-tenant deployment
- Different organizations/teams sharing infrastructure
- Compliance requirements (SOC 2, ISO 27001, GDPR)
- Security scanning or Terraform Cloud scenarios

## Security Solutions

We provide three approaches based on your deployment needs:

### Solution 1: Sidecar Pattern (Recommended)

**Best for:** Kubernetes deployments, Terraform Cloud, CI/CD runners

Deploy one Goblet instance per workload using Kubernetes sidecars:

```yaml
# Each pod gets its own isolated cache
containers:
  - name: application
  - name: goblet-sidecar
    volumeMounts:
      - name: cache
        mountPath: /cache
volumes:
  - name: cache
    emptyDir: {}
```

**Benefits:**
- Perfect isolation (no shared cache)
- No code changes required
- Deploy today
- Natural Kubernetes-style scaling

**See:** [Multi-Tenant Deployment Guide](multi-tenant-deployment.md#sidecar-pattern)

### Solution 2: Namespace Isolation

**Best for:** Enterprise Kubernetes, compliance requirements

Deploy separate Goblet instances per tenant in isolated namespaces:

```yaml
# Namespace per tenant with NetworkPolicy
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-acme-corp
---
# Goblet deployment with tenant-specific configuration
# ...
```

**Benefits:**
- Strong Kubernetes-native isolation
- Network-level security
- Resource quotas per tenant
- Audit trail per namespace

**See:** [Multi-Tenant Deployment Guide](multi-tenant-deployment.md#namespace-isolation)

### Solution 3: Application-Level Isolation

**Best for:** Custom deployments, future enhancement

Implement tenant-aware cache partitioning at the application level:

```go
// Cache path includes tenant identifier
/cache/tenant-{id}/{repo-host}/{repo-path}
```

**Status:** Framework implemented, requires integration (4 hours)

**Benefits:**
- Fine-grained control
- Efficient resource utilization
- Flexible policy management

**See:** [Isolation Strategies](isolation-strategies.md)

## Implementation Guide

### Immediate Actions (Do Now)

1. **Assess your deployment:**
   ```bash
   # Count unique users
   kubectl logs deployment/goblet | grep -o 'user=[^,]*' | sort -u | wc -l

   # If > 1 user AND private repos: Action required
   ```

2. **Review configurations:**
   - Check if users have different access permissions
   - Identify private repositories in cache
   - Document compliance requirements

3. **Choose a solution:**
   - Simple deployment → Sidecar Pattern
   - Enterprise/Compliance → Namespace Isolation
   - Custom requirements → Application-Level Isolation

### Quick Mitigation

If you need immediate security improvement:

```bash
# Option A: Deploy sidecar pattern (1 hour)
kubectl apply -f examples/kubernetes-sidecar-secure.yaml

# Option B: Temporarily restrict to single tenant
# Add NetworkPolicy to limit access to single namespace
kubectl apply -f examples/single-tenant-network-policy.yaml
```

## Security Checklist

Before deploying Goblet in production:

### Configuration Security
- [ ] Authentication configured (OAuth2/OIDC)
- [ ] TLS enabled for client connections
- [ ] TLS configured for upstream connections
- [ ] Strong cipher suites enforced (TLS 1.3)

### Tenant Isolation
- [ ] Isolation strategy selected and documented
- [ ] Cross-tenant access tested (must fail)
- [ ] Cache directories have appropriate permissions
- [ ] File system quotas configured (if applicable)

### Data Protection
- [ ] Encrypted volumes for cache storage
- [ ] Backup and disaster recovery tested
- [ ] Cache eviction policy defined
- [ ] Compliance requirements documented

### Monitoring & Audit
- [ ] Access logging enabled
- [ ] Security events monitored
- [ ] Alerting configured for:
  - Authentication failures
  - Unauthorized access attempts
  - Unusual cache access patterns
- [ ] Audit log retention policy defined

### Operational Security
- [ ] Non-root container user configured
- [ ] Resource limits set
- [ ] Network policies enforced
- [ ] Security scanning in CI/CD
- [ ] Incident response plan documented

## Compliance Considerations

### SOC 2 Type II

**Key Controls:**
- CC6.1: Logical access controls → Tenant isolation
- CC6.6: Encryption of data at rest → Encrypted volumes
- CC6.7: Encryption of data in transit → TLS 1.3
- CC7.2: System monitoring → Audit logging

### ISO 27001

**Key Requirements:**
- A.9.4.1: Information access restriction → Authentication + isolation
- A.10.1.1: Cryptographic controls → TLS + volume encryption
- A.12.4.1: Event logging → Audit trails
- A.18.1.5: IT security in supplier relationships → Vendor assessment

### GDPR

**Key Provisions:**
- Article 32: Security of processing → Encryption + access controls
- Article 33: Breach notification → Monitoring + alerting
- Article 17: Right to erasure → Cache eviction capability
- Article 30: Records of processing activities → Audit logs

## Testing Security

### Test Cross-Tenant Isolation

```bash
# Deploy test environment
kubectl apply -f examples/security-test.yaml

# Test as Tenant A
export TOKEN_A=$(get-token-for tenant-a)
curl -H "Authorization: Bearer $TOKEN_A" \
  http://goblet/github.com/tenant-a/repo

# Test as Tenant B accessing Tenant A's repo
export TOKEN_B=$(get-token-for tenant-b)
curl -H "Authorization: Bearer $TOKEN_B" \
  http://goblet/github.com/tenant-a/repo

# Expected: 403 Forbidden or separate cache
```

### Penetration Testing

Recommended tests:
- Path traversal attempts
- Authentication bypass attempts
- Authorization bypass attempts
- Cross-tenant access attempts
- Cache poisoning attempts
- Resource exhaustion (DoS)

## Reporting Security Issues

If you discover a security vulnerability:

1. **Do not** open a public GitHub issue
2. Email security@example.com with:
   - Description of vulnerability
   - Steps to reproduce
   - Affected versions
   - Suggested remediation (if any)
3. Allow 90 days for patch before public disclosure

## Additional Resources

- [Isolation Strategies](isolation-strategies.md) - Detailed technical implementation
- [Multi-Tenant Deployment](multi-tenant-deployment.md) - Step-by-step deployment guide
- [Threat Model](threat-model.md) - Complete threat analysis
- [Architecture Decisions](../architecture/design-decisions.md) - Security architecture rationale

## Summary

**Key Takeaways:**

1. Default Goblet is safe for single-tenant deployments
2. Multi-tenant with private repos requires isolation
3. Sidecar pattern provides immediate security (deploy today)
4. Namespace isolation provides enterprise-grade security
5. Application-level isolation offers maximum flexibility

**Next Steps:**

- ✅ Single-tenant: Deploy with confidence
- ⚠️ Multi-tenant: Review [Isolation Strategies](isolation-strategies.md)
- 🚨 High-risk: Implement sidecar pattern immediately

For questions: See [Getting Help](../getting-started.md#getting-help)
