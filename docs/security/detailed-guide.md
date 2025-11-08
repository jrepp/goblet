# Security Considerations for Goblet

## ⚠️ CRITICAL: Multi-Tenant Security Warning

**Goblet's default configuration is UNSAFE for multi-tenant deployments with private repositories.**

### The Problem

By default, Goblet caches repositories using only `{host}/{repo-path}` as the cache key, with **no user or tenant isolation**. This creates a security vulnerability:

```
1. User Alice (authorized) → fetches github.com/company/private-repo
   → Cached at: /cache/github.com/company/private-repo

2. User Bob (unauthorized) → requests github.com/company/private-repo
   → Authenticates successfully (valid user)
   → Served from cache WITHOUT checking Bob's repo permissions
   → 🚨 Bob gains unauthorized access to private repository
```

### Who Is Affected?

You are affected if:
- ✅ Multiple users/tenants use the same Goblet instance
- ✅ Users access private repositories
- ✅ Users have different access permissions to repositories
- ✅ Use case: Terraform Cloud, risk scanning, multi-org SaaS

You are NOT affected if:
- ⬜ Single user/service account per instance (sidecar pattern)
- ⬜ Only public repositories
- ⬜ All users have identical access permissions

## Solutions

### Recommended Approaches (In Order of Preference)

#### 1. Sidecar Pattern (Simplest, Most Secure)

**Deploy one Goblet instance per user/workload as a sidecar container.**

```yaml
# Kubernetes Pod
containers:
  - name: app
  - name: goblet-sidecar  # Dedicated instance
    env:
      - name: GOBLET_ISOLATION_MODE
        value: "sidecar"
```

**Pros:**
- ✅ Perfect isolation (no code changes needed)
- ✅ Simple deployment model
- ✅ Works with existing Goblet

**Use for:** Terraform agents, CI/CD runners, per-workspace caching

**See:** `loadtest/kubernetes-sidecar-deployment.yaml`

---

#### 2. User-Scoped Cache Isolation

**Enable user-scoped isolation mode (requires code integration).**

```go
config := &goblet.IsolationConfig{
    Mode:         goblet.IsolationUser,
    UserClaimKey: "email",
}
```

Cache structure: `/cache/user-alice@company.com/github.com/org/repo`

**Pros:**
- ✅ Perfect isolation per user
- ✅ Simple logic

**Cons:**
- ❌ Cache duplication (higher storage)
- ❌ Requires code changes

**Use for:** Risk scanning, development environments

**See:** `examples/isolation-config-user.go`

---

#### 3. Tenant-Scoped Cache Isolation

**Enable tenant-scoped isolation mode (requires code integration).**

```go
config := &goblet.IsolationConfig{
    Mode:            goblet.IsolationTenant,
    TenantHeaderKey: "X-Tenant-ID",
}
```

Cache structure: `/cache/tenant-org1/github.com/org/repo`

**Pros:**
- ✅ Good isolation per organization
- ✅ Better cache efficiency than user-scoped

**Cons:**
- ❌ Users within tenant share cache (acceptable if intended)
- ❌ Requires code changes

**Use for:** Terraform Cloud (workspace isolation), SaaS platforms

**See:** `examples/isolation-config-tenant.go`

---

#### 4. Network Isolation (Deployment-Level)

**Deploy separate Goblet instances per tenant in isolated namespaces.**

```yaml
# Namespace: tenant-org1
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goblet
  namespace: tenant-org1  # Isolated

---
# NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-cross-tenant
spec:
  podSelector: {}
  policyTypes:
    - Ingress
```

**Pros:**
- ✅ Perfect isolation
- ✅ No code changes

**Cons:**
- ❌ Higher infrastructure cost
- ❌ Operational overhead

**Use for:** Compliance-sensitive workloads, dedicated tenants

**See:** `loadtest/kubernetes-sidecar-secure.yaml`

---

## Implementation Status

### ✅ Implemented (Available Now)

1. **Isolation Framework** - `isolation.go`
   - IsolationMode types (none, user, tenant, sidecar)
   - User/tenant identifier extraction
   - Cache path generation with isolation prefix
   - Configuration validation

2. **Example Configurations** - `examples/`
   - User-scoped isolation examples
   - Tenant-scoped isolation examples
   - Terraform Cloud integration

3. **Secure Deployment Manifests** - `loadtest/`
   - Kubernetes sidecar (secure)
   - Network policies
   - Security contexts
   - Resource quotas

4. **Documentation** - `loadtest/SECURITY_ISOLATION.md`
   - Threat model
   - Isolation strategies
   - Configuration guide
   - Migration path

### 🚧 Requires Integration (Future Work)

1. **ServerConfig Integration**
   - Add `IsolationConfig` field to `ServerConfig`
   - Wire isolation logic into cache path generation
   - Update `getManagedRepo()` in `managed_repository.go`

