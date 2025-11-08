# Security & Tenant Isolation Design

## ⚠️ CRITICAL SECURITY FINDINGS

### Current State: UNSAFE for Multi-Tenant Deployments

**Vulnerability:** Private repository data leakage between users/tenants

**Attack Scenario:**
```
1. User Alice (has access) → fetches github.com/company/secrets
   → Cached at: /cache/github.com/company/secrets

2. User Bob (NO access) → requests github.com/company/secrets
   → Authenticates successfully (valid user)
   → Served from cache WITHOUT checking Bob's repo permissions
   → Bob gains unauthorized access to private repository
```

**Root Causes:**
1. ❌ Cache keyed only by `{host}/{repo-path}` - NO user/tenant identifier
2. ❌ Authorization validates "is user valid" NOT "can user access THIS repo"
3. ❌ Claims extracted from OIDC but never used for access control
4. ❌ No encryption at rest
5. ❌ Cached data served without re-checking permissions

**Code Locations:**
- `managed_repository.go:77` - Cache key without user context
- `http_proxy_server.go:49` - Authorization validates user, not repo access
- `managed_repository.go:434-444` - Serves cached data without re-auth

---

## Security Requirements for Multi-Tenant Deployment

### 1. Tenant Isolation
- ✅ **MUST**: Each tenant's cached data isolated from others
- ✅ **MUST**: Cache keys include tenant/user identifier
- ✅ **MUST**: No cross-tenant data access possible

### 2. Authorization Enforcement
- ✅ **MUST**: Verify user has permission for SPECIFIC repository
- ✅ **MUST**: Re-check permissions even when serving from cache
- ✅ **SHOULD**: Cache permission results (with TTL)

### 3. Data Protection
- ✅ **SHOULD**: Encrypt sensitive data at rest
- ✅ **MUST**: Secure file permissions on cache directories
- ✅ **SHOULD**: Support key rotation for encryption

### 4. Audit & Compliance
- ✅ **SHOULD**: Log all cache access with user identity
- ✅ **SHOULD**: Track which users accessed which repos
- ✅ **MUST**: Support cache eviction for compliance (GDPR, data retention)

### 5. Resource Limits
- ✅ **MUST**: Prevent one tenant from exhausting cache
- ✅ **SHOULD**: Per-tenant cache quotas
- ✅ **SHOULD**: LRU eviction per tenant

---

## Proposed Isolation Strategies

### Strategy 1: USER-SCOPED CACHE (Strongest Isolation)

**Design:**
```
Cache Layout:
/cache/
  ├── user-alice@company.com/
  │   └── github.com/org/repo-a/
  ├── user-bob@company.com/
  │   └── github.com/org/repo-b/
  └── user-charlie@company.com/
      └── github.com/org/repo-a/  (duplicate OK - isolated)
```

**Implementation:**
```go
// Cache key includes user identifier
userID := getUserIdentifier(r) // From OIDC claims or OAuth email
localDiskPath := filepath.Join(
    config.LocalDiskCacheRoot,
    sanitizeUserID(userID),
    u.Host,
    u.Path,
)
```

**Pros:**
- ✅ Perfect isolation - impossible to access other user's data
- ✅ No complex ACL logic needed
- ✅ Simple to implement
- ✅ Audit trail built-in (cache path = user)

**Cons:**
- ❌ Cache duplication (same repo cached multiple times)
- ❌ Higher storage costs
- ❌ Lower cache hit rate

**Use Cases:**
- Risk scanning services (different teams scan different repos)
- CI/CD with user-owned credentials
- Development environments

---

### Strategy 2: TENANT-SCOPED CACHE (Balanced)

**Design:**
```
Cache Layout:
/cache/
  ├── tenant-org1/
  │   └── github.com/org1/repo-a/
  ├── tenant-org2/
  │   └── github.com/org2/repo-b/
  └── shared/  (optional: for public repos)
      └── github.com/kubernetes/kubernetes/
```

**Implementation:**
```go
tenantID := getTenantIdentifier(r) // From OIDC groups claim or custom header

// For private repos: tenant-scoped
localDiskPath := filepath.Join(
    config.LocalDiskCacheRoot,
    "tenant-" + sanitizeTenantID(tenantID),
    u.Host,
    u.Path,
)

// For public repos: shared cache (optional optimization)
if isPublicRepo(u) {
    localDiskPath = filepath.Join(
        config.LocalDiskCacheRoot,
        "shared",
        u.Host,
        u.Path,
    )
}
```

