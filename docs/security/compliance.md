# Compliance Guide

This guide provides information about using Goblet in compliance-sensitive environments.

## Supported Compliance Frameworks

### SOC 2 Type II

**Data Security Controls:**
- Encryption at rest using AES-256-GCM
- TLS 1.3 for data in transit
- Audit logging for all access events
- Role-based access control (RBAC)

**Availability Controls:**
- Health check endpoints
- Prometheus metrics for monitoring
- High availability deployment patterns
- Automated failover support

**Confidentiality Controls:**
- Multi-tenant isolation strategies
- Network segmentation with NetworkPolicy
- Secure credential management
- Data residency controls

### ISO 27001

**Access Control (A.9):**
- Authentication via OAuth2/OIDC
- Authorization at cache key level
- Session management
- Audit trails

**Cryptography (A.10):**
- Industry-standard encryption algorithms
- Secure key management with envelope encryption
- Certificate management for TLS

**Operations Security (A.12):**
- Malware protection (container image scanning)
- Backup procedures
- Logging and monitoring
- Vulnerability management

**Communications Security (A.13):**
- Network segregation
- TLS enforcement
- Secure protocols only

### GDPR

**Data Protection:**
- Data minimization (only cache what's needed)
- Encryption at rest and in transit
- Access controls per tenant
- Audit logging

**Data Subject Rights:**
- Right to erasure (cache eviction API)
- Data portability (standard Git protocol)
- Right to access (audit logs)

## Deployment Checklist

### Pre-Deployment

- [ ] Complete security assessment
- [ ] Review [Security Overview](README.md)
- [ ] Choose appropriate [Isolation Strategy](isolation-strategies.md)
- [ ] Configure encryption (see [Detailed Guide](detailed-guide.md))
- [ ] Set up audit logging
- [ ] Define data retention policies

### Deployment

- [ ] Deploy with namespace isolation for enterprise
- [ ] Configure NetworkPolicy rules
- [ ] Enable TLS for all connections
- [ ] Set up RBAC policies
- [ ] Configure resource quotas
- [ ] Enable Pod Security Standards

### Post-Deployment

- [ ] Test isolation between tenants
- [ ] Verify encryption at rest
- [ ] Verify TLS connectivity
- [ ] Configure monitoring and alerting
- [ ] Set up log aggregation
- [ ] Perform security audit
- [ ] Document configuration

## Audit Logging

### Required Events

**Authentication:**
- User login attempts (success/failure)
- Token validation
- Authorization decisions

**Data Access:**
- Repository access (read/write)
- Cache hits and misses
- Upstream fetch events

**Administrative:**
- Configuration changes
- Cache eviction events
- Security policy updates

### Log Format

```json
{
  "timestamp": "2025-11-07T10:00:00Z",
  "event_type": "cache_access",
  "user_id": "user@example.com",
  "tenant_id": "tenant-123",
  "repository": "github.com/org/repo",
  "action": "read",
  "result": "success",
  "source_ip": "10.0.1.5",
  "duration_ms": 45
}
```

### Log Retention

**Recommendation:**
- Security logs: 1 year minimum
- Access logs: 90 days minimum
- Audit logs: 7 years for regulated industries

## Data Residency

### Regional Deployment

Deploy Goblet in specific regions to meet data residency requirements:

**EU Deployments:**
```yaml
# kubernetes deployment
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: topology.kubernetes.io/region
            operator: In
            values:
            - eu-west-1
            - eu-central-1
```

**Storage Location:**
- Configure cloud storage in compliant regions
- Use regional PersistentVolumes
- Ensure backup storage is in same region

## Incident Response

### Security Incident Process

1. **Detection**: Monitor alerts and logs
2. **Containment**: Isolate affected instances
3. **Investigation**: Review audit logs
4. **Remediation**: Apply fixes and patches
5. **Documentation**: Record incident details
6. **Review**: Update security controls

### Contact Information

- **Security Team**: security@example.com
- **On-Call**: See PagerDuty rotation
- **Escalation**: See incident response playbook

## Evidence Collection

### For Audits

**System Documentation:**
- Architecture diagrams (see [Design Decisions](../architecture/design-decisions.md))
- Network diagrams with NetworkPolicy
- Data flow diagrams
- Deployment configurations

**Security Controls:**
- Encryption configuration
- Access control policies
- Audit log samples
- Monitoring dashboards

**Testing Evidence:**
- Penetration test reports
- Vulnerability scan results
- Isolation test results (see [Testing Isolation](multi-tenant-deployment.md#testing-isolation))

## Compliance Testing

### Automated Tests

```bash
# Test encryption at rest
./scripts/test-encryption.sh

# Test tenant isolation
./scripts/test-isolation.sh tenant-a tenant-b

# Test audit logging
./scripts/test-audit-logs.sh

# Test TLS enforcement
./scripts/test-tls.sh
```

### Manual Verification

1. **Access Control**: Verify RBAC policies prevent unauthorized access
2. **Encryption**: Verify data is encrypted at rest
3. **Network Segmentation**: Verify NetworkPolicy blocks cross-tenant traffic
4. **Audit Logs**: Verify all required events are logged

## Related Documentation

- [Security Overview](README.md)
- [Isolation Strategies](isolation-strategies.md)
- [Detailed Security Guide](detailed-guide.md)
- [Multi-Tenant Deployment](multi-tenant-deployment.md)
- [Threat Model](threat-model.md)
