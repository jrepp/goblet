// Copyright 2025 Jacob Repp <jacobrepp@gmail.com>
//
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

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Status constants for metrics.
const (
	statusSuccess = "success"
	statusFailure = "failure"
	errorTypeNone = "none"
)

// Metric keys for storage operations.
var (
	// StorageOperationKey identifies the type of storage operation.
	StorageOperationKey tag.Key
	// StorageProviderKey identifies the storage provider (gcs, s3, etc).
	StorageProviderKey tag.Key
	// StorageStatusKey indicates success or failure.
	StorageStatusKey tag.Key
	// StorageErrorTypeKey categorizes the type of error.
	StorageErrorTypeKey tag.Key
)

// Metrics for storage operations.
var (
	// StorageOperationCount counts storage operations by type and status.
	StorageOperationCount = stats.Int64(
		"goblet/storage/operations",
		"Number of storage operations",
		stats.UnitDimensionless,
	)

	// StorageOperationLatency measures operation duration.
	StorageOperationLatency = stats.Float64(
		"goblet/storage/latency",
		"Storage operation latency in milliseconds",
		stats.UnitMilliseconds,
	)

	// StorageBytesTransferred tracks bytes read/written.
	StorageBytesTransferred = stats.Int64(
		"goblet/storage/bytes",
		"Bytes transferred in storage operations",
		stats.UnitBytes,
	)
)

func init() {
	var err error
	StorageOperationKey, err = tag.NewKey("operation")
	if err != nil {
		panic(err)
	}
	StorageProviderKey, err = tag.NewKey("provider")
	if err != nil {
		panic(err)
	}
	StorageStatusKey, err = tag.NewKey("status")
	if err != nil {
		panic(err)
	}
	StorageErrorTypeKey, err = tag.NewKey("error_type")
	if err != nil {
		panic(err)
	}
}

// StorageViews returns all storage-related metric views.
func StorageViews() []*view.View {
	return []*view.View{
		{
			Name:        "goblet/storage/operations_count",
			Description: "Count of storage operations by type and status",
			Measure:     StorageOperationCount,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{StorageOperationKey, StorageProviderKey, StorageStatusKey, StorageErrorTypeKey},
		},
		{
			Name:        "goblet/storage/latency_distribution",
			Description: "Distribution of storage operation latencies",
			Measure:     StorageOperationLatency,
			Aggregation: view.Distribution(0, 10, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
			TagKeys:     []tag.Key{StorageOperationKey, StorageProviderKey, StorageStatusKey},
		},
		{
			Name:        "goblet/storage/bytes_total",
			Description: "Total bytes transferred",
			Measure:     StorageBytesTransferred,
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{StorageOperationKey, StorageProviderKey},
		},
	}
}

// MetricsProvider wraps a Provider with metrics instrumentation.
type MetricsProvider struct {
	provider     Provider
	providerType string
}

// NewMetricsProvider creates a new metrics-instrumented provider.
func NewMetricsProvider(provider Provider, providerType string) Provider {
	return &MetricsProvider{
		provider:     provider,
		providerType: providerType,
	}
}

// Writer returns a writer for the given object path with metrics.
func (m *MetricsProvider) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
	start := time.Now()
	writer, err := m.provider.Writer(ctx, path)

	status := statusSuccess
	errorType := errorTypeNone
	if err != nil {
		status = statusFailure
		errorType = categorizeError(err)
	}

	m.recordMetrics(ctx, "writer", status, errorType, time.Since(start))

	if err != nil {
		return nil, err
	}

	return &metricsWriter{
		writer:       writer,
		ctx:          ctx,
		providerType: m.providerType,
	}, nil
}

// Reader returns a reader for the given object path with metrics.
func (m *MetricsProvider) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	start := time.Now()
	reader, err := m.provider.Reader(ctx, path)

	status := statusSuccess
	errorType := errorTypeNone
	if err != nil {
		status = statusFailure
		errorType = categorizeError(err)
	}

	m.recordMetrics(ctx, "reader", status, errorType, time.Since(start))

	if err != nil {
		return nil, err
	}

	return &metricsReader{
		reader:       reader,
		ctx:          ctx,
		providerType: m.providerType,
	}, nil
}

