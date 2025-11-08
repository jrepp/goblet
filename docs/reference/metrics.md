# Metrics Reference

Complete reference for Prometheus metrics exposed by Goblet.

## Request Metrics

### goblet_requests_total

Total number of requests processed.

**Labels:**
- `operation`: fetch, ls-refs
- `status`: success, error
- `cache`: hit, miss

**Type:** Counter

### goblet_request_duration_seconds

Request duration histogram.

**Labels:**
- `operation`: fetch, ls-refs

**Type:** Histogram

## Cache Metrics

### goblet_cache_hits_total

Total number of cache hits.

**Type:** Counter

### goblet_cache_misses_total

Total number of cache misses.

**Type:** Counter

### goblet_cache_size_bytes

Current cache size in bytes.

**Type:** Gauge

## Error Metrics

### goblet_errors_total

Total number of errors.

**Labels:**
- `type`: upstream, auth, internal

**Type:** Counter

## System Metrics

### goblet_disk_usage_bytes

Disk usage in bytes.

**Type:** Gauge

### goblet_disk_capacity_bytes

Total disk capacity in bytes.

**Type:** Gauge

## Related Documentation

- [Monitoring Guide](../operations/monitoring.md)
- [API Reference](api.md)
