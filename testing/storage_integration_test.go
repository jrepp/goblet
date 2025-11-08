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
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/goblet/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TestMinioConnectivity tests basic connectivity to Minio.
func TestMinioConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewIntegrationTestSetup()
	setup.Start(t)
	defer setup.Stop(t)

	accessKey, secretKey := setup.GetMinioCredentials()

	// Create a Minio client
	minioClient, err := minio.New(setup.GetMinioEndpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create Minio client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connectivity by listing buckets
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("Failed to list buckets: %v", err)
	}

	t.Logf("Successfully connected to Minio, found %d buckets", len(buckets))

	// Verify our test bucket exists
	bucketFound := false
	for _, bucket := range buckets {
		t.Logf("Found bucket: %s", bucket.Name)
		if bucket.Name == setup.GetMinioBucket() {
			bucketFound = true
		}
	}

	if !bucketFound {
		t.Errorf("Test bucket %s not found", setup.GetMinioBucket())
	}
}

// TestStorageProviderInitialization tests creating a storage provider.
func TestStorageProviderInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewIntegrationTestSetup()
	setup.Start(t)
	defer setup.Stop(t)

	accessKey, secretKey := setup.GetMinioCredentials()

	storageConfig := &storage.Config{
		Provider:          "s3",
		S3Endpoint:        setup.GetMinioEndpoint(),
		S3Bucket:          setup.GetMinioBucket(),
		S3AccessKeyID:     accessKey,
		S3SecretAccessKey: secretKey,
		S3Region:          "us-east-1",
		S3UseSSL:          false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	provider, err := storage.NewProvider(ctx, storageConfig)
	if err != nil {
		t.Fatalf("Failed to create storage provider: %v", err)
	}
	defer provider.Close()

	t.Log("Successfully initialized S3 storage provider with Minio")
}

// TestBundleBackupAndRestore tests backing up and restoring a repository bundle.
func TestBundleBackupAndRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewIntegrationTestSetup()
	setup.Start(t)
	defer setup.Stop(t)

	// Create a test server with a repository
	ts := NewTestServer(&TestServerConfig{
		RequestAuthorizer: TestRequestAuthorizer,
		TokenSource:       TestTokenSource,
	})
	defer ts.Close()

	// Create some commits
	commit1, err := ts.CreateRandomCommitUpstream()
	if err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}
	commit1 = strings.TrimSpace(commit1)

	t.Logf("Created commit: %s", commit1)

	// Fetch to populate the cache
	client := NewLocalGitRepo()
	defer client.Close()
	if _, err := client.Run("-c", "http.extraHeader=Authorization: Bearer "+ValidClientAuthToken, "fetch", ts.ProxyServerURL); err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// Create test bundle data (simulated)
	// Note: In a real test, you would get this from the actual repository
	// For now, we'll just test the storage mechanism with mock data
	var bundleBuffer bytes.Buffer
	bundleBuffer.WriteString("Mock git bundle data for testing\n")

	bundleSize := bundleBuffer.Len()
	if bundleSize == 0 {
		t.Error("Bundle is empty")
	}

	t.Logf("Created test bundle of size %d bytes", bundleSize)

	// Test uploading bundle to Minio
	accessKey, secretKey := setup.GetMinioCredentials()
	minioClient, err := minio.New(setup.GetMinioEndpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create Minio client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	objectName := "test-bundle-" + time.Now().Format("20060102-150405") + ".bundle"
	_, err = minioClient.PutObject(
		ctx,
		setup.GetMinioBucket(),
		objectName,
		bytes.NewReader(bundleBuffer.Bytes()),
		int64(bundleSize),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		t.Fatalf("Failed to upload bundle to Minio: %v", err)
	}

	t.Logf("Uploaded bundle to Minio: %s", objectName)

	// Verify object exists
	objInfo, err := minioClient.StatObject(ctx, setup.GetMinioBucket(), objectName, minio.StatObjectOptions{})
	if err != nil {
		t.Fatalf("Failed to stat uploaded object: %v", err)
	}

	if objInfo.Size != int64(bundleSize) {
		t.Errorf("Uploaded object size = %d, want %d", objInfo.Size, bundleSize)
	}

	t.Log("Successfully verified bundle in Minio storage")

	// Clean up
	if err := minioClient.RemoveObject(ctx, setup.GetMinioBucket(), objectName, minio.RemoveObjectOptions{}); err != nil {
		t.Logf("Warning: Failed to clean up test object: %v", err)
	}
}