// Delete removes an object at the given path with metrics.
func (m *MetricsProvider) Delete(ctx context.Context, path string) error {
	start := time.Now()
	err := m.provider.Delete(ctx, path)

	status := statusSuccess
	errorType := errorTypeNone
	if err != nil {
		status = statusFailure
		errorType = categorizeError(err)
	}

	m.recordMetrics(ctx, "delete", status, errorType, time.Since(start))
	return err
}

// List returns an iterator for objects with the given prefix with metrics.
func (m *MetricsProvider) List(ctx context.Context, prefix string) ObjectIterator {
	start := time.Now()
	iter := m.provider.List(ctx, prefix)

	// Record list operation start
	m.recordMetrics(ctx, "list", statusSuccess, errorTypeNone, time.Since(start))

	return &metricsIterator{
		iterator:     iter,
		ctx:          ctx,
		providerType: m.providerType,
	}
}

// Close closes the provider with metrics.
func (m *MetricsProvider) Close() error {
	start := time.Now()
	err := m.provider.Close()

	status := statusSuccess
	errorType := errorTypeNone
	if err != nil {
		status = statusFailure
		errorType = categorizeError(err)
	}

	m.recordMetrics(context.Background(), "close", status, errorType, time.Since(start))
	return err
}

func (m *MetricsProvider) recordMetrics(ctx context.Context, operation, status, errorType string, latency time.Duration) {
	_ = stats.RecordWithTags(ctx,
		[]tag.Mutator{
			tag.Upsert(StorageOperationKey, operation),
			tag.Upsert(StorageProviderKey, m.providerType),
			tag.Upsert(StorageStatusKey, status),
			tag.Upsert(StorageErrorTypeKey, errorType),
		},
		StorageOperationCount.M(1),
		StorageOperationLatency.M(float64(latency.Milliseconds())),
	)
}

// metricsWriter wraps an io.WriteCloser to track bytes written.
type metricsWriter struct {
	writer       io.WriteCloser
	ctx          context.Context
	providerType string
	bytesWritten int64
}

func (mw *metricsWriter) Write(p []byte) (n int, err error) {
	n, err = mw.writer.Write(p)
	mw.bytesWritten += int64(n)
	return n, err
}

func (mw *metricsWriter) Close() error {
	err := mw.writer.Close()

	// Record bytes transferred
	_ = stats.RecordWithTags(mw.ctx,
		[]tag.Mutator{
			tag.Upsert(StorageOperationKey, "write"),
			tag.Upsert(StorageProviderKey, mw.providerType),
		},
		StorageBytesTransferred.M(mw.bytesWritten),
	)

	return err
}

// metricsReader wraps an io.ReadCloser to track bytes read.
type metricsReader struct {
	reader       io.ReadCloser
	ctx          context.Context
	providerType string
	bytesRead    int64
}

func (mr *metricsReader) Read(p []byte) (n int, err error) {
	n, err = mr.reader.Read(p)
	mr.bytesRead += int64(n)
	return n, err
}

func (mr *metricsReader) Close() error {
	err := mr.reader.Close()

	// Record bytes transferred
	_ = stats.RecordWithTags(mr.ctx,
		[]tag.Mutator{
			tag.Upsert(StorageOperationKey, "read"),
			tag.Upsert(StorageProviderKey, mr.providerType),
		},
		StorageBytesTransferred.M(mr.bytesRead),
	)

	return err
}

// metricsIterator wraps an ObjectIterator to track iteration metrics.
type metricsIterator struct {
	iterator     ObjectIterator
	ctx          context.Context
	providerType string
	objectCount  int64
}

func (mi *metricsIterator) Next() (*ObjectAttrs, error) {
	attrs, err := mi.iterator.Next()
	if err == nil && attrs != nil {
		mi.objectCount++
	}
	return attrs, err
}

// categorizeError categorizes errors for metrics tagging.
func categorizeError(err error) string {
	if err == nil {
		return errorTypeNone
	}

	errStr := err.Error()
	switch {
	case contains(errStr, "not found", "no such", "does not exist"):
		return "not_found"
	case contains(errStr, "permission", "denied", "forbidden", "unauthorized"):
		return "permission_denied"
	case contains(errStr, "timeout", "deadline exceeded"):
		return "timeout"
	case contains(errStr, "connection", "network", "dial"):
		return "network"
	case contains(errStr, "context canceled"):
		return "canceled"
	case contains(errStr, "invalid", "malformed"):
		return "invalid_argument"
	default:
		return "unknown"
	}
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