**Pros:**
- ✅ Good isolation per organization/workspace
- ✅ Better cache efficiency within tenant
- ✅ Reduced storage vs user-scoped
- ✅ Can optimize with shared public cache

**Cons:**
- ❌ Requires tenant identification mechanism
- ❌ May not work if users span multiple tenants
- ❌ Need public/private repo detection

**Use Cases:**
- Terraform Cloud (workspace isolation)
- GitHub Apps (installation isolation)
- SaaS platforms (organization isolation)

---

### Strategy 3: ACL-BASED SHARED CACHE (Complex)

**Design:**
```
Cache Layout:
/cache/
  └── github.com/org/repo-a/
      ├── .git/  (bare repository)
      └── .acl   (access control list)
```

**ACL File Format:**
```json
{
  "repository": "github.com/org/repo-a",
  "allowed_users": ["alice@company.com", "bob@company.com"],
  "allowed_groups": ["org:engineering"],
  "cached_at": "2025-11-07T10:00:00Z",
  "ttl_seconds": 3600
}
```

**Implementation:**
```go
// Before serving from cache
func checkCacheACL(repoPath string, userClaims Claims) error {
    aclPath := filepath.Join(repoPath, ".acl")
    acl := loadACL(aclPath)

    if !acl.Allows(userClaims) {
        // Option A: Deny access
        return ErrUnauthorized

        // Option B: Verify with upstream API
        if !verifyUpstreamAccess(userClaims, repoURL) {
            return ErrUnauthorized
        }
        // Update ACL
        acl.AddUser(userClaims.Email)
        saveACL(aclPath, acl)
    }

    return nil
}
```

**Pros:**
- ✅ Best cache efficiency (shared cache)
- ✅ No duplication
- ✅ Lowest storage costs
- ✅ Fine-grained access control

**Cons:**
- ❌ Complex implementation
- ❌ Need upstream API access for verification
- ❌ ACL staleness issues (permissions change)
- ❌ Performance overhead (ACL checks)
- ❌ ACL cache poisoning risks

**Use Cases:**
- Large deployments with limited private repos
- When storage costs are critical
- When upstream has reliable API for access checks

---

### Strategy 4: SIDECAR-PER-TENANT (Deployment-Level Isolation)

**Design:**
```
Kubernetes Namespace: tenant-org1
  Pod: terraform-agent-1
    └── Sidecar: goblet (cache: /cache-org1)

Kubernetes Namespace: tenant-org2
  Pod: terraform-agent-2
    └── Sidecar: goblet (cache: /cache-org2)
```

**Implementation:**
- Deploy separate Goblet instances per tenant/namespace
- No code changes needed
- Network policies enforce isolation
- Each tenant has independent infrastructure

**Pros:**
- ✅ Perfect isolation (network + storage)
- ✅ No code changes required
- ✅ Can use existing single-tenant Goblet
- ✅ Kubernetes-native security model

**Cons:**
- ❌ Higher infrastructure costs
- ❌ More operational overhead
- ❌ Resource overhead per tenant
- ❌ Doesn't work for shared services

**Use Cases:**
- SaaS with dedicated namespaces per customer
- Compliance requirements (data residency)
- When infrastructure costs are acceptable

---

## Recommended Strategy by Use Case

| Use Case | Recommended Strategy | Rationale |
|----------|---------------------|-----------|
| **Terraform Cloud** | Tenant-Scoped Cache | Workspace isolation, good cache efficiency |
| **Risk Scanning SaaS** | User-Scoped Cache | Different teams, different repos, isolation critical |
| **CI/CD Shared Service** | ACL-Based Shared | Many users, limited repos, storage optimization |
| **Enterprise Internal** | Sidecar-Per-Tenant | Strong isolation, compliance requirements |
| **Development/Testing** | User-Scoped Cache | Simple, safe, storage not critical |

---

## Implementation Plan

### Phase 1: Quick Wins (Security Hardening)

**Goal:** Make current system safer without breaking changes

1. **Add isolation mode configuration:**
   ```go
   type IsolationMode string
   const (
       IsolationNone     IsolationMode = "none"     // Current behavior
       IsolationUser     IsolationMode = "user"     // User-scoped
       IsolationTenant   IsolationMode = "tenant"   // Tenant-scoped
       IsolationSidecar  IsolationMode = "sidecar"  // Single-user (default)
   )
   ```

