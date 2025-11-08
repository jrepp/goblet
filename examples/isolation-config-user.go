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

// Package examples demonstrates how to configure user-scoped cache isolation
package examples

import (
	"net/http"

	"github.com/google/goblet"
)

// ConfigureUserIsolation demonstrates user-scoped isolation for a risk scanning service.
//
// Use case: Different teams scanning different private repositories
// Each user gets their own cache directory
// Cache structure: /cache/user-alice@company.com/github.com/org/repo
func ConfigureUserIsolation() *goblet.ServerConfig {
	// Create isolation config
	isolationConfig := &goblet.IsolationConfig{
		Mode:         goblet.IsolationUser,
		UserClaimKey: "email", // Extract user from OIDC email claim

		// Optional: Hash identifiers for privacy
		// HashIdentifiers: true,  // Uses SHA256 hash instead of raw email
	}

	// Validate configuration
	if err := isolationConfig.Validate(); err != nil {
		panic(err)
	}

	// Print security warning/confirmation
	println(isolationConfig.SecurityWarning())

	// Create server config
	config := &goblet.ServerConfig{
		LocalDiskCacheRoot: "/cache",

		// OIDC-based authorization
		RequestAuthorizer: createOIDCAuthorizer(), // See auth/oidc package

		// Add isolation config (extend ServerConfig to support this)
		// IsolationConfig: isolationConfig,
	}

	return config
}

// ConfigureUserIsolationWithHashing demonstrates user-scoped isolation with hashed identifiers.
//
// Use case: Privacy-sensitive deployments where you don't want user emails in filesystem
// Cache structure: /cache/user-a1b2c3d4/github.com/org/repo
func ConfigureUserIsolationWithHashing() *goblet.IsolationConfig {
	return &goblet.IsolationConfig{
		Mode:            goblet.IsolationUser,
		UserClaimKey:    "email",
		HashIdentifiers: true, // SHA256 hash of email
	}
}

// ConfigureUserIsolationWithSubject demonstrates user-scoped isolation with subject (UUID).
//
// Use case: When OIDC subject (typically UUID) is preferred over email
// Cache structure: /cache/user-123e4567-e89b-12d3-a456-426614174000/github.com/org/repo
func ConfigureUserIsolationWithSubject() *goblet.IsolationConfig {
	return &goblet.IsolationConfig{
		Mode:         goblet.IsolationUser,
		UserClaimKey: "sub", // Use OIDC subject claim
	}
}

// Placeholder for example
func createOIDCAuthorizer() func(*http.Request) error {
	// See auth/oidc package for real implementation
	return func(r *http.Request) error {
		return nil
	}
}
