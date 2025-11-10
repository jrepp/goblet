# RFC-002: GitHub OAuth and Multi-Tenancy Authentication Architecture

**Status:** Draft
**Author:** System Architecture Team
**Created:** 2025-11-07
**Related:** PR #7, RFC-001 (Secure Multi-Tenant Cache)

## Executive Summary

This RFC provides a comprehensive analysis of GitHub Enterprise and public GitHub OAuth support in the context of multi-tenant Git caching proxy deployments. It examines the authentication models, authorization flows, token management strategies, and isolation requirements necessary for secure multi-tenant operations.

**Key Finding:** PR #7's changes (upstream URL-aware token generation + dynamic token type support) are **critical enablers** for secure multi-tenant GitHub caching but are **not sufficient** on their own. Complete multi-tenant isolation requires integration with request-level authorization and cache partitioning.

## Table of Contents

1. [Background](#background)
2. [Current State Analysis](#current-state-analysis)
3. [GitHub Authentication Models](#github-authentication-models)
4. [Multi-Tenancy Isolation Requirements](#multi-tenancy-isolation-requirements)
5. [Technical Architecture](#technical-architecture)
6. [Implementation Strategy](#implementation-strategy)
7. [Tradeoffs and Recommendations](#tradeoffs-and-recommendations)
8. [Security Considerations](#security-considerations)
9. [Migration Path](#migration-path)

---

## Background

### Problem Statement

Goblet currently implements a Git caching proxy with two distinct authentication layers:

1. **Client → Goblet Authentication** (RequestAuthorizer)
   - Who can access the cache?
   - Uses OIDC/Bearer tokens
   - Identity: email, groups, subject

2. **Goblet → Upstream Authentication** (TokenSource)
   - How does Goblet authenticate to upstream Git servers?
   - Uses OAuth2 tokens
   - **Current limitation:** Single token for all upstreams

### The Challenge

In multi-tenant scenarios with private repositories:

```
Tenant A (Org: acme-corp)          Tenant B (Org: megacorp)
  ↓ (OIDC: alice@acme.com)           ↓ (OIDC: bob@mega.com)
  ↓                                   ↓
  Goblet Cache
  ↓ (Need: acme-corp token)      ↓ (Need: megacorp token)
  ↓                                   ↓
github.com/acme-corp/repo       github.com/megacorp/repo
```

**Problems without PR #7:**
- ❌ Single `TokenSource` for all organizations
- ❌ Cannot generate org-specific tokens
- ❌ Hardcoded "Bearer" breaks GitHub Enterprise

**Problems even with PR #7:**
- ⚠️ No automatic tenant → upstream mapping
- ⚠️ No cache isolation enforcement
- ⚠️ No token scope validation

---

## Current State Analysis

### Existing Architecture

```mermaid
sequenceDiagram
    participant Client
    participant Goblet
    participant GitHub

    Client->>Goblet: HTTP Request + Authorization: Bearer <OIDC>
    Note over Goblet: RequestAuthorizer validates OIDC token
    Note over Goblet: Extracts: email, groups, sub
    Goblet->>GitHub: git-upload-pack + Authorization: <OAuth2>
    Note over Goblet: TokenSource(upstreamURL) generates token
    GitHub-->>Goblet: Pack data
    Goblet-->>Client: Cached response
```

### Current Authentication Flows

#### 1. Client Authentication (Inbound)

**Location:** `auth/oidc/authorizer.go`

```go
type Claims struct {
    Email         string   `json:"email"`
    EmailVerified bool     `json:"email_verified"`
    Name          string   `json:"name"`
    Groups        []string `json:"groups"`
    Subject       string   `json:"sub"`
}

func (a *Authorizer) AuthorizeRequest(r *http.Request) error {
    token := ExtractBearerToken(r)
    idToken, err := a.verifier.VerifyIDToken(r.Context(), token)
    claims, err := GetClaims(idToken)
    // Store claims in context
    ctx := context.WithValue(r.Context(), claimsKey, claims)
    *r = *r.WithContext(ctx)
    return nil
}
```

**Capabilities:**
- ✅ Verifies user identity via OIDC
- ✅ Extracts user metadata (email, groups)
- ✅ Stores claims in request context
- ❌ Does NOT enforce repository-level authorization
- ❌ Does NOT map user to upstream credentials

#### 2. Upstream Authentication (Outbound)

**Location:** `goblet.go`, `managed_repository.go`

**Before PR #7:**
```go
type ServerConfig struct {
    TokenSource oauth2.TokenSource  // Single token for ALL upstreams
}

// In managed_repository.go
t, err := r.config.TokenSource.Token()
req.Header.Add("Authorization", "Bearer "+t.AccessToken)
```

**After PR #7:**
```go
type ServerConfig struct {
    TokenSource func(upstreamURL *url.URL) (*oauth2.Token, error)
}

// In managed_repository.go
t, err := r.config.TokenSource(r.upstreamURL)
if t.AccessToken != "" {
    req.Header.Add("Authorization", t.Type()+" "+t.AccessToken)
}
```

**Improvements from PR #7:**
- ✅ Upstream URL passed to token generator
- ✅ Dynamic token type (Bearer vs Basic)
- ✅ Enables org-specific token generation
- ✅ GitHub Enterprise PAT support

---

## GitHub Authentication Models

### 1. GitHub.com (Public GitHub)

#### Personal Access Tokens (PATs)

**Classic PATs:**
```
Format: ghp_xxxxxxxxxxxxxxxxxxxx
Type: Bearer (or Basic for GHE)
Scope: User-level permissions
Max: 1 year expiration
```

**Fine-Grained PATs:**
```
Format: github_pat_xxxxxxxxxxxxxxxxxxxx
Type: Bearer
Scope: Repository-specific permissions
Max: 1 year expiration
Per-organization access control
```

**Pros:**
- Simple to generate
- No app registration needed
- Per-repo fine-grained permissions

**Cons:**
- User-scoped (not organization)
- Manual rotation required
- Cannot distinguish between tenants
- Requires secure storage

**Multi-Tenant Viability:** ⚠️ Limited
- Cannot map user identity to PAT
- Each tenant needs separate PAT
- No automatic rotation

#### GitHub Apps

**Installation Tokens:**
```
Format: ghs_xxxxxxxxxxxxxxxxxxxx
Type: Bearer
Scope: Per-installation (org-level)
Expiration: 1 hour (automatic)
```

**Architecture:**
```
GitHub App (app_id: 123456)
  ├── Installation 1 (org: acme-corp, id: 111)
  ├── Installation 2 (org: megacorp, id: 222)
  └── Installation 3 (org: startup-co, id: 333)

Each installation → independent token
```

**Token Generation:**
```go
func getInstallationToken(orgName string) (*oauth2.Token, error) {
    // 1. Create JWT signed with app private key
    jwt := createJWT(appID, privateKey)

    // 2. Look up installation ID for org
    installationID := getInstallationID(orgName)

    // 3. Request installation token
    token := exchangeJWTForToken(jwt, installationID)

    return &oauth2.Token{
        AccessToken: token,
        TokenType:   "Bearer",
        Expiry:      time.Now().Add(1 * time.Hour),
    }, nil
}
```

**Pros:**
- ✅ Automatic token rotation (1 hour)
- ✅ Organization-scoped
- ✅ Audit log per installation
- ✅ Fine-grained repository permissions
- ✅ No user credentials needed

**Cons:**
- Requires app registration and approval
- Complex setup (JWT signing)
- Rate limits per installation

**Multi-Tenant Viability:** ✅ Excellent
- Perfect isolation (org → installation → token)
- Automatic expiration
- Audit trail

#### OAuth Apps

**OAuth Access Tokens:**
```
Format: gho_xxxxxxxxxxxxxxxxxxxx
Type: Bearer
Scope: User permissions on behalf of user
Expiration: No automatic expiry
```

**Pros:**
- User-level authorization
- Can act on behalf of user

**Cons:**
- ❌ User must be online to authorize
- ❌ No organization-level control
- ❌ Manual token management

**Multi-Tenant Viability:** ❌ Poor
- Requires user interaction
- Not suitable for server-to-server

### 2. GitHub Enterprise Server (GHE)

#### Key Differences

**Authentication:**
```
Public GitHub:  Authorization: Bearer ghp_xxxx
GHE:           Authorization: Basic <base64(username:token)>
               OR
               Authorization: token ghp_xxxx
```

**Token Type Handling:**
```go
// PR #7 enables this:
token := &oauth2.Token{
    AccessToken: "ghp_enterprise_token",
    TokenType:   "Basic",  // GHE expects Basic
}

// managed_repository.go now uses:
header := "Authorization: " + t.Type() + " " + t.AccessToken
// Result: "Authorization: Basic ghp_enterprise_token"
```

**SAML/LDAP Integration:**
- GHE often uses SAML SSO
- Token generation may require SAML assertion
- Additional complexity for token mapping

---

## Multi-Tenancy Isolation Requirements

### Security Requirements Matrix

| Requirement | Current State | With PR #7 | Full Implementation |
|-------------|---------------|------------|---------------------|
| **Client Authentication** | ✅ OIDC | ✅ OIDC | ✅ OIDC |
| **Client Authorization** | ❌ No repo-level checks | ❌ No repo-level checks | ✅ Per-repo ACL |
| **Upstream Token Selection** | ❌ Single token | ✅ URL-based | ✅ Tenant-aware |
| **Cache Isolation** | ❌ Shared cache | ❌ Shared cache | ✅ Partitioned |
| **Token Type Support** | ❌ Bearer only | ✅ Dynamic | ✅ Dynamic |
| **Audit Logging** | ⚠️ Partial | ⚠️ Partial | ✅ Complete |

### Threat Model

#### T1: Cross-Tenant Cache Access

**Scenario:**
```
1. Alice (tenant-a) requests: github.com/acme/secret-repo
2. Goblet caches to: /cache/github.com/acme/secret-repo
3. Bob (tenant-b) requests: github.com/acme/secret-repo
4. Goblet serves cached data to Bob (❌ UNAUTHORIZED)
```

**Current Mitigation:** None
**With PR #7:** None (PR #7 doesn't address cache isolation)
**Required:** Cache partitioning by tenant ID

#### T2: Token Misuse

**Scenario:**
```
1. Goblet has token for org-a
2. User from org-b requests: github.com/org-a/repo
3. Goblet uses org-a token to fetch repo
4. User gains access to org-a data via org-b credentials
```

**Current Mitigation:** None
**With PR #7:** Enables org-specific tokens but doesn't enforce mapping
**Required:** Authorization layer to validate user→org→token

#### T3: Token Leakage

**Scenario:**
```
1. Installation token expires
2. Stale token remains in memory/logs
3. Attacker extracts token from logs
4. 1-hour window for misuse
```

**Current Mitigation:** None
**With PR #7:** None (token storage unchanged)
**Required:** Secure token storage, automatic rotation, audit logging

---

## Technical Architecture

### Proposed Multi-Tenant Authentication Flow

```mermaid
sequenceDiagram
    participant Client
    participant Goblet
    participant AuthZ as Authorization Layer
    participant TokenMgr as Token Manager
    participant GitHub

    Client->>Goblet: git fetch + Auth: Bearer <OIDC>
    Goblet->>AuthZ: ValidateRequest(claims, repoURL)
    Note over AuthZ: Extract tenant ID from claims
    Note over AuthZ: Check if tenant can access repo
    AuthZ-->>Goblet: OK (tenant_id, org_name)

    Goblet->>TokenMgr: GetToken(upstreamURL, tenant_id)
    Note over TokenMgr: Map tenant → GitHub org → installation
    Note over TokenMgr: Generate/refresh token for org
    TokenMgr-->>Goblet: Token (type, value, expiry)

    Goblet->>GitHub: git-upload-pack + Auth: <token>
    GitHub-->>Goblet: Pack data
    Note over Goblet: Cache in tenant-specific path
    Note over Goblet: /cache/<tenant_id>/<repo>
    Goblet-->>Client: Response
```

### Component Design

#### 1. Authorization Layer

**Purpose:** Enforce repository-level access control

```go
type Authorizer interface {
    // ValidateAccess checks if a user can access a repository
    ValidateAccess(ctx context.Context, claims *oidc.Claims, repoURL *url.URL) (*AuthzDecision, error)
}

type AuthzDecision struct {
    Allowed      bool
    TenantID     string     // tenant-a, tenant-b
    OrgName      string     // acme-corp, megacorp
    RepoAccess   []string   // ["read", "clone"]
    DenialReason string
}

type PolicyEngine struct {
    // Map user claims → tenant → allowed repos
    policies map[string]*TenantPolicy
}

type TenantPolicy struct {
    TenantID string
    GitHubOrgs []string
    AllowedRepos []string  // Patterns: "github.com/acme/*"
    DeniedRepos  []string  // Patterns: "github.com/acme/secret-*"
}

func (pe *PolicyEngine) ValidateAccess(
    ctx context.Context,
    claims *oidc.Claims,
    repoURL *url.URL,
) (*AuthzDecision, error) {
    // 1. Extract tenant ID from claims
    tenantID := extractTenantFromEmail(claims.Email)
    // alice@acme-corp.com → tenant_id: acme-corp

    // 2. Load tenant policy
    policy := pe.policies[tenantID]
    if policy == nil {
        return &AuthzDecision{
            Allowed: false,
            DenialReason: "no policy for tenant",
        }, nil
    }

    // 3. Check if repo matches allowed patterns
    allowed := matchesPatterns(repoURL.String(), policy.AllowedRepos)
    denied := matchesPatterns(repoURL.String(), policy.DeniedRepos)

    if denied {
        return &AuthzDecision{
            Allowed: false,
            DenialReason: "repo explicitly denied",
        }, nil
    }

    if !allowed {
        return &AuthzDecision{
            Allowed: false,
            DenialReason: "repo not in allow list",
        }, nil
    }

    // 4. Extract GitHub org from URL
    orgName := extractGitHubOrg(repoURL)
    // github.com/acme-corp/repo → org: acme-corp

    return &AuthzDecision{
        Allowed: true,
        TenantID: tenantID,
        OrgName: orgName,
        RepoAccess: []string{"read", "clone"},
    }, nil
}
```

#### 2. Token Manager

**Purpose:** Generate org-specific tokens with proper lifecycle

```go
type TokenManager interface {
    GetToken(upstreamURL *url.URL, tenantID string) (*oauth2.Token, error)
}

type GitHubAppTokenManager struct {
    appID          int64
    privateKey     *rsa.PrivateKey
    installations  map[string]int64  // org_name → installation_id
    tokenCache     map[string]*cachedToken
    mu             sync.RWMutex
}

type cachedToken struct {
    token      *oauth2.Token
    expiry     time.Time
    orgName    string
    tenantID   string
}

func (tm *GitHubAppTokenManager) GetToken(
    upstreamURL *url.URL,
    tenantID string,
) (*oauth2.Token, error) {
    // 1. Extract org from URL
    orgName := extractGitHubOrg(upstreamURL)
    // github.com/acme-corp/repo → acme-corp

    // 2. Check cache
    cacheKey := fmt.Sprintf("%s:%s", tenantID, orgName)
    tm.mu.RLock()
    cached := tm.tokenCache[cacheKey]
    tm.mu.RUnlock()

    if cached != nil && time.Now().Before(cached.expiry.Add(-5*time.Minute)) {
        // Return cached token (with 5min safety margin)
        return cached.token, nil
    }

    // 3. Look up installation ID
    installationID, ok := tm.installations[orgName]
    if !ok {
        return nil, fmt.Errorf("no GitHub App installation for org: %s", orgName)
    }

    // 4. Generate JWT for GitHub App authentication
    jwt, err := tm.generateJWT()
    if err != nil {
        return nil, fmt.Errorf("failed to generate JWT: %w", err)
    }

    // 5. Request installation token
    token, err := tm.exchangeJWTForInstallationToken(jwt, installationID)
    if err != nil {
        return nil, fmt.Errorf("failed to get installation token: %w", err)
    }

    // 6. Cache token
    tm.mu.Lock()
    tm.tokenCache[cacheKey] = &cachedToken{
        token:    token,
        expiry:   token.Expiry,
        orgName:  orgName,
        tenantID: tenantID,
    }
    tm.mu.Unlock()

    return token, nil
}

func (tm *GitHubAppTokenManager) generateJWT() (string, error) {
    // Create JWT claims
    now := time.Now()
    claims := jwt.MapClaims{
        "iat": now.Unix(),
        "exp": now.Add(10 * time.Minute).Unix(),
        "iss": strconv.FormatInt(tm.appID, 10),
    }

    // Sign with app private key
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    return token.SignedString(tm.privateKey)
}

func (tm *GitHubAppTokenManager) exchangeJWTForInstallationToken(
    jwtToken string,
    installationID int64,
) (*oauth2.Token, error) {
    // POST to GitHub API
    url := fmt.Sprintf(
        "https://api.github.com/app/installations/%d/access_tokens",
        installationID,
    )

    req, _ := http.NewRequest("POST", url, nil)
    req.Header.Set("Authorization", "Bearer "+jwtToken)
    req.Header.Set("Accept", "application/vnd.github+json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Token     string    `json:"token"`
        ExpiresAt time.Time `json:"expires_at"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return &oauth2.Token{
        AccessToken: result.Token,
        TokenType:   "Bearer",
        Expiry:      result.ExpiresAt,
    }, nil
}
```

#### 3. Cache Partitioning

**Purpose:** Isolate cached data per tenant

```go
func (ic *IsolationConfig) GetCachePath(
    r *http.Request,
    cacheRoot string,
    repoURL *url.URL,
) (string, error) {
    // Extract tenant ID from request context
    claims, ok := oidc.GetClaimsFromContext(r.Context())
    if !ok {
        return "", fmt.Errorf("no claims in context")
    }

    tenantID := extractTenantFromEmail(claims.Email)
    // alice@acme-corp.com → tenant_id: acme-corp

    // Construct tenant-isolated cache path
    basePath := filepath.Join(
        repoURL.Host,
        repoURL.Path,
    )

    return filepath.Join(cacheRoot, tenantID, basePath), nil
}
```

### Integration with PR #7

**PR #7 provides the foundation:**

```go
// In goblet-server/main.go
TokenSource: func(upstreamURL *url.URL) (*oauth2.Token, error) {
    // BEFORE PR #7: Could not access upstreamURL
    // AFTER PR #7: Can generate org-specific tokens

    // Extract tenant from request context (requires additional work)
    tenantID := getTenantFromContext()

    // Use TokenManager to get org-specific token
    return tokenManager.GetToken(upstreamURL, tenantID)
}
```

**What PR #7 enables:**
1. ✅ Upstream URL available for token generation
2. ✅ Can extract GitHub org from URL
3. ✅ Can map org → installation → token
4. ✅ Dynamic token type for GHE support

**What still needs implementation:**
1. ❌ Pass tenant context to TokenSource
2. ❌ Implement TokenManager
3. ❌ Integrate Authorization Layer
4. ❌ Implement cache partitioning

---

## Implementation Strategy

### Phase 1: Foundation (PR #7) ✅ COMPLETE

**Goal:** Enable upstream-aware token generation

**Completed:**
- Modified `TokenSource` signature to accept `upstreamURL`
- Updated all token generation call sites
- Dynamic token type support (Bearer vs Basic)

### Phase 2: Authorization Layer

**Goal:** Enforce repository-level access control

**Tasks:**
1. Define `Authorizer` interface
2. Implement `PolicyEngine` with tenant policies
3. Integrate with `http_proxy_server.go`
4. Extract and validate tenant from OIDC claims
5. Validate repo access per tenant
6. Add audit logging for authorization decisions

**Estimated Effort:** 2-3 weeks

**Files to modify:**
- `http_proxy_server.go` - Add authorization check
- `auth/authz/` - New package for authorization
- `auth/oidc/authorizer.go` - Extract tenant ID

### Phase 3: Token Manager with GitHub App Support

**Goal:** Implement org-specific token generation

**Tasks:**
1. Implement `TokenManager` interface
2. Create `GitHubAppTokenManager` with JWT signing
3. Configuration for GitHub App (app_id, private_key, installations)
4. Token caching with automatic refresh
5. Integration with `TokenSource` function
6. Support for fallback to PATs

**Estimated Effort:** 3-4 weeks

**Files to modify:**
- `auth/tokens/` - New package for token management
- `goblet-server/main.go` - Configure TokenManager
- Configuration file - Add GitHub App settings

### Phase 4: Cache Partitioning

**Goal:** Isolate cache per tenant

**Tasks:**
1. Integrate isolation.go into main codebase
2. Modify cache path generation to include tenant ID
3. Update all cache read/write operations
4. Migration tool for existing cache
5. Testing with multiple tenants

**Estimated Effort:** 2-3 weeks

**Files to modify:**
- `managed_repository.go` - Use partitioned cache paths
- `isolation.go` - Move from prototype to pkg/
- Cache migration tool

### Phase 5: Security Hardening

**Goal:** Production-ready security

**Tasks:**
1. Secure token storage (encrypted at rest)
2. Token rotation automation
3. Comprehensive audit logging
4. Rate limiting per tenant
5. Security testing and penetration testing

**Estimated Effort:** 2-3 weeks

---

## Tradeoffs and Recommendations

### Token Management Strategies

#### Option 1: GitHub Apps (RECOMMENDED)

**Pros:**
- ✅ Automatic token rotation (1 hour expiry)
- ✅ Organization-scoped isolation
- ✅ Fine-grained repository permissions
- ✅ Audit trail per installation
- ✅ No user credentials required

**Cons:**
- Requires GitHub App registration per tenant
- Complex setup (JWT signing, private key management)
- Rate limits per installation (5000 req/hour)

**Use Case:** Multi-tenant SaaS with many organizations

**Recommendation:** **Primary choice for production multi-tenant deployments**

#### Option 2: Fine-Grained PATs

**Pros:**
- ✅ Repository-specific permissions
- ✅ Simple to generate
- ✅ Per-organization control

**Cons:**
- ❌ Manual rotation (max 1 year)
- ❌ User-scoped (not org-scoped)
- ❌ Requires secure storage
- ❌ Must be generated per tenant

**Use Case:** Small number of tenants with manual management

**Recommendation:** **Fallback option or development/testing**

#### Option 3: Classic PATs

**Pros:**
- ✅ Simplest to implement
- ✅ Works with GHE (with Basic auth)

**Cons:**
- ❌ Broad permissions
- ❌ Manual rotation
- ❌ No tenant isolation

**Use Case:** Single-tenant deployments, GHE legacy

**Recommendation:** **Not recommended for multi-tenant**

### Cache Isolation Strategies

#### Option 1: Tenant-Partitioned Cache (RECOMMENDED)

**Design:**
```
/cache/
  ├── tenant-acme/
  │   └── github.com/acme-corp/repo1
  └── tenant-mega/
      └── github.com/megacorp/repo2
```

**Pros:**
- ✅ Perfect isolation
- ✅ Simple to implement
- ✅ Easy to audit

**Cons:**
- Higher storage usage (no sharing)
- Duplicate data for public repos

**Recommendation:** **Primary choice for security-critical deployments**

#### Option 2: Shared Cache with ACL

**Design:**
```
/cache/
  └── github.com/acme-corp/repo1
      + ACL: [tenant-acme]
```

**Pros:**
- Lower storage usage
- Shared public repos

**Cons:**
- ❌ Complex ACL management
- ❌ Risk of ACL misconfiguration
- ❌ Performance overhead

**Recommendation:** **Not recommended - complexity outweighs benefits**

### Deployment Patterns

#### Pattern 1: Sidecar with GitHub Apps

**Architecture:**
```
Tenant A Namespace
  └── Pod
      ├── App Container
      └── Goblet Sidecar
          - GitHub App Installation: acme-corp
          - Cache: /cache/tenant-a/

Tenant B Namespace
  └── Pod
      ├── App Container
      └── Goblet Sidecar
          - GitHub App Installation: megacorp
          - Cache: /cache/tenant-b/
```

**Security:** ✅ Excellent (namespace + network + cache isolation)
**Cost:** High (separate Goblet per tenant)
**Complexity:** Low

**Recommendation:** **Best for regulated industries, compliance requirements**

#### Pattern 2: Shared Goblet with Authorization Layer

**Architecture:**
```
                 Shared Goblet Instance
                          |
        +-----------------+------------------+
        |                                    |
    Tenant A                             Tenant B
    (GitHub App: acme)                   (GitHub App: mega)
    (Cache: /cache/tenant-a/)            (Cache: /cache/tenant-b/)
```

**Security:** ✅ Good (authorization + cache partitioning)
**Cost:** Low (single instance)
**Complexity:** High (requires PR #7 + full implementation)

**Recommendation:** **Best for SaaS platforms with cost optimization**

---

## Security Considerations

### Token Security

**Storage:**
- Use Kubernetes Secrets for GitHub App private keys
- Encrypt tokens at rest
- Never log tokens (redact in logs)

**Rotation:**
- GitHub App tokens auto-rotate (1 hour)
- PATs require manual rotation
- Monitor token expiry and renew proactively

**Scope:**
- Minimum required permissions
- Repository-level granularity where possible
- Regular permission audits

### Audit Requirements

**Log all:**
1. Authentication attempts (success/failure)
2. Authorization decisions (allow/deny with reason)
3. Token generation events
4. Cache access patterns
5. Upstream fetch operations

**Retention:**
- Security logs: 1 year minimum
- Access logs: 90 days minimum
- Compliance: per industry requirements (7 years for financial)

### Rate Limiting

**Per-tenant limits:**
- Requests per second: 100
- Concurrent operations: 50
- Cache storage: 100GB

**GitHub API limits:**
- GitHub Apps: 5000 req/hour per installation
- PATs: 5000 req/hour per token
- Implement backoff and retry

---

## Migration Path

### Phase 0: Current State

```go
// Single token for everything
TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
    AccessToken: "ghp_single_token",
    TokenType:   "Bearer",
})
```

### Phase 1: PR #7 Deployed (URL-Aware Tokens)

```go
// Can now use URL to select token
TokenSource: func(upstreamURL *url.URL) (*oauth2.Token, error) {
    // Extract org from URL
    org := extractOrg(upstreamURL)

    // Simple map-based selection
    tokens := map[string]string{
        "acme-corp": "ghp_acme_token",
        "megacorp":  "ghp_mega_token",
    }

    token := tokens[org]
    if token == "" {
        token = "ghp_default_token"
    }

    return &oauth2.Token{
        AccessToken: token,
        TokenType:   "Bearer",
    }, nil
}
```

**Benefits:**
- ✅ Org-specific tokens
- ✅ GHE support with Basic auth

**Limitations:**
- ❌ No tenant validation
- ❌ No cache isolation
- ❌ Manual token management

### Phase 2: Authorization Layer Added

```go
// In http_proxy_server.go
func (s *httpProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Step 1: Authenticate (existing)
    if err := s.config.RequestAuthorizer(r); err != nil {
        // Unauthenticated
        return
    }

    // Step 2: Authorize (NEW)
    claims, _ := oidc.GetClaimsFromContext(r.Context())
    repoURL, _ := extractRepoURL(r)

    decision, err := authorizationEngine.ValidateAccess(r.Context(), claims, repoURL)
    if err != nil || !decision.Allowed {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // Step 3: Store tenant context for later use
    ctx := context.WithValue(r.Context(), tenantKey, decision.TenantID)
    *r = *r.WithContext(ctx)

    // Continue with request...
}
```

### Phase 3: GitHub App Token Manager

```go
// In goblet-server/main.go
tokenManager := &GitHubAppTokenManager{
    appID: 123456,
    privateKey: loadPrivateKey(),
    installations: map[string]int64{
        "acme-corp": 111,
        "megacorp":  222,
    },
}

config := &goblet.ServerConfig{
    TokenSource: func(upstreamURL *url.URL) (*oauth2.Token, error) {
        // Get tenant from request context
        tenantID := getTenantFromRequestContext()

        // TokenManager handles org mapping + caching
        return tokenManager.GetToken(upstreamURL, tenantID)
    },
}
```

### Phase 4: Cache Partitioning

```go
// In managed_repository.go
func openManagedRepository(config *ServerConfig, u *url.URL) (*managedRepository, error) {
    // Get tenant from request context
    tenantID := getTenantFromContext()

    // Generate tenant-partitioned cache path
    cachePath := filepath.Join(
        config.LocalDiskCacheRoot,
        tenantID,
        u.Host,
        u.Path,
    )

    // Rest of implementation...
}
```

---

## Conclusion

### Summary

PR #7 provides **critical foundational changes** for multi-tenant GitHub authentication:

1. **Upstream URL-aware token generation** enables org-specific tokens
2. **Dynamic token type support** enables GitHub Enterprise compatibility
3. **Foundation for TokenManager** that can implement complex token strategies

However, PR #7 alone is **not sufficient** for secure multi-tenant operation. Complete implementation requires:

1. **Authorization Layer** - Enforce repo-level access control
2. **Token Manager** - Implement GitHub App token generation with caching
3. **Cache Partitioning** - Isolate cached data per tenant
4. **Audit Logging** - Track all access and authorization decisions

### Recommended Implementation

**For production multi-tenant deployments:**

1. **Deploy PR #7** - Foundation for all other work
2. **Implement Authorization Layer** (Phase 2) - Critical for security
3. **Deploy GitHub Apps** (Phase 3) - Best token management
4. **Enable Cache Partitioning** (Phase 4) - Complete isolation
5. **Security Hardening** (Phase 5) - Production readiness

**Estimated Timeline:** 12-16 weeks for complete implementation

**Alternative for Faster Deployment:**
- Use sidecar pattern (one Goblet per tenant)
- Simpler setup, higher cost
- Can migrate to shared model later

### Risk Assessment

**Without Full Implementation:**
- 🔴 **Critical:** Cross-tenant cache access (CVSS 8.1)
- 🔴 **High:** Token misuse across organizations
- 🟡 **Medium:** Manual token rotation burden
- 🟡 **Medium:** No audit trail for compliance

**With PR #7 + Full Implementation:**
- ✅ **Secure:** Complete tenant isolation
- ✅ **Compliant:** Full audit trail
- ✅ **Scalable:** Automatic token management
- ✅ **Maintainable:** Clear security boundaries

---

## References

1. [GitHub Apps Documentation](https://docs.github.com/en/apps)
2. [GitHub OAuth2 Token Types](https://docs.github.com/en/authentication)
3. [GitHub Enterprise Authentication](https://docs.github.com/en/enterprise-server/authentication)
4. [RFC-001: Secure Multi-Tenant Cache](secure-multi-tenant-rfc.md)
5. [PR #7: Upstream Authentication Improvements](https://github.com/jrepp/goblet/pull/7)
6. [Isolation Strategies](../security/isolation-strategies.md)

---

## Appendix A: GitHub API Endpoints

### GitHub App Authentication

```bash
# 1. Create JWT
JWT=$(generate_jwt $APP_ID $PRIVATE_KEY)

# 2. List installations
curl -H "Authorization: Bearer $JWT" \
     -H "Accept: application/vnd.github+json" \
     https://api.github.com/app/installations

# 3. Get installation token
curl -X POST \
     -H "Authorization: Bearer $JWT" \
     -H "Accept: application/vnd.github+json" \
     https://api.github.com/app/installations/$INSTALLATION_ID/access_tokens
```

### Token Permissions

**GitHub App - Minimum Permissions:**
```json
{
  "permissions": {
    "contents": "read",
    "metadata": "read"
  },
  "repositories": ["repo1", "repo2"]
}
```

**Fine-Grained PAT - Minimum Permissions:**
```
Repository access: Selected repositories
Permissions:
  - Contents: Read-only
  - Metadata: Read-only
```

## Appendix B: Configuration Examples

### GitHub App Configuration

```yaml
# goblet-config.yaml
github:
  app_id: 123456
  private_key_path: /secrets/github-app.pem
  installations:
    - org: acme-corp
      installation_id: 111
    - org: megacorp
      installation_id: 222

tenants:
  - id: tenant-acme
    name: "Acme Corporation"
    github_orgs: ["acme-corp"]
    allowed_repos:
      - "github.com/acme-corp/*"

  - id: tenant-mega
    name: "MegaCorp Industries"
    github_orgs: ["megacorp"]
    allowed_repos:
      - "github.com/megacorp/*"
```

### Environment Variables

```bash
# GitHub App
export GITHUB_APP_ID=123456
export GITHUB_APP_PRIVATE_KEY_PATH=/secrets/github-app.pem

# OIDC
export OIDC_ISSUER_URL=https://auth.example.com
export OIDC_CLIENT_ID=goblet-cache

# Cache
export CACHE_ROOT=/cache
export CACHE_ISOLATION_MODE=tenant
```
