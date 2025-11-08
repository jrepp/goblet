# Storage Cost Optimization for Goblet

## Overview

Git caches can grow to hundreds of GB per tenant. This document provides strategies to minimize storage costs while maintaining performance using cloud provider tiered storage.

---

## Storage Cost Comparison (per TB/month, 2025)

| Tier | AWS | GCP | Azure | Use Case | Access Time |
|------|-----|-----|-------|----------|-------------|
| **Hot** | $23 | $20 | $18 | Active repos | < 10ms |
| **Cool** | $10 | $10 | $10 | Recent repos | < 100ms |
| **Archive** | $1 | $1.20 | $0.99 | Old repos | Minutes-hours |
| **Cold Archive** | $0.36 | $0.40 | $0.18 | Compliance | Hours |

**Cost Reduction:** Up to **98% savings** with proper tiering

---

## Recommended Architecture

### Three-Tier Strategy

```
┌──────────────────────────────────────────────────────────┐
│                    Hot Tier (NVMe SSD)                   │
│  • Last accessed: < 7 days                               │
│  • Cost: $20-23/TB/month                                 │
│  • Access: < 10ms                                        │
│  • Size: 10-20% of total                                 │
└────────────────┬─────────────────────────────────────────┘
                 │ Automatic tiering (7 days)
┌────────────────▼─────────────────────────────────────────┐
│                    Cool Tier (HDD/S3)                    │
│  • Last accessed: 7-90 days                              │
│  • Cost: $10/TB/month                                    │
│  • Access: < 100ms                                       │
│  • Size: 30-50% of total                                 │
└────────────────┬─────────────────────────────────────────┘
                 │ Automatic tiering (90 days)
┌────────────────▼─────────────────────────────────────────┐
│                 Archive Tier (Glacier/Coldline)          │
│  • Last accessed: > 90 days                              │
│  • Cost: $1/TB/month                                     │
│  • Access: Minutes-hours                                 │
│  • Size: 30-60% of total                                 │
└──────────────────────────────────────────────────────────┘
```

### Cost Savings Example

**Scenario:** 1TB cache, 60% cold data

| Storage Strategy | Cost/month | Annual Cost |
|-----------------|------------|-------------|
| All Hot (SSD) | $20 | $240 |
| **Tiered** (40% hot, 30% cool, 30% archive) | **$9.30** | **$111.60** |
| **Savings** | **54%** | **$128.40** |

---

## AWS Implementation

### Strategy: S3 Intelligent-Tiering + EBS

#### Architecture

```
┌─────────────────────────────────────────────┐
│  EC2 Instance (Goblet)                      │
│  ┌──────────────────────────────────────┐   │
│  │ Active Cache (EBS gp3)               │   │
│  │ /cache/hot/                          │   │
│  │ Last 7 days: 200GB                   │   │
│  └──────────────────────────────────────┘   │
└─────────┬───────────────────────────────────┘
          │ Sync every 1 hour
┌─────────▼───────────────────────────────────┐
│ S3 Intelligent-Tiering Bucket               │
│ s3://goblet-cache-tenant-{id}/              │
│                                             │
│ Auto-tiering:                               │
│ • 0-30 days → Frequent Access   $23/TB     │
│ • 30-90 days → Infrequent       $12.50/TB  │
│ • 90+ days → Archive            $4/TB      │
│ • 180+ days → Deep Archive      $1/TB      │
└─────────────────────────────────────────────┘
```

#### Implementation

```yaml
# goblet-config.yaml
storage:
  primary:
    type: "ebs"
    mount: "/cache/hot"
    size_gb: 200
    volume_type: "gp3"  # $0.08/GB/month = $16/month for 200GB
    iops: 3000
    throughput_mbps: 125

  tiering:
    enabled: true
    provider: "aws-s3"

    # S3 bucket with Intelligent-Tiering
    s3:
      bucket: "goblet-cache-${TENANT_ID}"
      region: "us-east-1"
      storage_class: "INTELLIGENT_TIERING"

    # Tiering rules
    rules:
      - name: "sync-to-s3"
        condition: "age > 1 hour AND access_count = 0"
        action: "upload"
        delete_local: false

      - name: "evict-from-local"
        condition: "age > 7 days"
        action: "delete"
        keep_in_s3: true

      - name: "restore-on-access"
        condition: "cache_miss AND exists_in_s3"
        action: "download"
        priority: "high"
```

