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

package goblet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

// TestManagedRepository_TokenSourceCalled verifies that TokenSource is called
// with the correct upstream URL when fetching from upstream.
func TestManagedRepository_TokenSourceCalled(t *testing.T) {
	var capturedURL *url.URL
	var tokenCallCount int
	var mu sync.Mutex

	upstreamURL, _ := url.Parse("https://github.com/test-org/test-repo")

	config := &ServerConfig{
		LocalDiskCacheRoot: t.TempDir(),
		URLCanonializer: func(u *url.URL) (*url.URL, error) {
			return upstreamURL, nil
		},
		RequestAuthorizer: func(r *http.Request) error {
			return nil
		},
		TokenSource: func(u *url.URL) (*oauth2.Token, error) {
			mu.Lock()
			capturedURL = u
			tokenCallCount++
			mu.Unlock()

			return &oauth2.Token{
				AccessToken: "test-token",
				TokenType:   "Bearer",
			}, nil
		},
	}

	repo, err := openManagedRepository(config, upstreamURL)
	if err != nil {
		t.Fatalf("Failed to open managed repository: %v", err)
	}

	// Trigger a fetch to invoke TokenSource
	// Note: This will likely fail without a real upstream, but TokenSource will still be called
	_ = repo.fetchUpstream()

	// Verify the URL was captured
	mu.Lock()
	defer mu.Unlock()

	if capturedURL == nil {
		t.Fatal("TokenSource was not called with upstream URL")
	}

	if capturedURL.String() != upstreamURL.String() {
		t.Errorf("TokenSource called with URL %q, want %q", capturedURL.String(), upstreamURL.String())
	}

	if tokenCallCount == 0 {
		t.Error("TokenSource was never called")
	}

	t.Logf("TokenSource called %d times with URL: %s", tokenCallCount, capturedURL)
}

// TestManagedRepository_DifferentTokenTypes tests that different token types
// (Bearer, Basic) are correctly applied to upstream requests.
func TestManagedRepository_DifferentTokenTypes(t *testing.T) {
	tests := []struct {
		name                string
		tokenType           string
		accessToken         string
		wantAuthHeaderStart string
	}{
		{
			name:                "Bearer token for public GitHub",
			tokenType:           "Bearer",
			accessToken:         "ghp_public_token",
			wantAuthHeaderStart: "Bearer ghp_public_token",
		},
		{
			name:                "Basic token for GitHub Enterprise",
			tokenType:           "Basic",
			accessToken:         "ghp_enterprise_token",
			wantAuthHeaderStart: "Basic ghp_enterprise_token",
		},
		{
			name:                "token type (lowercase)",
			tokenType:           "token",
			accessToken:         "ghp_custom_token",
			wantAuthHeaderStart: "token ghp_custom_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedAuthHeader string
			var mu sync.Mutex

			// Create a test upstream server that captures the Authorization header
			upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				capturedAuthHeader = r.Header.Get("Authorization")
				mu.Unlock()

				// Return a minimal git response
				w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("0000")) // Git flush packet
			}))
			defer upstreamServer.Close()

			upstreamURL, _ := url.Parse(upstreamServer.URL + "/test-repo")

			config := &ServerConfig{
				LocalDiskCacheRoot: t.TempDir(),
				URLCanonializer: func(u *url.URL) (*url.URL, error) {
					return upstreamURL, nil
				},
				RequestAuthorizer: func(r *http.Request) error {
					return nil
				},
				TokenSource: func(u *url.URL) (*oauth2.Token, error) {
					return &oauth2.Token{
						AccessToken: tt.accessToken,
						TokenType:   tt.tokenType,
					}, nil
				},
			}

			repo, err := openManagedRepository(config, upstreamURL)
			if err != nil {
				t.Fatalf("Failed to open managed repository: %v", err)
			}

			// Force an upstream fetch to trigger token usage
			// Note: This will fail but we're just testing that the auth header is set
			_ = repo.fetchUpstream()

			mu.Lock()
			authHeader := capturedAuthHeader
			mu.Unlock()

			if authHeader == "" {
				t.Error("Authorization header was not set on upstream request")
				return
			}

			if authHeader != tt.wantAuthHeaderStart {
				t.Errorf("Authorization header = %q, want %q", authHeader, tt.wantAuthHeaderStart)
			}

			t.Logf("Correct Authorization header set: %s", authHeader)
		})
	}
}

