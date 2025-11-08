# Scaling Strategies

How to scale Goblet for high-traffic deployments.

## Vertical Scaling

Increase resources for single instance:

- **CPU:** 2-8 cores
- **Memory:** 4-16GB  
- **Disk:** Fast SSD, 100GB-1TB
- **Capacity:** Up to 1,000 req/sec

## Horizontal Scaling

Add more instances:

1. **Sidecar Pattern:** N instances (one per workload)
2. **Sharded Pattern:** HAProxy with consistent hashing
3. **Regional Pattern:** Instance per region

See [Deployment Patterns](../operations/deployment-patterns.md) for details.

## Auto-Scaling

Kubernetes HPA configuration:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: goblet-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: goblet
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

## Related Documentation

- [Deployment Patterns](../operations/deployment-patterns.md)
- [Load Testing](../operations/load-testing.md)
- [Design Decisions](design-decisions.md)
