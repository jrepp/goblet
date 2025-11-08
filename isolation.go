// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goblet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// IsolationMode defines how cache is isolated between users/tenants.
type IsolationMode string

const (
	// IsolationNone - No isolation (UNSAFE for multi-tenant with private repos)
	// All users share the same cache. Use only for:
	//   - Single-user deployments
	//   - Public repositories only
	//   - Trusted environments where all users have same access
	IsolationNone IsolationMode = "none"

	// IsolationUser - Cache isolated per user (SAFE, moderate storage)
	// Each user gets their own cache directory based on user identifier.
	// Example: /cache/user-alice@company.com/github.com/org/repo
	// Use for:
	//   - Risk scanning services (different teams scan different repos)
	//   - Development environments
	//   - When users have different access permissions
	IsolationUser IsolationMode = "user"

	// IsolationTenant - Cache isolated per tenant/organization (SAFE, better efficiency)
	// Users within same tenant share cache, isolated across tenants.
	// Example: /cache/tenant-org1/github.com/org/repo
	// Use for:
	//   - Terraform Cloud (workspace isolation)
	//   - SaaS platforms (organization isolation)
	//   - When users within tenant have similar access
	IsolationTenant IsolationMode = "tenant"

	// IsolationSidecar - Single-user mode (SAFE, default)
	// Assumes single user/service account per instance (sidecar pattern).
	// No isolation prefix added. Equivalent to IsolationNone but explicitly
	// documents deployment intention.
	// Use for:
	//   - Kubernetes sidecar deployments
	//   - Single service account
	//   - Pod-scoped cache
	IsolationSidecar IsolationMode = "sidecar"
)

// IsolationConfig defines how to extract and apply isolation.
type IsolationConfig struct {
	// Mode specifies the isolation strategy
	Mode IsolationMode

	// UserClaimKey specifies which OIDC claim contains user identifier
	// Examples: "email", "sub", "preferred_username"
	// Used when Mode = IsolationUser
	UserClaimKey string

	// TenantClaimKey specifies which OIDC claim contains tenant identifier
	// Examples: "groups", "org", "tenant_id"
	// Used when Mode = IsolationTenant
	TenantClaimKey string

	// TenantRegex extracts tenant ID from claim value
	// Example: "^org:(.*)" extracts "engineering" from "org:engineering"
	// Used when Mode = IsolationTenant
	TenantRegex *regexp.Regexp

	// TenantHeaderKey specifies HTTP header containing tenant ID
	// Example: "X-Tenant-ID"
	// Alternative to claim-based tenant extraction
	TenantHeaderKey string

	// HashIdentifiers when true, uses SHA256 hash of identifier instead of raw value
	// Useful for privacy or to handle special characters in identifiers
	// Example: alice@company.com -> sha256(alice@company.com) = "a1b2c3..."
	HashIdentifiers bool
}

// DefaultIsolationConfig returns safe defaults.
func DefaultIsolationConfig() *IsolationConfig {
	return &IsolationConfig{
		Mode:            IsolationSidecar,
		UserClaimKey:    "email",
		TenantClaimKey:  "groups",
		HashIdentifiers: false,
	}
}

// GetCachePath returns the cache path with appropriate isolation prefix.
func (ic *IsolationConfig) GetCachePath(r *http.Request, cacheRoot string, repoURL *url.URL) (string, error) {
	if ic == nil {
		ic = DefaultIsolationConfig()
	}

	// Base path without isolation
	basePath := filepath.Join(repoURL.Host, repoURL.Path)

	switch ic.Mode {
	case IsolationNone, IsolationSidecar:
		// No isolation prefix
		return filepath.Join(cacheRoot, basePath), nil

	case IsolationUser:
		userID, err := ic.getUserIdentifier(r)
		if err != nil {
			return "", fmt.Errorf("failed to get user identifier: %w", err)
		}
		return filepath.Join(cacheRoot, userID, basePath), nil

	case IsolationTenant:
		tenantID, err := ic.getTenantIdentifier(r)
		if err != nil {
			return "", fmt.Errorf("failed to get tenant identifier: %w", err)
		}
		return filepath.Join(cacheRoot, tenantID, basePath), nil

	default:
		return "", fmt.Errorf("unknown isolation mode: %s", ic.Mode)
	}
}

// getUserIdentifier extracts user identifier from request.
func (ic *IsolationConfig) getUserIdentifier(r *http.Request) (string, error) {
	// Try to get from OIDC claims context
	claims := GetClaimsFromContext(r.Context())
	if claims != nil {
		// Use configured claim key
		claimKey := ic.UserClaimKey
		if claimKey == "" {
			claimKey = "email" // default
		}

		var userID string
		switch claimKey {
		case "email":
			userID = claims.Email
		case "sub":
			userID = claims.Subject
		default:
			// Try to extract from claims map if available
			return "", fmt.Errorf("unsupported user claim key: %s", claimKey)
		}

		if userID == "" {
			return "", fmt.Errorf("user identifier not found in claims (key: %s)", claimKey)
		}

		return ic.sanitizeIdentifier(userID, "user"), nil
	}

	// Fallback: try to extract from authorization header
	// This is less secure but provides compatibility
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Extract email from Bearer token if it's a JWT (simplified)
		// In production, this should be properly parsed
		// For now, return a generic identifier
		return ic.sanitizeIdentifier("unknown-user", "user"), fmt.Errorf("could not extract user from token")
	}

	return "", fmt.Errorf("no user identity found in request")
}