#### Terraform Configuration

```hcl
# S3 bucket with Intelligent-Tiering
resource "aws_s3_bucket" "goblet_cache" {
  for_each = var.tenants

  bucket = "goblet-cache-${each.key}"

  tags = {
    Tenant = each.key
    Purpose = "git-cache"
  }
}

resource "aws_s3_bucket_intelligent_tiering_configuration" "goblet_cache" {
  for_each = var.tenants

  bucket = aws_s3_bucket.goblet_cache[each.key].id
  name   = "EntireCache"

  tiering {
    access_tier = "ARCHIVE_ACCESS"
    days        = 90
  }

  tiering {
    access_tier = "DEEP_ARCHIVE_ACCESS"
    days        = 180
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "goblet_cache" {
  for_each = var.tenants

  bucket = aws_s3_bucket.goblet_cache[each.key].id

  rule {
    id     = "abort-incomplete-uploads"
    status = "Enabled"

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  rule {
    id     = "delete-old-versions"
    status = "Enabled"

    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

# EBS volume for hot cache
resource "aws_ebs_volume" "goblet_hot_cache" {
  for_each = var.goblet_instances

  availability_zone = each.value.az
  size              = 200  # GB
  type              = "gp3"
  iops              = 3000
  throughput        = 125
  encrypted         = true
  kms_key_id        = aws_kms_key.goblet_cache.arn

  tags = {
    Name = "goblet-hot-cache-${each.key}"
    Tier = "hot"
  }
}
```

#### Cost Breakdown

```
Hot Cache (EBS gp3): 200GB × $0.08/GB = $16/month
S3 Intelligent-Tiering:
  - 400GB × $0.023/GB (frequent, 0-30 days) = $9.20
  - 300GB × $0.0125/GB (infrequent, 30-90 days) = $3.75
  - 100GB × $0.004/GB (archive, 90+ days) = $0.40

Total: $29.35/month for 1TB (vs $80 all-EBS)
Savings: 63%
```

---

## GCP Implementation

### Strategy: Persistent Disk + Cloud Storage Autoclass

#### Architecture

```
┌─────────────────────────────────────────────┐
│  GKE Node (Goblet Pod)                      │
│  ┌──────────────────────────────────────┐   │
│  │ Active Cache (SSD PD)                │   │
│  │ /cache/hot/                          │   │
│  │ Last 7 days: 200GB                   │   │
│  └──────────────────────────────────────┘   │
└─────────┬───────────────────────────────────┘
          │ Sync with Cloud Storage Fuse (gcsfuse)
┌─────────▼───────────────────────────────────┐
│ Cloud Storage Autoclass Bucket              │
│ gs://goblet-cache-tenant-{id}/              │
│                                             │
│ Auto-tiering:                               │
│ • Frequent Access → Standard    $20/TB     │
│ • Infrequent      → Nearline    $10/TB     │
│ • Archive         → Coldline    $4/TB      │
│ • Deep Archive    → Archive     $1.20/TB   │
└─────────────────────────────────────────────┘
```

#### Implementation

