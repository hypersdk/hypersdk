# HyperSDK Operations Runbook

Production operations guide for supporting HyperSDK in Kubernetes environments.

## Quick Reference

### Common Commands

```bash
# Check deployment health
kubectl get pods -n hypersdk
kubectl get deploy -n hypersdk hypersdk

# View logs
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk --tail=100 -f

# Check API health
kubectl exec -n hypersdk deploy/hypersdk -- curl http://localhost:8080/health

# View metrics
kubectl port-forward -n hypersdk svc/hypersdk 8081:8081
curl http://localhost:8081/metrics

# Scale deployment
kubectl scale -n hypersdk deploy/hypersdk --replicas=3

# Restart deployment
kubectl rollout restart -n hypersdk deploy/hypersdk

# Check resource usage
kubectl top pods -n hypersdk
```

### Emergency Contacts

- **On-call Engineer**: [PagerDuty/Opsgenie rotation]
- **Platform Team**: platform-team@example.com
- **Database Admin**: dba-team@example.com
- **Network Team**: network-ops@example.com

## Health Checks

### Service Health

```bash
# API health endpoint
curl http://hypersdk.example.com/health

# Expected response:
{
  "status": "healthy",
  "version": "0.2.0",
  "uptime": "72h15m32s"
}
```

### Pod Health

```bash
# Check pod status
kubectl get pods -n hypersdk

# Expected output:
NAME                       READY   STATUS    RESTARTS   AGE
hypersdk-7d6f9c8b5-abcd1   1/1     Running   0          2d
hypersdk-7d6f9c8b5-abcd2   1/1     Running   0          2d
hypersdk-7d6f9c8b5-abcd3   1/1     Running   0          2d
```

### Service Availability

```bash
# Check service endpoints
kubectl get svc -n hypersdk
kubectl get endpoints -n hypersdk

# Test service connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://hypersdk.hypersdk:8080/health
```

## Common Issues and Solutions

### Issue 1: Pods Crashing (CrashLoopBackOff)

**Symptoms:**
- Pods in CrashLoopBackOff state
- Deployment not ready

**Diagnosis:**
```bash
# Check pod status
kubectl get pods -n hypersdk

# View pod logs
kubectl logs -n hypersdk <pod-name> --previous

# Describe pod for events
kubectl describe pod -n hypersdk <pod-name>
```

**Common Causes & Solutions:**

#### A. Database Connection Failure

```bash
# Check database credentials
kubectl get secret -n hypersdk

# Verify database connectivity
kubectl exec -n hypersdk deploy/hypersdk -- \
  nc -zv <database-host> <database-port>
```

**Solution:**
```bash
# Update database configuration
kubectl edit configmap -n hypersdk hypersdk

# Restart pods
kubectl rollout restart -n hypersdk deploy/hypersdk
```

#### B. Out of Memory

```bash
# Check memory usage
kubectl top pods -n hypersdk

# View memory limits
kubectl describe pod -n hypersdk <pod-name> | grep -A 5 "Limits:"
```

**Solution:**
```bash
# Increase memory limits
helm upgrade hypersdk hypersdk/hypersdk \
  --reuse-values \
  --set resources.limits.memory=4Gi \
  --namespace hypersdk
```

#### C. Invalid Configuration

```bash
# Check ConfigMap
kubectl get configmap -n hypersdk hypersdk -o yaml

# Validate configuration
kubectl exec -n hypersdk deploy/hypersdk -- \
  /usr/local/bin/hypervisord --validate-config
```

**Solution:**
```bash
# Fix configuration
kubectl edit configmap -n hypersdk hypersdk

# Apply changes
kubectl rollout restart -n hypersdk deploy/hypersdk
```

### Issue 2: High CPU/Memory Usage

**Symptoms:**
- Slow API responses
- Pods being OOMKilled
- High resource utilization

**Diagnosis:**
```bash
# Check resource usage
kubectl top pods -n hypersdk

# Check metrics
kubectl port-forward -n hypersdk svc/hypersdk 8081:8081
curl http://localhost:8081/metrics | grep process_

# Check active jobs
curl http://hypersdk.example.com/api/v1/status
```

**Solutions:**

