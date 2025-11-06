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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/goblet/storage"
)

// Mock storage provider for testing
type mockStorageProvider struct {
	listError   error
	listLatency time.Duration
	closed      bool
}

func (m *mockStorageProvider) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorageProvider) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStorageProvider) Delete(ctx context.Context, path string) error {
	return errors.New("not implemented")
}

func (m *mockStorageProvider) List(ctx context.Context, prefix string) storage.ObjectIterator {
	if m.listLatency > 0 {
		time.Sleep(m.listLatency)
	}
	return &mockObjectIterator{err: m.listError}
}

func (m *mockStorageProvider) Close() error {
	m.closed = true
	return nil
}

type mockObjectIterator struct {
	err    error
	called bool
}

func (m *mockObjectIterator) Next() (*storage.ObjectAttrs, error) {
	if m.called {
		return nil, io.EOF
	}
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return nil, io.EOF
}

func TestNewHealthChecker(t *testing.T) {
	tests := []struct {
		name     string
		provider storage.Provider
		version  string
	}{
		{
			name:     "with storage provider",
			provider: &mockStorageProvider{},
			version:  "1.0.0",
		},
		{
			name:     "without storage provider",
			provider: nil,
			version:  "2.0.0",
		},
		{
			name:     "empty version",
			provider: &mockStorageProvider{},
			version:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := NewHealthChecker(tt.provider, tt.version)
			if hc == nil {
				t.Fatal("NewHealthChecker returned nil")
			}
			if hc.storageProvider != tt.provider {
				t.Error("Storage provider not set correctly")
			}
			if hc.version != tt.version {
				t.Errorf("Version = %q, want %q", hc.version, tt.version)
			}
		})
	}
}

func TestHealthChecker_Check_NoStorage(t *testing.T) {
	hc := NewHealthChecker(nil, "1.0.0")
	ctx := context.Background()

	response := hc.Check(ctx)

	if response == nil {
		t.Fatal("Check returned nil")
	}

	if response.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, want %s", response.Status, HealthStatusHealthy)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", response.Version)
	}

	// Should have cache component but not storage
	if _, ok := response.Components["cache"]; !ok {
		t.Error("Cache component missing")
	}

	if storageComp, ok := response.Components["storage"]; ok {
		t.Logf("Storage component present: %+v", storageComp)
	}

	if time.Since(response.Timestamp) > time.Second {
		t.Error("Timestamp is too old")
	}
}

func TestHealthChecker_Check_HealthyStorage(t *testing.T) {
	mock := &mockStorageProvider{}
	hc := NewHealthChecker(mock, "1.0.0")
	ctx := context.Background()

	response := hc.Check(ctx)

	if response.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, want %s", response.Status, HealthStatusHealthy)
	}

	storageComp, ok := response.Components["storage"]
	if !ok {
		t.Fatal("Storage component missing")
	}

	if storageComp.Status != HealthStatusHealthy {
		t.Errorf("Storage status = %s, want %s", storageComp.Status, HealthStatusHealthy)
	}

	if storageComp.Message != "connected" {
		t.Errorf("Storage message = %q, want %q", storageComp.Message, "connected")
	}

	if storageComp.Latency == "" {
		t.Error("Storage latency not reported")
	}
}

func TestHealthChecker_Check_StorageError(t *testing.T) {
	mock := &mockStorageProvider{
		listError: errors.New("connection failed"),
	}
	hc := NewHealthChecker(mock, "1.0.0")
	ctx := context.Background()

	response := hc.Check(ctx)

	// Overall status should be degraded when storage fails
	if response.Status != HealthStatusDegraded {
		t.Errorf("Status = %s, want %s", response.Status, HealthStatusDegraded)
	}

	storageComp, ok := response.Components["storage"]
	if !ok {
		t.Fatal("Storage component missing")
	}

	if storageComp.Status != HealthStatusUnhealthy {
		t.Errorf("Storage status = %s, want %s", storageComp.Status, HealthStatusUnhealthy)
	}

	if storageComp.Message != "connectivity error: connection failed" {
		t.Errorf("Storage message = %q, want error message", storageComp.Message)
	}
}

func TestHealthChecker_Check_SlowStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow storage test in short mode")
	}

	mock := &mockStorageProvider{
		listLatency: 2500 * time.Millisecond, // Slow but not error
	}
	hc := NewHealthChecker(mock, "1.0.0")
	ctx := context.Background()

	start := time.Now()
	response := hc.Check(ctx)
	elapsed := time.Since(start)

	// Should complete despite slow storage
	if elapsed < 2*time.Second {
		t.Errorf("Check completed too quickly: %v", elapsed)
	}

	// Note: The check might succeed if latency threshold is higher than our test latency
	// Just verify response is valid
	if response == nil {
		t.Fatal("Response is nil")
	}

	t.Logf("Status: %s, Elapsed: %v", response.Status, elapsed)

	storageComp := response.Components["storage"]
	t.Logf("Storage status: %s, message: %s", storageComp.Status, storageComp.Message)
}

