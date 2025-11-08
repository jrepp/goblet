# API Reference

HTTP endpoints exposed by Goblet.

## Git Protocol Endpoints

### POST /{repo}/git-upload-pack

Git protocol v2 upload-pack endpoint for fetch operations.

**Request:**
```
POST /github.com/kubernetes/kubernetes/git-upload-pack
Content-Type: application/x-git-upload-pack-request
Git-Protocol: version=2

<git protocol v2 data>
```

**Response:**
```
HTTP/1.1 200 OK
Content-Type: application/x-git-upload-pack-result

<git pack data>
```

### GET /{repo}/info/refs

Git smart HTTP info/refs endpoint (legacy).

## Health & Monitoring

### GET /healthz

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "uptime": "48h30m",
  "cache_size": "45GB"
}
```

### GET /metrics

Prometheus metrics endpoint.

**Response:**
```
# HELP goblet_requests_total Total number of requests
# TYPE goblet_requests_total counter
goblet_requests_total{operation="fetch",status="success"} 12345
...
```

## Related Documentation

- [Metrics Reference](metrics.md)
- [Monitoring Guide](../operations/monitoring.md)