#### A. Too Many Concurrent Jobs

```bash
# Reduce concurrent workers
kubectl set env -n hypersdk deploy/hypersdk \
  DOWNLOAD_WORKERS=2

# Or via ConfigMap
kubectl edit configmap -n hypersdk hypersdk
# Update: download_workers: 2
```

#### B. Memory Leak

```bash
# Check memory trend
kubectl top pods -n hypersdk --use-protocol-buffers

# Restart pods as workaround
kubectl rollout restart -n hypersdk deploy/hypersdk

# File bug report with heap dump
kubectl exec -n hypersdk <pod-name> -- \
  curl -X POST http://localhost:8080/debug/pprof/heap > heap.prof
```

#### C. Scale Out

```bash
# Horizontal scaling
helm upgrade hypersdk hypersdk/hypersdk \
  --reuse-values \
  --set replicaCount=5 \
  --namespace hypersdk

# Enable HPA
helm upgrade hypersdk hypersdk/hypersdk \
  --reuse-values \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=3 \
  --set autoscaling.maxReplicas=10 \
  --namespace hypersdk
```

### Issue 3: Storage Full (PVC)

**Symptoms:**
- Export jobs failing
- Database write errors
- "No space left on device" errors

**Diagnosis:**
```bash
# Check PVC usage
kubectl exec -n hypersdk deploy/hypersdk -- df -h /data /exports

# Check PVC status
kubectl get pvc -n hypersdk
kubectl describe pvc -n hypersdk
```

**Solutions:**

#### A. Clean Old Exports

```bash
# List old exports
kubectl exec -n hypersdk deploy/hypersdk -- \
  find /exports -type f -mtime +30

# Remove old exports (be careful!)
kubectl exec -n hypersdk deploy/hypersdk -- \
  find /exports -type f -mtime +30 -delete
```

#### B. Expand PVC

```bash
# Check if storage class supports expansion
kubectl get sc <storage-class> -o yaml | grep allowVolumeExpansion

# Expand PVC
kubectl edit pvc -n hypersdk hypersdk-exports
# Update: spec.resources.requests.storage: 2Ti

# Wait for expansion
kubectl get pvc -n hypersdk -w
```

#### C. Add New PVC

```bash
# Create additional PVC
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-exports-2
  namespace: hypersdk
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Ti
  storageClassName: fast-ssd
EOF

# Update deployment to use both PVCs
# (requires Helm chart modification)
```

### Issue 4: Network Connectivity Issues

**Symptoms:**
- Cannot connect to vSphere/cloud providers
- Timeouts during exports
- Connection refused errors

**Diagnosis:**
```bash
# Test connectivity from pod
kubectl exec -n hypersdk deploy/hypersdk -- \
  nc -zv vcenter.example.com 443

# Check DNS resolution
kubectl exec -n hypersdk deploy/hypersdk -- \
  nslookup vcenter.example.com

# Check network policies
kubectl get networkpolicy -n hypersdk
```

**Solutions:**

#### A. DNS Issues

```bash
# Check DNS config
kubectl exec -n hypersdk deploy/hypersdk -- cat /etc/resolv.conf

# Test with specific nameserver
kubectl exec -n hypersdk deploy/hypersdk -- \
  nslookup vcenter.example.com 8.8.8.8
```

#### B. Network Policy Blocking

```bash
# Check network policies
kubectl describe networkpolicy -n hypersdk

# Temporarily disable (for testing only!)
kubectl delete networkpolicy -n hypersdk --all

# Re-enable after testing
helm upgrade hypersdk hypersdk/hypersdk \
  --reuse-values \
  --set networkPolicy.enabled=true \
  --namespace hypersdk
```

#### C. Firewall Rules

```bash
# Verify firewall allows traffic
# (Check with network team)

# Test from worker node directly
ssh worker-node
curl -v https://vcenter.example.com
```

### Issue 5: Certificate/TLS Issues

**Symptoms:**
- TLS handshake failures
- Certificate verification errors
- "x509: certificate signed by unknown authority"

