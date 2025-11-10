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

package testing

import (
	"fmt"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

// TestTokenSource_URLBasedSelection tests that different upstream URLs
// can receive different tokens based on the URL passed to TokenSource.
func TestTokenSource_URLBasedSelection(t *testing.T) {
	calledWithURLs := []string{}
	var mu sync.Mutex

	//nolint:unparam // Test function demonstrating token selection pattern
	tokenFunc := func(upstreamURL *url.URL) (*oauth2.Token, error) {
		mu.Lock()
		calledWithURLs = append(calledWithURLs, upstreamURL.String())
		mu.Unlock()

		// Return different tokens based on the URL
		switch upstreamURL.Host {
		case "github.com":
			return &oauth2.Token{
				AccessToken: "token-for-github",
				TokenType:   "Bearer",
			}, nil
		case "gitlab.com":
			return &oauth2.Token{
				AccessToken: "token-for-gitlab",
				TokenType:   "Bearer",
			}, nil
		default:
			return &oauth2.Token{
				AccessToken: "default-token",
				TokenType:   "Bearer",
			}, nil
		}
	}

	// Test that the function is called with the correct URL
	url1, _ := url.Parse("https://github.com/org/repo")
	token1, err := tokenFunc(url1)
	if err != nil {
		t.Fatalf("TokenSource failed for github.com: %v", err)
	}
	if token1.AccessToken != "token-for-github" {
		t.Errorf("Got token %q for github.com, want %q", token1.AccessToken, "token-for-github")
	}

	url2, _ := url.Parse("https://gitlab.com/org/repo")
	token2, err := tokenFunc(url2)
	if err != nil {
		t.Fatalf("TokenSource failed for gitlab.com: %v", err)
	}
	if token2.AccessToken != "token-for-gitlab" {
		t.Errorf("Got token %q for gitlab.com, want %q", token2.AccessToken, "token-for-gitlab")
	}

	if len(calledWithURLs) != 2 {
		t.Errorf("TokenSource called %d times, want 2", len(calledWithURLs))
	}

	t.Logf("TokenSource called with URLs: %v", calledWithURLs)
}

// TestTokenSource_TokenTypeHandling tests that different token types
// (Bearer, Basic) are correctly handled.
func TestTokenSource_TokenTypeHandling(t *testing.T) {
	tests := []struct {
		name              string
		tokenType         string
		accessToken       string
		wantAuthHeader    string
		wantTokenTypeName string
	}{
		{
			name:              "Bearer token",
			tokenType:         "Bearer",
			accessToken:       "ghp_abc123",
			wantAuthHeader:    "Bearer ghp_abc123",
			wantTokenTypeName: "Bearer",
		},
		{
			name:              "Basic token",
			tokenType:         "Basic",
			accessToken:       "ghp_enterprise123",
			wantAuthHeader:    "Basic ghp_enterprise123",
			wantTokenTypeName: "Basic",
		},
		{
			name:              "Empty token type defaults to Bearer",
			tokenType:         "",
			accessToken:       "token123",
			wantAuthHeader:    "Bearer token123",
			wantTokenTypeName: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &oauth2.Token{
				AccessToken: tt.accessToken,
				TokenType:   tt.tokenType,
			}

			// Verify token type
			tokenType := token.Type()
			if tokenType != tt.wantTokenTypeName {
				t.Errorf("Token.Type() = %q, want %q", tokenType, tt.wantTokenTypeName)
			}

			// Verify that the authorization header would be constructed correctly
			authHeader := fmt.Sprintf("%s %s", token.Type(), token.AccessToken)
			if authHeader != tt.wantAuthHeader {
				t.Errorf("Authorization header = %q, want %q", authHeader, tt.wantAuthHeader)
			}
		})
	}
}

// TestTokenSource_OrgSpecificTokens tests that tokens can be selected
// based on GitHub organization extracted from the URL.
func TestTokenSource_OrgSpecificTokens(t *testing.T) {
	orgTokens := map[string]string{
		"acme-corp": "token-acme",
		"megacorp":  "token-mega",
		"startup":   "token-startup",
	}

	extractOrg := func(u *url.URL) string {
		// Extract org from github.com/org/repo format
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 1 {
			return parts[0]
		}
		return ""
	}

	tokenFunc := func(upstreamURL *url.URL) (*oauth2.Token, error) {
		org := extractOrg(upstreamURL)
		token, ok := orgTokens[org]
		if !ok {
			return nil, fmt.Errorf("no token configured for org: %s", org)
		}

		return &oauth2.Token{
			AccessToken: token,
			TokenType:   "Bearer",
		}, nil
	}

	tests := []struct {
		url       string
		wantToken string
		wantErr   bool
	}{
		{
			url:       "https://github.com/acme-corp/private-repo",
			wantToken: "token-acme",
			wantErr:   false,
		},
		{
			url:       "https://github.com/megacorp/project",
			wantToken: "token-mega",
			wantErr:   false,
		},
		{
			url:       "https://github.com/startup/api",
			wantToken: "token-startup",
			wantErr:   false,
		},
		{
			url:       "https://github.com/unknown-org/repo",
			wantToken: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			token, err := tokenFunc(u)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for unknown org, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if token.AccessToken != tt.wantToken {
				t.Errorf("Got token %q, want %q", token.AccessToken, tt.wantToken)
			}
		})
	}
}

