# Configuration Reference

Complete reference for all Goblet configuration options.

## Command-Line Flags

### Basic Options

```bash
--port int
    HTTP server port (default: 8080)

--cache_root string
    Root directory for cache storage (default: "/cache")

--upstream_timeout duration
    Timeout for upstream requests (default: 30s)

--log_level string
    Log level: debug, info, warn, error (default: "info")
```

### Authentication

```bash
--auth_type string
    Authentication type: oauth2, oidc (default: "oauth2")

--oauth2_client_id string
    OAuth2 client ID

--oidc_issuer string
    OIDC issuer URL

--oidc_client_id string
    OIDC client ID
```

### Storage

```bash
--storage_provider string
    Storage provider: local, s3, gcs (default: "local")

--storage_bucket string
    Cloud storage bucket name

--backup_interval duration
    Backup interval for cloud storage (default: 1h)
```

## Environment Variables

All command-line flags can be set via environment variables:

```bash
GOBLET_PORT=8080
GOBLET_CACHE_ROOT=/cache
GOBLET_LOG_LEVEL=info
GOBLET_AUTH_TYPE=oidc
GOBLET_OIDC_ISSUER=https://auth.example.com
GOBLET_OIDC_CLIENT_ID=goblet
```

## Configuration File

Create `/etc/goblet/config.yaml`:

```yaml
server:
  port: 8080
  cache_root: /cache
  upstream_timeout: 30s
  log_level: info

auth:
  type: oidc
  oidc:
    issuer: https://auth.example.com
    client_id: goblet

storage:
  provider: local
  backup:
    enabled: true
    interval: 1h
    provider: gcs
    bucket: goblet-backups
```

## Isolation Configuration

See [Isolation Strategies](../security/isolation-strategies.md) for multi-tenant configuration.

## Related Documentation

- [Getting Started](../getting-started.md)
- [Security Guide](../security/README.md)
- [Deployment Patterns](../operations/deployment-patterns.md)
