# Deployment Patterns

This guide describes proven deployment patterns for Goblet based on your scale and requirements.

## Pattern Selection

Choose a deployment pattern based on your needs:

| Pattern | Best For | Isolation | Complexity | Cost |
|---------|----------|-----------|------------|------|
| [Single Instance](#single-instance) | Development, < 1K req/day | N/A | Low | $ |
| [Sidecar](#sidecar-pattern) | Multi-tenant, CI/CD | Perfect | Low | $$ |
| [Namespace](#namespace-isolation) | Enterprise, compliance | High | Medium | $$$ |
| [Sharded](#sharded-cluster) | High traffic > 10K req/day | Good | High | $$$$ |

## Single Instance

### Overview

One Goblet instance serves all requests. Suitable for development or single-tenant production use.

```
┌─────────────┐
│   Clients   │
└──────┬──────┘
       │
┌──────▼──────┐
│   Goblet    │
│   Instance  │
└──────┬──────┘
       │
  ┌────▼────┐
  │  Cache  │
  └─────────┘
```

### When to Use

- Development and testing
- Single user or service account
- Public repositories only
- Low traffic (< 1,000 requests/day)

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goblet
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: goblet
        image: goblet:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: cache
          mountPath: /cache
      volumes:
      - name: cache
        persistentVolumeClaim:
          claimName: goblet-cache
---
apiVersion: v1
kind: Service
metadata:
  name: goblet
spec:
  selector:
    app: goblet
  ports:
  - port: 80
    targetPort: 8080
```

### Scaling Limits

- **Throughput:** 500-1,000 requests/second
- **Concurrent users:** 100-500
- **Cache size:** 100GB-1TB
- **Single point of failure**

## Sidecar Pattern

### Overview

Each workload gets its own Goblet instance as a sidecar container. Provides perfect isolation with minimal configuration.

```
┌────────────────────────────────┐
│  Pod (Workload)                │
│  ┌──────────┐  ┌────────────┐  │
│  │   App    │  │  Goblet    │  │
│  │Container │──│  Sidecar   │  │
│  └──────────┘  └─────┬──────┘  │
│                      │          │
│                 ┌────▼──────┐   │
│                 │   Cache   │   │
│                 │ (emptyDir)│   │
│                 └───────────┘   │
└────────────────────────────────┘
```

### When to Use

- ✅ **Recommended default for multi-tenant deployments**
- Multiple users with different access permissions
- Terraform Cloud, security scanning
- CI/CD runners
- Kubernetes-native environments

### Benefits

- **Perfect isolation:** Each workload has dedicated cache
- **No shared state:** Eliminates cross-tenant risks
- **Simple scaling:** Add pods for more capacity
- **Zero network latency:** Localhost communication
- **No code changes:** Deploy with existing Goblet

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: terraform-agent
spec:
  replicas: 10  # Scale as needed
  template:
    spec:
      containers:
      # Main application
      - name: terraform-agent
        image: terraform:latest
        env:
        - name: HTTP_PROXY
          value: "http://localhost:8080"
        - name: HTTPS_PROXY
          value: "http://localhost:8080"

      # Goblet sidecar
      - name: goblet-cache
        image: goblet:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: cache
          mountPath: /cache
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 1
            memory: 2Gi

      volumes:
      - name: cache
        emptyDir:
          sizeLimit: 10Gi
```

### Auto-Scaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: terraform-agent-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: terraform-agent
  minReplicas: 10
  maxReplicas: 100
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Cost Analysis

**Example:** 100 pods for 1M requests/month
- Per pod: ~10,000 requests/month
- CPU: 50m average, 500m burst
- Memory: 1GB
- Cache: 10GB per pod
- **Total cost:** ~$155/month (varies by provider)

### Capacity Planning

| Pods | Requests/Month | Cost/Month | Use Case |
|------|----------------|------------|----------|
| 10 | 100K | $15 | Small team |
| 50 | 500K | $75 | Growing team |
| 100 | 1M | $155 | Enterprise |
| 500 | 5M | $775 | Large scale |

## Namespace Isolation

### Overview

Separate Goblet deployments per tenant in isolated Kubernetes namespaces with network policies.

```
┌─────────────────────────────────────┐
│  Namespace: tenant-acme             │
│  ┌──────────┐   ┌──────────┐       │
│  │  Goblet  │───│  Network │       │
│  │  Deploy  │   │  Policy  │       │
│  └────┬─────┘   └──────────┘       │
│       │                             │
│  ┌────▼─────┐                       │
│  │  Cache   │                       │
│  │  (PVC)   │                       │
│  └──────────┘                       │
└─────────────────────────────────────┘
        ┼
┌─────────────────────────────────────┐
│  Namespace: tenant-bigcorp          │
│  ┌──────────┐   ┌──────────┐       │
│  │  Goblet  │───│  Network │       │
│  │  Deploy  │   │  Policy  │       │
│  └────┬─────┘   └──────────┘       │
│       │                             │
│  ┌────▼─────┐                       │
│  │  Cache   │                       │
│  │  (PVC)   │                       │
│  └──────────┘                       │
└─────────────────────────────────────┘
```

### When to Use

- Enterprise multi-tenant deployments
- Compliance requirements (SOC 2, ISO 27001)
- Strong isolation needed
- Different SLAs per tenant
- Resource quotas per tenant

### Deployment

```yaml
# Create namespace per tenant
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-acme-corp
  labels:
    tenant: acme-corp
---
# Network policy for isolation
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: goblet-isolation
  namespace: tenant-acme-corp
spec:
  podSelector:
    matchLabels:
      app: goblet
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Only from same namespace
  - from:
    - namespaceSelector:
        matchLabels:
          tenant: acme-corp
    ports:
    - port: 8080
  egress:
  # DNS, KMS, upstream only
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - port: 53
      protocol: UDP
---
# Resource quota per tenant
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-quota
  namespace: tenant-acme-corp
spec:
  hard:
    requests.cpu: "10"
    requests.memory: "20Gi"
    persistentvolumeclaims: "10"
---
# Goblet deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goblet
  namespace: tenant-acme-corp
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: goblet
        image: goblet:latest
        volumeMounts:
        - name: cache
          mountPath: /cache
      volumes:
      - name: cache
        persistentVolumeClaim:
          claimName: goblet-cache-acme-corp
```

### Management Script

```bash
#!/bin/bash
# deploy-tenant.sh

TENANT=$1

kubectl create namespace tenant-$TENANT
kubectl label namespace tenant-$TENANT tenant=$TENANT

# Apply network policy
kubectl apply -f network-policy.yaml -n tenant-$TENANT

# Apply resource quota
kubectl apply -f resource-quota.yaml -n tenant-$TENANT

# Deploy goblet
kubectl apply -f goblet-deployment.yaml -n tenant-$TENANT

echo "Tenant $TENANT deployed successfully"
```

## Sharded Cluster

### Overview

Multiple Goblet instances with load balancer using consistent hashing to route requests.

```
        ┌───────────────┐
        │ Load Balancer │
        │(Consistent    │
        │ Hash on URL)  │
        └───────┬───────┘
                │
    ┌───────────┼───────────┐
    │           │           │
┌───▼───┐   ┌───▼───┐   ┌───▼───┐
│Goblet │   │Goblet │   │Goblet │
│  -1   │   │  -2   │   │  -3   │
└───┬───┘   └───┬───┘   └───┬───┘
    │           │           │
┌───▼───┐   ┌───▼───┐   ┌───▼───┐
│Cache-1│   │Cache-2│   │Cache-3│
└───────┘   └───────┘   └───────┘
```

### When to Use

- High traffic (> 10,000 requests/day)
- Need high availability
- Want to share cache across team
- Have operational expertise

### Load Balancer Configuration

```
# HAProxy config
backend goblet_shards
    balance uri whole
    hash-type consistent

    # Route same repo to same instance
    server goblet-1 10.0.1.1:8080 check
    server goblet-2 10.0.1.2:8080 check
    server goblet-3 10.0.1.3:8080 check
```

### Deployment

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: goblet
spec:
  serviceName: goblet
  replicas: 3
  template:
    spec:
      containers:
      - name: goblet
        image: goblet:latest
        volumeMounts:
        - name: cache
          mountPath: /cache
  volumeClaimTemplates:
  - metadata:
      name: cache
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
```

### Scaling Considerations

**Adding a node:**
```bash
# Gradually increases StatefulSet replicas
kubectl scale statefulset goblet --replicas=4

# HAProxy automatically includes new instance
# Some repositories will migrate to new instance
```

**Removing a node:**
```bash
# Drain node gracefully
kubectl drain node-4 --ignore-daemonsets

# Scale down
kubectl scale statefulset goblet --replicas=3

# Repositories redistribute to remaining instances
```

## Hybrid Patterns

### Sidecar + Namespace

Combine sidecar pattern with namespace isolation for maximum security:

```yaml
# Each tenant gets own namespace
# Each workload in namespace gets sidecar
# Network policy enforces namespace boundary
```

**Best for:** Enterprise SaaS platforms

### Sharded + Sidecar

Use sharding for shared resources, sidecar for user workloads:

```
Shared Infrastructure (sharded):
  ├─ Common public repositories
  └─ Terraform modules

User Workloads (sidecar):
  ├─ Private repositories
  └─ User-specific caches
```

**Best for:** Hybrid cloud/on-premise deployments

## Migration Paths

### From Single Instance to Sidecar

```bash
# 1. Deploy sidecar pattern in new namespace
kubectl create ns goblet-v2
kubectl apply -f sidecar-deployment.yaml -n goblet-v2

# 2. Gradually migrate workloads
kubectl label namespace app-team-1 goblet-version=v2

# 3. Monitor both versions
kubectl logs -l app=goblet -n goblet-v1
kubectl logs -l app=goblet -n goblet-v2

# 4. Decommission old instance when ready
kubectl delete deployment goblet -n goblet-v1
```

### From Sidecar to Namespace

```bash
# Create tenant namespaces
for tenant in acme bigcorp startup; do
  kubectl create ns tenant-$tenant
  kubectl apply -f tenant-deployment.yaml -n tenant-$tenant
done

# Migrate workloads namespace by namespace
kubectl move-workloads tenant-acme
```

## Monitoring Deployments

### Key Metrics by Pattern

| Pattern | Key Metrics |
|---------|-------------|
| Single Instance | Request rate, cache hit rate, disk usage |
| Sidecar | Pods running, cache size per pod, memory usage |
| Namespace | Quota utilization, cross-namespace calls (should be 0) |
| Sharded | Load distribution, rebalancing events |

### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
- name: goblet
  rules:
  # Low cache hit rate
  - alert: LowCacheHitRate
    expr: rate(cache_hits_total[5m]) / rate(requests_total[5m]) < 0.5
    for: 10m

  # High error rate
  - alert: HighErrorRate
    expr: rate(errors_total[5m]) / rate(requests_total[5m]) > 0.05
    for: 5m

  # Disk space low
  - alert: LowDiskSpace
    expr: disk_usage_bytes / disk_capacity_bytes > 0.9
    for: 5m
```

## Best Practices

### General

1. **Start simple:** Use sidecar pattern unless specific needs require alternatives
2. **Monitor first:** Instrument before scaling
3. **Test isolation:** Verify cross-tenant access fails
4. **Document decisions:** Record why you chose a pattern

### Sidecar Pattern

1. Set appropriate `emptyDir` size limits
2. Use resource requests/limits
3. Configure HPA for auto-scaling
4. Monitor per-pod cache hit rates

### Namespace Isolation

1. Use NetworkPolicy to enforce boundaries
2. Set ResourceQuota per namespace
3. Monitor quota utilization
4. Audit cross-namespace access

### Sharded Cluster

1. Use consistent hashing in load balancer
2. Monitor load distribution
3. Plan shard additions carefully
4. Test failover scenarios

## Troubleshooting

### Sidecar Not Starting

```bash
# Check container logs
kubectl logs pod-name -c goblet-cache

# Check events
kubectl describe pod pod-name

# Common issues:
# - Resource limits too low
# - Volume mount permissions
# - Image pull errors
```

### High Memory Usage

```bash
# Check cache size
kubectl exec pod-name -c goblet-cache -- du -sh /cache

# Reduce cache size limit
# Edit deployment: emptyDir.sizeLimit
```

### Cross-Tenant Access

```bash
# Test isolation
./test-isolation.sh tenant-a tenant-b

# If test fails:
# - Verify NetworkPolicy applied
# - Check namespace labels
# - Review RBAC rules
```

## Summary

**Quick Decision Guide:**

- **Starting out?** → Sidecar Pattern
- **Enterprise compliance?** → Namespace Isolation
- **High traffic (> 10K req/day)?** → Sharded Cluster
- **Development only?** → Single Instance

**Next Steps:**

1. Review your requirements
2. Choose a pattern
3. Deploy to dev/staging
4. Monitor and validate
5. Deploy to production

For detailed implementation, see example configurations in [`examples/`](../../examples/).
