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

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Provider implements Provider for S3-compatible storage (including Minio)
type S3Provider struct {
	client     *minio.Client
	bucketName string
}

// NewS3Provider creates a new S3/Minio storage provider
func NewS3Provider(ctx context.Context, config *Config) (*S3Provider, error) {
	client, err := minio.New(config.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.S3AccessKeyID, config.S3SecretAccessKey, ""),
		Secure: config.S3UseSSL,
		Region: config.S3Region,
	})
	if err != nil {
		return nil, err
	}

	// Ensure bucket exists
	exists, err := client.BucketExists(ctx, config.S3Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = client.MakeBucket(ctx, config.S3Bucket, minio.MakeBucketOptions{
			Region: config.S3Region,
		})
		if err != nil {
			return nil, err
		}
	}

	return &S3Provider{
		client:     client,
		bucketName: config.S3Bucket,
	}, nil
}

// Writer returns a writer for the given object path
func (s *S3Provider) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		_, err := s.client.PutObject(ctx, s.bucketName, path, pr, -1, minio.PutObjectOptions{})
		if err != nil {
			pr.CloseWithError(err)
		} else {
			pr.Close()
		}
	}()

	return pw, nil
}

// Reader returns a reader for the given object path
func (s *S3Provider) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucketName, path, minio.GetObjectOptions{})
}

// Delete removes an object at the given path
func (s *S3Provider) Delete(ctx context.Context, path string) error {
	return s.client.RemoveObject(ctx, s.bucketName, path, minio.RemoveObjectOptions{})
}

// List returns an iterator for objects with the given prefix
func (s *S3Provider) List(ctx context.Context, prefix string) ObjectIterator {
	ch := s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	return &s3Iterator{
		ch:  ch,
		ctx: ctx,
	}
}

// Close closes the S3 client (no-op for Minio client)
func (s *S3Provider) Close() error {
	return nil
}

// s3Iterator wraps the S3 object channel
type s3Iterator struct {
	ch  <-chan minio.ObjectInfo
	ctx context.Context
}

// Next returns the next object attributes
func (i *s3Iterator) Next() (*ObjectAttrs, error) {
	select {
	case obj, ok := <-i.ch:
		if !ok {
			return nil, io.EOF
		}
		if obj.Err != nil {
			return nil, obj.Err
		}

		name := obj.Key
		prefix := ""
		if obj.Key == "" {
			// This is a prefix/directory entry
			prefix = obj.Key
		}

		return &ObjectAttrs{
			Name:    name,
			Prefix:  prefix,
			Created: obj.LastModified,
			Updated: obj.LastModified,
			Size:    obj.Size,
		}, nil
	case <-i.ctx.Done():
		return nil, i.ctx.Err()
	}
}