**Diagnosis:**
```bash
# Test TLS connection
kubectl exec -n hypersdk deploy/hypersdk -- \
  openssl s_client -connect vcenter.example.com:443

# Check certificate validity
kubectl exec -n hypersdk deploy/hypersdk -- \
  curl -v https://vcenter.example.com 2>&1 | grep -i certificate
```

**Solutions:**

#### A. Add CA Certificate

```bash
# Create ConfigMap with CA cert
kubectl create configmap -n hypersdk ca-certificates \
  --from-file=ca.crt=/path/to/ca.crt

# Update deployment to mount CA cert
helm upgrade hypersdk hypersdk/hypersdk \
  --reuse-values \
  --set-file extraCACerts=ca.crt \
  --namespace hypersdk
```

#### B. Skip Verification (Insecure - Development Only!)

```bash
# Update vSphere credentials
kubectl edit secret -n hypersdk vsphere-credentials
# Add: insecure: "1"

# Restart pods
kubectl rollout restart -n hypersdk deploy/hypersdk
```

## Backup and Recovery

### Database Backup

```bash
# Backup SQLite database
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db ".backup /data/backup.db"

# Copy backup to local machine
kubectl cp hypersdk/<pod-name>:/data/backup.db \
  ./hypersdk-backup-$(date +%Y%m%d).db
```

### PVC Snapshots

```bash
# Create VolumeSnapshot (if supported)
kubectl apply -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: hypersdk-data-snapshot
  namespace: hypersdk
spec:
  volumeSnapshotClassName: csi-snapclass
  source:
    persistentVolumeClaimName: hypersdk-data
EOF

# List snapshots
kubectl get volumesnapshot -n hypersdk
```

### Disaster Recovery

```bash
# 1. Backup Helm values
helm get values -n hypersdk hypersdk > hypersdk-values-backup.yaml

# 2. Backup secrets
kubectl get secret -n hypersdk -o yaml > secrets-backup.yaml

# 3. Backup database
# (See Database Backup above)

# 4. Restore in new cluster
helm install hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --create-namespace \
  --values hypersdk-values-backup.yaml

# 5. Restore secrets
kubectl apply -f secrets-backup.yaml

# 6. Restore database
kubectl cp hypersdk-backup.db hypersdk/<pod-name>:/data/hypersdk.db
kubectl rollout restart -n hypersdk deploy/hypersdk
```

## Performance Tuning

### Optimize for High Throughput

```yaml
# values.yaml
replicaCount: 5

config:
  downloadWorkers: 5
  chunkSize: 50MB

resources:
  requests:
    memory: 2Gi
    cpu: 1000m
  limits:
    memory: 8Gi
    cpu: 4000m

autoscaling:
  enabled: true
  minReplicas: 5
  maxReplicas: 20
  targetCPUUtilizationPercentage: 70
```

### Optimize for Low Latency

```yaml
# values.yaml
replicaCount: 3

config:
  downloadWorkers: 2
  chunkSize: 10MB

resources:
  requests:
    memory: 1Gi
    cpu: 500m
  limits:
    memory: 4Gi
    cpu: 2000m

# Use fast SSD storage
persistence:
  data:
    storageClass: premium-ssd
```

## Monitoring and Alerts

### Key Metrics to Watch

```bash
# Request rate
rate(hypersdk_http_requests_total[5m])

# Error rate
rate(hypersdk_http_requests_total{status=~"5.."}[5m])

# Active jobs
hypersdk_active_jobs

# Job completion rate
rate(hypersdk_jobs_completed_total[5m])

# Memory usage
container_memory_usage_bytes{pod=~"hypersdk-.*"}

# CPU usage
rate(container_cpu_usage_seconds_total{pod=~"hypersdk-.*"}[5m])
```

### Alert Thresholds

| Alert | Threshold | Severity |
|-------|-----------|----------|
| Pod Down | Any pod down > 2min | Critical |
| High Error Rate | Error rate > 5% | Warning |
| High Memory | Memory > 90% | Warning |
| PVC Full | Storage > 85% | Warning |
| Job Failures | Failure rate > 10% | Critical |

## Maintenance Windows

### Rolling Update Procedure

