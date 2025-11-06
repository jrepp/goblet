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
	"strings"
	"testing"
	"time"
)

// TestBasicFetchOperation tests a basic git fetch through the proxy
func TestBasicFetchOperation(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Create a commit on the upstream
	commitHash, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit on upstream: %v", err)
	}

	t.Logf("Created commit %s on upstream", commitHash)

	// Create a client and fetch from proxy
	client := NewLocalGitRepo()
	defer client.Close()

	output, err := client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Failed to fetch from proxy: %v", err)
	}

	t.Logf("Fetch output: %s", output)

	// Verify we got the correct commit
	fetchHead, err := client.Run("rev-parse", "FETCH_HEAD")
	if err != nil {
		t.Fatalf("Failed to parse FETCH_HEAD: %v", err)
	}

	fetchHead = strings.TrimSpace(fetchHead)
	commitHash = strings.TrimSpace(commitHash)

	if fetchHead != commitHash {
		t.Errorf("FETCH_HEAD = %s, want %s", fetchHead, commitHash)
	}

	t.Log("Basic fetch operation successful")
}

// TestMultipleFetchOperations tests multiple fetch operations
func TestMultipleFetchOperations(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	client := NewLocalGitRepo()
	defer client.Close()

	// Add remote
	if _, err := client.Run("remote", "add", "origin", ts.ProxyServerURL); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	commits := make([]string, 3)

	// Create multiple commits and fetch each one
	for i := 0; i < 3; i++ {
		commitHash, err := ts.CreateRandomCommitUpstream()
		if err != nil {
			t.Fatalf("Failed to create commit %d: %v", i, err)
		}
		commits[i] = strings.TrimSpace(commitHash)

		t.Logf("Created commit %d: %s", i, commitHash)

		// Fetch the commit (using HEAD since branch name may vary)
		_, err = client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", "origin")
		if err != nil {
			t.Fatalf("Failed to fetch commit %d: %v", i, err)
		}

		// Verify FETCH_HEAD matches
		fetchHead, err := client.Run("rev-parse", "FETCH_HEAD")
		if err != nil {
			t.Fatalf("Failed to parse FETCH_HEAD for commit %d: %v", i, err)
		}

		fetchHead = strings.TrimSpace(fetchHead)
		if fetchHead != commits[i] {
			t.Errorf("Commit %d: FETCH_HEAD = %s, want %s", i, fetchHead, commits[i])
		}
	}

	t.Log("Multiple fetch operations successful")
}

// TestFetchWithProtocolV2 verifies that protocol v2 is being used
func TestFetchWithProtocolV2(t *testing.T) {
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

	// Explicitly set protocol version to 2
	output, err := client.Run(
		"-c", "protocol.version=2",
		"-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken,
		"fetch", ts.ProxyServerURL,
	)
	if err != nil {
		t.Fatalf("Failed to fetch with protocol v2: %v", err)
	}

	t.Logf("Protocol v2 fetch output: %s", output)
	t.Log("Protocol v2 fetch successful")
}

// TestFetchPerformance tests the performance of fetch operations
func TestFetchPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Create a commit
	_, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	client := NewLocalGitRepo()
	defer client.Close()

	// First fetch (cold cache)
	start := time.Now()
	_, err = client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Failed first fetch: %v", err)
	}
	firstFetchDuration := time.Since(start)

	// Second fetch (warm cache) - same client
	client2 := NewLocalGitRepo()
	defer client2.Close()

	start = time.Now()
	_, err = client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Failed second fetch: %v", err)
	}
	secondFetchDuration := time.Since(start)

	t.Logf("First fetch (cold cache): %v", firstFetchDuration)
	t.Logf("Second fetch (warm cache): %v", secondFetchDuration)

	// The second fetch should typically be faster, but we're not enforcing this
	// as it depends on many factors. Just log the times.
	if secondFetchDuration < firstFetchDuration {
		t.Logf("Cache improved performance by %v", firstFetchDuration-secondFetchDuration)
	}
}

// TestFetchAfterUpstreamUpdate tests fetching after upstream has been updated
func TestFetchAfterUpstreamUpdate(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Initial commit
	firstCommit, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create first commit: %v", err)
	}
	firstCommit = strings.TrimSpace(firstCommit)

	client := NewLocalGitRepo()
	defer client.Close()

	if _, err := client.Run("remote", "add", "origin", ts.ProxyServerURL); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	// First fetch
	if _, err := client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", "origin"); err != nil {
		t.Fatalf("Failed first fetch: %v", err)
	}

	// Create another commit
	secondCommit, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create second commit: %v", err)
	}
	secondCommit = strings.TrimSpace(secondCommit)

	// Second fetch should get the new commit (using HEAD since branch name may vary)
	if _, err := client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", "origin"); err != nil {
		t.Fatalf("Failed second fetch: %v", err)
	}

	fetchHead, err := client.Run("rev-parse", "FETCH_HEAD")
	if err != nil {
		t.Fatalf("Failed to parse FETCH_HEAD: %v", err)
	}
	fetchHead = strings.TrimSpace(fetchHead)

	if fetchHead != secondCommit {
		t.Errorf("FETCH_HEAD = %s, want %s", fetchHead, secondCommit)
	}

	if fetchHead == firstCommit {
		t.Error("FETCH_HEAD still points to first commit, update didn't work")
	}

	t.Log("Fetch after upstream update successful")
}