2. **Document security risks:**
   - Add WARNING in README
   - Add security guide
   - Add deployment recommendations

3. **Add cache isolation helper:**
   ```go
   func getCachePath(config ServerConfig, claims Claims, repoURL url.URL) string {
       switch config.IsolationMode {
       case IsolationUser:
           return filepath.Join(config.CacheRoot, claims.Email, repoURL.Host, repoURL.Path)
       case IsolationTenant:
           tenant := getTenantFromClaims(claims)
           return filepath.Join(config.CacheRoot, tenant, repoURL.Host, repoURL.Path)
       default:
           return filepath.Join(config.CacheRoot, repoURL.Host, repoURL.Path)
       }
   }
   ```

### Phase 2: Tenant Isolation (Recommended)

**Goal:** Support multi-tenant deployments safely

1. **Implement tenant-scoped cache keys**
2. **Add tenant extraction from OIDC claims**
3. **Add per-tenant cache quotas**
4. **Add tenant isolation tests**

### Phase 3: Encryption at Rest (Optional)

**Goal:** Protect sensitive data on disk

1. **Add encryption layer using age/crypto**
2. **Support key management (KMS)**
3. **Transparent encryption/decryption**

### Phase 4: ACL-Based Sharing (Future)

**Goal:** Optimize storage with safe sharing

1. **Implement ACL checking**
2. **Add upstream API integration**
3. **Add ACL caching with TTL**

---

## Configuration Examples

### User-Scoped Isolation

```yaml
# goblet-config.yaml
isolation_mode: user
cache_root: /cache
auth:
  type: oidc
  issuer_url: https://auth.company.com
  client_id: goblet
  user_claim: email  # Use email as user identifier
```

### Tenant-Scoped Isolation

```yaml
# goblet-config.yaml
isolation_mode: tenant
cache_root: /cache
auth:
  type: oidc
  issuer_url: https://auth.company.com
  client_id: goblet
  tenant_claim: groups  # Extract tenant from groups claim
  tenant_regex: "^org:(.*)"  # Parse "org:engineering" -> "engineering"
```

### Sidecar Mode (Single-User)

```yaml
# goblet-config.yaml
isolation_mode: sidecar  # Default
cache_root: /cache
auth:
  type: google_oauth2
  service_account: agent@project.iam.gserviceaccount.com
```

---

## Deployment Patterns with Security

### INSECURE: Shared Goblet with No Isolation ❌

```
Load Balancer
    |
    +-- Goblet (shared cache)
            |
    +-------+-------+
    |       |       |
  User A  User B  User C
```

**Risk:** User A can access User B's private repos
**Verdict:** UNSAFE - Do not use in production

---

### SECURE: User-Scoped Cache ✅

```
Load Balancer
    |
    +-- Goblet (user-scoped cache)
            |
    Cache Structure:
    /cache/
      ├── alice@company.com/
      ├── bob@company.com/
      └── charlie@company.com/
```

**Risk:** None - perfect isolation
**Verdict:** SAFE - Recommended for risk scanning

---

### SECURE: Tenant-Scoped Cache ✅

```
Load Balancer
    |
    +-- Goblet (tenant-scoped cache)
            |
    Cache Structure:
    /cache/
      ├── tenant-org1/
      ├── tenant-org2/
      └── tenant-org3/
```

**Risk:** Users within same tenant share cache (acceptable if intended)
**Verdict:** SAFE - Recommended for Terraform Cloud

---

### SECURE: Sidecar Per Tenant ✅

```
Tenant Org1 Namespace:
  Pod-1: App + Goblet-Sidecar (cache-1)
  Pod-2: App + Goblet-Sidecar (cache-2)

Tenant Org2 Namespace:
  Pod-3: App + Goblet-Sidecar (cache-3)
  Pod-4: App + Goblet-Sidecar (cache-4)
```

**Risk:** None - network + storage isolation
**Verdict:** SAFE - Best for compliance-sensitive workloads

---

## Security Checklist

Before deploying Goblet in production with private repositories:

- [ ] **Isolation Mode Configured**: Set `isolation_mode` appropriately
- [ ] **User/Tenant Identification**: Ensure claims extraction works
- [ ] **Authorization Tested**: Verify cross-tenant access blocked
- [ ] **File Permissions**: Ensure cache directories have restrictive permissions
- [ ] **Network Policies**: Implement if using Kubernetes
- [ ] **Audit Logging**: Enable access logs with user context
- [ ] **Encryption at Rest**: Consider for highly sensitive repos
- [ ] **Cache Eviction**: Implement for compliance (GDPR, retention)
- [ ] **Monitoring**: Alert on unauthorized access attempts
- [ ] **Documentation**: Document security model for users

---

## Testing Isolation

### Test User Isolation

```bash
# User A fetches private repo
curl -H "Authorization: Bearer $TOKEN_USER_A" \
  http://goblet:8080/github.com/company/secrets/info/refs

# Verify cached at /cache/user-a@company.com/...
ls /cache/user-a@company.com/github.com/company/secrets

# User B attempts to access same repo
curl -H "Authorization: Bearer $TOKEN_USER_B" \
  http://goblet:8080/github.com/company/secrets/info/refs

# Should either:
# 1. Return 403 Forbidden (if User B has no access)
# 2. Cache separately at /cache/user-b@company.com/... (if has access)

# MUST NOT serve from User A's cache
ls /cache/user-b@company.com/  # Should be separate or empty
```

### Test Tenant Isolation

```python
# Python test script
def test_tenant_isolation():
    # Tenant 1 fetches repo
    headers_t1 = {"Authorization": f"Bearer {token_tenant1}"}
    resp1 = requests.get(f"{goblet_url}/github.com/company/repo", headers=headers_t1)
    assert resp1.status_code == 200

    # Tenant 2 attempts access
    headers_t2 = {"Authorization": f"Bearer {token_tenant2}"}
    resp2 = requests.get(f"{goblet_url}/github.com/company/repo", headers=headers_t2)

    # Should fail if Tenant 2 has no access
    assert resp2.status_code == 403

    # Verify separate cache paths
    assert os.path.exists("/cache/tenant-1/github.com/company/repo")
    assert not os.path.exists("/cache/tenant-2/github.com/company/repo")
```

---

## Encryption at Rest (Future Enhancement)

### Design

```go
type EncryptedStorage struct {
    backend    Storage
    keyManager KeyManager
}

func (e *EncryptedStorage) Write(path string, data []byte) error {
    encryptedData := e.keyManager.Encrypt(data)
    return e.backend.Write(path, encryptedData)
}

func (e *EncryptedStorage) Read(path string) ([]byte, error) {
    encryptedData, err := e.backend.Read(path)
    if err != nil {
        return nil, err
    }
    return e.keyManager.Decrypt(encryptedData)
}
```

### Key Management Options

1. **Local Key File**: Simple, for single-instance
2. **Environment Variable**: For containers
3. **KMS Integration**: AWS KMS, Google Cloud KMS, HashiCorp Vault
4. **Per-Tenant Keys**: Different key per tenant for isolation

---

## Migration Guide

### From Shared Cache to User-Scoped

```bash
# 1. Stop Goblet
systemctl stop goblet

# 2. Backup existing cache
mv /cache /cache.backup

# 3. Update configuration
cat > /etc/goblet/config.yaml <<EOF
isolation_mode: user
cache_root: /cache
EOF

# 4. Start Goblet (cache will warm up fresh)
systemctl start goblet

# 5. Monitor for cache misses (expected during warm-up)
curl http://localhost:8080/metrics | grep cache_miss
```

### From Single-Instance to Tenant-Scoped

```bash
# 1. Identify tenants in existing cache
# 2. Reorganize cache by tenant (if reusing data)
mkdir -p /cache-new/tenant-{1,2,3}
# ... migrate data ...

# 3. Update config and restart
# 4. Verify isolation with tests
```

---

## Conclusion

**Current State:** Goblet is UNSAFE for multi-tenant private repository scenarios

**Required Changes:**
1. Implement isolation mode configuration
2. Add user/tenant-scoped cache keys
3. Document security model clearly
4. Add isolation tests

**Recommended Approach:**
- **Terraform Cloud**: Tenant-scoped cache + sidecar deployment
- **Risk Scanning**: User-scoped cache
- **Enterprise**: Sidecar-per-tenant with network policies

**Next Steps:**
1. Review this document with team
2. Choose isolation strategy for your use case
3. Implement isolation mode (see implementation PR)
4. Test thoroughly before production deployment