```yaml
# goblet-gcp-config.yaml
storage:
  primary:
    type: "gcp-persistent-disk"
    mount: "/cache/hot"
    size_gb: 200
    disk_type: "pd-ssd"  # $0.17/GB/month = $34/month

  tiering:
    enabled: true
    provider: "gcp-gcs"

    gcs:
      bucket: "goblet-cache-${TENANT_ID}"
      location: "us-central1"
      storage_class: "AUTOCLASS"  # Automatic tiering

    # Mount GCS as filesystem using gcsfuse
    gcsfuse:
      enabled: true
      mount: "/cache/cold"
      cache_max_size_mb: 1024  # Local cache for GCS data
      stat_cache_ttl: "1h"

    rules:
      - name: "sync-to-gcs"
        condition: "age > 6 hours"
        action: "upload"
        delete_local: false

      - name: "evict-from-pd"
        condition: "age > 7 days"
        action: "delete"
        keep_in_gcs: true

      - name: "lazy-load"
        condition: "cache_miss"
        action: "mount"  # Access via gcsfuse, auto-download
```

#### Terraform Configuration

```hcl
# GCS bucket with Autoclass
resource "google_storage_bucket" "goblet_cache" {
  for_each = var.tenants

  name          = "goblet-cache-${each.key}"
  location      = "US"
  storage_class = "STANDARD"  # Autoclass starts here

  autoclass {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 180
    }
    action {
      type          = "SetStorageClass"
      storage_class = "ARCHIVE"
    }
  }

  lifecycle_rule {
    condition {
      age = 365
      with_state = "ARCHIVED"
    }
    action {
      type = "Delete"
    }
  }

  encryption {
    default_kms_key_name = google_kms_crypto_key.goblet_cache.id
  }
}

# Persistent disk for hot cache
resource "google_compute_disk" "goblet_hot_cache" {
  for_each = var.goblet_instances

  name  = "goblet-hot-cache-${each.key}"
  type  = "pd-ssd"
  zone  = each.value.zone
  size  = 200  # GB

  disk_encryption_key {
    kms_key_self_link = google_kms_crypto_key.goblet_cache.id
  }

  labels = {
    tier = "hot"
    tenant = each.key
  }
}

# Kubernetes PVC using the disk
resource "kubernetes_persistent_volume_claim" "goblet_hot_cache" {
  for_each = var.goblet_instances

  metadata {
    name      = "goblet-hot-cache"
    namespace = "tenant-${each.key}"
  }

  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        storage = "200Gi"
      }
    }
    storage_class_name = "ssd-retain"
  }
}
```

#### Cost Breakdown

```
Hot Cache (PD-SSD): 200GB × $0.17/GB = $34/month
GCS Autoclass: 800GB average across tiers
  - 300GB × $0.020/GB (standard) = $6.00
  - 300GB × $0.010/GB (nearline) = $3.00
  - 200GB × $0.004/GB (coldline) = $0.80

Total: $43.80/month for 1TB (vs $170 all-SSD)
Savings: 74%
```

---

## Azure Implementation

### Strategy: Premium SSD + Blob Storage with Access Tiers

#### Architecture

```
┌─────────────────────────────────────────────┐
│  AKS Node (Goblet Pod)                      │
│  ┌──────────────────────────────────────┐   │
│  │ Active Cache (Premium SSD)           │   │
│  │ /cache/hot/                          │   │
│  │ Last 7 days: 200GB                   │   │
│  └──────────────────────────────────────┘   │
└─────────┬───────────────────────────────────┘
          │ Sync with Blob Storage using Blobfuse2
┌─────────▼───────────────────────────────────┐
│ Azure Blob Storage (Lifecycle Management)   │
│ container: goblet-cache-tenant-{id}         │
│                                             │
│ Auto-tiering:                               │
│ • 0-30 days → Hot               $18/TB     │
│ • 30-90 days → Cool             $10/TB     │
│ • 90+ days → Archive            $0.99/TB   │
│ • 180+ days → Cold Archive (opt) $0.18/TB  │
└─────────────────────────────────────────────┘
```

#### Implementation

