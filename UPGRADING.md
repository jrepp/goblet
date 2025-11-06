# Upgrading Guide

## 2025-11 Update

### Go Version Update

The project has been upgraded from Go 1.12 to Go 1.24.0, bringing modern language features and improved performance.

### Module Updates

All Go modules have been updated to their latest versions:

**Major Updates:**
- `cloud.google.com/go/logging`: v1.4.2 → v1.13.1
- `cloud.google.com/go/storage`: v1.16.0 → v1.57.1
- `github.com/go-git/go-git/v5`: v5.4.2 → v5.16.3
- `google.golang.org/api`: v0.50.0 → v0.255.0
- `google.golang.org/grpc`: v1.39.0 → v1.76.0
- `google.golang.org/protobuf`: v1.27.1 → v1.36.10

**New Dependencies:**
- OpenTelemetry instrumentation packages (v1.38.0)
- Minio Go SDK (v7.0.97) for S3 support
- Cloud monitoring and tracing support

### Breaking Changes

#### Storage Backend Configuration

The storage configuration has been modernized to support multiple providers.

**Old Configuration (GCS only):**
```bash
-backup_bucket_name=my-bucket
-backup_manifest_name=my-manifest
```

**New Configuration:**

For GCS:
```bash
-storage_provider=gcs
-backup_bucket_name=my-bucket
-backup_manifest_name=my-manifest
```

For S3/Minio:
```bash
-storage_provider=s3
-s3_endpoint=localhost:9000
-s3_bucket=goblet-backups
-s3_access_key=minioadmin
-s3_secret_key=minioadmin
-s3_region=us-east-1
-s3_use_ssl=false
-backup_manifest_name=my-manifest
```

#### API Changes

The `google.RunBackupProcess` function signature has changed:

**Before:**
```go
func RunBackupProcess(config *goblet.ServerConfig, bh *storage.BucketHandle, manifestName string, logger *log.Logger)
```

**After:**
```go
func RunBackupProcess(config *goblet.ServerConfig, provider storage.Provider, manifestName string, logger *log.Logger)
```

### Migration Steps

1. **Update Go Installation:**
   ```bash
   # Install Go 1.24 or later
   go version # Should show go1.24 or higher
   ```

2. **Update Dependencies:**
   ```bash
   go mod tidy
   go build ./...
   ```

3. **Update Configuration:**
   - Add `-storage_provider` flag to your deployment
   - For GCS: `-storage_provider=gcs`
   - For S3/Minio: Add S3 configuration flags

4. **Test Changes:**
   ```bash
   go test ./...
   ```

5. **Deploy:**
   - Update your deployment scripts with new configuration flags
   - For Docker deployments, see docker-compose.yml for examples

### Backwards Compatibility

The changes maintain backwards compatibility for deployments without backup configured. If no storage provider is specified, the server will run without backup functionality.

### Docker Deployment

A new docker-compose.yml has been added for local testing with Minio:

```bash
docker-compose up -d
```

This will start:
- Goblet server on port 8080
- Minio S3 on port 9000 (API) and 9001 (Console)

### Environment Variables

S3 credentials can also be provided via environment variables:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

For production deployments, prefer environment variables or secrets management over command-line flags.
