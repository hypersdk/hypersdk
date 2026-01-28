# NFS Deployment Architecture

Comprehensive architecture documentation for HyperSDK deployment with NFS shared storage across Kubernetes and native Linux environments.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Cloud Provider (vSphere/AWS/Azure/GCP)            │
│                                     │                                        │
│                              ┌──────▼────────┐                              │
│                              │  Virtual VMs  │                              │
│                              └──────┬────────┘                              │
└─────────────────────────────────────┼──────────────────────────────────────┘
                                      │ Export via API
                                      │
┌─────────────────────────────────────▼──────────────────────────────────────┐
│                         Kubernetes Cluster                                  │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────┐      │
│  │                    hypersdk Namespace                           │      │
│  │                                                                  │      │
│  │  ┌──────────────────────┐         ┌──────────────────────┐     │      │
│  │  │  HyperSDK Pod 1      │         │  HyperSDK Pod 2      │     │      │
│  │  │  ┌────────────────┐  │         │  ┌────────────────┐  │     │      │
│  │  │  │ hypervisord    │  │         │  │ hypervisord    │  │     │      │
│  │  │  │ :8080 (API)    │  │         │  │ :8080 (API)    │  │     │      │
│  │  │  │ :8081 (metrics)│  │         │  │ :8081 (metrics)│  │     │      │
│  │  │  └────────────────┘  │         │  └────────────────┘  │     │      │
│  │  │         │             │         │         │             │     │      │
│  │  │  ┌──────▼──────────┐ │         │  ┌──────▼──────────┐ │     │      │
│  │  │  │  /exports       │ │         │  │  /exports       │ │     │      │
│  │  │  │  (NFS Mount)    │ │         │  │  (NFS Mount)    │ │     │      │
│  │  │  └──────┬──────────┘ │         │  └──────┬──────────┘ │     │      │
│  │  └─────────┼─────────────┘         └─────────┼─────────────┘     │      │
│  │            │                                  │                   │      │
│  │            └──────────────────┬───────────────┘                   │      │
│  │                               │                                   │      │
│  │                    ┌──────────▼───────────┐                       │      │
│  │                    │  PersistentVolumeClaim│                      │      │
│  │                    │  hypersdk-exports     │                      │      │
│  │                    │  AccessMode: RWX      │                      │      │
│  │                    └──────────┬────────────┘                      │      │
│  │                               │                                   │      │
│  │  ┌──────────────────────────┐ │  ┌──────────────────────────┐   │      │
│  │  │  hyper2kvm Pod           │ │  │  Other Consumer Pods     │   │      │
│  │  │  ┌────────────────────┐  │ │  │  ┌────────────────────┐  │   │      │
│  │  │  │ /imports (RO)      │  │ │  │  │ /data (RO/RW)      │  │   │      │
│  │  │  │ (NFS Mount)        │◄─┘ └─►│  │ (NFS Mount)        │  │   │      │
│  │  │  └────────────────────┘  │    │  └────────────────────┘  │   │      │
│  │  └──────────────────────────┘    └──────────────────────────┘   │      │
│  │                                                                  │      │
│  └──────────────────────────────────────────────────────────────────┘      │
│                                      │                                      │
└──────────────────────────────────────┼──────────────────────────────────────┘
                                       │
                            ┌──────────▼───────────┐
                            │  PersistentVolume    │
                            │  Type: NFS           │
                            │  AccessMode: RWX     │
                            └──────────┬───────────┘
                                       │
                          ┌────────────▼────────────┐
                          │     NFS Server          │
                          │  192.168.1.100:2049     │
                          │  /exports/hypersdk      │
                          │                         │
                          │  Storage: 2Ti           │
                          │  Protocol: NFSv4.1      │
                          └────────────┬────────────┘
                                       │
                                       │ NFS Mount
                      ┌────────────────┼────────────────┐
                      │                │                │
         ┌────────────▼─────┐  ┌───────▼───────┐  ┌───▼──────────────┐
         │ Native Linux     │  │ Native Linux  │  │ Native Linux     │
         │ Host 1           │  │ Host 2        │  │ Host 3           │
         │                  │  │               │  │                  │
         │ /mnt/hypersdk-   │  │ /mnt/hypersdk-│  │ /mnt/hypersdk-   │
         │ exports          │  │ exports       │  │ exports          │
         │                  │  │               │  │                  │
         │ ┌──────────────┐ │  │ ┌───────────┐ │  │ ┌──────────────┐ │
         │ │ hyper2kvm    │ │  │ │ KVM/libvirt│ │  │ │Custom Tools │ │
         │ │ CLI          │ │  │ │           │ │  │ │             │ │
         │ └──────────────┘ │  │ └───────────┘ │  │ └──────────────┘ │
         └──────────────────┘  └───────────────┘  └──────────────────┘
