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
	"net/http"
	"time"

	"github.com/google/goblet/storage"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	// HealthStatusHealthy indicates all systems are operational
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates some non-critical systems are impaired
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy indicates critical systems are failing
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
	Latency string       `json:"latency,omitempty"`
}

// HealthCheckResponse represents the full health check response
type HealthCheckResponse struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version,omitempty"`
	Components map[string]ComponentHealth `json:"components"`
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	storageProvider storage.Provider
	version         string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(provider storage.Provider, version string) *HealthChecker {
	return &HealthChecker{
		storageProvider: provider,
		version:         version,
	}
}

// Check performs a health check and returns the status
func (hc *HealthChecker) Check(ctx context.Context) *HealthCheckResponse {
	response := &HealthCheckResponse{
		Status:     HealthStatusHealthy,
		Timestamp:  time.Now(),
		Version:    hc.version,
		Components: make(map[string]ComponentHealth),
	}

	// Check storage connectivity if configured
	if hc.storageProvider != nil {
		storageHealth := hc.checkStorage(ctx)
		response.Components["storage"] = storageHealth

		// Degrade overall status if storage is unhealthy
		// Note: Storage issues are not critical for read operations
		if storageHealth.Status == HealthStatusUnhealthy {
			response.Status = HealthStatusDegraded
		}
	}

	// Check disk cache - always present
	cacheHealth := hc.checkCache()
	response.Components["cache"] = cacheHealth
	if cacheHealth.Status == HealthStatusUnhealthy {
		response.Status = HealthStatusUnhealthy
	}

	return response
}

// checkStorage checks the storage provider connectivity
func (hc *HealthChecker) checkStorage(ctx context.Context) ComponentHealth {
	if hc.storageProvider == nil {
		return ComponentHealth{
			Status:  HealthStatusHealthy,
			Message: "not configured",
		}
	}

	start := time.Now()

	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to list objects - this tests connectivity and permissions
	iter := hc.storageProvider.List(checkCtx, "")
	_, err := iter.Next()

	latency := time.Since(start)

	if err != nil && err.Error() != "EOF" {
		// Real error (not just empty listing)
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "connectivity error: " + err.Error(),
			Latency: latency.String(),
		}
	}

	// Check if latency is concerning
	if latency > 2*time.Second {
		return ComponentHealth{
			Status:  HealthStatusDegraded,
			Message: "slow response",
			Latency: latency.String(),
		}
	}

	return ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "connected",
		Latency: latency.String(),
	}
}

// checkCache checks the local disk cache health
func (hc *HealthChecker) checkCache() ComponentHealth {
	// For now, we assume cache is healthy if the service is running
	// In a real implementation, you'd check disk space, permissions, etc.
	return ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "operational",
	}
}

// ServeHTTP implements http.Handler for health check endpoint
func (hc *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Support both simple and detailed health checks
	detailed := r.URL.Query().Get("detailed") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	health := hc.Check(ctx)

	if !detailed {
		// Simple health check - just return status code and text
		if health.Status == HealthStatusHealthy {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok\n"))
			return
		}

		status := http.StatusServiceUnavailable
		if health.Status == HealthStatusDegraded {
			status = http.StatusOK // Still OK for degraded
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		w.Write([]byte(string(health.Status) + "\n"))
		return
	}

	// Detailed health check - return JSON
	w.Header().Set("Content-Type", "application/json")

	status := http.StatusOK
	if health.Status == HealthStatusUnhealthy {
		status = http.StatusServiceUnavailable
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(health)
}
