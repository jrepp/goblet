# Troubleshooting Guide

Common issues and solutions for Goblet deployments.

## Quick Diagnostics

```bash
# Check pod status
kubectl get pods -l app=goblet

# View logs
kubectl logs -f deployment/goblet

# Check metrics
curl http://localhost:8080/metrics

# Test connectivity
curl http://localhost:8080/healthz
```

## Common Issues

### High Latency

**Symptoms:** P95 latency > 1000ms

**Causes:**
- Low cache hit rate
- Slow disk I/O
- Network issues with upstream
- Resource constraints

**Solutions:**
1. Check cache hit rate: `curl http://localhost:8080/metrics | grep cache_hit`
2. Check disk I/O: `iostat -x 1`
3. Increase cache size
4. Use faster storage (SSD)
5. Pre-warm cache

See [Monitoring Guide](monitoring.md) for detailed metrics.

### High Error Rate

**Symptoms:** Errors > 5%

**Causes:**
- Upstream connectivity issues
- Authentication failures
- Rate limiting
- Misconfigurations

**Solutions:**
1. Check logs for error patterns
2. Verify upstream connectivity
3. Check authentication configuration
4. Review rate limits

### Out of Disk Space

**Symptoms:** "no space left on device"

**Causes:**
- Cache grew beyond capacity
- No eviction policy
- Large repositories

**Solutions:**
1. Implement [tiered storage](../architecture/storage-optimization.md)
2. Add LRU eviction
3. Increase disk size
4. Clean old repositories

### Cross-Tenant Access

**Symptoms:** User A can access User B's repositories

**Cause:** Missing tenant isolation

**Solution:** Implement [isolation strategy](../security/isolation-strategies.md)

### Pod Won't Start

**Symptoms:** CrashLoopBackOff

**Diagnostics:**
```bash
kubectl describe pod <pod-name>
kubectl logs <pod-name> --previous
```

**Common Causes:**
- Image pull errors
- Resource limits too low
- Volume mount issues
- Configuration errors

## Debugging Commands

### Check Configuration
```bash
kubectl get configmap goblet-config -o yaml
```

### View Full Logs
```bash
kubectl logs deployment/goblet --all-containers=true --tail=100
```

### Check Resource Usage
```bash
kubectl top pod -l app=goblet
```

### Test Cache
```bash
# Clone repo twice, second should be faster
time git clone https://github.com/kubernetes/kubernetes.git test1
rm -rf test1
time git clone https://github.com/kubernetes/kubernetes.git test2
```

## Getting Help

1. Check this guide
2. Review [documentation](../index.md)
3. Search [GitHub issues](https://github.com/google/goblet/issues)
4. Ask in [discussions](https://github.com/google/goblet/discussions)

## Related Documentation

- [Monitoring Guide](monitoring.md)
- [Load Testing](load-testing.md)
- [Deployment Patterns](deployment-patterns.md)