```

## Data Flow Architecture

### Export Flow (vSphere → Kubernetes → NFS)

```
┌──────────────┐
│   vSphere    │
│   vCenter    │
└──────┬───────┘
       │ 1. API Request
       │    (GOVC)
       ▼
┌──────────────────────────┐
│  HyperSDK Pod (K8s)      │
│  ┌────────────────────┐  │
│  │  API Handler       │  │ 2. Job Queued
│  └─────────┬──────────┘  │
│            │              │
│  ┌─────────▼──────────┐  │
│  │  Export Worker     │  │ 3. VM Export
│  │  ┌──────────────┐  │  │    Process
│  │  │ Download VM  │  │  │
│  │  │ Stream to    │  │  │
│  │  │ /exports     │  │  │
│  │  └──────┬───────┘  │  │
│  └─────────┼──────────┘  │
│            │              │
│  ┌─────────▼──────────┐  │
│  │  /exports Volume   │  │ 4. Write to NFS
│  │  (NFS PVC)         │  │
│  └─────────┬──────────┘  │
└────────────┼─────────────┘
             │
             │ NFS Protocol (NFSv4.1)
             │ 1MB chunks
             ▼
    ┌────────────────┐
    │  NFS Server    │ 5. Store on Disk
    │  Filesystem    │
    │  /exports/     │
    │  hypersdk/     │
    │  vm-name/      │
    │    vm.ova      │
    └────────────────┘
```

### Import Flow (NFS → Native Linux / Kubernetes)

```
┌────────────────────┐
│   NFS Server       │
│   /exports/        │
│   hypersdk/        │
│   vm-name/         │
│     vm.ova         │
└────────┬───────────┘
         │
         │ NFS Mount
         │
    ┌────┴─────────────────────┐
    │                          │
    ▼                          ▼
┌─────────────────┐    ┌──────────────────────┐
│ Kubernetes Pod  │    │  Native Linux Host   │
│                 │    │                      │
│ /imports/       │    │  /mnt/hypersdk-      │
│ vm-name/        │    │  exports/vm-name/    │
│   vm.ova        │    │    vm.ova            │
│                 │    │                      │
│ ┌─────────────┐ │    │  ┌─────────────────┐ │
│ │ hyper2kvm   │ │    │  │   hyper2kvm     │ │
│ │ convert     │ │    │  │   CLI           │ │
│ │             │ │    │  │                 │ │
│ │ vm.ova →    │ │    │  │   vm.ova →      │ │
│ │ vm.qcow2    │ │    │  │   vm.qcow2      │ │
│ └─────────────┘ │    │  └─────────────────┘ │
│                 │    │                      │
│ ┌─────────────┐ │    │  ┌─────────────────┐ │
│ │/var/lib/    │ │    │  │ /var/lib/       │ │
│ │libvirt/     │ │    │  │ libvirt/images/ │ │
│ │images/      │ │    │  │   vm.qcow2      │ │
│ └─────────────┘ │    │  └─────────────────┘ │
└─────────────────┘    └──────────────────────┘
```

## Network Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster Network                     │
│                         10.244.0.0/16                             │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  hypersdk Namespace                                         │ │
│  │                                                             │ │
│  │  Pod Network: 10.244.1.0/24                                │ │
│  │                                                             │ │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │ │
│  │  │ HyperSDK Pod │    │ HyperSDK Pod │    │ hyper2kvm    │ │ │
│  │  │ 10.244.1.10  │    │ 10.244.1.11  │    │ Pod          │ │ │
│  │  │              │    │              │    │ 10.244.1.20  │ │ │
│  │  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘ │ │
│  │         │                   │                   │         │ │
│  │         └───────────────────┼───────────────────┘         │ │
│  │                             │                             │ │
│  │                    ┌────────▼────────┐                    │ │
│  │                    │  Service        │                    │ │
│  │                    │  hypervisord    │                    │ │
│  │                    │  ClusterIP      │                    │ │
│  │                    │  10.96.1.100    │                    │ │
│  │                    └────────┬────────┘                    │ │
│  │                             │                             │ │
│  └─────────────────────────────┼─────────────────────────────┘ │
│                                │                               │
│  ┌─────────────────────────────▼─────────────────────────────┐ │
│  │              Ingress Controller                           │ │
│  │              (nginx/traefik)                              │ │
│  │              External IP: 192.168.1.50                    │ │
│  └─────────────────────────────┬─────────────────────────────┘ │
│                                │                               │
└────────────────────────────────┼───────────────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   External Network      │
                    │   192.168.1.0/24        │
                    └────────────┬────────────┘
                                 │
                 ┌───────────────┼───────────────┐
                 │               │               │
        ┌────────▼────────┐ ┌───▼──────┐ ┌─────▼────────┐
        │  NFS Server     │ │  Native  │ │  Native      │
        │  192.168.1.100  │ │  Linux   │ │  Linux       │
        │  :2049          │ │  Host 1  │ │  Host 2      │
        └─────────────────┘ └──────────┘ └──────────────┘

NFS Traffic Flow:
- Kubernetes Pods → NFS Server: 10.244.x.x → 192.168.1.100:2049
- Native Linux → NFS Server: 192.168.1.x → 192.168.1.100:2049
```

