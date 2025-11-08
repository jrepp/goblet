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

// Package examples demonstrates how to configure tenant-scoped cache isolation
package examples

import (
	"regexp"

	"github.com/google/goblet"
)

// ConfigureTenantIsolation demonstrates tenant-scoped isolation for Terraform Cloud.
//
// Use case: Multiple organizations/workspaces, users within org share cache
// Cache structure: /cache/tenant-org1/github.com/org/repo
func ConfigureTenantIsolation() *goblet.IsolationConfig {
	return &goblet.IsolationConfig{
		Mode:           goblet.IsolationTenant,
		TenantClaimKey: "groups", // Extract tenant from OIDC groups claim

		// Extract tenant from group format: "org:engineering" -> "engineering"
		TenantRegex: regexp.MustCompile(`^org:(.*)`),
	}
}

// ConfigureTenantIsolationWithHeader demonstrates tenant-scoped isolation with HTTP header.
//
// Use case: Tenant ID passed via custom header (common in proxies)
// Header: X-Tenant-ID: acme-corp
// Cache structure: /cache/tenant-acme-corp/github.com/org/repo
func ConfigureTenantIsolationWithHeader() *goblet.IsolationConfig {
	return &goblet.IsolationConfig{
		Mode:            goblet.IsolationTenant,
		TenantHeaderKey: "X-Tenant-ID", // Custom header for tenant
	}
}

// ConfigureTenantIsolationWithHashing demonstrates tenant-scoped isolation with hashing.
//
// Use case: Privacy or special characters in tenant names
// Cache structure: /cache/tenant-a1b2c3d4/github.com/org/repo
func ConfigureTenantIsolationWithHashing() *goblet.IsolationConfig {
	return &goblet.IsolationConfig{
		Mode:            goblet.IsolationTenant,
		TenantHeaderKey: "X-Tenant-ID",
		HashIdentifiers: true, // SHA256 hash of tenant ID
	}
}

// ConfigureTerraformCloudIsolation demonstrates Terraform Cloud workspace isolation.
//
// Use case: Terraform workspaces with different credentials per workspace
// Header format: X-TFC-Workspace-ID: ws-abc123xyz
// Cache structure: /cache/tenant-ws-abc123xyz/github.com/hashicorp/terraform-aws-vpc
func ConfigureTerraformCloudIsolation() *goblet.IsolationConfig {
	config := &goblet.IsolationConfig{
		Mode:            goblet.IsolationTenant,
		TenantHeaderKey: "X-TFC-Workspace-ID",
		HashIdentifiers: false, // Keep workspace ID readable for debugging
	}

	// Validate
	if err := config.Validate(); err != nil {
		panic(err)
	}

	return config
}

// ConfigureMultiOrgSaaS demonstrates multi-organization SaaS isolation.
//
// Use case: SaaS platform with multiple customer organizations
// OIDC claim: groups = ["org:customer-123", "team:security"]
// Extract first matching org:* group
// Cache structure: /cache/tenant-customer-123/github.com/org/repo
func ConfigureMultiOrgSaaS() *goblet.IsolationConfig {
	config := &goblet.IsolationConfig{
		Mode:           goblet.IsolationTenant,
		TenantClaimKey: "groups",

		// Extract organization ID from groups claim
		// Matches "org:customer-123" and extracts "customer-123"
		TenantRegex: regexp.MustCompile(`^org:(.+)`),
	}

	// Validate
	if err := config.Validate(); err != nil {
		panic(err)
	}

	println(config.SecurityWarning())

	return config
}
