# Performance Tuning Guide

## Table of Contents

1. [Export Performance](#export-performance)
2. [Network Optimization](#network-optimization)
3. [Disk I/O Optimization](#disk-io-optimization)
4. [Memory Management](#memory-management)
5. [CPU Optimization](#cpu-optimization)
6. [Database Performance](#database-performance)
7. [Benchmarking](#benchmarking)

## Export Performance

### Parallel Downloads

```yaml
export:
  parallel_downloads: 8  # Default is 4
  download_workers: 8
```

```bash
# CLI override
./hyperexport -vm myvm -parallel 8
```

### Connection Pooling

```yaml
connection_pool:
  max_connections: 10  # More concurrent vSphere connections
  min_connections: 2
  idle_timeout: "10m"
  max_lifetime: "2h"
```

### Compression vs Speed

```yaml
export:
  compress: true
  compression_level: 1  # Fastest (default is 6)
  # 1 = fastest, 9 = best compression
```

```bash
# For maximum speed, disable compression
./hyperexport -vm myvm -compress=false

# For maximum compression
./hyperexport -vm myvm -compress -compression-level 9
```

### Buffer Size

```yaml
export:
  buffer_size: 16777216  # 16MB buffer (default 8MB)
  read_buffer_size: 8388608  # 8MB
  write_buffer_size: 8388608  # 8MB
```

### Concurrent Jobs

```yaml
daemon:
  max_concurrent_jobs: 20  # Process more jobs simultaneously
  download_workers: 8
```

### Skip Verification

```bash
# Skip checksum verification for speed (not recommended)
./hyperexport -vm myvm -verify=false
```

## Network Optimization

### Network Bandwidth

```yaml
export:
  bandwidth_limit: 0  # Unlimited (default)
  # Or limit: "100M" for 100 Mbps
```

### TCP Tuning

```bash
# Increase TCP buffer sizes
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.wmem_max=134217728
sudo sysctl -w net.ipv4.tcp_rmem='4096 87380 67108864'
sudo sysctl -w net.ipv4.tcp_wmem='4096 65536 67108864'
sudo sysctl -w net.ipv4.tcp_congestion_control=bbr

# Make permanent
cat | sudo tee -a /etc/sysctl.conf <<EOF
net.core.rmem_max=134217728
net.core.wmem_max=134217728
net.ipv4.tcp_rmem=4096 87380 67108864
net.ipv4.tcp_wmem=4096 65536 67108864
net.ipv4.tcp_congestion_control=bbr
EOF

sudo sysctl -p
```

### Network Interface Tuning

```bash
# Increase ring buffer
sudo ethtool -G eth0 rx 4096 tx 4096

# Enable TCP offloading
sudo ethtool -K eth0 tso on
sudo ethtool -K eth0 gso on
sudo ethtool -K eth0 gro on

# Increase queue length
sudo ifconfig eth0 txqueuelen 10000
```

### MTU Optimization

```bash
# Test optimal MTU
ping -M do -s 8972 vcenter.example.com

# Set jumbo frames (if supported)
sudo ip link set eth0 mtu 9000
```

## Disk I/O Optimization

### Filesystem Selection

```bash
# XFS for large files (recommended)
sudo mkfs.xfs -f -l size=128m -d agcount=32 /dev/sdb
sudo mount -o noatime,nodiratime,logbufs=8,logbsize=256k /dev/sdb /exports

# Or ext4 with optimizations
sudo mkfs.ext4 -E lazy_itable_init=0,lazy_journal_init=0 /dev/sdb
sudo mount -o noatime,nodiratime,data=writeback,barrier=0,commit=60 /dev/sdb /exports
```

### I/O Scheduler

```bash
# Use none (noop) for SSD
echo none | sudo tee /sys/block/sdb/queue/scheduler

# Or deadline for HDD
echo deadline | sudo tee /sys/block/sdb/queue/scheduler

# Make permanent
cat | sudo tee /etc/udev/rules.d/60-scheduler.rules <<EOF
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/rotational}=="0", ATTR{queue/scheduler}="none"
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/rotational}=="1", ATTR{queue/scheduler}="deadline"
EOF
```

### Read-Ahead

```bash
# Increase read-ahead for sequential reads
sudo blockdev --setra 8192 /dev/sdb

# Make permanent
echo 'ACTION=="add|change", KERNEL=="sdb", ATTR{bdi/read_ahead_kb}="8192"' | \
  sudo tee /etc/udev/rules.d/60-readahead.rules
```

### Direct I/O

```yaml
export:
  direct_io: true  # Bypass page cache
```

### RAID Configuration

```bash
# RAID 0 for maximum speed (no redundancy)
sudo mdadm --create /dev/md0 --level=0 --raid-devices=4 \
  /dev/sd[bcde]

# RAID 10 for speed + redundancy
sudo mdadm --create /dev/md0 --level=10 --raid-devices=4 \
  /dev/sd[bcde]

# Format and mount
sudo mkfs.xfs -f /dev/md0
sudo mount -o noatime /dev/md0 /exports
```

## Memory Management

### Increase Memory

```yaml
daemon:
  max_memory: "16GB"  # Limit daemon memory usage
```

### Page Cache

```bash
# Drop cache before large exports
sudo sync
sudo echo 3 | sudo tee /proc/sys/vm/drop_caches

# Adjust swappiness (prefer RAM over swap)
sudo sysctl -w vm.swappiness=10

# Make permanent
echo "vm.swappiness=10" | sudo tee -a /etc/sysctl.conf
```

### Huge Pages

```bash
# Enable transparent huge pages
echo always | sudo tee /sys/kernel/mm/transparent_hugepage/enabled

# Or configure static huge pages
sudo sysctl -w vm.nr_hugepages=1024
echo "vm.nr_hugepages=1024" | sudo tee -a /etc/sysctl.conf
```

## CPU Optimization

### CPU Affinity

```bash
# Pin daemon to specific CPUs
sudo systemctl edit hypervisord

[Service]
CPUAffinity=0-7  # Use CPUs 0-7
```

### CPU Governor

```bash
# Use performance governor
sudo cpupower frequency-set -g performance

# Or on-demand
sudo cpupower frequency-set -g ondemand
```

### NUMA Optimization

```bash
# Check NUMA layout
numactl --hardware

# Run on specific NUMA node
numactl --cpunodebind=0 --membind=0 hypervisord
```

### Goroutine Pool

```yaml
daemon:
  max_goroutines: 10000  # Limit concurrent goroutines
  goroutine_pool_size: 1000
```

## Database Performance

### SQLite Optimization

```yaml
database:
  type: "sqlite"
  path: "/var/lib/hypersdk/hypersdk.db"
  pragmas:
    journal_mode: "WAL"  # Write-Ahead Logging
    synchronous: "NORMAL"
    cache_size: -64000  # 64MB cache
    mmap_size: 268435456  # 256MB mmap
    page_size: 4096
    temp_store: "MEMORY"
```

### PostgreSQL Optimization

```yaml
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  database: "hypersdk"
  max_connections: 100
  max_idle_connections: 10
  connection_lifetime: "1h"
```

```sql
-- PostgreSQL tuning
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;
ALTER SYSTEM SET work_mem = '10MB';
ALTER SYSTEM SET min_wal_size = '1GB';
ALTER SYSTEM SET max_wal_size = '4GB';

-- Restart PostgreSQL
SELECT pg_reload_conf();
```

## Benchmarking

### Export Benchmark

```bash
#!/bin/bash
# benchmark-export.sh

VM="/datacenter/vm/benchmark-vm"
ITERATIONS=5

echo "Running export benchmark..."
for i in $(seq 1 $ITERATIONS); do
  echo "Iteration $i..."

  START=$(date +%s)

  ./hyperexport -vm "$VM" \
    -output /tmp/benchmark-$i \
    -format ova \
    -compress \
    -verify

  END=$(date +%s)
  DURATION=$((END - START))

  SIZE=$(du -sh /tmp/benchmark-$i | cut -f1)

  echo "Duration: ${DURATION}s, Size: $SIZE"

  rm -rf /tmp/benchmark-$i
done
```

### Network Benchmark

```bash
# iperf3 test
iperf3 -c vcenter.example.com -t 60 -P 4

# Test vSphere network
time govc vm.info -vm.path="$VM" > /dev/null
```

### Disk Benchmark

```bash
# Sequential write
dd if=/dev/zero of=/exports/test bs=1M count=10240 oflag=direct

# Sequential read
dd if=/exports/test of=/dev/null bs=1M iflag=direct

# fio comprehensive test
fio --name=sequential-write --ioengine=libaio --direct=1 --bs=1M \
  --iodepth=64 --rw=write --size=10G --filename=/exports/test

fio --name=sequential-read --ioengine=libaio --direct=1 --bs=1M \
  --iodepth=64 --rw=read --size=10G --filename=/exports/test

fio --name=random-read --ioengine=libaio --direct=1 --bs=4k \
  --iodepth=256 --rw=randread --size=10G --filename=/exports/test
```

### API Benchmark

```bash
# Apache Bench
ab -n 1000 -c 10 http://localhost:8080/health

# wrk
wrk -t 4 -c 100 -d 30s http://localhost:8080/status

# Custom benchmark
for i in {1..100}; do
  time curl -s http://localhost:8080/health > /dev/null
done
```

## Performance Monitoring

### System Monitoring

```bash
# Real-time system stats
htop
iotop
nethogs
nmon

# CPU usage
mpstat 1

# Disk I/O
iostat -xz 1

# Network
iftop -i eth0
```

### Application Monitoring

```bash
# Daemon metrics
curl http://localhost:8080/metrics

# Profiling
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

### Prometheus Metrics

```yaml
metrics:
  enabled: true
  prometheus_enabled: true
  prometheus_path: "/metrics"
```

```bash
# Scrape metrics
curl http://localhost:8080/metrics

# Example metrics:
# hypersdk_export_duration_seconds
# hypersdk_export_bytes_total
# hypersdk_jobs_active
# hypersdk_connection_pool_size
```

## Optimization Checklist

- [ ] Parallel downloads configured (8+)
- [ ] Connection pooling enabled
- [ ] TCP buffer sizes increased
- [ ] MTU optimized for network
- [ ] Filesystem tuned (XFS with noatime)
- [ ] I/O scheduler optimized
- [ ] Read-ahead configured
- [ ] Page cache settings tuned
- [ ] CPU governor set to performance
- [ ] Database optimized (WAL mode for SQLite)
- [ ] Monitoring enabled
- [ ] Benchmarks run and baselined

## Performance Targets

### Export Performance

- **Small VM (< 50GB)**: 5-10 minutes
- **Medium VM (50-200GB)**: 15-30 minutes
- **Large VM (200GB-1TB)**: 1-3 hours

### API Performance

- **Health check**: < 10ms
- **Job submission**: < 100ms
- **Job query**: < 50ms
- **WebSocket latency**: < 100ms

### Resource Usage

- **Memory**: < 2GB for daemon
- **CPU**: < 50% during exports
- **Disk I/O**: Limited by network bandwidth
- **Network**: Saturate available bandwidth

## See Also

- [Configuration Reference](configuration-reference.md)
- [Troubleshooting Guide](troubleshooting-guide.md)
- [System Requirements](installation-guide.md#system-requirements)