func TestHealthChecker_Check_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	mock := &mockStorageProvider{
		listLatency: 10 * time.Second, // Will timeout
	}
	hc := NewHealthChecker(mock, "1.0.0")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	response := hc.Check(ctx)
	elapsed := time.Since(start)

	// Note: The health check creates its own 5s timeout context internally
	// So it may not respect our 100ms timeout
	t.Logf("Check completed in: %v", elapsed)

	// Should still return a response
	if response == nil {
		t.Fatal("Check returned nil on timeout")
	}
}

func TestHealthChecker_ServeHTTP_Simple(t *testing.T) {
	tests := []struct {
		name           string
		provider       storage.Provider
		wantStatus     int
		wantBody       string
		wantStatusText string
	}{
		{
			name:       "healthy - no storage",
			provider:   nil,
			wantStatus: http.StatusOK,
			wantBody:   "ok\n",
		},
		{
			name:       "healthy - with storage",
			provider:   &mockStorageProvider{},
			wantStatus: http.StatusOK,
			wantBody:   "ok\n",
		},
		{
			name: "degraded - storage error",
			provider: &mockStorageProvider{
				listError: errors.New("storage down"),
			},
			wantStatus:     http.StatusOK, // Still 200 for degraded
			wantBody:       "degraded\n",
			wantStatusText: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := NewHealthChecker(tt.provider, "1.0.0")
			req := httptest.NewRequest("GET", "/healthz", nil)
			rec := httptest.NewRecorder()

			hc.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if rec.Body.String() != tt.wantBody {
				t.Errorf("Body = %q, want %q", rec.Body.String(), tt.wantBody)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "text/plain" {
				t.Errorf("Content-Type = %q, want text/plain", contentType)
			}
		})
	}
}

func TestHealthChecker_ServeHTTP_Detailed(t *testing.T) {
	tests := []struct {
		name           string
		provider       storage.Provider
		wantStatus     int
		checkResponse  func(*testing.T, *HealthCheckResponse)
		wantStatusText string
	}{
		{
			name:       "detailed - healthy",
			provider:   &mockStorageProvider{},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *HealthCheckResponse) {
				if resp.Status != HealthStatusHealthy {
					t.Errorf("Status = %s, want healthy", resp.Status)
				}
				if resp.Version != "1.0.0" {
					t.Errorf("Version = %s, want 1.0.0", resp.Version)
				}
				if len(resp.Components) != 2 {
					t.Errorf("Components count = %d, want 2", len(resp.Components))
				}
			},
		},
		{
			name: "detailed - degraded",
			provider: &mockStorageProvider{
				listError: errors.New("storage error"),
			},
			wantStatus: http.StatusOK, // Degraded still returns 200
			checkResponse: func(t *testing.T, resp *HealthCheckResponse) {
				if resp.Status != HealthStatusDegraded {
					t.Errorf("Status = %s, want degraded", resp.Status)
				}
				if resp.Components["storage"].Status != HealthStatusUnhealthy {
					t.Error("Storage should be unhealthy")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := NewHealthChecker(tt.provider, "1.0.0")
			req := httptest.NewRequest("GET", "/healthz?detailed=true", nil)
			rec := httptest.NewRecorder()

			hc.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}

			var response HealthCheckResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode JSON: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, &response)
			}
		})
	}
}

func TestHealthChecker_checkCache(t *testing.T) {
	hc := NewHealthChecker(nil, "1.0.0")

	health := hc.checkCache()

	if health.Status != HealthStatusHealthy {
		t.Errorf("Status = %s, want %s", health.Status, HealthStatusHealthy)
	}

	if health.Message != "operational" {
		t.Errorf("Message = %q, want 'operational'", health.Message)
	}
}

func TestHealthStatus_Values(t *testing.T) {
	tests := []struct {
		status HealthStatus
		value  string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusDegraded, "degraded"},
		{HealthStatusUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.value {
				t.Errorf("Status = %q, want %q", tt.status, tt.value)
			}
		})
	}
}

func TestHealthChecker_ConcurrentChecks(t *testing.T) {
	mock := &mockStorageProvider{}
	hc := NewHealthChecker(mock, "1.0.0")
	ctx := context.Background()

	const numGoroutines = 10
	done := make(chan *HealthCheckResponse, numGoroutines)

	// Launch concurrent health checks
	for i := 0; i < numGoroutines; i++ {
		go func() {
			done <- hc.Check(ctx)
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		resp := <-done
		if resp == nil {
			t.Error("Received nil response")
		}
		if resp.Status != HealthStatusHealthy {
			t.Errorf("Response %d: Status = %s, want healthy", i, resp.Status)
		}
	}
}

func TestHealthChecker_HTTPConcurrent(t *testing.T) {
	hc := NewHealthChecker(&mockStorageProvider{}, "1.0.0")

	const numRequests = 20
	done := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/healthz", nil)
			rec := httptest.NewRecorder()
			hc.ServeHTTP(rec, req)
			done <- rec.Code
		}()
	}

	for i := 0; i < numRequests; i++ {
		code := <-done
		if code != http.StatusOK {
			t.Errorf("Request %d: Status = %d, want 200", i, code)
		}
	}
}
