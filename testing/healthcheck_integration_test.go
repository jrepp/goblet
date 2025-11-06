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
	"io"
	"net/http"
	"testing"
	"time"
)

// TestHealthCheckEndpoint tests the /healthz endpoint
func TestHealthCheckEndpoint(t *testing.T) {
	// Setup test server
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	tests := []struct {
		name           string
		endpoint       string
		wantStatus     int
		wantBody       string
		wantStatusCode int
	}{
		{
			name:           "health check returns ok",
			endpoint:       "/healthz",
			wantStatus:     http.StatusOK,
			wantBody:       "ok\n",
			wantStatusCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make request to the health endpoint
			resp, err := http.Get(ts.ProxyServerURL + tt.endpoint)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if string(body) != tt.wantBody {
				t.Errorf("Response body = %q, want %q", string(body), tt.wantBody)
			}

			// Verify content type
			contentType := resp.Header.Get("Content-Type")
			if contentType != "text/plain" {
				t.Errorf("Content-Type = %q, want %q", contentType, "text/plain")
			}
		})
	}
}

// TestHealthCheckWithMinio tests health check with actual Minio instance
func TestHealthCheckWithMinio(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewIntegrationTestSetup()
	setup.Start(t)
	defer setup.Stop(t)

	// Test Minio health endpoint
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://" + setup.GetMinioEndpoint() + "/minio/health/live")
	if err != nil {
		t.Fatalf("Failed to connect to Minio: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Minio health check failed with status: %d", resp.StatusCode)
	}

	t.Log("Minio is healthy and responding")
}

// TestServerReadiness tests that the server becomes ready quickly
func TestServerReadiness(t *testing.T) {
	start := time.Now()

	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	elapsed := time.Since(start)

	// Server should be ready in under 5 seconds
	if elapsed > 5*time.Second {
		t.Errorf("Server took too long to start: %v", elapsed)
	}

	// Verify it responds to health checks
	resp, err := http.Get(ts.ProxyServerURL + "/healthz")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check failed with status: %d", resp.StatusCode)
	}

	t.Logf("Server became ready in %v", elapsed)
}
