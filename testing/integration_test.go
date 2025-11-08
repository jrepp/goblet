// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at.
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software.
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and.
// limitations under the License.

// Package testing provides integration tests for the Goblet server.
// These tests require Docker to be running and will start a Minio container.
package testing

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// IntegrationTestSetup manages the Docker Compose environment for integration tests.
type IntegrationTestSetup struct {
	composeFile   string
	projectName   string
	useComposeV2  bool
}

// NewIntegrationTestSetup creates a new integration test setup.
func NewIntegrationTestSetup() *IntegrationTestSetup {
	return &IntegrationTestSetup{
		composeFile: "../docker-compose.test.yml",
		projectName: "goblet-test",
	}
}

// getComposeCommand returns the appropriate docker compose command based on what's available.
func (its *IntegrationTestSetup) getComposeCommand(ctx context.Context, args ...string) *exec.Cmd {
	if its.useComposeV2 {
		// Use docker compose (v2)
		composeArgs := append([]string{"compose", "-f", its.composeFile, "-p", its.projectName}, args...)
		return exec.CommandContext(ctx, "docker", composeArgs...)
	}
	// Use docker-compose (v1)
	composeArgs := append([]string{"-f", its.composeFile, "-p", its.projectName}, args...)
	return exec.CommandContext(ctx, "docker-compose", composeArgs...)
}

// Start brings up the Docker Compose environment.
func (its *IntegrationTestSetup) Start(t *testing.T) {
	t.Helper()

	// Check if Docker is available
	if _, err := exec.LookPath("docker-compose"); err != nil {
		if _, err := exec.LookPath("docker"); err != nil {
			t.Skip("Docker is not available, skipping integration tests")
			return
		}
		// Try docker compose (new style)
		cmd := exec.Command("docker", "compose", "version")
		if err := cmd.Run(); err != nil {
			t.Skip("Docker Compose is not available, skipping integration tests")
			return
		}
		its.useComposeV2 = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("Starting Docker Compose environment for integration tests...")

	// Stop any existing services first
	stopCmd := its.getComposeCommand(ctx, "down", "-v")
	stopCmd.Stdout = os.Stdout
	stopCmd.Stderr = os.Stderr
	_ = stopCmd.Run() // Ignore errors if nothing is running

	// Start services
	startCmd := its.getComposeCommand(ctx, "up", "-d")
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	if err := startCmd.Run(); err != nil {
		t.Fatalf("Failed to start Docker Compose: %v", err)
	}

	// Wait for services to be healthy
	t.Log("Waiting for services to be healthy...")
	time.Sleep(10 * time.Second)
}

// Stop tears down the Docker Compose environment.
func (its *IntegrationTestSetup) Stop(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Stopping Docker Compose environment...")
	cmd := its.getComposeCommand(ctx, "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to stop Docker Compose: %v", err)
	}
}

// GetMinioEndpoint returns the Minio endpoint for tests.
func (its *IntegrationTestSetup) GetMinioEndpoint() string {
	return "localhost:9000"
}

// GetMinioCredentials returns the Minio credentials for tests.
func (its *IntegrationTestSetup) GetMinioCredentials() (accessKey, secretKey string) {
	return "minioadmin", "minioadmin"
}

// GetMinioBucket returns the Minio bucket name for tests.
func (its *IntegrationTestSetup) GetMinioBucket() string {
	return "goblet-test"
}