## Storage Architecture

### NFS Volume Hierarchy

```
NFS Server: /exports/hypersdk/
│
├── vm-production-web-01/
│   ├── vm-production-web-01.ova          [5.2 GB]
│   ├── vm-production-web-01.vmdk         [4.8 GB]
│   ├── vm-production-web-01.vmx
│   ├── metadata.json
│   └── checksums.sha256
│
├── vm-production-db-01/
│   ├── vm-production-db-01.ova           [12.5 GB]
│   └── metadata.json
│
├── vm-development-test/
│   └── ...
│
└── .hypersdk/
    ├── locks/                             # Job coordination
    ├── cache/                             # Metadata cache
    └── logs/                              # Export logs
```

### Kubernetes Storage Stack

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
│  ┌────────────────┐         ┌───────────────────┐          │
│  │  HyperSDK Pod  │         │  hyper2kvm Pod    │          │
│  │  /exports      │         │  /imports         │          │
│  └────────┬───────┘         └─────────┬─────────┘          │
└───────────┼───────────────────────────┼─────────────────────┘
            │                           │
┌───────────▼───────────────────────────▼─────────────────────┐
│                  Kubernetes Volume Layer                     │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  PersistentVolumeClaim: hypersdk-exports             │   │
│  │  - Namespace: hypersdk                               │   │
│  │  - AccessMode: ReadWriteMany (RWX)                   │   │
│  │  - Capacity: 2Ti                                     │   │
│  │  - VolumeMode: Filesystem                            │   │
│  └────────────────────────┬─────────────────────────────┘   │
└───────────────────────────┼─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│              PersistentVolume Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  PersistentVolume: hypersdk-exports-nfs              │   │
│  │  - Type: NFS                                         │   │
│  │  - Server: 192.168.1.100                            │   │
│  │  - Path: /exports/hypersdk                          │   │
│  │  - ReclaimPolicy: Retain                            │   │
│  │  - MountOptions:                                    │   │
│  │    * nfsvers=4.1                                    │   │
│  │    * hard                                           │   │
│  │    * rsize=1048576 (1MB)                           │   │
│  │    * wsize=1048576 (1MB)                           │   │
│  └────────────────────────┬─────────────────────────────┘   │
└───────────────────────────┼─────────────────────────────────┘
                            │
                            │ NFS Protocol (TCP/IP)
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                    NFS Server Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  NFS Server Daemon (nfsd)                            │   │
│  │  - Protocol: NFSv4.1                                 │   │
│  │  - Port: 2049                                        │   │
│  │  - Threads: 32                                       │   │
│  │  - Export: /exports/hypersdk                         │   │
│  │  - Options: rw,sync,no_subtree_check,no_root_squash │   │
│  └────────────────────────┬─────────────────────────────┘   │
└───────────────────────────┼─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                  Physical Storage Layer                      │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Filesystem: ext4 / xfs                              │   │
│  │  Path: /exports/hypersdk                             │   │
│  │  Size: 2TB                                           │   │
│  │  Permissions: 755 (1000:1000)                        │   │
│  └────────────────────────┬─────────────────────────────┘   │
│                            │                                 │
│  ┌────────────────────────▼─────────────────────────────┐   │
│  │  Block Storage / SAN / Local Disk                    │   │
│  │  - RAID 10 / SSD / NVMe                             │   │
│  │  - High IOPS for VM exports                         │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Security Architecture

