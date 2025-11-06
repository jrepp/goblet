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
	"sync"
	"testing"
	"time"
)

// TestCacheHitBehavior tests that subsequent fetches use the cache
func TestCacheHitBehavior(t *testing.T) {
	// Track requests to upstream
	var upstreamRequests int
	var mu sync.Mutex

	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
		RequestLogger: func(r *http.Request, status int, requestSize, responseSize int64, latency time.Duration) {
			mu.Lock()
			defer mu.Unlock()
			upstreamRequests++
			t.Logf("Request: %s %s, Status: %d, Latency: %v", r.Method, r.URL.Path, status, latency)
		},
	})
	defer ts.Close()

	_, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// First fetch - should miss cache
	client1 := NewLocalGitRepo()
	defer client1.Close()

	start := time.Now()
	_, err = client1.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("First fetch failed: %v", err)
	}
	firstFetchTime := time.Since(start)

	// Second fetch - should hit cache
	client2 := NewLocalGitRepo()
	defer client2.Close()

	start = time.Now()
	_, err = client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
	if err != nil {
		t.Fatalf("Second fetch failed: %v", err)
	}
	secondFetchTime := time.Since(start)

	t.Logf("First fetch time: %v", firstFetchTime)
	t.Logf("Second fetch time: %v", secondFetchTime)

	// Verify both fetches got the same content
	hash1, _ := client1.Run("rev-parse", "FETCH_HEAD")
	hash2, _ := client2.Run("rev-parse", "FETCH_HEAD")

	if strings.TrimSpace(hash1) != strings.TrimSpace(hash2) {
		t.Errorf("Fetches got different commits: %s vs %s", hash1, hash2)
	}
}

// TestCacheConsistency tests that multiple concurrent fetches remain consistent
func TestCacheConsistency(t *testing.T) {
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

	// Launch multiple concurrent fetches
	const numClients = 5
	var wg sync.WaitGroup
	results := make([]string, numClients)
	errors := make([]error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			client := NewLocalGitRepo()
			defer client.Close()

			_, err := client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL)
			if err != nil {
				errors[idx] = err
				return
			}

			hash, err := client.Run("rev-parse", "FETCH_HEAD")
			if err != nil {
				errors[idx] = err
				return
			}

			results[idx] = strings.TrimSpace(hash)
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			t.Errorf("Client %d failed: %v", i, err)
		}
	}

	// Check all results are consistent
	for i, hash := range results {
		if hash != commitHash {
			t.Errorf("Client %d got hash %s, want %s", i, hash, commitHash)
		}
	}

	t.Logf("All %d concurrent clients got consistent results: %s", numClients, commitHash)
}

// TestCacheInvalidationOnUpdate tests that cache updates when upstream changes
func TestCacheInvalidationOnUpdate(t *testing.T) {
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// First commit
	firstCommit, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create first commit: %v", err)
	}
	firstCommit = strings.TrimSpace(firstCommit)

	// First fetch
	client1 := NewLocalGitRepo()
	defer client1.Close()
	if _, err := client1.Run("remote", "add", "origin", ts.ProxyServerURL); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	if _, err := client1.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", "origin"); err != nil {
		t.Fatalf("First fetch failed: %v", err)
	}

	hash1, _ := client1.Run("rev-parse", "FETCH_HEAD")
	hash1 = strings.TrimSpace(hash1)

	if hash1 != firstCommit {
		t.Errorf("First fetch: got %s, want %s", hash1, firstCommit)
	}

	// Update upstream
	secondCommit, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create second commit: %v", err)
	}
	secondCommit = strings.TrimSpace(secondCommit)

	// Wait a bit to ensure update is visible
	time.Sleep(100 * time.Millisecond)

	// Second fetch should get updated content
	client2 := NewLocalGitRepo()
	defer client2.Close()
	if _, err := client2.Run("remote", "add", "origin", ts.ProxyServerURL); err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}
	// Fetch all refs to get the update
	if _, err := client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", "origin"); err != nil {
		t.Fatalf("Second fetch failed: %v", err)
	}

	hash2, _ := client2.Run("rev-parse", "FETCH_HEAD")
	hash2 = strings.TrimSpace(hash2)

	if hash2 != secondCommit {
		t.Errorf("Second fetch: got %s, want %s", hash2, secondCommit)
	}

	if hash2 == hash1 {
		t.Error("Cache not updated after upstream change")
	}

	t.Logf("Cache invalidation successful: %s -> %s", firstCommit, secondCommit)
}

// TestCacheWithDifferentRepositories tests caching across different repositories
func TestCacheWithDifferentRepositories(t *testing.T) {
	// Create two separate test servers (representing different repositories)
	ts1 := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts1.Close()

	ts2 := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts2.Close()

	// Create commits in both
	commit1, err := ts1.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit in repo 1: %v", err)
	}
	commit1 = strings.TrimSpace(commit1)

	commit2, err := ts2.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit in repo 2: %v", err)
	}
	commit2 = strings.TrimSpace(commit2)

	// Fetch from both
	client1 := NewLocalGitRepo()
	defer client1.Close()
	if _, err := client1.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts1.ProxyServerURL); err != nil {
		t.Fatalf("Failed to fetch from repo 1: %v", err)
	}

	client2 := NewLocalGitRepo()
	defer client2.Close()
	if _, err := client2.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts2.ProxyServerURL); err != nil {
		t.Fatalf("Failed to fetch from repo 2: %v", err)
	}

	// Verify we got different commits
	hash1, _ := client1.Run("rev-parse", "FETCH_HEAD")
	hash2, _ := client2.Run("rev-parse", "FETCH_HEAD")

	hash1 = strings.TrimSpace(hash1)
	hash2 = strings.TrimSpace(hash2)

	if hash1 != commit1 {
		t.Errorf("Repo 1: got %s, want %s", hash1, commit1)
	}

	if hash2 != commit2 {
		t.Errorf("Repo 2: got %s, want %s", hash2, commit2)
	}

	if hash1 == hash2 {
		t.Error("Different repositories should not have the same commits")
	}

	t.Log("Cache correctly isolates different repositories")
}