// getTenantIdentifier extracts tenant identifier from request.
func (ic *IsolationConfig) getTenantIdentifier(r *http.Request) (string, error) {
	// Try custom header first (higher priority)
	if ic.TenantHeaderKey != "" {
		tenantID := r.Header.Get(ic.TenantHeaderKey)
		if tenantID != "" {
			return ic.sanitizeIdentifier(tenantID, "tenant"), nil
		}
	}

	// Try to get from OIDC claims
	claims := GetClaimsFromContext(r.Context())
	if claims != nil {
		claimKey := ic.TenantClaimKey
		if claimKey == "" {
			claimKey = "groups" // default
		}

		var tenantID string
		switch claimKey {
		case "groups":
			// Extract from groups claim
			if len(claims.Groups) > 0 {
				// Use first group or apply regex
				groupValue := claims.Groups[0]
				if ic.TenantRegex != nil {
					matches := ic.TenantRegex.FindStringSubmatch(groupValue)
					if len(matches) > 1 {
						tenantID = matches[1]
					}
				} else {
					tenantID = groupValue
				}
			}
		default:
			return "", fmt.Errorf("unsupported tenant claim key: %s", claimKey)
		}

		if tenantID == "" {
			return "", fmt.Errorf("tenant identifier not found in claims (key: %s)", claimKey)
		}

		return ic.sanitizeIdentifier(tenantID, "tenant"), nil
	}

	return "", fmt.Errorf("no tenant identity found in request")
}

// sanitizeIdentifier makes identifier safe for use in filesystem paths.
func (ic *IsolationConfig) sanitizeIdentifier(identifier, prefix string) string {
	if ic.HashIdentifiers {
		// Use SHA256 hash for privacy and filesystem safety
		hash := sha256.Sum256([]byte(identifier))
		return prefix + "-" + hex.EncodeToString(hash[:8]) // Use first 8 bytes (16 hex chars)
	}

	// Replace unsafe characters for filesystem
	// Keep: alphanumeric, dash, underscore, period
	// Replace: everything else with dash
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_' || r == '.':
			return r
		case r == '@':
			return '-' // Common in emails
		default:
			return '-'
		}
	}, identifier)

	// Ensure prefix for clarity
	if !strings.HasPrefix(safe, prefix+"-") {
		safe = prefix + "-" + safe
	}

	return safe
}

// Validate checks if configuration is valid.
func (ic *IsolationConfig) Validate() error {
	if ic == nil {
		return fmt.Errorf("isolation config is nil")
	}

	switch ic.Mode {
	case IsolationNone, IsolationSidecar:
		// No additional validation needed
		return nil

	case IsolationUser:
		if ic.UserClaimKey == "" {
			return fmt.Errorf("UserClaimKey must be set for IsolationUser mode")
		}
		return nil

	case IsolationTenant:
		if ic.TenantClaimKey == "" && ic.TenantHeaderKey == "" {
			return fmt.Errorf("TenantClaimKey or TenantHeaderKey must be set for IsolationTenant mode")
		}
		return nil

	default:
		return fmt.Errorf("unknown isolation mode: %s", ic.Mode)
	}
}

// Claims represents authentication claims from OIDC/OAuth2.
type Claims struct {
	Email   string
	Subject string
	Groups  []string
}

// claimsContextKey is the key for storing claims in request context.
type claimsContextKey struct{}

// SetClaimsInContext stores claims in request context.
func SetClaimsInContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// GetClaimsFromContext retrieves claims from request context.
func GetClaimsFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(claimsContextKey{}).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// SecurityWarning returns a warning message if configuration is unsafe.
func (ic *IsolationConfig) SecurityWarning() string {
	if ic == nil {
		return "WARNING: No isolation config - using defaults"
	}

	switch ic.Mode {
	case IsolationNone:
		return "⚠️  WARNING: Isolation mode 'none' is UNSAFE for multi-tenant deployments with private repositories. " +
			"All users share the same cache. Use only for single-user deployments or public repositories. " +
			"Consider using 'user', 'tenant', or 'sidecar' mode."
	case IsolationSidecar:
		return "✓ Isolation mode 'sidecar' - safe for single-user/single-service-account deployments"
	case IsolationUser:
		return "✓ Isolation mode 'user' - safe for multi-tenant deployments (user-scoped cache)"
	case IsolationTenant:
		return "✓ Isolation mode 'tenant' - safe for multi-tenant deployments (tenant-scoped cache)"
	default:
		return "⚠️  WARNING: Unknown isolation mode"
	}
}