### Network Security

```
┌─────────────────────────────────────────────────────────────┐
│                 Kubernetes Network Policies                  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  NetworkPolicy: hypersdk-allow-nfs                   │   │
│  │                                                      │   │
│  │  Egress Rules:                                       │   │
│  │  - Allow to: 192.168.1.100:2049 (NFS)              │   │
│  │  - Protocol: TCP                                     │   │
│  │                                                      │   │
│  │  Ingress Rules:                                      │   │
│  │  - Allow from: Ingress Controller                    │   │
│  │  - Ports: 8080 (API), 8081 (Metrics)               │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Network Policy Enforcement
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                    NFS Server Firewall                       │
│                                                              │
│  iptables / firewalld Rules:                                │
│                                                              │
│  - Accept from: 10.244.0.0/16 (K8s Pod Network)            │
│  - Accept from: 192.168.1.0/24 (Trusted Network)           │
│  - Port: 2049 (NFS), 111 (RPC), 20048 (mountd)             │
│  - Drop all other traffic                                   │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    NFS Export Security                       │
│                                                              │
│  /etc/exports Configuration:                                │
│                                                              │
│  /exports/hypersdk 10.244.0.0/16(rw,sync,no_root_squash)   │
│  /exports/hypersdk 192.168.1.0/24(rw,sync,no_root_squash)  │
│                                                              │
│  Security Options:                                          │
│  - sec=sys (default) or sec=krb5p (Kerberos encryption)    │
│  - no_subtree_check (performance)                           │
│  - sync (data safety)                                       │
└─────────────────────────────────────────────────────────────┘
```

### Access Control

```
┌──────────────────────────────────────────────────────────────┐
│                  Kubernetes RBAC                              │
│                                                               │
│  ServiceAccount: hypersdk                                    │
│  ├─ Role: hypersdk-pvc-user                                 │
│  │  └─ Rules:                                               │
│  │     - persistentvolumeclaims: [get, list, create]       │
│  │                                                           │
│  └─ RoleBinding: hypersdk-pvc-binding                       │
│     └─ Binds: hypersdk ServiceAccount → hypersdk-pvc-user   │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│              Filesystem Permissions (NFS Server)              │
│                                                               │
│  /exports/hypersdk/                                          │
│  - Owner: 1000:1000 (hypersdk user)                         │
│  - Permissions: 755 (rwxr-xr-x)                              │
│  - SELinux: nfs_t context (if enabled)                       │
│                                                               │
│  Files created by HyperSDK:                                  │
│  - Owner: 1000:1000                                          │
│  - Permissions: 644 (rw-r--r--)                              │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                 Pod Security Context                          │
│                                                               │
│  securityContext:                                            │
│    runAsUser: 1000                                           │
│    runAsGroup: 1000                                          │
│    fsGroup: 1000                                             │
│    runAsNonRoot: true                                        │
│    capabilities:                                             │
│      drop: [ALL]                                             │
└──────────────────────────────────────────────────────────────┘
```

## High Availability Architecture

### HA NFS Server (Active-Passive)

```
┌─────────────────────────────────────────────────────────────┐
│                   Load Balancer / VIP                        │
│                   192.168.1.100:2049                         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ Failover via Pacemaker/Corosync
                         │
        ┌────────────────┴────────────────┐
        │                                 │
┌───────▼──────────┐           ┌──────────▼────────┐
│ NFS Primary      │           │ NFS Secondary     │
│ (Active)         │◄─────────►│ (Standby)         │
│ 192.168.1.101    │ DRBD/GFS2 │ 192.168.1.102     │
│                  │ Sync      │                   │
│ ┌──────────────┐ │           │ ┌──────────────┐  │
│ │ /exports/    │ │           │ │ /exports/    │  │
│ │ hypersdk/    │ │  Replicate│ │ hypersdk/    │  │
│ │              │─┼───────────┼─►│              │  │
│ └──────────────┘ │           │ └──────────────┘  │
└──────────────────┘           └───────────────────┘
```

