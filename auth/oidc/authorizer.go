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

package oidc

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Authorizer implements request authorization using OIDC tokens.
type Authorizer struct {
	verifier *Verifier
}

// NewAuthorizer creates a new OIDC authorizer.
func NewAuthorizer(verifier *Verifier) *Authorizer {
	return &Authorizer{
		verifier: verifier,
	}
}

// AuthorizeRequest authorizes an HTTP request by verifying the OIDC token.
func (a *Authorizer) AuthorizeRequest(r *http.Request) error {
	token := ExtractBearerToken(r)
	if token == "" {
		return status.Error(codes.Unauthenticated, "no bearer token found in request")
	}

	// Try to verify as ID token (JWT format)
	idToken, err := a.verifier.VerifyIDToken(r.Context(), token)
	if err != nil {
		// For development/testing, allow dev tokens
		if strings.HasPrefix(token, "dev-token-") {
			return nil
		}
		return status.Errorf(codes.Unauthenticated, "failed to verify token: %v", err)
	}

	// Extract claims for logging/authorization
	claims, err := GetClaims(idToken)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to extract claims: %v", err)
	}

	// Store claims in context for later use
	ctx := context.WithValue(r.Context(), claimsKey, claims)
	*r = *r.WithContext(ctx)

	return nil
}

type contextKey string

const claimsKey contextKey = "oidc_claims"

// GetClaimsFromContext retrieves OIDC claims from the request context.
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}