// TestManagedRepository_EmptyToken tests that requests without tokens
// (for public repositories) work correctly.
func TestManagedRepository_EmptyToken(t *testing.T) {
	var authHeaderSet bool
	var mu sync.Mutex

	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		authHeaderSet = r.Header.Get("Authorization") != ""
		mu.Unlock()

		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("0000"))
	}))
	defer upstreamServer.Close()

	upstreamURL, _ := url.Parse(upstreamServer.URL + "/public-repo")

	config := &ServerConfig{
		LocalDiskCacheRoot: t.TempDir(),
		URLCanonializer: func(u *url.URL) (*url.URL, error) {
			return upstreamURL, nil
		},
		RequestAuthorizer: func(r *http.Request) error {
			return nil
		},
		TokenSource: func(u *url.URL) (*oauth2.Token, error) {
			// Return token with empty access token for public repos
			return &oauth2.Token{
				AccessToken: "",
				TokenType:   "Bearer",
			}, nil
		},
	}

	repo, err := openManagedRepository(config, upstreamURL)
	if err != nil {
		t.Fatalf("Failed to open managed repository: %v", err)
	}

	// Trigger upstream operation
	_ = repo.fetchUpstream()

	mu.Lock()
	defer mu.Unlock()

	if authHeaderSet {
		t.Error("Authorization header should not be set for empty token")
	}

	t.Log("Empty token handled correctly - no Authorization header set")
}

// TestManagedRepository_TokenSourceError tests error handling when
// TokenSource returns an error.
func TestManagedRepository_TokenSourceError(t *testing.T) {
	upstreamURL, _ := url.Parse("https://github.com/org/repo")

	config := &ServerConfig{
		LocalDiskCacheRoot: t.TempDir(),
		URLCanonializer: func(u *url.URL) (*url.URL, error) {
			return upstreamURL, nil
		},
		RequestAuthorizer: func(r *http.Request) error {
			return nil
		},
		TokenSource: func(u *url.URL) (*oauth2.Token, error) {
			return nil, fmt.Errorf("failed to generate token: installation not found")
		},
	}

	repo, err := openManagedRepository(config, upstreamURL)
	if err != nil {
		t.Fatalf("Failed to open managed repository: %v", err)
	}

	// Attempt to fetch - should fail with token error
	err = repo.fetchUpstream()
	if err == nil {
		t.Error("Expected error when TokenSource fails, got nil")
	}

	if !strings.Contains(err.Error(), "token") {
		t.Errorf("Error should mention token, got: %v", err)
	}

	t.Logf("Token error correctly propagated: %v", err)
}

// TestManagedRepository_MultipleTokenCalls tests that TokenSource can be
// called multiple times for the same repository (e.g., for token refresh).
func TestManagedRepository_MultipleTokenCalls(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("0000"))
	}))
	defer upstreamServer.Close()

	upstreamURL, _ := url.Parse(upstreamServer.URL + "/repo")

	config := &ServerConfig{
		LocalDiskCacheRoot: t.TempDir(),
		URLCanonializer: func(u *url.URL) (*url.URL, error) {
			return upstreamURL, nil
		},
		RequestAuthorizer: func(r *http.Request) error {
			return nil
		},
		TokenSource: func(u *url.URL) (*oauth2.Token, error) {
			mu.Lock()
			callCount++
			currentCount := callCount
			mu.Unlock()

			// Return different tokens to simulate refresh
			return &oauth2.Token{
				AccessToken: fmt.Sprintf("token-%d", currentCount),
				TokenType:   "Bearer",
			}, nil
		},
	}

	repo, err := openManagedRepository(config, upstreamURL)
	if err != nil {
		t.Fatalf("Failed to open managed repository: %v", err)
	}

	// Make multiple fetch attempts
	for i := 0; i < 3; i++ {
		_ = repo.fetchUpstream()
	}

	mu.Lock()
	defer mu.Unlock()

	if callCount < 3 {
		t.Errorf("TokenSource called %d times, expected at least 3", callCount)
	}

	t.Logf("TokenSource called %d times for token refresh", callCount)
}