### Cloud-Managed NFS (Highly Available)

```
┌─────────────────────────────────────────────────────────────┐
│                   Cloud Provider Region                      │
│                                                              │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐│
│  │ Availability   │  │ Availability   │  │ Availability   ││
│  │ Zone 1         │  │ Zone 2         │  │ Zone 3         ││
│  │                │  │                │  │                ││
│  │ ┌────────────┐ │  │ ┌────────────┐ │  │ ┌────────────┐││
│  │ │NFS Endpoint│ │  │ │NFS Endpoint│ │  │ │NFS Endpoint│││
│  │ └──────┬─────┘ │  │ └──────┬─────┘ │  │ └──────┬─────┘││
│  └────────┼────────┘  └────────┼────────┘  └────────┼──────┘│
│           └──────────────┬─────┴────────────────────┘       │
│                          │                                   │
│              ┌───────────▼──────────┐                       │
│              │  Managed NFS Service │                       │
│              │  - AWS EFS           │                       │
│              │  - Azure Files       │                       │
│              │  - GCP Filestore     │                       │
│              │                      │                       │
│              │  Features:           │                       │
│              │  - Auto-scaling      │                       │
│              │  - Auto-backup       │                       │
│              │  - Encryption        │                       │
│              │  - 99.99% SLA        │                       │
│              └──────────────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

## Monitoring Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Metrics Collection                        │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  HyperSDK Pod                                        │   │
│  │  ┌────────────────────────────────────────────────┐  │   │
│  │  │  Prometheus Metrics Endpoint (:8081/metrics)   │  │   │
│  │  │  - hypersdk_export_bytes_written               │  │   │
│  │  │  - hypersdk_nfs_operations_total               │  │   │
│  │  │  - hypersdk_nfs_latency_seconds                │  │   │
│  │  └────────────────────────────────────────────────┘  │   │
│  └────────────────────────┬─────────────────────────────┘   │
└───────────────────────────┼─────────────────────────────────┘
                            │ Scrape (15s interval)
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                    Prometheus Server                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Time Series Database                                │   │
│  │  - Retention: 15 days                                │   │
│  │  - Storage: 50GB                                     │   │
│  └────────────────────────┬─────────────────────────────┘   │
│                            │                                 │
│  ┌────────────────────────▼─────────────────────────────┐   │
│  │  Alert Rules                                         │   │
│  │  - NFS server unavailable                            │   │
│  │  - High latency (>500ms)                            │   │
│  │  - Disk usage >80%                                   │   │
│  └────────────────────────┬─────────────────────────────┘   │
└───────────────────────────┼─────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌───────▼────────┐  ┌──────▼──────────┐
│  Grafana       │  │  Alertmanager  │  │  Alert Webhook  │
│  Dashboards    │  │  Notifications │  │  (Slack/Email)  │
└────────────────┘  └────────────────┘  └─────────────────┘
```

## Deployment Sequence Diagram

