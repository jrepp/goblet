# Threat Model

Security threat analysis for Goblet deployments.

## Threat Categories

### 1. Cross-Tenant Data Access

**Threat:** User A accesses User B's cached private repositories

**Attack Vector:** Shared cache without tenant isolation

**Severity:** Critical (CVSS 8.1)

**Mitigation:** Implement [isolation strategies](isolation-strategies.md)

### 2. Cache Poisoning

**Threat:** Attacker injects malicious content into cache

**Attack Vector:** Compromised upstream or MITM

**Severity:** High

**Mitigation:**
- TLS for all upstream connections
- Verify upstream certificates
- Checksum validation

### 3. Unauthorized Access

**Threat:** Unauthenticated users access cache

**Attack Vector:** Missing or weak authentication

**Severity:** High

**Mitigation:**
- Enforce OAuth2/OIDC authentication
- Use strong tokens
- Regular token rotation

### 4. Data Exposure

**Threat:** Sensitive repository data exposed via filesystem

**Attack Vector:** Unauthorized filesystem access

**Severity:** Medium

**Mitigation:**
- Encrypted volumes
- Restrictive file permissions
- Pod security policies

### 5. Denial of Service

**Threat:** Cache exhaustion or resource starvation

**Attack Vector:** Malicious or excessive requests

**Severity:** Medium

**Mitigation:**
- Resource limits
- Rate limiting
- Cache quotas per tenant

## Security Controls

See [Detailed Security Guide](detailed-guide.md) for complete controls.

## Related Documentation

- [Security Overview](README.md)
- [Isolation Strategies](isolation-strategies.md)
- [Detailed Guide](detailed-guide.md)