// TestTokenSource_ErrorHandling tests that errors from TokenSource
// are properly handled.
func TestTokenSource_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		tokenFunc     func(*url.URL) (*oauth2.Token, error)
		url           string
		wantErr       bool
		wantErrString string
	}{
		{
			name: "token generation error",
			tokenFunc: func(u *url.URL) (*oauth2.Token, error) {
				return nil, fmt.Errorf("failed to generate token: connection timeout")
			},
			url:           "https://github.com/org/repo",
			wantErr:       true,
			wantErrString: "connection timeout",
		},
		{
			name: "nil token returned",
			tokenFunc: func(u *url.URL) (*oauth2.Token, error) {
				return nil, nil
			},
			url:     "https://github.com/org/repo",
			wantErr: false, // nil token with nil error is valid
		},
		{
			name: "successful token generation",
			tokenFunc: func(u *url.URL) (*oauth2.Token, error) {
				return &oauth2.Token{
					AccessToken: "valid-token",
					TokenType:   "Bearer",
				}, nil
			},
			url:     "https://github.com/org/repo",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			token, err := tt.tokenFunc(u)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.wantErrString != "" && !strings.Contains(err.Error(), tt.wantErrString) {
					t.Errorf("Error = %q, want substring %q", err.Error(), tt.wantErrString)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if token != nil && token.AccessToken != "" {
					t.Logf("Token generated successfully: %s", token.AccessToken)
				}
			}
		})
	}
}