```
User    K8s API   Controller   HyperSDK   NFS    Native
 │         │          │           │        │     Linux
 │         │          │           │        │       │
 │─────────┼──────────┼───────────┼────────┼───────┼─── Deploy Phase
 │         │          │           │        │       │
 │ kubectl │          │           │        │       │
 │ apply   │          │           │        │       │
 ├────────>│          │           │        │       │
 │         │          │           │        │       │
 │         │ Create   │           │        │       │
 │         │ PV/PVC   │           │        │       │
 │         ├─────────>│           │        │       │
 │         │          │           │        │       │
 │         │          │ Mount NFS │        │       │
 │         │          ├──────────────────>│       │
 │         │          │           │  NFS   │       │
 │         │          │<──────────────────┤       │
 │         │          │           │  Mount │       │
 │         │          │           │   OK   │       │
 │         │          │           │        │       │
 │         │          │ Create Pod│        │       │
 │         │          ├──────────>│        │       │
 │         │          │           │        │       │
 │         │          │           │ Mount  │       │
 │         │          │           │ /exports       │
 │         │          │           ├───────>│       │
 │         │          │           │<───────┤       │
 │         │          │           │   OK   │       │
 │         │          │<──────────┤        │       │
 │         │<─────────┤  Running  │        │       │
 │<────────┤          │           │        │       │
 │  Ready  │          │           │        │       │
 │         │          │           │        │       │
 │─────────┼──────────┼───────────┼────────┼───────┼─── Export Phase
 │         │          │           │        │       │
 │ Submit  │          │           │        │       │
 │ Export  │          │           │        │       │
 │ Job     │          │           │        │       │
 ├────────────────────────────────>│       │       │
 │         │          │           │        │       │
 │         │          │           │ Export │       │
 │         │          │           │ VM to  │       │
 │         │          │           │ /exports       │
 │         │          │           ├───────>│       │
 │         │          │           │  Write │       │
 │         │          │           │  Data  │       │
 │         │          │           ├───────>│       │
 │         │          │           │<───────┤       │
 │         │          │           │   OK   │       │
 │         │          │           │        │       │
 │─────────┼──────────┼───────────┼────────┼───────┼─── Import Phase
 │         │          │           │        │       │
 │         │          │           │        │       │ mount
 │         │          │           │        │<──────┤ NFS
 │         │          │           │        ├──────>│
 │         │          │           │        │  OK   │
 │         │          │           │        │       │
 │         │          │           │        │       │ hyper2kvm
 │         │          │           │        │       │ convert
 │         │          │           │        │<──────┤ Read
 │         │          │           │        ├──────>│ VM data
 │         │          │           │        │       │
 │         │          │           │        │       │ Convert
 │         │          │           │        │       │ to qcow2
 │         │          │           │        │       │
```

## Multi-Architecture Support

### Container Image Architecture

HyperSDK supports multiple CPU architectures:

```
Container Images: ghcr.io/ssahani/hypersdk-*:latest

├── linux/amd64 (x86_64)
│   - Size: ~72 MB (hypervisord)
│   - Use: Intel/AMD servers, most cloud providers
│
└── linux/arm64 (aarch64)
    - Size: ~68 MB (hypervisord)
    - Use: ARM servers, Apple Silicon, AWS Graviton
```

### NFS Client Support

NFS protocol is architecture-independent and works on both:

```
✓ x86_64 (amd64)
  - Traditional Intel/AMD servers
  - Cloud instances (AWS, Azure, GCP)
  - Most Kubernetes clusters

✓ ARM64 (aarch64)
  - AWS Graviton instances
  - Raspberry Pi 4/5 (development)
  - Apple Silicon (development)
  - Ampere Altra servers

✓ Other architectures (with manual build)
  - ppc64le (IBM POWER)
  - s390x (IBM Z)
```

## Performance Characteristics

### Throughput Benchmarks

```
Environment          Read Speed    Write Speed   Latency
────────────────────────────────────────────────────────
K8s Pod → NFS       850 MB/s      780 MB/s      ~2ms
Native Linux → NFS  920 MB/s      850 MB/s      ~1ms
K8s Pod → Local     2.1 GB/s      1.8 GB/s      <1ms

Network: 10 Gigabit Ethernet
NFS Version: 4.1
Mount Options: rsize=1048576, wsize=1048576
```

### Scaling Limits

```
Component               Max Supported    Recommended
──────────────────────────────────────────────────────
Concurrent HyperSDK Pods    100          3-5
Concurrent NFS Clients      1000         50
NFS Storage Size            16 TB        2-8 TB
Concurrent Exports          50           5-10
Export File Size            2 TB         50-500 GB
```

## Summary

This architecture provides:

✅ **High Availability** - Multiple HyperSDK pods, HA NFS options
✅ **Cross-Environment** - Shared storage between K8s and native Linux
✅ **Scalability** - ReadWriteMany allows horizontal scaling
✅ **Performance** - Optimized NFS mount options, 1MB chunk sizes
✅ **Security** - Network policies, RBAC, filesystem permissions
✅ **Monitoring** - Full observability with Prometheus/Grafana
✅ **Multi-Architecture** - Support for amd64 and arm64

## References

- [NFS Shared Storage Tutorial](../tutorials/nfs-shared-storage.md)
- [Kubernetes Deployment Guide](../../deployments/kubernetes/README.md)
- [Configuration Reference](../reference/configuration-reference.md)