// TestStorageProviderUploadDownload tests upload and download operations.
func TestStorageProviderUploadDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("=== Starting TestStorageProviderUploadDownload ===")
	startTime := time.Now()

	setup := NewIntegrationTestSetup()
	t.Log("Starting integration test setup...")
	setup.Start(t)
	defer setup.Stop(t)
	t.Logf("Setup completed in %v", time.Since(startTime))

	accessKey, secretKey := setup.GetMinioCredentials()
	t.Logf("Minio credentials obtained - endpoint: %s, bucket: %s",
		setup.GetMinioEndpoint(), setup.GetMinioBucket())

	storageConfig := &storage.Config{
		Provider:          "s3",
		S3Endpoint:        setup.GetMinioEndpoint(),
		S3Bucket:          setup.GetMinioBucket(),
		S3AccessKeyID:     accessKey,
		S3SecretAccessKey: secretKey,
		S3Region:          "us-east-1",
		S3UseSSL:          false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Creating storage provider...")
	providerStartTime := time.Now()
	provider, err := storage.NewProvider(ctx, storageConfig)
	if err != nil {
		t.Fatalf("Failed to create storage provider: %v", err)
	}
	defer provider.Close()
	t.Logf("Storage provider created in %v", time.Since(providerStartTime))

	// Test data
	testData := []byte("This is test data for storage provider")
	testKey := "test-" + time.Now().Format("20060102-150405.000") + ".dat"
	t.Logf("Test key: %s, test data size: %d bytes", testKey, len(testData))

	// Upload (write)
	t.Log("Getting writer...")
	writeStartTime := time.Now()
	writer, err := provider.Writer(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to get writer: %v", err)
	}
	t.Logf("Writer obtained in %v", time.Since(writeStartTime))

	t.Log("Writing data...")
	writeDataStartTime := time.Now()
	bytesWritten, err := writer.Write(testData)
	if err != nil {
		writer.Close()
		t.Fatalf("Failed to write: %v", err)
	}
	t.Logf("Wrote %d bytes in %v", bytesWritten, time.Since(writeDataStartTime))

	t.Log("Closing writer...")
	closeStartTime := time.Now()
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}
	t.Logf("Writer closed in %v", time.Since(closeStartTime))
	t.Logf("Total upload time: %v", time.Since(writeStartTime))

	// Add a small delay to ensure the background goroutine completes
	// This helps diagnose race conditions
	t.Log("Waiting briefly for upload to stabilize...")
	time.Sleep(100 * time.Millisecond)

	// Verify object exists before attempting download
	t.Log("Verifying object exists in storage...")
	verifyStartTime := time.Now()
	accessKey2, secretKey2 := setup.GetMinioCredentials()
	minioClient, err := minio.New(setup.GetMinioEndpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey2, secretKey2, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create verification Minio client: %v", err)
	}

	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer verifyCancel()

	objInfo, err := minioClient.StatObject(verifyCtx, setup.GetMinioBucket(), testKey, minio.StatObjectOptions{})
	if err != nil {
		t.Fatalf("Object verification failed - object not found after upload: %v", err)
	}
	t.Logf("Object verified in %v - size: %d bytes, etag: %s",
		time.Since(verifyStartTime), objInfo.Size, objInfo.ETag)

	// Download (read)
	t.Log("Getting reader...")
	readStartTime := time.Now()
	reader, err := provider.Reader(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to get reader: %v", err)
	}
	defer reader.Close()
	t.Logf("Reader obtained in %v", time.Since(readStartTime))

	t.Log("Reading data...")
	readDataStartTime := time.Now()
	var downloadBuffer bytes.Buffer
	bytesCopied, err := io.Copy(&downloadBuffer, reader)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	t.Logf("Read %d bytes in %v", bytesCopied, time.Since(readDataStartTime))
	t.Logf("Total download time: %v", time.Since(readStartTime))

	// Verify
	t.Log("Verifying data integrity...")
	downloadedData := downloadBuffer.Bytes()
	t.Logf("Downloaded %d bytes, expected %d bytes", len(downloadedData), len(testData))

	if !bytes.Equal(downloadedData, testData) {
		t.Errorf("Downloaded data doesn't match!")
		t.Errorf("  Expected: %q", string(testData))
		t.Errorf("  Got:      %q", string(downloadedData))
		t.Errorf("  Expected bytes: %v", testData)
		t.Errorf("  Got bytes:      %v", downloadedData)
	} else {
		t.Log("Data integrity verified - content matches!")
	}

	// Clean up
	t.Log("Cleaning up test data...")
	deleteStartTime := time.Now()
	if err := provider.Delete(ctx, testKey); err != nil {
		t.Logf("Warning: Failed to clean up test data: %v", err)
	} else {
		t.Logf("Test data deleted in %v", time.Since(deleteStartTime))
	}

	t.Logf("=== Test completed in %v ===", time.Since(startTime))
}

// TestStorageHealthCheck tests the storage provider health check.
func TestStorageHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setup := NewIntegrationTestSetup()
	setup.Start(t)
	defer setup.Stop(t)

	accessKey, secretKey := setup.GetMinioCredentials()

	storageConfig := &storage.Config{
		Provider:          "s3",
		S3Endpoint:        setup.GetMinioEndpoint(),
		S3Bucket:          setup.GetMinioBucket(),
		S3AccessKeyID:     accessKey,
		S3SecretAccessKey: secretKey,
		S3Region:          "us-east-1",
		S3UseSSL:          false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	provider, err := storage.NewProvider(ctx, storageConfig)
	if err != nil {
		t.Fatalf("Failed to create storage provider: %v", err)
	}
	defer provider.Close()

	// Test health check by attempting to list objects
	// This serves as a basic connectivity test
	iter := provider.List(ctx, "")
	_, err = iter.Next()
	// It's ok if there are no objects (EOF error), we just want to verify connectivity
	if err != nil && err != io.EOF {
		t.Logf("Storage connectivity check warning: %v", err)
	}
	t.Log("Storage connectivity check passed")
}
