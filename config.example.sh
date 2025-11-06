#!/bin/bash
# Example configuration for Goblet server
# Source this file or copy values to your deployment script

# Server configuration
export PORT=8080
export CACHE_ROOT="/var/cache/goblet"

# Storage provider: "gcs" or "s3"
export STORAGE_PROVIDER="s3"

# Backup manifest name (required if storage provider is set)
export BACKUP_MANIFEST_NAME="production"

# GCS configuration (if STORAGE_PROVIDER=gcs)
export BACKUP_BUCKET_NAME="my-gcs-bucket"

# S3/Minio configuration (if STORAGE_PROVIDER=s3)
export S3_ENDPOINT="s3.amazonaws.com"         # or "localhost:9000" for Minio
export S3_BUCKET="goblet-backups"
export S3_ACCESS_KEY="your-access-key"
export S3_SECRET_KEY="your-secret-key"
export S3_REGION="us-east-1"
export S3_USE_SSL="true"                      # "false" for local Minio

# Google Cloud Stackdriver configuration (optional)
export STACKDRIVER_PROJECT=""
export STACKDRIVER_LOGGING_LOG_ID=""

# Run the server
# ./goblet-server \
#   -port=$PORT \
#   -cache_root=$CACHE_ROOT \
#   -storage_provider=$STORAGE_PROVIDER \
#   -backup_manifest_name=$BACKUP_MANIFEST_NAME \
#   -s3_endpoint=$S3_ENDPOINT \
#   -s3_bucket=$S3_BUCKET \
#   -s3_access_key=$S3_ACCESS_KEY \
#   -s3_secret_key=$S3_SECRET_KEY \
#   -s3_region=$S3_REGION \
#   -s3_use_ssl=$S3_USE_SSL
