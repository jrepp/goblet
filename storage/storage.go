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
	"context"
	"io"
	"time"
)

// Provider defines the interface for object storage backends
type Provider interface {
	// Writer returns a writer for the given object path
	Writer(ctx context.Context, path string) (io.WriteCloser, error)

	// Reader returns a reader for the given object path
	Reader(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes an object at the given path
	Delete(ctx context.Context, path string) error

	// List returns an iterator for objects with the given prefix
	List(ctx context.Context, prefix string) ObjectIterator

	// Close closes the provider connection
	Close() error
}

// ObjectIterator provides iteration over storage objects
type ObjectIterator interface {
	// Next returns the next object attributes
	Next() (*ObjectAttrs, error)
}

// ObjectAttrs represents object metadata
type ObjectAttrs struct {
	Name    string
	Prefix  string
	Created time.Time
	Updated time.Time
	Size    int64
}

// Config holds storage provider configuration
type Config struct {
	// Provider type: "gcs" or "s3"
	Provider string

	// For GCS
	GCSBucket string

	// For S3/Minio
	S3Endpoint        string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3Region          string
	S3UseSSL          bool
}

// NewProvider creates a new storage provider based on configuration
func NewProvider(ctx context.Context, config *Config) (Provider, error) {
	switch config.Provider {
	case "gcs":
		return NewGCSProvider(ctx, config.GCSBucket)
	case "s3":
		return NewS3Provider(ctx, config)
	default:
		return nil, nil // No backup configured
	}
}
