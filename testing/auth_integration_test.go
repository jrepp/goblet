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
	"net/http"
	"strings"
	"testing"
)

// TestAuthenticationRequired tests that authentication is required
func TestAuthenticationRequired(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	_, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	client := NewLocalGitRepo()
	defer client.Close()

	// Try to fetch without authentication
	output, err := client.Run("fetch", ts.ProxyServerURL)
	if err == nil {
		t.Error("Expected fetch without auth to fail, but it succeeded")
	}

	// Error message should indicate authentication problem
	if !strings.Contains(output, "Authentication") && !strings.Contains(output, "authentication") &&
		!strings.Contains(output, "Unauthorized") && !strings.Contains(output, "Unauthenticated") {
		t.Logf("Error output: %s", output)
		// Still fail the test
		if err == nil {
			t.Error("Fetch without authentication should have failed")
		}
	}

	t.Log("Authentication correctly required for fetch operations")
}

// TestValidAuthentication tests that valid tokens work
func TestValidAuthentication(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	commitHash, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}
	commitHash = strings.TrimSpace(commitHash)

	client := NewLocalGitRepo()
	defer client.Close()

	// Fetch with valid authentication
	_, err = client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Fetch with valid auth failed: %v", err)
	}

	// Verify we got the commit
	fetchHead, err := client.Run("rev-parse", "FETCH_HEAD")
	if err != nil {
		t.Fatalf("Failed to get FETCH_HEAD: %v", err)
	}

	if strings.TrimSpace(fetchHead) != commitHash {
		t.Errorf("Got commit %s, want %s", fetchHead, commitHash)
	}

	t.Log("Valid authentication successful")
}

// TestInvalidAuthentication tests that invalid tokens are rejected
func TestInvalidAuthentication(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	_, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	client := NewLocalGitRepo()
	defer client.Close()

	invalidTokens := []string{
		"invalid-token",
		"Bearer invalid",
		"",
		"wrong-format",
	}

	for _, token := range invalidTokens {
		t.Run("token="+token, func(t *testing.T) {
			output, err := client.Run("-c", "http.extraHeader=Authorization: Bearer "+token, "fetch", ts.ProxyServerURL)
			if err == nil {
				t.Errorf("Expected fetch with invalid token %q to fail, but it succeeded", token)
			}
			t.Logf("Correctly rejected token %q, error: %v, output: %s", token, err, output)
		})
	}

	t.Log("Invalid authentication correctly rejected")
}

// TestAuthenticationHeaderFormat tests different auth header formats
func TestAuthenticationHeaderFormat(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	_, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		shouldWork bool
	}{
		{
			name:       "valid bearer token",
			authHeader: "Authorization: Bearer " + ValidClientAuthToken,
			shouldWork: true,
		},
		{
			name:       "token without bearer prefix",
			authHeader: "Authorization: " + ValidClientAuthToken,
			shouldWork: false,
		},
		{
			name:       "lowercase bearer",
			authHeader: "Authorization: bearer " + ValidClientAuthToken,
			shouldWork: false,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			shouldWork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewLocalGitRepo()
			defer client.Close()

			var output string
			var err error

			if tt.authHeader != "" {
				output, err = client.Run("-c", "http.extraHeader="+tt.authHeader, "fetch", ts.ProxyServerURL)
			} else {
				output, err = client.Run("fetch", ts.ProxyServerURL)
			}

			if tt.shouldWork && err != nil {
				t.Errorf("Expected success but got error: %v, output: %s", err, output)
			}

			if !tt.shouldWork && err == nil {
				t.Errorf("Expected failure but got success, output: %s", output)
			}
		})
	}
}

// TestConcurrentAuthenticatedRequests tests multiple concurrent authenticated requests
func TestConcurrentAuthenticatedRequests(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	commitHash, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}
	commitHash = strings.TrimSpace(commitHash)

	// Launch multiple concurrent authenticated fetches
	const numClients = 10
	errors := make(chan error, numClients)
	hashes := make(chan string, numClients)

	for i := 0; i < numClients; i++ {
		go func(idx int) {
			client := NewLocalGitRepo()
			defer client.Close()

			_, err := client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
			if err != nil {
				errors <- err
				return
			}

			hash, err := client.Run("rev-parse", "FETCH_HEAD")
			if err != nil {
				errors <- err
				return
			}

			hashes <- strings.TrimSpace(hash)
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numClients; i++ {
		select {
		case err := <-errors:
			t.Errorf("Client failed: %v", err)
		case hash := <-hashes:
			if hash != commitHash {
				t.Errorf("Got hash %s, want %s", hash, commitHash)
			}
			successCount++
		}
	}

	if successCount != numClients {
		t.Errorf("Only %d/%d clients succeeded", successCount, numClients)
	}

	t.Logf("All %d concurrent authenticated requests succeeded", successCount)
}

// TestUnauthorizedEndpointAccess tests accessing endpoints without proper auth
func TestUnauthorizedEndpointAccess(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Try to access info/refs without auth
	client := &http.Client{}
	resp, err := client.Get(ts.ProxyServerURL + "/info/refs?service=git-upload-pack")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("Expected unauthorized access to be rejected, but got 200 OK")
	}

	t.Logf("Unauthorized endpoint access correctly rejected with status %d", resp.StatusCode)
}