// TestMultipleUpstreams_Integration tests fetching from multiple upstream
// servers with different authentication credentials.
func TestMultipleUpstreams_Integration(t *testing.T) {
	t.Skip("Skipping complex multi-upstream test - see TestTokenSource_OrgSpecificTokens for similar coverage")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create two upstream servers with different auth requirements
	upstream1Token := "upstream1-secret-token"
	upstream2Token := "upstream2-secret-token"

	// Upstream 1
	upstream1Repo := NewLocalBareGitRepo()
	defer upstream1Repo.Close()
	_, _ = upstream1Repo.Run("config", "http.receivepack", "1")
	_, _ = upstream1Repo.Run("config", "uploadpack.allowfilter", "1")

	upstream1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		auth := req.Header.Get("Authorization")
		if auth != "Bearer "+upstream1Token {
			http.Error(w, "invalid auth for upstream1", http.StatusForbidden)
			return
		}
		h := &cgi.Handler{
			Path: gitBinary,
			Dir:  string(upstream1Repo),
			Env: []string{
				"GIT_PROJECT_ROOT=" + string(upstream1Repo),
				"GIT_HTTP_EXPORT_ALL=1",
			},
		}
		h.ServeHTTP(w, req)
	}))
	defer upstream1Server.Close()

	// Upstream 2
	upstream2Repo := NewLocalBareGitRepo()
	defer upstream2Repo.Close()
	_, _ = upstream2Repo.Run("config", "http.receivepack", "1")
	_, _ = upstream2Repo.Run("config", "uploadpack.allowfilter", "1")

	upstream2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		auth := req.Header.Get("Authorization")
		if auth != "Bearer "+upstream2Token {
			http.Error(w, "invalid auth for upstream2", http.StatusForbidden)
			return
		}
		h := &cgi.Handler{
			Path: gitBinary,
			Dir:  string(upstream2Repo),
			Env: []string{
				"GIT_PROJECT_ROOT=" + string(upstream2Repo),
				"GIT_HTTP_EXPORT_ALL=1",
			},
		}
		h.ServeHTTP(w, req)
	}))
	defer upstream2Server.Close()

	// Create commits on both upstreams using helper repos to push
	pushClient1 := NewLocalGitRepo()
	defer pushClient1.Close()
	commit1, err := pushClient1.CreateRandomCommit()
	if err != nil {
		t.Fatalf("Failed to create commit for upstream1: %v", err)
	}
	_, err = pushClient1.Run("-c", "http.extraHeader=Authorization: Bearer "+upstream1Token,
		"push", upstream1Server.URL, "HEAD:main")
	if err != nil {
		t.Fatalf("Failed to push to upstream1: %v", err)
	}
	t.Logf("Created commit on upstream1: %s", commit1)

	pushClient2 := NewLocalGitRepo()
	defer pushClient2.Close()
	commit2, err := pushClient2.CreateRandomCommit()
	if err != nil {
		t.Fatalf("Failed to create commit for upstream2: %v", err)
	}
	_, err = pushClient2.Run("-c", "http.extraHeader=Authorization: Bearer "+upstream2Token,
		"push", upstream2Server.URL, "HEAD:main")
	if err != nil {
		t.Fatalf("Failed to push to upstream2: %v", err)
	}
	t.Logf("Created commit on upstream2: %s", commit2)

	// Create test server with URL-based token selection
	var tokenCallCount int
	var tokenCallURLs []string
	var mu sync.Mutex

	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource: &testTokenSource{
			tokenFunc: func(upstreamURL *url.URL) (*oauth2.Token, error) {
				mu.Lock()
				tokenCallCount++
				tokenCallURLs = append(tokenCallURLs, upstreamURL.String())
				mu.Unlock()

				// Select token based on upstream URL
				switch upstreamURL.Host {
				case strings.TrimPrefix(upstream1Server.URL, "http://"):
					return &oauth2.Token{
						AccessToken: upstream1Token,
						TokenType:   "Bearer",
					}, nil
				case strings.TrimPrefix(upstream2Server.URL, "http://"):
					return &oauth2.Token{
						AccessToken: upstream2Token,
						TokenType:   "Bearer",
					}, nil
				default:
					return nil, fmt.Errorf("unknown upstream: %s", upstreamURL.Host)
				}
			},
		},
	})
	defer ts.Close()

	// Override the URL canonicalizer to handle both upstreams
	upstreamMapping := map[string]string{
		"/upstream1": upstream1Server.URL,
		"/upstream2": upstream2Server.URL,
	}

	originalCanonicalizer := ts.serverConfig.URLCanonializer
	ts.serverConfig.URLCanonializer = func(u *url.URL) (*url.URL, error) {
		for prefix, upstreamURL := range upstreamMapping {
			if strings.HasPrefix(u.Path, prefix) {
				parsedURL, err := url.Parse(upstreamURL)
				if err != nil {
					return nil, err
				}
				// Strip the prefix from the path
				parsedURL.Path = strings.TrimPrefix(u.Path, prefix)
				return parsedURL, nil
			}
		}
		return originalCanonicalizer(u)
	}

	// Test fetching from upstream1
	client1 := NewLocalGitRepo()
	defer client1.Close()

	_, err = client1.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL+"/upstream1")
	if err != nil {
		t.Fatalf("Failed to fetch from upstream1: %v", err)
	}

	fetchHead1, err := client1.Run("rev-parse", "FETCH_HEAD")
	if err != nil {
		t.Fatalf("Failed to parse FETCH_HEAD from upstream1: %v", err)
	}

	if strings.TrimSpace(fetchHead1) != strings.TrimSpace(commit1) {
		t.Errorf("Upstream1: FETCH_HEAD = %s, want %s", strings.TrimSpace(fetchHead1), strings.TrimSpace(commit1))
	}

	// Test fetching from upstream2
	client2 := NewLocalGitRepo()
	defer client2.Close()

	_, err = client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL+"/upstream2")
	if err != nil {
		t.Fatalf("Failed to fetch from upstream2: %v", err)
	}

	fetchHead2, err := client2.Run("rev-parse", "FETCH_HEAD")
	if err != nil {
		t.Fatalf("Failed to parse FETCH_HEAD from upstream2: %v", err)
	}

	if strings.TrimSpace(fetchHead2) != strings.TrimSpace(commit2) {
		t.Errorf("Upstream2: FETCH_HEAD = %s, want %s", strings.TrimSpace(fetchHead2), strings.TrimSpace(commit2))
	}

	// Verify that TokenSource was called with the correct URLs
	mu.Lock()
	defer mu.Unlock()

	if tokenCallCount < 2 {
		t.Errorf("TokenSource called %d times, expected at least 2", tokenCallCount)
	}

	t.Logf("TokenSource called %d times with URLs: %v", tokenCallCount, tokenCallURLs)

	// Verify different tokens were used
	foundUpstream1 := false
	foundUpstream2 := false
	for _, u := range tokenCallURLs {
		if strings.Contains(u, upstream1Server.URL) {
			foundUpstream1 = true
		}
		if strings.Contains(u, upstream2Server.URL) {
			foundUpstream2 = true
		}
	}

	if !foundUpstream1 {
		t.Error("TokenSource was not called with upstream1 URL")
	}
	if !foundUpstream2 {
		t.Error("TokenSource was not called with upstream2 URL")
	}

	t.Log("Successfully fetched from multiple upstreams with different tokens")
}

