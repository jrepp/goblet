// Copyright 2025 Jacob Repp <jacobrepp@gmail.com>
//
// Copyright 2019 Google LLC
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

// Package oidc provides OIDC token verification for authentication.
package oidc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Verifier provides OIDC token verification.
type Verifier struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   *Config
}

// Config holds OIDC configuration.
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
}

// NewVerifier creates a new OIDC verifier.
func NewVerifier(ctx context.Context, config *Config) (*Verifier, error) {
	if config.IssuerURL == "" {
		return nil, fmt.Errorf("issuer URL is required")
	}
	if config.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	return &Verifier{
		provider: provider,
		verifier: verifier,
		config:   config,
	}, nil
}

// VerifyAccessToken verifies an access token (opaque token).
// For Dex, we need to verify it as an ID token or use introspection.
func (v *Verifier) VerifyAccessToken(ctx context.Context, token string) error {
	// Try to verify as ID token first
	_, err := v.verifier.Verify(ctx, token)
	if err != nil {
		// If that fails, we could implement token introspection
		// For now, return the error
		return fmt.Errorf("failed to verify token: %w", err)
	}
	return nil
}

// VerifyIDToken verifies an ID token (JWT).
func (v *Verifier) VerifyIDToken(ctx context.Context, token string) (*oidc.IDToken, error) {
	idToken, err := v.verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}
	return idToken, nil
}

// ExtractBearerToken extracts the bearer token from an HTTP request.
func ExtractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

// Claims represents the claims in an OIDC token.
type Claims struct {
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"`
	Groups        []string `json:"groups"`
	Subject       string   `json:"sub"`
}

// GetClaims extracts claims from an ID token.
func GetClaims(idToken *oidc.IDToken) (*Claims, error) {
	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}
	return &claims, nil
}