```yaml
# goblet-azure-config.yaml
storage:
  primary:
    type: "azure-disk"
    mount: "/cache/hot"
    size_gb: 200
    sku: "Premium_LRS"  # $0.128/GB/month = $25.60/month

  tiering:
    enabled: true
    provider: "azure-blob"

    blob:
      storage_account: "gobletcache${TENANT_ID}"
      container: "cache"
      access_tier: "Hot"  # Initial tier, will auto-tier

    # Mount using Blobfuse2
    blobfuse:
      enabled: true
      mount: "/cache/cold"
      tmp_path: "/mnt/blobfuse-tmp"
      cache_size_mb: 1024

    rules:
      - name: "sync-to-blob"
        condition: "age > 12 hours"
        action: "upload"
        access_tier: "Hot"

      - name: "tier-to-cool"
        condition: "age > 30 days"
        action: "change_tier"
        access_tier: "Cool"

      - name: "tier-to-archive"
        condition: "age > 90 days"
        action: "change_tier"
        access_tier: "Archive"

      - name: "evict-from-disk"
        condition: "age > 7 days"
        action: "delete"
        keep_in_blob: true

      - name: "rehydrate-on-access"
        condition: "cache_miss AND tier = Archive"
        action: "rehydrate"
        priority: "Standard"  # or "High" for faster (more expensive)
```

#### Terraform Configuration

```hcl
# Storage account
resource "azurerm_storage_account" "goblet_cache" {
  for_each = var.tenants

  name                     = "gobletcache${replace(each.key, "-", "")}"
  resource_group_name      = azurerm_resource_group.goblet.name
  location                 = azurerm_resource_group.goblet.location
  account_tier             = "Standard"
  account_replication_type = "LRS"

  blob_properties {
    versioning_enabled = true

    # Lifecycle management
    lifecycle_management {
      rule {
        name    = "tier-to-cool"
        enabled = true

        filters {
          blob_types   = ["blockBlob"]
          prefix_match = ["cache/"]
        }

        actions {
          base_blob {
            tier_to_cool_after_days_since_modification = 30
            tier_to_archive_after_days_since_modification = 90
            delete_after_days_since_modification = 365
          }
        }
      }
    }
  }

  tags = {
    Tenant = each.key
  }
}

# Container
resource "azurerm_storage_container" "goblet_cache" {
  for_each = var.tenants

  name                  = "cache"
  storage_account_name  = azurerm_storage_account.goblet_cache[each.key].name
  container_access_type = "private"
}

# Managed disk for hot cache
resource "azurerm_managed_disk" "goblet_hot_cache" {
  for_each = var.goblet_instances

  name                 = "goblet-hot-cache-${each.key}"
  location             = azurerm_resource_group.goblet.location
  resource_group_name  = azurerm_resource_group.goblet.name
  storage_account_type = "Premium_LRS"
  create_option        = "Empty"
  disk_size_gb         = 200

  encryption_settings {
    enabled = true
    disk_encryption_key {
      secret_url      = azurerm_key_vault_secret.disk_encryption_key.id
      source_vault_id = azurerm_key_vault.goblet.id
    }
  }

  tags = {
    tier   = "hot"
    tenant = each.key
  }
}

# Kubernetes PVC
resource "kubernetes_persistent_volume_claim" "goblet_hot_cache" {
  for_each = var.goblet_instances

  metadata {
    name      = "goblet-hot-cache"
    namespace = "tenant-${each.key}"
  }

  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        storage = "200Gi"
      }
    }
    storage_class_name = "managed-premium-retain"
  }
}
```

#### Cost Breakdown

```
Hot Cache (Premium SSD): 200GB × $0.128/GB = $25.60/month
Blob Storage: 800GB across tiers
  - 300GB × $0.018/GB (hot, 0-30 days) = $5.40
  - 300GB × $0.010/GB (cool, 30-90 days) = $3.00
  - 200GB × $0.00099/GB (archive, 90+ days) = $0.20

Total: $34.20/month for 1TB (vs $128 all-Premium)
Savings: 73%
```

---

## Comparison Matrix

### Cost Comparison (1TB cache over 1 year)

| Provider | All Hot | Tiered | Savings |
|----------|---------|--------|---------|
| AWS | $960 | $352 | **$608 (63%)** |
| GCP | $2,040 | $526 | **$1,514 (74%)** |
| Azure | $1,536 | $410 | **$1,126 (73%)** |

