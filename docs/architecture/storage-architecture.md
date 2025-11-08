# Storage Architecture

## Overview

Goblet uses object storage backends to persist git repository backups. The storage architecture has been redesigned to support multiple providers through a common interface, enabling deployment flexibility.

## Design Principles

1. **Provider Abstraction**: A common `storage.Provider` interface abstracts storage operations
2. **Pluggable Backends**: Easy to add new storage providers
3. **Backward Compatible**: Existing GCS deployments work with minimal changes
4. **Configuration-driven**: Provider selection via command-line flags

## Architecture

### Storage Interface

The `storage.Provider` interface defines the contract for all storage backends:

```go
type Provider interface {
    Writer(ctx context.Context, path string) (io.WriteCloser, error)
    Reader(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    List(ctx context.Context, prefix string) ObjectIterator
    Close() error
}
```

### Object Iteration

Storage providers implement a consistent iterator pattern:

```go
type ObjectIterator interface {
    Next() (*ObjectAttrs, error)
}

type ObjectAttrs struct {
    Name    string
    Prefix  string
    Created time.Time
    Updated time.Time
    Size    int64
}
```

### Supported Providers

#### 1. Google Cloud Storage (GCS)

**Implementation**: `storage/gcs.go`

Uses the official `cloud.google.com/go/storage` SDK.

**Configuration:**
```bash
-storage_provider=gcs
-backup_bucket_name=my-gcs-bucket
-backup_manifest_name=production
```

**Authentication:**
- Uses Application Default Credentials (ADC)
- Service account JSON key via GOOGLE_APPLICATION_CREDENTIALS
- Workload Identity in GKE

**Features:**
- Automatic retry and exponential backoff
- Strong consistency
- Lifecycle policies for old manifests

#### 2. S3-Compatible Storage (S3/Minio)

**Implementation**: `storage/s3.go`

Uses the Minio Go SDK (`github.com/minio/minio-go/v7`) which supports:
- Amazon S3
- Minio
- DigitalOcean Spaces
- Wasabi
- Any S3-compatible storage

**Configuration:**
```bash
-storage_provider=s3
-s3_endpoint=s3.amazonaws.com          # or localhost:9000 for Minio
-s3_bucket=my-s3-bucket
-s3_access_key=AKIAIOSFODNN7EXAMPLE
-s3_secret_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
-s3_region=us-east-1
-s3_use_ssl=true                       # false for local Minio
-backup_manifest_name=production
```

**Authentication:**
- Static credentials via flags/environment variables
- IAM roles (for AWS EC2/ECS)
- Environment variables: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY

**Features:**
- Multipart upload for large objects
- Bucket auto-creation
- Streaming uploads via io.Pipe

## Storage Operations

### Backup Process

The backup process runs on a configurable frequency (default: 1 hour):

1. **List Managed Repositories**: Get all cached repositories
2. **Check Latest Bundle**: Verify if backup is up-to-date
3. **Create Bundle**: Generate git bundle from repository
4. **Upload Bundle**: Write bundle to storage provider
5. **Update Manifest**: Write manifest file with repository list
6. **Garbage Collection**: Remove old bundles and manifests

### Recovery Process

On startup, the server can recover from backups:

1. **List Manifests**: Find all manifest files
2. **Read Manifest**: Parse repository URLs
3. **Download Bundles**: Fetch git bundles from storage
4. **Restore Repositories**: Initialize local repositories from bundles

### Storage Layout

```
bucket/
├── goblet-repository-manifests/
│   └── {manifest-name}/
│       ├── {timestamp1}           # Manifest file
│       └── {timestamp2}           # Manifest file
└── github.com/
    └── {owner}/
        └── {repo}/
            └── {timestamp}        # Git bundle
```

**Manifest File Format:**
```
https://github.com/owner/repo1
https://github.com/owner/repo2
https://github.com/owner/repo3
```

**Bundle Naming:**
- Timestamp format: 12-digit Unix timestamp (e.g., `000001699999999`)
- Enables chronological sorting
- Garbage collection keeps only the latest bundle

## Provider Selection

The `storage.NewProvider()` factory function creates the appropriate provider:

```go
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
```

## Adding New Providers

To add a new storage provider:

