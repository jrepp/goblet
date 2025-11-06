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

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSProvider implements Provider for Google Cloud Storage
type GCSProvider struct {
	client *storage.Client
	bucket *storage.BucketHandle
}

// NewGCSProvider creates a new GCS storage provider
func NewGCSProvider(ctx context.Context, bucketName string) (*GCSProvider, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &GCSProvider{
		client: client,
		bucket: client.Bucket(bucketName),
	}, nil
}

// Writer returns a writer for the given object path
func (g *GCSProvider) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
	return g.bucket.Object(path).NewWriter(ctx), nil
}

// Reader returns a reader for the given object path
func (g *GCSProvider) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	return g.bucket.Object(path).NewReader(ctx)
}

// Delete removes an object at the given path
func (g *GCSProvider) Delete(ctx context.Context, path string) error {
	return g.bucket.Object(path).Delete(ctx)
}

// List returns an iterator for objects with the given prefix
func (g *GCSProvider) List(ctx context.Context, prefix string) ObjectIterator {
	query := &storage.Query{
		Delimiter: "/",
		Prefix:    prefix,
	}
	return &gcsIterator{
		iter: g.bucket.Objects(ctx, query),
	}
}

// Close closes the GCS client
func (g *GCSProvider) Close() error {
	return g.client.Close()
}

// gcsIterator wraps the GCS iterator
type gcsIterator struct {
	iter *storage.ObjectIterator
}

// Next returns the next object attributes
func (i *gcsIterator) Next() (*ObjectAttrs, error) {
	attrs, err := i.iter.Next()
	if err == iterator.Done {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}

	return &ObjectAttrs{
		Name:    attrs.Name,
		Prefix:  attrs.Prefix,
		Created: attrs.Created,
		Updated: attrs.Updated,
		Size:    attrs.Size,
	}, nil
}