**Winner: Azure** (lowest tiered cost)

### Performance Comparison

| Metric | AWS | GCP | Azure |
|--------|-----|-----|-------|
| Hot tier latency | 5ms (gp3) | 3ms (SSD) | 4ms (Premium) |
| Cool tier latency | 50ms (S3) | 40ms (GCS) | 60ms (Blob) |
| Archive restore | 3-5 hours | 12 hours | 15 hours |
| Throughput (hot) | 125MB/s | 120MB/s | 120MB/s |

**Winner: GCP** (lowest latency for cool tier)

### Feature Comparison

| Feature | AWS | GCP | Azure |
|---------|-----|-----|-------|
| Automatic tiering | ✅ Intelligent-Tiering | ✅ Autoclass | ⚠️  Manual lifecycle |
| FUSE mounting | ⚠️  s3fs (3rd party) | ✅ gcsfuse (official) | ✅ Blobfuse2 (official) |
| Encryption | ✅ KMS | ✅ KMS | ✅ Key Vault |
| Multi-region | ✅ S3 Replication | ✅ Dual-region | ✅ GRS/RA-GRS |
| Cost explorer | ✅ Excellent | ✅ Good | ⚠️  Basic |

**Winner: AWS** (best automation and tooling)

---

## Hybrid Strategy: Multi-Cloud Cost Optimization

### Recommended Approach

Use cheapest storage for each tier across providers:

```
Hot Tier: GCP Persistent Disk SSD ($34/month for 200GB)
  └─ Lowest latency, good price

Cool Tier: Azure Blob Cool ($3/month for 300GB)
  └─ Best cool tier pricing

Archive: AWS S3 Deep Archive ($0.36/month for 200GB)
  └─ Cheapest long-term storage
```

**Total hybrid cost:** $37.36/month for 700GB actively managed cache

**Challenges:**
- Complexity of multi-cloud orchestration
- Data transfer costs between providers
- Operational overhead

**Verdict:** Only for very large deployments (100+ TB)

---

## Best Practices

### 1. Access Pattern Analysis

```bash
# Analyze cache access patterns
./scripts/analyze-access-patterns.sh /cache

# Output:
# Repository Access Report (Last 90 days):
#   github.com/acme/app: 1,234 accesses (hot)
#   github.com/acme/lib: 45 accesses (cool)
#   github.com/acme/archive: 2 accesses (archive candidate)
```

### 2. Tiering Policy Configuration

```yaml
# Customize based on your access patterns
tiering:
  policies:
    - name: "frequently-accessed"
      condition: "access_count > 10/week"
      tier: "hot"
      cost_optimized: false

    - name: "occasionally-accessed"
      condition: "access_count 1-10/week"
      tier: "cool"
      cost_optimized: true

    - name: "rarely-accessed"
      condition: "access_count < 1/week"
      tier: "archive"
      cost_optimized: true
      rehydration: "standard"  # 15-hour restore

    - name: "compliance-only"
      condition: "age > 365 days"
      tier: "cold-archive"
      cost_optimized: true
      rehydration: "bulk"  # 48-hour restore
```

### 3. Cache Warming

```go
// Pre-warm cache for known access patterns
func (c *CacheManager) WarmCache(ctx context.Context, repos []string) error {
    for _, repoURL := range repos {
        // Check current tier
        tier, err := c.storage.GetTier(repoURL)
        if err != nil {
            return err
        }

        // Rehydrate if archived
        if tier == "archive" || tier == "cold-archive" {
            log.Printf("Rehydrating %s (currently in %s)", repoURL, tier)
            if err := c.storage.Rehydrate(repoURL, "expedited"); err != nil {
                return err
            }
        }

        // Move to hot tier
        if err := c.storage.SetTier(repoURL, "hot"); err != nil {
            return err
        }
    }

    return nil
}

// Example: Warm cache before business hours
func (c *CacheManager) ScheduledWarmup() {
    // Daily at 6 AM
    cron.Schedule("0 6 * * *", func() {
        repos := c.getFrequentlyAccessedRepos()
        c.WarmCache(context.Background(), repos)
    })
}
```