2. **Claims Propagation**
   - OIDC authorizer sets claims in request context
   - Claims available for isolation logic
   - Update `auth/oidc/authorizer.go`

3. **Testing**
   - Unit tests for isolation modes
   - Integration tests for cross-tenant access
   - Security test suite

4. **Encryption at Rest** (Optional)
   - Transparent encryption layer
   - KMS integration
   - Key rotation support

---

## Quick Start: Secure Deployment

### For Terraform Cloud (Tenant Isolation)

```bash
# 1. Build Goblet with isolation support
docker build -t goblet:secure .

# 2. Deploy with tenant isolation
kubectl apply -f loadtest/kubernetes-sidecar-secure.yaml

# 3. Configure Terraform to pass workspace ID
# In Terraform agent:
export TFC_WORKSPACE_ID="ws-abc123"
export GIT_CONFIG_COUNT=2
export GIT_CONFIG_KEY_0="http.proxy"
export GIT_CONFIG_VALUE_0="http://localhost:8080"
export GIT_CONFIG_KEY_1="http.extraHeader"
export GIT_CONFIG_VALUE_1="X-TFC-Workspace-ID: $TFC_WORKSPACE_ID"
```

### For Risk Scanning (User Isolation)

```bash
# 1. Configure user-scoped isolation
cat > config.yaml <<EOF
isolation:
  mode: user
  user_claim: email
EOF

# 2. Deploy
kubectl apply -f loadtest/kubernetes-sidecar-secure.yaml

# 3. Ensure OIDC claims include email
# Authorization will extract user from email claim
```

---

## Security Checklist

Before deploying Goblet with private repositories:

### Configuration
- [ ] Isolation mode configured (`sidecar`, `user`, or `tenant`)
- [ ] User/tenant identification mechanism validated
- [ ] Test cross-tenant access (should be blocked)
- [ ] Review security warning from `IsolationConfig.SecurityWarning()`

### Deployment
- [ ] Use non-root user (UID 1000)
- [ ] Read-only root filesystem (where possible)
- [ ] Drop all capabilities
- [ ] Network policies enforce isolation
- [ ] Resource limits prevent DoS
- [ ] Service account with minimal permissions

### Data Protection
- [ ] Cache directory has restrictive permissions (0700)
- [ ] Consider encryption at rest for sensitive repos
- [ ] Implement cache eviction for compliance (GDPR)
- [ ] Audit logging enabled with user context

### Monitoring
- [ ] Alert on authentication failures
- [ ] Monitor cache access patterns
- [ ] Track unauthorized access attempts
- [ ] Review audit logs regularly

### Testing
- [ ] Verify user A cannot access user B's cache
- [ ] Test with revoked credentials
- [ ] Validate tenant isolation
- [ ] Load test with multiple tenants

---

## Vulnerability Disclosure

If you discover a security vulnerability in Goblet, please email:
security@example.com (Update with actual contact)

Please include:
1. Description of the vulnerability
2. Steps to reproduce
3. Affected versions
4. Suggested fix (if any)

---

## Frequently Asked Questions

### Q: Is the default configuration secure?

**A:** Only for single-user deployments or public repositories. For multi-tenant with private repos: **NO**.

### Q: Can I use the current version in production with private repos?

**A:** Only if you deploy using the **sidecar pattern** (one instance per user/workload). Do NOT share a Goblet instance across users with current default configuration.

### Q: What's the safest option?

**A:** Sidecar pattern (one instance per pod/user). No code changes needed, perfect isolation.

### Q: Will isolation modes reduce cache efficiency?

**A:** Yes. User-scoped has lowest efficiency (most duplication). Tenant-scoped is better (shared within tenant). Sidecar is equivalent to user-scoped but simpler to deploy.

### Q: Do I need encryption at rest?

**A:** Depends on your threat model. If disk access is restricted and you trust the infrastructure, encryption may not be necessary. For highly sensitive repos or compliance requirements, consider encryption.

### Q: How do I migrate from shared to isolated cache?

**A:** See `loadtest/SECURITY_ISOLATION.md` → Migration Guide section. Generally: stop instance, backup cache, reconfigure isolation mode, restart. Cache will warm up fresh.

### Q: Can I use different isolation modes for different repos?

**A:** Not currently. Isolation mode is instance-wide. For mixed requirements, deploy multiple instances with different configurations.

---

## Additional Resources

- **Threat Model & Analysis:** `loadtest/SECURITY_ISOLATION.md`
- **Configuration Examples:** `examples/isolation-config-*.go`
- **Secure Deployments:** `loadtest/kubernetes-sidecar-secure.yaml`
- **Load Testing:** `loadtest/README.md`

---

## License

This security documentation is provided under the same Apache 2.0 license as Goblet.

**Disclaimer:** Use Goblet at your own risk. The maintainers are not responsible for security breaches resulting from misconfiguration or improper deployment.
