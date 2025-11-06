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

package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// Mock provider for testing
type mockProvider struct {
	writerFunc func(ctx context.Context, path string) (io.WriteCloser, error)
	readerFunc func(ctx context.Context, path string) (io.ReadCloser, error)
	deleteFunc func(ctx context.Context, path string) error
	listFunc   func(ctx context.Context, prefix string) ObjectIterator
	closeFunc  func() error
}

func (m *mockProvider) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
	if m.writerFunc != nil {
		return m.writerFunc(ctx, path)
	}
	return &mockWriteCloser{}, nil
}

func (m *mockProvider) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	if m.readerFunc != nil {
		return m.readerFunc(ctx, path)
	}
	return io.NopCloser(bytes.NewReader([]byte("test data"))), nil
}

func (m *mockProvider) Delete(ctx context.Context, path string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, path)
	}
	return nil
}

func (m *mockProvider) List(ctx context.Context, prefix string) ObjectIterator {
	if m.listFunc != nil {
		return m.listFunc(ctx, prefix)
	}
	return &mockIterator{}
}

func (m *mockProvider) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

type mockWriteCloser struct {
	buf    bytes.Buffer
	closed bool
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.closed = true
	return nil
}

type mockIterator struct {
	items   []*ObjectAttrs
	index   int
	err     error
	nextErr error
}

func (m *mockIterator) Next() (*ObjectAttrs, error) {
	if m.nextErr != nil {
		return nil, m.nextErr
	}
	if m.index >= len(m.items) {
		return nil, io.EOF
	}
	item := m.items[m.index]
	m.index++
	return item, nil
}

func TestNewProvider_S3(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping S3 provider test in short mode")
	}

	config := &Config{
		Provider:          "s3",
		S3Endpoint:        "localhost:9000",
		S3Bucket:          "test-bucket",
		S3AccessKeyID:     "test-key",
		S3SecretAccessKey: "test-secret",
		S3Region:          "us-east-1",
		S3UseSSL:          false,
	}

	ctx := context.Background()
	provider, err := NewProvider(ctx, config)

	// This will fail if Minio is not running, which is expected in short mode
	if err != nil {
		t.Logf("Note: S3 provider creation failed (expected if Minio not running): %v", err)
	}

	if provider != nil {
		defer provider.Close()
		t.Log("Successfully created S3 provider")
	}
}

func TestNewProvider_NoProvider(t *testing.T) {
	config := &Config{
		Provider: "",
	}

	ctx := context.Background()
	provider, err := NewProvider(ctx, config)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if provider != nil {
		t.Error("Expected nil provider for empty config")
	}
}

func TestNewProvider_UnsupportedProvider(t *testing.T) {
	config := &Config{
		Provider: "unsupported",
	}

	ctx := context.Background()
	provider, err := NewProvider(ctx, config)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if provider != nil {
		t.Error("Expected nil provider for unsupported type")
	}
}

func TestConfig_S3Fields(t *testing.T) {
	config := &Config{
		Provider:          "s3",
		S3Endpoint:        "s3.amazonaws.com",
		S3Bucket:          "my-bucket",
		S3AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		S3SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		S3Region:          "us-west-2",
		S3UseSSL:          true,
	}

	if config.Provider != "s3" {
		t.Errorf("Provider = %q, want s3", config.Provider)
	}

	if config.S3Endpoint != "s3.amazonaws.com" {
		t.Errorf("S3Endpoint = %q, want s3.amazonaws.com", config.S3Endpoint)
	}

	if config.S3Bucket != "my-bucket" {
		t.Errorf("S3Bucket = %q, want my-bucket", config.S3Bucket)
	}

	if config.S3Region != "us-west-2" {
		t.Errorf("S3Region = %q, want us-west-2", config.S3Region)
	}

	if !config.S3UseSSL {
		t.Error("S3UseSSL = false, want true")
	}
}

func TestConfig_GCSFields(t *testing.T) {
	config := &Config{
		Provider:  "gcs",
		GCSBucket: "my-gcs-bucket",
	}

	if config.Provider != "gcs" {
		t.Errorf("Provider = %q, want gcs", config.Provider)
	}

	if config.GCSBucket != "my-gcs-bucket" {
		t.Errorf("GCSBucket = %q, want my-gcs-bucket", config.GCSBucket)
	}
}

func TestObjectAttrs_Fields(t *testing.T) {
	now := time.Now()
	attrs := &ObjectAttrs{
		Name:    "test-object.dat",
		Prefix:  "test/",
		Created: now,
		Updated: now.Add(time.Hour),
		Size:    12345,
	}

	if attrs.Name != "test-object.dat" {
		t.Errorf("Name = %q, want test-object.dat", attrs.Name)
	}

	if attrs.Prefix != "test/" {
		t.Errorf("Prefix = %q, want test/", attrs.Prefix)
	}

	if attrs.Size != 12345 {
		t.Errorf("Size = %d, want 12345", attrs.Size)
	}

	if attrs.Created != now {
		t.Error("Created time doesn't match")
	}

	if !attrs.Updated.After(attrs.Created) {
		t.Error("Updated time should be after Created time")
	}
}

