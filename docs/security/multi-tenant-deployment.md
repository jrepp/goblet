# Multi-Tenant Deployment Guide

This guide provides step-by-step instructions for securely deploying Goblet in multi-tenant environments.

## Overview

Multi-tenant deployments require additional security measures to prevent private repository data from being accessed by unauthorized users. This guide covers the recommended deployment patterns and configurations.

## Quick Start

**Recommended:** Deploy using the sidecar pattern for immediate, secure multi-tenant support.

```bash
kubectl apply -f examples/kubernetes-sidecar-secure.yaml
```

## Deployment Patterns

See [Deployment Patterns](../operations/deployment-patterns.md) for detailed architecture options.

### Sidecar Pattern

**Recommended for most deployments.**

Deploy one Goblet instance per workload:
- Perfect isolation (no shared cache)
- No code changes required
- Simple scaling

See detailed guide: [Sidecar Pattern](../operations/deployment-patterns.md#sidecar-pattern)

### Namespace Isolation

**For enterprise deployments with compliance requirements.**

Deploy separate instances per tenant in isolated namespaces:
- Strong Kubernetes-native isolation
- NetworkPolicy enforcement
- Resource quotas per tenant

See detailed guide: [Namespace Isolation](../operations/deployment-patterns.md#namespace-isolation)

### Application-Level Isolation

**For advanced deployments requiring code integration.**

Implement tenant-aware cache partitioning:
- Requires code changes
- Fine-grained control
- Flexible policies

See [Isolation Strategies](isolation-strategies.md) for implementation details

## Security Checklist

Before deploying in production:

- [ ] Choose isolation pattern based on requirements
- [ ] Review [Security Overview](README.md)
- [ ] Implement [Isolation Strategy](isolation-strategies.md)
- [ ] Configure authentication (OAuth2/OIDC)
- [ ] Enable TLS for all connections
- [ ] Set up audit logging
- [ ] Configure NetworkPolicy
- [ ] Set resource limits
- [ ] Test cross-tenant access (must fail)
- [ ] Review [Security Detailed Guide](detailed-guide.md)

## Configuration Example

```yaml
# Sidecar with tenant isolation
apiVersion: apps/v1
kind: Deployment
metadata:
  name: terraform-agent
spec:
  template:
    spec:
      containers:
      - name: app
        env:
        - name: HTTP_PROXY
          value: "http://localhost:8080"

      - name: goblet-sidecar
        image: goblet:latest
        env:
        - name: GOBLET_ISOLATION_MODE
          value: "sidecar"
        volumeMounts:
        - name: cache
          mountPath: /cache

      volumes:
      - name: cache
        emptyDir:
          sizeLimit: 10Gi
```

## Testing Isolation

Verify that isolation works correctly:

```bash
# Test that tenant A cannot access tenant B's data
./scripts/test-isolation.sh tenant-a tenant-b
```

## Next Steps

- Review [Operations Guide](../operations/deployment-patterns.md)
- Set up [Monitoring](../operations/monitoring.md)
- Configure [Load Testing](../operations/load-testing.md)

## Related Documentation

- [Security Overview](README.md)
- [Isolation Strategies](isolation-strategies.md)
- [Deployment Patterns](../operations/deployment-patterns.md)
- [Getting Started](../getting-started.md)