// TestManagedRepository_URLPassedToTokenSource verifies that the exact
// upstream URL is passed to TokenSource, including host, path, etc.
func TestManagedRepository_URLPassedToTokenSource(t *testing.T) {
	tests := []struct {
		name        string
		upstreamURL string
	}{
		{
			name:        "GitHub public",
			upstreamURL: "https://github.com/org/repo",
		},
		{
			name:        "GitHub Enterprise",
			upstreamURL: "https://github.enterprise.com/org/repo",
		},
		{
			name:        "GitLab",
			upstreamURL: "https://gitlab.com/group/project",
		},
		{
			name:        "Custom git server with port",
			upstreamURL: "https://git.example.com:8443/path/to/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL *url.URL
			var mu sync.Mutex

			upstreamURL, _ := url.Parse(tt.upstreamURL)

			config := &ServerConfig{
				LocalDiskCacheRoot: t.TempDir(),
				URLCanonializer: func(u *url.URL) (*url.URL, error) {
					return upstreamURL, nil
				},
				RequestAuthorizer: func(r *http.Request) error {
					return nil
				},
				TokenSource: func(u *url.URL) (*oauth2.Token, error) {
					mu.Lock()
					capturedURL = u
					mu.Unlock()

					return &oauth2.Token{
						AccessToken: "test-token",
						TokenType:   "Bearer",
					}, nil
				},
			}

			repo, err := openManagedRepository(config, upstreamURL)
			if err != nil {
				t.Fatalf("Failed to open managed repository: %v", err)
			}

			// Trigger a fetch to invoke TokenSource
			_ = repo.fetchUpstream()

			mu.Lock()
			defer mu.Unlock()

			if capturedURL == nil {
				t.Fatal("TokenSource was not called")
			}

			// Verify complete URL match
			if capturedURL.Scheme != upstreamURL.Scheme {
				t.Errorf("Scheme = %q, want %q", capturedURL.Scheme, upstreamURL.Scheme)
			}
			if capturedURL.Host != upstreamURL.Host {
				t.Errorf("Host = %q, want %q", capturedURL.Host, upstreamURL.Host)
			}
			if capturedURL.Path != upstreamURL.Path {
				t.Errorf("Path = %q, want %q", capturedURL.Path, upstreamURL.Path)
			}

			t.Logf("Correct upstream URL passed: %s", capturedURL)
		})
	}
}

// TestManagedRepository_ConcurrentTokenRequests tests that concurrent
// operations on the same repository correctly handle token requests.
func TestManagedRepository_ConcurrentTokenRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	var tokenCallCount int
	var mu sync.Mutex

	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("0000"))
	}))
	defer upstreamServer.Close()

	upstreamURL, _ := url.Parse(upstreamServer.URL + "/repo")

	config := &ServerConfig{
		LocalDiskCacheRoot: t.TempDir(),
		URLCanonializer: func(u *url.URL) (*url.URL, error) {
			return upstreamURL, nil
		},
		RequestAuthorizer: func(r *http.Request) error {
			return nil
		},
		TokenSource: func(u *url.URL) (*oauth2.Token, error) {
			mu.Lock()
			tokenCallCount++
			mu.Unlock()

			return &oauth2.Token{
				AccessToken: "concurrent-token",
				TokenType:   "Bearer",
			}, nil
		},
	}

	repo, err := openManagedRepository(config, upstreamURL)
	if err != nil {
		t.Fatalf("Failed to open managed repository: %v", err)
	}

	// Launch concurrent fetch operations
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = repo.fetchUpstream()
		}()
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if tokenCallCount == 0 {
		t.Error("TokenSource was never called during concurrent operations")
	}

	t.Logf("TokenSource handled %d concurrent calls successfully", tokenCallCount)
}
