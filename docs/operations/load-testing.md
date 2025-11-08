# Load Testing

This guide explains how to load test Goblet to validate performance and capacity before production deployment.

## Overview

Load testing helps you:
- Validate deployment capacity
- Identify performance bottlenecks
- Tune cache sizes and resource limits
- Establish baseline metrics
- Test failure scenarios

## Quick Start

### Using Docker Compose

The fastest way to run load tests:

```bash
cd loadtest

# Start test environment (3 Goblet instances + monitoring)
make start

# Run Python-based load test
make loadtest-python

# View results
open http://localhost:8404  # HAProxy stats
open http://localhost:9090  # Prometheus
open http://localhost:3000  # Grafana

# Cleanup
make stop
```

##

 Test Environment Architecture

```
┌──────────────┐
│   HAProxy    │  Load balancer with consistent hashing
│   (port 8080)│
└───────┬──────┘
        │
    ┌───┴────┬────────┐
    │        │        │
┌───▼───┐ ┌──▼───┐ ┌──▼───┐
│Goblet │ │Goblet│ │Goblet│
│  -1   │ │  -2  │ │  -3  │
└───┬───┘ └──┬───┘ └──┬───┘
    │        │        │
┌───▼────────▼────────▼───┐
│      Prometheus          │
│  (port 9090)             │
└────────┬─────────────────┘
         │
┌────────▼─────────────────┐
│      Grafana             │
│  (port 3000)             │
└──────────────────────────┘
```

## Test Tools

### Python Load Test

Flexible, easy to customize:

```bash
python3 loadtest/loadtest.py \
  --url http://localhost:8080 \
  --workers 20 \
  --requests 100 \
  --repos github.com/kubernetes/kubernetes \
          github.com/golang/go
```

**Options:**
- `--url`: Target URL
- `--workers`: Concurrent workers
- `--requests`: Requests per worker
- `--think-time`: Delay between requests (ms)
- `--repos`: Repository list to test
- `--output`: JSON output file

**Output:**
```
=== Load Test Summary ===

Total Requests:    2000
Successful:        1995
Failed:            5
Success Rate:      99.75%
Total Duration:    45.23s
Requests/sec:      44.21

Response Times (ms):
  Min:             12.34
  Max:             456.78
  Mean:            89.45
  Median:          67.89
  P95:             234.56
  P99:             389.12
```

### k6 Load Test

Advanced load testing with gradual ramp-up:

```bash
# Run k6 test
docker-compose --profile loadtest up k6
```

**Test stages:**
- Ramp to 10 VUs (2 min)
- Stay at 10 VUs (5 min)
- Ramp to 50 VUs (2 min)
- Stay at 50 VUs (5 min)
- Ramp to 100 VUs (2 min)
- Stay at 100 VUs (5 min)
- Ramp down (2 min)

## Test Scenarios

### Scenario 1: Cache Warm-up

Test cache efficiency after warm-up period:

```bash
# Phase 1: Populate cache
python3 loadtest.py --workers 5 --requests 50

# Phase 2: Test cache hits
python3 loadtest.py --workers 20 --requests 200

# Expected: >80% cache hit rate in Phase 2
```

### Scenario 2: Cold Start

Test behavior with empty cache:

```bash
# Clear caches
docker-compose down -v
docker-compose up -d

# Run test
python3 loadtest.py --workers 10 --requests 100

# Expected: Higher latency, all cache misses initially
```

### Scenario 3: High Concurrency

Test maximum concurrent requests:

```bash
python3 loadtest.py \
  --workers 100 \
  --requests 50 \
  --think-time 0

# Monitor: CPU, memory, connection count
```

### Scenario 4: Repository Diversity

Test with many different repositories:

```bash
python3 loadtest.py \
  --workers 20 \
  --requests 100 \
  --repos $(cat popular-repos.txt)

# Tests cache distribution and eviction
```

### Scenario 5: Sustained Load

Test stability over time:

```bash
# Run for 1 hour
python3 loadtest.py \
  --workers 10 \
  --requests 3600 \
  --think-time 1000

# Monitor: memory leaks, cache growth, error rates
```

## Interpreting Results

### Key Metrics

**Success Rate:**
- Target: > 99%
- Warning: < 99%
- Critical: < 95%

**Response Time (P95):**
- Excellent: < 100ms
- Good: 100-500ms
- Acceptable: 500-1000ms
- Poor: > 1000ms

**Cache Hit Rate:**
- Excellent: > 90%
- Good: 80-90%
- Acceptable: 70-80%
- Poor: < 70%

**Throughput:**
- Single instance: 500-1000 req/sec
- Per sidecar: 50-100 req/sec (sufficient for most workloads)

### Performance Baselines

**Cached requests (hit):**
```
Min:    5-10ms    (memory access)
P50:    10-20ms   (disk read)
P95:    50-100ms  (cold disk cache)
P99:    100-200ms (contention)
Max:    500ms+    (GC pauses)
```

**Cache miss (fetch from upstream):**
```
Min:    100ms     (small repo, fast network)
P50:    500ms     (typical)
P95:    2000ms    (large repo)
P99:    5000ms    (very large repo)
Max:    30000ms   (timeout)
```

## Capacity Planning

### Single Instance Capacity

Based on typical workloads:

| Metric | Value |
|--------|-------|
| Max requests/sec | 500-1000 |
| Concurrent connections | 1000 |
| Cache size | 100GB-1TB |
| CPU (sustained) | 2-4 cores |
| Memory | 4-8GB |

### Sidecar Capacity

Per-pod capacity:

| Metric | Value |
|--------|-------|
| Requests/hour | 100-1000 |
| Peak requests/sec | 10-50 |
| Cache size | 1-10GB |
| CPU | 250m-1 core |
| Memory | 512MB-2GB |

### Scaling Formula

```
Required pods = (Peak requests/sec) / (Requests per pod/sec)

Example:
- Peak traffic: 1000 req/sec
- Per pod capacity: 10 req/sec
- Required pods: 100

With 50% buffer: 150 pods
```

## Monitoring During Tests

### HAProxy Stats

```bash
open http://localhost:8404

# Key metrics:
# - Request distribution across instances
# - Health check status
# - Error rates per backend
```

### Prometheus Queries

```promql
# Cache hit rate
rate(cache_hits_total[5m]) / rate(requests_total[5m])

# Request latency (P95)
histogram_quantile(0.95, rate(request_duration_seconds_bucket[5m]))

# Error rate
rate(errors_total[5m]) / rate(requests_total[5m])

# Requests per second
rate(requests_total[5m])
```

### System Metrics

```bash
# CPU usage
docker stats goblet-1 goblet-2 goblet-3

# Disk I/O
docker exec goblet-1 iostat -x 1

# Network
docker exec goblet-1 iftop -i eth0
```

## Troubleshooting

### High Latency

**Symptoms:** P95 > 1000ms

**Diagnosis:**
```bash
# Check cache hit rate
curl http://localhost:8080/metrics | grep cache_hit_rate

# Check disk I/O
docker exec goblet-1 iostat -x

# Check network latency to upstream
docker exec goblet-1 ping -c 10 github.com
```

**Solutions:**
- Increase cache size
- Use faster storage (SSD)
- Add more instances
- Pre-warm cache

### High Error Rate

**Symptoms:** Errors > 5%

**Diagnosis:**
```bash
# Check logs
docker-compose logs goblet-1 | grep ERROR

# Check upstream connectivity
docker exec goblet-1 curl -I https://github.com
```

**Solutions:**
- Verify upstream connectivity
- Check authentication
- Increase timeout values
- Review rate limiting

### Uneven Load Distribution

**Symptoms:** One instance much busier than others

**Diagnosis:**
```bash
# Check HAProxy distribution
curl http://localhost:8404 | grep -A 20 goblet_shards
```

**Solutions:**
- Verify consistent hashing configured
- Check if specific repos dominate traffic
- Review routing algorithm

### Memory Growth

**Symptoms:** Memory usage increases over time

**Diagnosis:**
```bash
# Monitor memory over time
watch -n 5 'docker stats --no-stream goblet-1'

# Check cache size
docker exec goblet-1 du -sh /cache
```

**Solutions:**
- Set cache size limits
- Enable LRU eviction
- Increase memory limits
- Review for memory leaks

## Best Practices

### Before Testing

1. **Define objectives:**
   - What are you testing?
   - What metrics matter?
   - What's the success criteria?

2. **Prepare environment:**
   - Clean state (clear caches if needed)
   - Monitoring configured
   - Baseline metrics captured

3. **Plan test scenarios:**
   - Realistic traffic patterns
   - Representative repository mix
   - Appropriate duration

### During Testing

1. **Monitor actively:**
   - Watch dashboards
   - Check logs for errors
   - Note any anomalies

2. **Document observations:**
   - Screenshot metrics
   - Record configuration
   - Note any changes made

3. **Adjust gradually:**
   - Change one variable at a time
   - Allow time to stabilize
   - Compare with baseline

### After Testing

1. **Analyze results:**
   - Compare against targets
   - Identify bottlenecks
   - Document findings

2. **Save data:**
   - Export metrics
   - Save logs
   - Archive configurations

3. **Create action items:**
   - Performance improvements needed
   - Configuration changes
   - Scaling requirements

## Example Test Plan

### Objective
Validate Goblet can handle 1M requests/month with sidecar pattern.

### Setup
- 10 pods with sidecars
- 1GB cache per pod
- Representative repo mix

### Test Phases

**Phase 1: Baseline (30 min)**
```bash
# Light load to warm up cache
python3 loadtest.py --workers 5 --requests 100
```
*Expected: Establish baseline latency and hit rate*

**Phase 2: Normal Load (1 hour)**
```bash
# Simulate average daily traffic
python3 loadtest.py --workers 10 --requests 1000
```
*Expected: P95 < 500ms, hit rate > 80%*

**Phase 3: Peak Load (30 min)**
```bash
# Simulate 10x peak
python3 loadtest.py --workers 100 --requests 100
```
*Expected: P95 < 1000ms, no errors*

**Phase 4: Sustained Peak (2 hours)**
```bash
# Validate stability at peak
python3 loadtest.py --workers 50 --requests 2000
```
*Expected: Stable performance, no memory leaks*

### Success Criteria
- ✅ Success rate > 99%
- ✅ P95 latency < 500ms (normal), < 1000ms (peak)
- ✅ Cache hit rate > 80%
- ✅ No memory leaks
- ✅ No errors under sustained load

## Summary

**Quick Reference:**

```bash
# Start environment
cd loadtest && make start

# Run test
make loadtest-python

# View stats
open http://localhost:8404

# Cleanup
make stop
```

**Key Takeaways:**

1. Start with warm-up phase
2. Test realistic scenarios
3. Monitor actively
4. Document everything
5. Plan for peak + buffer

**Next Steps:**

- Run baseline tests in dev
- Validate capacity planning
- Test failure scenarios
- Move to staging
- Production rollout with monitoring

For detailed test scripts, see [`loadtest/`](../../loadtest/) directory.