```bash
# 1. Announce maintenance window
# Send notification to users

# 2. Scale up for redundancy
kubectl scale -n hypersdk deploy/hypersdk --replicas=6

# 3. Perform upgrade
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --values production-values.yaml \
  --wait \
  --timeout 10m

# 4. Verify health
kubectl rollout status -n hypersdk deploy/hypersdk
curl http://hypersdk.example.com/health

# 5. Monitor for issues (15 minutes)
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk -f

# 6. Scale back to normal
kubectl scale -n hypersdk deploy/hypersdk --replicas=3
```

### Emergency Rollback

```bash
# Quick rollback to previous version
helm rollback hypersdk -n hypersdk

# Or specify revision
helm history -n hypersdk hypersdk
helm rollback hypersdk 5 -n hypersdk
```

## Security

### Rotate Credentials

```bash
# 1. Create new secret
kubectl create secret generic vsphere-credentials-new \
  --from-literal=url=https://vcenter.example.com/sdk \
  --from-literal=username=admin \
  --from-literal=password=NEW_PASSWORD \
  --namespace hypersdk

# 2. Update deployment to use new secret
helm upgrade hypersdk hypersdk/hypersdk \
  --reuse-values \
  --set credentials.vsphere.existingSecret=vsphere-credentials-new \
  --namespace hypersdk

# 3. Verify functionality
# Test exports with new credentials

# 4. Delete old secret
kubectl delete secret -n hypersdk vsphere-credentials
```

### Security Scanning

```bash
# Scan container image for vulnerabilities
trivy image ghcr.io/ssahani/hypersdk-hypervisord:0.2.0

# Check for security issues in Kubernetes manifests
kubesec scan deployment.yaml

# Audit RBAC permissions
kubectl auth can-i --list --as=system:serviceaccount:hypersdk:hypersdk
```

## Capacity Planning

### Resource Usage Trends

```promql
# Average memory usage over 7 days
avg_over_time(container_memory_usage_bytes{pod=~"hypersdk-.*"}[7d])

# Peak CPU usage over 7 days
max_over_time(rate(container_cpu_usage_seconds_total{pod=~"hypersdk-.*"}[5m])[7d:1h])

# Storage growth rate
rate(kubelet_volume_stats_used_bytes{persistentvolumeclaim="hypersdk-exports"}[7d])
```

### Scaling Recommendations

| Metric | Threshold | Action |
|--------|-----------|--------|
| CPU > 80% | Sustained > 1h | Add replicas or increase CPU limits |
| Memory > 85% | Sustained > 30m | Increase memory limits |
| Storage > 80% | - | Expand PVC or clean old data |
| Job Queue > 100 | Sustained > 15m | Scale out workers |

## On-Call Playbook

### Severity 1: Service Down

**Response Time:** Immediate

**Steps:**
1. Acknowledge alert
2. Check pod status: `kubectl get pods -n hypersdk`
3. Check recent changes: `helm history -n hypersdk hypersdk`
4. Review logs for errors: `kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk --tail=100`
5. If recent deploy: Rollback immediately
6. If infrastructure: Check node/cluster health
7. Escalate to platform team if needed
8. Post-incident review required

### Severity 2: Degraded Performance

**Response Time:** 15 minutes

**Steps:**
1. Check resource usage: `kubectl top pods -n hypersdk`
2. Check metrics for anomalies
3. Review recent job activity
4. Scale if needed: `kubectl scale deploy/hypersdk --replicas=5 -n hypersdk`
5. Monitor for improvement
6. Investigate root cause

### Severity 3: Warning

**Response Time:** Next business day

**Steps:**
1. Create ticket
2. Investigate during business hours
3. Plan remediation
4. Schedule maintenance if needed

## Contact and Escalation

1. **First Response:** On-call engineer (via PagerDuty)
2. **Escalation (15min):** Senior engineer
3. **Escalation (30min):** Engineering manager
4. **Escalation (1h):** CTO / VP Engineering

## Documentation Links

- [Architecture Documentation](../../docs/architecture/)
- [API Reference](../../docs/api/)
- [Troubleshooting Guide](../../docs/reference/troubleshooting-guide.md)
- [Monitoring Dashboards](http://grafana.example.com/d/hypersdk)
- [Alert Rules](http://prometheus.example.com/alerts)
