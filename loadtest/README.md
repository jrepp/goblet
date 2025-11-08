# Goblet Load Testing & Deployment Patterns

This directory contains load testing infrastructure and deployment patterns for scaling Goblet in production environments.

## ⚠️ CRITICAL SECURITY NOTICE

**Before deploying Goblet with private repositories, read the [Security Isolation Guide](../docs/security/isolation-strategies.md)**

Goblet's default configuration is **UNSAFE for multi-tenant deployments with private repositories**. Users can access each other's cached private repos. See [Security](#security-considerations) section below.

## Table of Contents

1. [Security Considerations](#security-considerations)
2. [Architecture Overview](#architecture-overview)
3. [Load Testing Setup](#load-testing-setup)
4. [Deployment Patterns](#deployment-patterns)
5. [Scaling Considerations](#scaling-considerations)
6. [Sidecar Pattern for Terraform](#sidecar-pattern-for-terraform)

---

## Security Considerations

### The Problem

**Default cache key:** `/cache/{host}/{repo-path}` - NO user/tenant identifier
**Risk:** User A's private repos accessible to User B

### Solutions (Pick One)

| Pattern | Security | Storage | Complexity | Use Case |
|---------|----------|---------|------------|----------|
| **Sidecar** (Recommended) | ✅ Perfect | Medium | Low | Terraform, CI/CD |
| **User-Scoped** | ✅ Perfect | High | Medium | Risk scanning |
| **Tenant-Scoped** | ✅ Good | Medium | Medium | Terraform Cloud |
| **Network Isolation** | ✅ Perfect | Low | High | Compliance |
| ❌ **Default (None)** | ❌ UNSAFE | Low | Low | Public repos only |

**Quick Fix:** Use sidecar pattern (one instance per pod). See [`kubernetes-sidecar-deployment.yaml`](./kubernetes-sidecar-deployment.yaml)

**Detailed Guide:** See [Security Isolation Strategies](../docs/security/isolation-strategies.md)

**Architecture:** See [Design Decisions](../docs/architecture/design-decisions.md)

---

## Architecture Overview

### Stateful vs Stateless

**Goblet is a STATEFUL caching proxy** with the following characteristics:

- **File-based cache**: Bare Git repositories stored on local disk
- **In-process state**: `sync.Map` for repository management with per-repo mutexes
- **Single-writer assumption**: Git operations expect exclusive access to repositories
- **No distributed coordination**: No distributed locks or leader election

### Scaling Implications

❌ **NOT SAFE**: Multiple instances sharing the same cache directory
- Git operations will race and corrupt repositories
- In-memory locks are process-local

✅ **SAFE**:
- Single instance per cache directory
- Multiple instances with repository sharding
- Sidecar pattern (one cache per application pod)

---

## Load Testing Setup

### Prerequisites

- Docker and Docker Compose
- Python 3.8+ (for Python-based load test)
- OR k6 (for JavaScript-based load test)

### Quick Start

1. **Start the load test environment:**

   ```bash
   docker-compose -f docker-compose.loadtest.yml up -d
   ```

   This starts:
   - 3 Goblet instances (goblet-1, goblet-2, goblet-3)
   - HAProxy load balancer with consistent hashing (port 8080)
   - Prometheus metrics collector (port 9090)
   - Grafana dashboard (port 3000)

2. **View HAProxy stats:**

   ```bash
   open http://localhost:8404
   ```

3. **Run Python load test:**

   ```bash
   python3 loadtest/loadtest.py \
     --url http://localhost:8080 \
     --workers 20 \
     --requests 100 \
     --repos github.com/kubernetes/kubernetes github.com/golang/go
   ```

4. **Run k6 load test:**

   ```bash
   docker-compose -f docker-compose.loadtest.yml --profile loadtest up k6
   ```

5. **View Grafana dashboards:**

   ```bash
   open http://localhost:3000
   # Login: admin/admin
   ```

### Load Test Scripts

#### Python Script (`loadtest.py`)

Flexible, easy-to-customize load test script:

```bash
python3 loadtest/loadtest.py \
  --url http://localhost:8080 \
  --workers 50 \
  --requests 200 \
  --think-time 50 \
  --repos github.com/user/repo1 github.com/user/repo2 \
  --output results.json
```

**Options:**
- `--url`: Target URL (default: http://localhost:8080)
- `--workers`: Number of concurrent workers (default: 10)
- `--requests`: Requests per worker (default: 100)
- `--think-time`: Delay between requests in ms (default: 100)
- `--repos`: List of repository paths to test
- `--output`: JSON output file for results

#### k6 Script (`k6-script.js`)

Advanced load testing with gradual ramp-up:

```javascript
// Stages defined in k6-script.js:
// - Ramp up to 10 VUs over 2 minutes
// - Stay at 10 VUs for 5 minutes
// - Ramp up to 50 VUs over 2 minutes
// - Stay at 50 VUs for 5 minutes
// - Ramp up to 100 VUs over 2 minutes
// - Stay at 100 VUs for 5 minutes
// - Ramp down over 2 minutes
```

Customize repositories in `k6-script.js` line 22.

---

## Deployment Patterns

### Pattern 1: Repository Sharding with HAProxy

**Use case**: Centralized cache with horizontal scaling

**Architecture:**
```
              HAProxy (consistent hashing on URL)
                        |
        +---------------+---------------+
        |               |               |
    Goblet-1        Goblet-2        Goblet-3
   (repos A-H)     (repos I-P)     (repos Q-Z)
        |               |               |
    Cache Dir 1     Cache Dir 2     Cache Dir 3
```

**Implementation:**

```yaml
# See docker-compose.loadtest.yml
# HAProxy uses: balance uri whole
```

**Pros:**
- True horizontal scaling
- Linear throughput increase
- Each instance caches a subset of repos

**Cons:**
- Cache efficiency reduced (each instance has partial cache)
- Need sticky routing per repository
- Adds load balancer complexity

### Pattern 2: Sidecar Pattern (Recommended for Terraform)

**Use case**: Large-scale deployments with millions of requests per month

**Architecture:**
```
Kubernetes Pod
  |
  +-- Terraform Agent Container
  |     (git -> http://localhost:8080)
  |
  +-- Goblet Sidecar Container
        (port 8080, cache: /cache)
        |
        +-- EmptyDir Volume (10Gi)
```

**Implementation:**

See `kubernetes-sidecar-deployment.yaml`

**Benefits:**
- ✅ Zero network latency (localhost)
- ✅ Pod-scoped cache lifecycle
- ✅ Natural workload partitioning
- ✅ No coordination needed
- ✅ Scales linearly with pod count
- ✅ Perfect for Terraform Cloud Agents

**Configuration:**

```yaml
# In Terraform agent container:
env:
  - name: HTTP_PROXY
    value: "http://localhost:8080"
  # OR
  - name: GIT_CONFIG_KEY_0
    value: "http.proxy"
  - name: GIT_CONFIG_VALUE_0
    value: "http://localhost:8080"
```

### Pattern 3: Regional Instances

**Use case**: Multi-region deployments with geo-distributed teams

**Architecture:**
```
US-EAST Region          EU-WEST Region          APAC Region
  |                       |                       |
Goblet Instance       Goblet Instance       Goblet Instance
(10GB cache)          (10GB cache)          (10GB cache)
```

**Pros:**
- Low latency for regional users
- Independent failure domains
- Simple deployment model

**Cons:**
- Cache duplication across regions
- Higher storage costs

---

## Scaling Considerations

### When to Scale

**Vertical Scaling (increase instance size):**
- CPU bound: Many concurrent requests, protocol parsing
- Memory bound: Large number of cached repositories
- Disk I/O bound: Frequent cache misses, large repos

**Horizontal Scaling (add instances):**
- Request rate exceeds single instance capacity (~1000 req/s)
- Need high availability / redundancy
- Regional distribution required
- Workload naturally partitioned (e.g., per-tenant)

### Metrics to Monitor

1. **Request Rate**: requests/sec per instance
2. **Cache Hit Rate**: % of requests served from cache
3. **Response Latency**: p50, p95, p99 latencies
4. **Disk Usage**: cache directory size
5. **Git Fetch Duration**: time to fetch from upstream
6. **Error Rate**: failed requests / total requests

### Capacity Planning

**Single Instance Capacity (estimated):**
- **Request Rate**: 500-1000 req/s (depends on cache hit rate)
- **Concurrent Connections**: 1000+
- **Cached Repositories**: 100-1000 (depends on size)
- **Disk I/O**: ~100 MB/s sustained

**For millions of requests/month:**
```
1,000,000 requests/month = ~0.4 requests/sec average
With peak factor 10x = ~4 requests/sec peak
Single instance: SUFFICIENT for average load
Sidecar pattern: BETTER for peak handling + resilience
```

### Recommended Architecture for Terraform Cloud Scale

**Deployment:**
- 100 Terraform Agent pods
- Each pod with Goblet sidecar
- 10GB cache per pod
- HPA (Horizontal Pod Autoscaler): 100-500 pods

**Expected Performance:**
- 1M requests/month = ~10K requests/pod/month
- Avg: 0.004 req/sec per pod (trivial)
- Peak (10x): 0.04 req/sec per pod (trivial)
- **Cache hit rate**: 80-95% (after warm-up)

**Benefits:**
- No shared state = no coordination overhead
- Linear scaling with pod count
- Cache warm-up happens naturally per pod
- Failed pods don't affect others
- Rolling updates are safe

---

## Sidecar Pattern for Terraform

### Why Sidecar for Terraform Agents?

1. **Workload Isolation**: Each Terraform run is independent
2. **Cache Locality**: Terraform runs often use same repos
3. **No Network Overhead**: Localhost communication
4. **Natural Partitioning**: No need for distributed coordination
5. **Pod Lifecycle**: Cache created/destroyed with pod

### Deployment Steps

1. **Build Goblet container image:**

   ```bash
   docker build -t goblet:latest .
   docker tag goblet:latest your-registry/goblet:v1.0.0
   docker push your-registry/goblet:v1.0.0
   ```

2. **Deploy to Kubernetes:**

   ```bash
   kubectl create namespace terraform-agents
   kubectl apply -f loadtest/kubernetes-sidecar-deployment.yaml
   ```

3. **Verify deployment:**

   ```bash
   kubectl get pods -n terraform-agents
   kubectl logs -n terraform-agents <pod-name> -c goblet-cache
   ```

4. **Monitor with Prometheus:**

   ```bash
   kubectl port-forward -n terraform-agents svc/terraform-agent-metrics 8080:8080
   curl http://localhost:8080/metrics
   ```

### Configuration Tips

**Cache Size:**
```yaml
volumes:
  - name: git-cache
    emptyDir:
      sizeLimit: 10Gi  # Adjust based on repo sizes
```

**Resource Allocation:**
```yaml
resources:
  requests:
    cpu: "500m"      # Increase for cache-heavy workloads
    memory: "1Gi"    # Increase for many repos
  limits:
    cpu: "1"
    memory: "2Gi"
```

**Autoscaling:**
```yaml
minReplicas: 10    # Baseline capacity
maxReplicas: 100   # Peak capacity
```

### Testing Sidecar Deployment

```bash
# Port forward to a pod
kubectl port-forward -n terraform-agents <pod-name> 8080:8080

# Test from your local machine
python3 loadtest/loadtest.py \
  --url http://localhost:8080 \
  --workers 5 \
  --requests 50
```

---

## Troubleshooting

### Load Balancer Issues

**Problem**: Requests not evenly distributed

**Check HAProxy stats:**
```bash
curl http://localhost:8404
```

**Solution**: Verify consistent hashing is working:
```bash
# Same repo should always go to same backend
for i in {1..10}; do
  curl -v http://localhost:8080/github.com/kubernetes/kubernetes/info/refs \
    2>&1 | grep "X-Served-By"
done
```

### Cache Corruption

**Problem**: Git errors, repository corruption

**Likely cause**: Multiple instances sharing same cache directory

**Solution**:
1. Stop all instances
2. Clear cache: `rm -rf /cache/*`
3. Ensure proper sharding/sidecar deployment
4. Restart with isolated caches

### High Memory Usage

**Problem**: Goblet using excessive memory

**Likely cause**: Many large repositories cached

**Solution**:
1. Reduce cache size with LRU eviction (future enhancement)
2. Increase sizeLimit for emptyDir volume
3. Partition repositories across more instances

### Slow Response Times

**Problem**: High p95/p99 latencies

**Diagnosis**:
```bash
# Check metrics
curl http://localhost:8080/metrics | grep git_fetch

# Check upstream latency
curl http://localhost:8080/metrics | grep upstream_duration
```

**Solutions**:
- Increase worker pool size
- Add more instances (sharding)
- Optimize upstream connectivity
- Add backup storage for cold starts

---

## Future Enhancements

### Distributed Coordination

To enable true shared-cache multi-instance deployment:

1. **Distributed locks** (Redis, etcd)
2. **Leader election** per repository
3. **Cache coherency protocol**
4. **Shared metadata store**

### Cache Management

1. **LRU eviction** for size-bounded cache
2. **Metrics-based warming** (pre-fetch popular repos)
3. **Tiered storage** (hot/cold separation)
4. **Cache replication** for HA

---

## Related Documentation

- [Goblet README](../README.md)
- [Offline Mode Documentation](../testing/TEST_COVERAGE.md)
- [Docker Compose Configuration](../docker-compose.loadtest.yml)
- [Kubernetes Deployment](./kubernetes-sidecar-deployment.yaml)

---

## Questions & Support

For issues or questions about load testing:
1. Check HAProxy stats: http://localhost:8404
2. Check Prometheus metrics: http://localhost:9090
3. Check Grafana dashboards: http://localhost:3000
4. Review container logs: `docker-compose logs -f goblet-1`