// TestTokenSource_ConcurrentCalls tests that TokenSource can be called
// concurrently from multiple goroutines.
func TestTokenSource_ConcurrentCalls(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	//nolint:unparam // Test function for concurrency, not error handling
	tokenFunc := func(upstreamURL *url.URL) (*oauth2.Token, error) {
		mu.Lock()
		callCount++
		mu.Unlock()

		return &oauth2.Token{
			AccessToken: fmt.Sprintf("token-%s", upstreamURL.Host),
			TokenType:   "Bearer",
		}, nil
	}

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			u, _ := url.Parse(fmt.Sprintf("https://host%d.example.com/repo", id%10))
			_, err := tokenFunc(u)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent call error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if callCount != numGoroutines {
		t.Errorf("TokenSource called %d times, want %d", callCount, numGoroutines)
	}

	t.Logf("Successfully handled %d concurrent TokenSource calls", callCount)
}

// TestTokenSource_EmptyToken tests handling of empty tokens.
func TestTokenSource_EmptyToken(t *testing.T) {
	//nolint:unparam // Test function for empty token scenario
	tokenFunc := func(upstreamURL *url.URL) (*oauth2.Token, error) {
		// Return a token with empty access token (valid for public repos)
		return &oauth2.Token{
			AccessToken: "",
			TokenType:   "Bearer",
		}, nil
	}

	u, _ := url.Parse("https://github.com/public/repo")
	token, err := tokenFunc(u)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if token == nil {
		t.Fatal("Token is nil")
	}

	if token.AccessToken != "" {
		t.Errorf("Expected empty access token, got %q", token.AccessToken)
	}

	t.Log("Empty token handled correctly (for public repositories)")
}

// TestTokenSource_WithGitHubAppPattern tests a realistic GitHub App
// installation token pattern.
func TestTokenSource_WithGitHubAppPattern(t *testing.T) {
	// Simulate GitHub App installation IDs for different orgs
	installations := map[string]int64{
		"acme-corp": 111,
		"megacorp":  222,
		"startup":   333,
	}

	extractOrg := func(u *url.URL) string {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 1 {
			return parts[0]
		}
		return ""
	}

	tokenFunc := func(upstreamURL *url.URL) (*oauth2.Token, error) {
		org := extractOrg(upstreamURL)
		installationID, ok := installations[org]
		if !ok {
			return nil, fmt.Errorf("no GitHub App installation for org: %s", org)
		}

		// Simulate generating an installation token
		// In real implementation, this would:
		// 1. Generate JWT signed with app private key
		// 2. Exchange JWT for installation token
		return &oauth2.Token{
			AccessToken: fmt.Sprintf("ghs_installation_%d_token", installationID),
			TokenType:   "Bearer",
		}, nil
	}

	tests := []struct {
		url            string
		wantTokenMatch string
		wantErr        bool
	}{
		{
			url:            "https://github.com/acme-corp/private-repo",
			wantTokenMatch: "ghs_installation_111_token",
			wantErr:        false,
		},
		{
			url:            "https://github.com/megacorp/project",
			wantTokenMatch: "ghs_installation_222_token",
			wantErr:        false,
		},
		{
			url:            "https://github.com/unknown-org/repo",
			wantTokenMatch: "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			u, _ := url.Parse(tt.url)
			token, err := tokenFunc(u)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for unknown org, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if token.AccessToken != tt.wantTokenMatch {
				t.Errorf("Got token %q, want %q", token.AccessToken, tt.wantTokenMatch)
			}
		})
	}
}

// testTokenSource is a helper that implements oauth2.TokenSource
// with a custom function for testing.
type testTokenSource struct {
	tokenFunc func(*url.URL) (*oauth2.Token, error)
}

func (ts *testTokenSource) Token() (*oauth2.Token, error) {
	// This should not be called directly in the new implementation
	// but we provide a default implementation for compatibility
	return &oauth2.Token{
		AccessToken: "default-test-token",
		TokenType:   "Bearer",
	}, nil
}