1. **Create Provider File**: `storage/{provider}.go`
2. **Implement Interface**: Implement `storage.Provider`
3. **Add to Factory**: Update `NewProvider()` in `storage/storage.go`
4. **Add Configuration**: Add flags in `goblet-server/main.go`
5. **Document**: Update this file

### Example Provider Template

```go
package storage

type MyProvider struct {
    client *SomeClient
}

func NewMyProvider(ctx context.Context, config *Config) (*MyProvider, error) {
    // Initialize client
    return &MyProvider{client: client}, nil
}

func (p *MyProvider) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
    // Return writer
}

func (p *MyProvider) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
    // Return reader
}

func (p *MyProvider) Delete(ctx context.Context, path string) error {
    // Delete object
}

func (p *MyProvider) List(ctx context.Context, prefix string) ObjectIterator {
    // Return iterator
}

func (p *MyProvider) Close() error {
    // Cleanup
}
```

## Performance Considerations

### GCS Provider
- **Latency**: Low latency within same region
- **Throughput**: High (multi-Gbps)
- **Consistency**: Strong consistency
- **Cost**: Pay for storage and operations

### S3 Provider
- **Latency**: Varies by provider
- **Throughput**: High for AWS S3
- **Consistency**: Strong consistency (as of Dec 2020)
- **Cost**: Varies by provider (Minio is self-hosted)

### Minio (Self-hosted)
- **Latency**: Very low (local network)
- **Throughput**: Limited by hardware
- **Consistency**: Strong consistency
- **Cost**: Infrastructure only

## Testing

### Local Testing with Minio

```bash
# Start services
docker-compose up -d

# Check Minio console
open http://localhost:9001
# Login: minioadmin / minioadmin

# View logs
docker-compose logs -f goblet

# Test backup by adding a repository
git clone --mirror https://github.com/some/repo /tmp/test.git

# Stop services
docker-compose down
```

### Unit Testing

Mock the `storage.Provider` interface for testing:

```go
type MockProvider struct {
    mock.Mock
}

func (m *MockProvider) Writer(ctx context.Context, path string) (io.WriteCloser, error) {
    args := m.Called(ctx, path)
    return args.Get(0).(io.WriteCloser), args.Error(1)
}

// ... implement other methods
```

## Security Considerations

1. **Credentials Management**
   - Never commit credentials to source control
   - Use environment variables or secrets management
   - Rotate credentials regularly

2. **Bucket Permissions**
   - Principle of least privilege
   - Separate buckets for different environments
   - Enable versioning for production

3. **Network Security**
   - Use SSL/TLS for remote storage (s3_use_ssl=true)
   - VPC endpoints for cloud storage
   - Network policies for Kubernetes

4. **Data Protection**
   - Enable encryption at rest
   - Use server-side encryption
   - Implement lifecycle policies

## Monitoring

Key metrics to monitor:

- **Backup Success Rate**: Percentage of successful backups
- **Backup Duration**: Time to complete backup cycle
- **Storage Size**: Total size of stored bundles
- **API Errors**: Storage provider error rates
- **Latency**: Read/write operation latency

## Troubleshooting

### Common Issues

**Connection Refused (Minio):**
- Check Minio is running: `docker-compose ps`
- Verify endpoint configuration
- Check network connectivity

**Authentication Failed (GCS):**
- Verify credentials: `gcloud auth application-default login`
- Check service account permissions
- Ensure storage.objects.* permissions

**Authentication Failed (S3):**
- Verify access key and secret key
- Check IAM policy has s3:* permissions
- Verify bucket exists and region is correct

**Slow Backups:**
- Check network bandwidth
- Monitor storage provider metrics
- Consider increasing backup frequency
- Verify no rate limiting

### Debug Logging

Enable verbose logging:
```bash
# Set log level
export GOBLET_LOG_LEVEL=debug

# Run with debug flags
./goblet-server -storage_provider=s3 ...
```

## Future Enhancements

Potential improvements to the storage architecture:

1. **Azure Blob Storage**: Add Azure support
2. **Compression**: Compress bundles before upload
3. **Encryption**: Client-side encryption for sensitive repos
4. **Deduplication**: Share common objects across bundles
5. **Incremental Backups**: Only backup changed objects
6. **Parallel Uploads**: Upload multiple bundles concurrently
7. **Backup Verification**: Periodic integrity checks
8. **Backup Metrics**: Expose Prometheus metrics