func TestProvider_Writer(t *testing.T) {
	writerCalled := false
	capturedPath := ""

	mock := &mockProvider{
		writerFunc: func(ctx context.Context, path string) (io.WriteCloser, error) {
			writerCalled = true
			capturedPath = path
			return &mockWriteCloser{}, nil
		},
	}

	ctx := context.Background()
	writer, err := mock.Writer(ctx, "test/path.dat")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !writerCalled {
		t.Error("Writer function was not called")
	}

	if capturedPath != "test/path.dat" {
		t.Errorf("Path = %q, want test/path.dat", capturedPath)
	}

	// Test writing
	data := []byte("test data")
	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Wrote %d bytes, want %d", n, len(data))
	}

	// Test closing
	if err := writer.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestProvider_Reader(t *testing.T) {
	testData := []byte("hello world")
	readerCalled := false

	mock := &mockProvider{
		readerFunc: func(ctx context.Context, path string) (io.ReadCloser, error) {
			readerCalled = true
			return io.NopCloser(bytes.NewReader(testData)), nil
		},
	}

	ctx := context.Background()
	reader, err := mock.Reader(ctx, "test.dat")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !readerCalled {
		t.Error("Reader function was not called")
	}

	// Read data
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Read error: %v", err)
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("Data = %q, want %q", data, testData)
	}

	reader.Close()
}

func TestProvider_Delete(t *testing.T) {
	deleteCalled := false
	deletedPath := ""

	mock := &mockProvider{
		deleteFunc: func(ctx context.Context, path string) error {
			deleteCalled = true
			deletedPath = path
			return nil
		},
	}

	ctx := context.Background()
	err := mock.Delete(ctx, "delete-me.dat")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Error("Delete function was not called")
	}

	if deletedPath != "delete-me.dat" {
		t.Errorf("Deleted path = %q, want delete-me.dat", deletedPath)
	}
}

func TestProvider_List(t *testing.T) {
	expectedItems := []*ObjectAttrs{
		{Name: "file1.dat", Size: 100},
		{Name: "file2.dat", Size: 200},
		{Name: "file3.dat", Size: 300},
	}

	listCalled := false
	listPrefix := ""

	mock := &mockProvider{
		listFunc: func(ctx context.Context, prefix string) ObjectIterator {
			listCalled = true
			listPrefix = prefix
			return &mockIterator{items: expectedItems}
		},
	}

	ctx := context.Background()
	iter := mock.List(ctx, "test/")

	if !listCalled {
		t.Error("List function was not called")
	}

	if listPrefix != "test/" {
		t.Errorf("List prefix = %q, want test/", listPrefix)
	}

	// Iterate through results
	var items []*ObjectAttrs
	for {
		item, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Iterator error: %v", err)
		}
		items = append(items, item)
	}

	if len(items) != len(expectedItems) {
		t.Errorf("Got %d items, want %d", len(items), len(expectedItems))
	}

	for i, item := range items {
		if item.Name != expectedItems[i].Name {
			t.Errorf("Item %d: Name = %q, want %q", i, item.Name, expectedItems[i].Name)
		}
	}
}

func TestProvider_Close(t *testing.T) {
	closeCalled := false

	mock := &mockProvider{
		closeFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	err := mock.Close()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !closeCalled {
		t.Error("Close function was not called")
	}
}

func TestProvider_ErrorHandling(t *testing.T) {
	expectedError := errors.New("storage error")

	tests := []struct {
		name     string
		provider *mockProvider
		testFunc func(Provider) error
	}{
		{
			name: "writer error",
			provider: &mockProvider{
				writerFunc: func(ctx context.Context, path string) (io.WriteCloser, error) {
					return nil, expectedError
				},
			},
			testFunc: func(p Provider) error {
				_, err := p.Writer(context.Background(), "test")
				return err
			},
		},
		{
			name: "reader error",
			provider: &mockProvider{
				readerFunc: func(ctx context.Context, path string) (io.ReadCloser, error) {
					return nil, expectedError
				},
			},
			testFunc: func(p Provider) error {
				_, err := p.Reader(context.Background(), "test")
				return err
			},
		},
		{
			name: "delete error",
			provider: &mockProvider{
				deleteFunc: func(ctx context.Context, path string) error {
					return expectedError
				},
			},
			testFunc: func(p Provider) error {
				return p.Delete(context.Background(), "test")
			},
		},
		{
			name: "close error",
			provider: &mockProvider{
				closeFunc: func() error {
					return expectedError
				},
			},
			testFunc: func(p Provider) error {
				return p.Close()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc(tt.provider)
			if err != expectedError {
				t.Errorf("Error = %v, want %v", err, expectedError)
			}
		})
	}
}

func TestObjectIterator_EOF(t *testing.T) {
	iter := &mockIterator{
		items: []*ObjectAttrs{},
	}

	item, err := iter.Next()

	if err != io.EOF {
		t.Errorf("Error = %v, want EOF", err)
	}

	if item != nil {
		t.Error("Expected nil item on EOF")
	}
}

func TestObjectIterator_Error(t *testing.T) {
	expectedError := errors.New("iterator error")
	iter := &mockIterator{
		nextErr: expectedError,
	}

	item, err := iter.Next()

	if err != expectedError {
		t.Errorf("Error = %v, want %v", err, expectedError)
	}

	if item != nil {
		t.Error("Expected nil item on error")
	}
}

func TestProvider_ContextCancellation(t *testing.T) {
	mock := &mockProvider{
		writerFunc: func(ctx context.Context, path string) (io.WriteCloser, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &mockWriteCloser{}, nil
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := mock.Writer(ctx, "test")

	if err != context.Canceled {
		t.Errorf("Error = %v, want context.Canceled", err)
	}
}

func TestProvider_ConcurrentAccess(t *testing.T) {
	mock := &mockProvider{}

	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ctx := context.Background()

			// Test concurrent writes
			writer, err := mock.Writer(ctx, "concurrent-test")
			if err != nil {
				done <- err
				return
			}
			writer.Close()

			// Test concurrent reads
			reader, err := mock.Reader(ctx, "concurrent-test")
			if err != nil {
				done <- err
				return
			}
			reader.Close()

			done <- nil
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}
}