### 4. Cost Monitoring

```go
type StorageCostTracker struct {
    provider    string
    tenantID    string
    prometheus  *prometheus.Client
}

func (s *StorageCostTracker) TrackCosts() {
    // Hot tier cost
    hotSize := s.getSize("hot")
    hotCost := hotSize * s.getPricing("hot")
    s.prometheus.RecordCost("hot", hotCost, s.tenantID)

    // Cool tier cost
    coolSize := s.getSize("cool")
    coolCost := coolSize * s.getPricing("cool")
    s.prometheus.RecordCost("cool", coolCost, s.tenantID)

    // Archive tier cost
    archiveSize := s.getSize("archive")
    archiveCost := archiveSize * s.getPricing("archive")
    s.prometheus.RecordCost("archive", archiveCost, s.tenantID)

    // Data transfer cost
    transferCost := s.getTransferCost()
    s.prometheus.RecordCost("transfer", transferCost, s.tenantID)

    // Total
    totalCost := hotCost + coolCost + archiveCost + transferCost
    s.prometheus.RecordCost("total", totalCost, s.tenantID)
}
```

---

## Recommendations by Scale

### Small (< 100GB, < 1000 req/day)

**Recommendation:** All-hot storage (simplest)

- AWS: EBS gp3
- GCP: Persistent Disk SSD
- Azure: Premium SSD

**Why:** Tiering overhead not worth it at this scale

---

### Medium (100GB - 1TB, 1000-10000 req/day)

**Recommendation:** Hot + Cool tiering

- **AWS:** EBS gp3 (hot) + S3 Intelligent-Tiering
- **GCP:** PD-SSD (hot) + GCS Autoclass
- **Azure:** Premium SSD (hot) + Blob Cool

**Savings:** 50-70%

---

### Large (1TB - 10TB, > 10000 req/day)

**Recommendation:** Hot + Cool + Archive

- **AWS:** EBS gp3 (hot, 200GB) + S3 Intelligent-Tiering (warm) + S3 Glacier (archive)
- **GCP:** PD-SSD (hot, 200GB) + GCS Nearline (warm) + GCS Coldline (archive)
- **Azure:** Premium SSD (hot, 200GB) + Blob Cool (warm) + Blob Archive

**Savings:** 70-85%

---

### Enterprise (> 10TB, > 100000 req/day)

**Recommendation:** Hot + Cool + Archive + Cold Archive + Multi-region

- **AWS:** EBS io2 Block Express (ultra-hot) + gp3 (hot) + S3 INT (warm) + Glacier (archive) + Deep Archive (cold)
- **GCP:** Local SSD (ultra-hot) + PD-SSD (hot) + GCS Standard (warm) + Coldline (archive) + Archive (cold)
- **Azure:** Ultra Disk (ultra-hot) + Premium SSD (hot) + Blob Hot (warm) + Cool (archive) + Archive (cold) + Cold Archive (long-term)

**Additional:** CDN for frequently accessed public repos

**Savings:** 80-95%

---

## Summary

**Recommended Providers by Priority:**

1. **AWS** - Best automation (Intelligent-Tiering), great tooling
2. **Azure** - Lowest cost for tiered storage
3. **GCP** - Best performance (gcsfuse), good auto-tiering

**Key Takeaways:**

- ✅ Tiering can save **60-95%** on storage costs
- ✅ Most repos accessed < once/week (ideal for archival)
- ✅ Automatic tiering (AWS/GCP) reduces operational overhead
- ✅ Monitor access patterns to optimize tier placement

**Action Items:**

1. Analyze current access patterns
2. Choose provider based on existing infrastructure
3. Implement hot + cool tiers initially
4. Add archive tier after 90 days of data
5. Monitor costs and adjust policies
