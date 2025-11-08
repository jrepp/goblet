# Monitoring Guide

Monitor Goblet's performance, health, and security with Prometheus metrics and alerting.

## Quick Start

```bash
# View metrics
curl http://localhost:8080/metrics

# Access Prometheus (if using load test environment)
open http://localhost:9090

# Access Grafana
open http://localhost:3000
```

## Key Metrics

### Performance Metrics

**Cache Hit Rate:**
```promql
rate(cache_hits_total[5m]) / rate(requests_total[5m])
```
- Target: > 80%
- Warning: < 70%
- Critical: < 50%

**Request Latency (P95):**
```promql
histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m]))
```
- Good: < 100ms
- Acceptable: 100-500ms
- Poor: > 500ms

**Error Rate:**
```promql
rate(errors_total[5m]) / rate(requests_total[5m])
```
- Target: < 1%
- Warning: > 5%
- Critical: > 10%

### Resource Metrics

**Disk Usage:**
```promql
disk_usage_bytes / disk_capacity_bytes
```
- Warning: > 80%
- Critical: > 90%

**Memory Usage:**
```promql
container_memory_usage_bytes{container="goblet"}
```

**CPU Usage:**
```promql
rate(container_cpu_usage_seconds_total{container="goblet"}[5m])
```

## Dashboards

### Grafana Dashboard

Import the Goblet dashboard (coming soon):
```bash
# Import dashboard JSON
kubectl create configmap goblet-dashboard \
  --from-file=dashboards/goblet.json
```

### Key Panels

1. **Request Overview**
   - Total requests/sec
   - Success rate
   - Error rate

2. **Cache Performance**
   - Hit rate over time
   - Cache size
   - Eviction rate

3. **Latency Distribution**
   - P50, P95, P99
   - By operation type
   - By repository

4. **Resource Utilization**
   - CPU usage
   - Memory usage
   - Disk usage
   - Network I/O

## Alerting Rules

### Prometheus Alerts

```yaml
groups:
- name: goblet
  rules:
  # Low cache hit rate
  - alert: GobletLowCacheHitRate
    expr: rate(cache_hits_total[5m]) / rate(requests_total[5m]) < 0.5
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Low cache hit rate ({{ $value | humanizePercentage }})"

  # High error rate
  - alert: GobletHighErrorRate
    expr: rate(errors_total[5m]) / rate(requests_total[5m]) > 0.05
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High error rate ({{ $value | humanizePercentage }})"

  # Disk space low
  - alert: GobletLowDiskSpace
    expr: disk_usage_bytes / disk_capacity_bytes > 0.9
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Low disk space ({{ $value | humanizePercentage }})"

  # High latency
  - alert: GobletHighLatency
    expr: histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m])) > 1.0
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "High P95 latency ({{ $value }}s)"
```

## Health Checks

### Liveness Probe

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
```

### Readiness Probe

```yaml
readinessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Logging

### Log Levels

- `debug`: Detailed debugging information
- `info`: General operational messages
- `warn`: Warning messages (e.g., cache misses, slow operations)
- `error`: Error messages

### Structured Logging

```json
{
  "level": "info",
  "timestamp": "2025-11-07T10:00:00Z",
  "message": "Cache hit",
  "repository": "github.com/kubernetes/kubernetes",
  "operation": "fetch",
  "duration_ms": 45,
  "cache_hit": true
}
```

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md) for common issues and solutions.

## Related Documentation

- [Load Testing](load-testing.md)
- [Deployment Patterns](deployment-patterns.md)
- [Troubleshooting](troubleshooting.md)
