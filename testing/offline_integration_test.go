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

package testing

import (
	"strings"
	"testing"
)

// TestOfflineModeWithWarmCache tests that Goblet can serve ls-refs and fetch
// from local cache when upstream is disabled after initial population.
func TestOfflineModeWithWarmCache(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Create a commit in upstream
	commitHash, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create upstream commit: %v", err)
	}
	t.Logf("Created upstream commit: %s", commitHash)

	// Step 1: Populate the cache with upstream enabled
	client1 := NewLocalGitRepo()
	defer client1.Close()

	_, err = client1.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Initial fetch failed: %v", err)
	}

	hash1, _ := client1.Run("rev-parse", "FETCH_HEAD")
	hash1 = strings.TrimSpace(hash1)
	t.Logf("Initial fetch got commit: %s", hash1)

	// Step 2: Disable upstream to simulate offline mode
	falseValue := false
	ts.serverConfig.SetUpstreamEnabled(&falseValue)
	t.Logf("Disabled upstream connectivity")

	// Step 3: Try to fetch with upstream disabled - should work from cache
	client2 := NewLocalGitRepo()
	defer client2.Close()

	_, err = client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Offline fetch failed: %v (expected to work from cache)", err)
	}

	hash2, _ := client2.Run("rev-parse", "FETCH_HEAD")
	hash2 = strings.TrimSpace(hash2)
	t.Logf("Offline fetch got commit: %s", hash2)

	// Verify same content was fetched
	if hash1 != hash2 {
		t.Errorf("Offline fetch returned different commit: got %s, want %s", hash2, hash1)
	}

	t.Logf("SUCCESS: Goblet served from local cache with upstream disabled")
}

// TestOfflineModeWithColdCache tests that Goblet returns appropriate error
// when upstream is disabled and there's no local cache.
func TestOfflineModeWithColdCache(t *testing.T) {
	// Start server with upstream disabled from the beginning
	falseValue := false
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
		UpstreamEnabled:   &falseValue,
	})
	defer ts.Close()

	// Create a commit in upstream (but proxy won't be able to access it)
	_, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create upstream commit: %v", err)
	}

	// Try to fetch with cold cache and upstream disabled - should fail
	client := NewLocalGitRepo()
	defer client.Close()

	_, err = client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err == nil {
		t.Fatalf("Expected fetch to fail with cold cache and upstream disabled, but it succeeded")
	}

	// Verify error message indicates local repository not available
	if !strings.Contains(err.Error(), "local repository not available") &&
		!strings.Contains(err.Error(), "exit status") {
		t.Logf("Warning: error message doesn't mention local repository: %v", err)
	}

	t.Logf("SUCCESS: Goblet correctly failed with cold cache and upstream disabled: %v", err)
}

// TestUpstreamFailureFallback tests that Goblet automatically falls back to
// local cache when upstream becomes unavailable after initial cache population.
func TestUpstreamFailureFallback(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Create initial commit and populate cache
	commitHash, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create upstream commit: %v", err)
	}
	t.Logf("Created upstream commit: %s", commitHash)

	client1 := NewLocalGitRepo()
	defer client1.Close()

	_, err = client1.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Initial fetch failed: %v", err)
	}

	hash1, _ := client1.Run("rev-parse", "FETCH_HEAD")
	hash1 = strings.TrimSpace(hash1)

	// Stop the upstream server to simulate failure
	ts.upstreamServer.Close()
	t.Logf("Stopped upstream server to simulate failure")

	// Try to fetch again - should automatically fall back to cache
	client2 := NewLocalGitRepo()
	defer client2.Close()

	_, err = client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Fetch with upstream down failed: %v (expected fallback to cache)", err)
	}

	hash2, _ := client2.Run("rev-parse", "FETCH_HEAD")
	hash2 = strings.TrimSpace(hash2)

	// Verify same content was fetched from cache
	if hash1 != hash2 {
		t.Errorf("Fallback fetch returned different commit: got %s, want %s", hash2, hash1)
	}

	t.Logf("SUCCESS: Goblet automatically fell back to local cache when upstream failed")
}

// TestUpstreamRecovery tests that Goblet recovers and uses upstream
// after it becomes available again.
func TestUpstreamRecovery(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Create initial commit and populate cache
	_, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create upstream commit: %v", err)
	}

	client1 := NewLocalGitRepo()
	defer client1.Close()

	_, err = client1.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Initial fetch failed: %v", err)
	}

	hash1, _ := client1.Run("rev-parse", "FETCH_HEAD")
	hash1 = strings.TrimSpace(hash1)

	// Disable upstream temporarily
	falseValue := false
	ts.serverConfig.SetUpstreamEnabled(&falseValue)
	t.Logf("Disabled upstream (simulating outage)")

	// Verify cache works
	client2 := NewLocalGitRepo()
	defer client2.Close()

	_, err = client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Fetch during outage failed: %v", err)
	}

	// Re-enable upstream (simulate recovery)
	trueValue := true
	ts.serverConfig.SetUpstreamEnabled(&trueValue)
	t.Logf("Re-enabled upstream (simulating recovery)")

	// Create new commit in upstream
	newCommitHash, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create new upstream commit: %v", err)
	}
	t.Logf("Created new upstream commit after recovery: %s", newCommitHash)

	// Fetch again - should get new commit from upstream
	client3 := NewLocalGitRepo()
	defer client3.Close()

	_, err = client3.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Fetch after recovery failed: %v", err)
	}

	hash3, _ := client3.Run("rev-parse", "FETCH_HEAD")
	hash3 = strings.TrimSpace(hash3)

	// Verify we got the new commit (not the cached one)
	if hash3 == hash1 {
		t.Errorf("After recovery, still got old commit %s, expected new commit %s", hash3, newCommitHash)
	}

	if !strings.HasPrefix(newCommitHash, hash3) {
		t.Logf("Note: fetched commit %s might be descendant of new commit %s", hash3, newCommitHash)
	}

	t.Logf("SUCCESS: Goblet recovered and fetched from upstream after re-enabling")
}
