# Troubleshooting FAQ - Common Issues & Solutions

Comprehensive FAQ covering common issues across deployment, operations, security, and performance.

## Quick Links

- [Installation Issues](#installation-issues)
- [Pod/Container Issues](#podcontainer-issues)
- [Storage Issues](#storage-issues)
- [Network Issues](#network-issues)
- [Security Issues](#security-issues)
- [Performance Issues](#performance-issues)
- [Backup/Recovery Issues](#backuprecovery-issues)
- [GitOps Issues](#gitops-issues)
- [Cost/Resource Issues](#costresource-issues)
- [Monitoring Issues](#monitoring-issues)

## Installation Issues

### Q: Helm install fails with "no matches for kind"

**Error:**
```
Error: unable to build kubernetes objects from release manifest:
unable to recognize "": no matches for kind "ServiceMonitor" in version "monitoring.coreos.com/v1"
```

**Cause:** Prometheus Operator CRDs not installed

**Solution:**
```bash
# Install Prometheus Operator CRDs first
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_prometheusrules.yaml

# Or disable ServiceMonitor
helm install hypersdk hypersdk/hypersdk --set monitoring.serviceMonitor.enabled=false
```

### Q: Helm install fails with "forbidden: violates PodSecurity"

**Error:**
```
Error: admission webhook "validation.gatekeeper.sh" denied the request:
[denied by autogen-pod-security-policy]
```

**Cause:** Namespace has Pod Security Standards enforcement

**Solution:**
```bash
# Option 1: Relax namespace policy
kubectl label namespace hypersdk pod-security.kubernetes.io/enforce=baseline

# Option 2: Use restricted-compliant values
helm install hypersdk hypersdk/hypersdk -f values-restricted.yaml
```

### Q: Chart not found in OCI registry

**Error:**
```
Error: failed to download "oci://ghcr.io/ssahani/charts/hypersdk"
```

**Cause:** Authentication required or chart doesn't exist

**Solution:**
```bash
# Login to registry
echo $GITHUB_TOKEN | helm registry login ghcr.io -u $GITHUB_USER --password-stdin

# Verify chart exists
helm show chart oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0

# Check available versions
helm search repo hypersdk --versions
```

### Q: Values file not being applied

**Error:** Configuration not taking effect

**Solution:**
```bash
# Verify values are applied
helm get values hypersdk -n hypersdk

# Check for YAML syntax errors
yamllint values.yaml

# Use -f flag explicitly
helm upgrade hypersdk hypersdk/hypersdk -f custom-values.yaml

# Verify rendered templates
helm template hypersdk hypersdk/hypersdk -f custom-values.yaml
```

## Pod/Container Issues

### Q: Pods stuck in Pending state

**Symptoms:**
```
NAME                        READY   STATUS    RESTARTS   AGE
hypersdk-7d6f9c8b5-abcd1    0/1     Pending   0          5m
```

**Diagnosis:**
```bash
# Check pod events
kubectl describe pod -n hypersdk hypersdk-7d6f9c8b5-abcd1

# Common causes:
# 1. Insufficient resources
# 2. PVC not bound
# 3. Node selector not matching
# 4. Taints/tolerations mismatch
```

**Solutions:**

**Insufficient Resources:**
```bash
# Check node capacity
kubectl describe nodes | grep -A 5 "Allocated resources"

# Reduce resource requests
helm upgrade hypersdk hypersdk/hypersdk \
  --set resources.requests.memory=256Mi \
  --set resources.requests.cpu=100m
```

**PVC Not Bound:**
```bash
# Check PVC status
kubectl get pvc -n hypersdk

# Check storage class
kubectl get storageclass

# Create if missing
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
EOF
```

### Q: CrashLoopBackOff - Database error

**Error in logs:**
```
unable to open database file: /data/hypersdk.db: permission denied
```

**Cause:** Permission issues with volume mount

**Solution:**
```bash
# Check security context
kubectl get pod -n hypersdk -o yaml | grep -A 10 securityContext

# Ensure fsGroup is set
helm upgrade hypersdk hypersdk/hypersdk \
  --set podSecurityContext.fsGroup=1000

# Or fix PVC permissions manually
kubectl exec -n hypersdk deploy/hypersdk -- chown -R 1000:1000 /data
```

### Q: CrashLoopBackOff - Invalid config

**Error in logs:**
```
Error: invalid configuration: field 'download_workers' cannot be negative
```

**Cause:** Invalid configuration in ConfigMap

**Solution:**
```bash
# Check current config
kubectl get configmap -n hypersdk hypersdk -o yaml

# Fix configuration
helm upgrade hypersdk hypersdk/hypersdk \
  --set config.downloadWorkers=3

# Verify
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk
```

### Q: High restart count

**Symptoms:**
```
NAME                        READY   STATUS    RESTARTS   AGE
hypersdk-7d6f9c8b5-abcd1    1/1     Running   47         2d
```

**Diagnosis:**
```bash
# Check why pods are restarting
kubectl logs -n hypersdk hypersdk-7d6f9c8b5-abcd1 --previous

# Common causes:
# 1. Memory limit too low (OOMKilled)
# 2. Liveness probe failing
# 3. Application crashes
```

**Solution for OOMKilled:**
```bash
# Check memory usage
kubectl top pod -n hypersdk

# Increase memory limit
helm upgrade hypersdk hypersdk/hypersdk \
  --set resources.limits.memory=4Gi
```

**Solution for failing probes:**
```bash
# Adjust probe timing
helm upgrade hypersdk hypersdk/hypersdk \
  --set livenessProbe.initialDelaySeconds=60 \
  --set livenessProbe.periodSeconds=30
```

## Storage Issues

### Q: PVC stuck in Pending

**Symptoms:**
```
NAME              STATUS    VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS
hypersdk-data     Pending                                      standard
```

**Diagnosis:**
```bash
# Check PVC events
kubectl describe pvc -n hypersdk hypersdk-data

# Check if StorageClass exists
kubectl get storageclass standard
```

**Solutions:**

**StorageClass doesn't exist:**
```bash
# List available storage classes
kubectl get storageclass

# Use existing one
helm upgrade hypersdk hypersdk/hypersdk \
  --set persistence.data.storageClass=gp2

# Or create custom StorageClass (see DEPLOYMENT.md)
```

**Dynamic provisioning not available:**
```bash
# Create PV manually for local testing
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolume
metadata:
  name: hypersdk-data-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /mnt/data
  storageClassName: standard
EOF
```

### Q: Out of disk space

**Error in logs:**
```
write /exports/vm-export.vmdk: no space left on device
```

**Diagnosis:**
```bash
# Check disk usage
kubectl exec -n hypersdk deploy/hypersdk -- df -h

# Check PVC size
kubectl get pvc -n hypersdk
```

**Solutions:**

**Expand PVC (if supported):**
```bash
# Check if storage class supports expansion
kubectl get storageclass -o yaml | grep allowVolumeExpansion

# Expand PVC
kubectl patch pvc hypersdk-exports -n hypersdk \
  -p '{"spec":{"resources":{"requests":{"storage":"1Ti"}}}}'

# Wait for expansion
kubectl get pvc -n hypersdk -w
```

**Clean old data:**
```bash
# Find old exports
kubectl exec -n hypersdk deploy/hypersdk -- \
  find /exports -type f -mtime +7

# Delete old exports
kubectl exec -n hypersdk deploy/hypersdk -- \
  find /exports -type f -mtime +7 -delete
```

### Q: Volume mount permission denied

**Error:**
```
mkdir: can't create directory '/data': Permission denied
```

**Cause:** Volume owned by different user

**Solution:**
```bash
# Add initContainer to fix permissions
helm upgrade hypersdk hypersdk/hypersdk \
  --set podSecurityContext.fsGroup=1000

# Or add initContainer manually
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      initContainers:
        - name: fix-permissions
          image: busybox
          command: ['sh', '-c', 'chmod -R 777 /data && chown -R 1000:1000 /data']
          volumeMounts:
            - name: data
              mountPath: /data
EOF
```

## Network Issues

### Q: Cannot access API from outside cluster

**Symptoms:** `curl: (7) Failed to connect to hypersdk.example.com`

**Diagnosis:**
```bash
# Check service type
kubectl get svc -n hypersdk

# Check ingress
kubectl get ingress -n hypersdk

# Check if service has endpoints
kubectl get endpoints -n hypersdk
```

**Solutions:**

**Service is ClusterIP (internal only):**
```bash
# Change to LoadBalancer
helm upgrade hypersdk hypersdk/hypersdk \
  --set service.type=LoadBalancer

# Or use port-forward for testing
kubectl port-forward -n hypersdk svc/hypersdk 8080:8080
```

**Ingress not working:**
```bash
# Check ingress controller is installed
kubectl get pods -n ingress-nginx

# Install if missing
helm install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx --create-namespace

# Enable ingress in chart
helm upgrade hypersdk hypersdk/hypersdk \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=hypersdk.example.com
```

### Q: NetworkPolicy blocking traffic

**Error:** Connection timeout to API

**Diagnosis:**
```bash
# Check if NetworkPolicy is enabled
kubectl get networkpolicy -n hypersdk

# Test from another pod
kubectl run test --image=curlimages/curl -it --rm -- \
  curl http://hypersdk.hypersdk:8080/health
```

**Solution:**
```bash
# Disable NetworkPolicy temporarily
helm upgrade hypersdk hypersdk/hypersdk \
  --set networkPolicy.enabled=false

# Or add ingress rule
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-ingress
  namespace: hypersdk
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: hypersdk
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
EOF
```

### Q: DNS resolution failing

**Error:** `dial tcp: lookup vcenter.example.com: no such host`

**Diagnosis:**
```bash
# Test DNS from pod
kubectl exec -n hypersdk deploy/hypersdk -- nslookup vcenter.example.com

# Check CoreDNS
kubectl get pods -n kube-system | grep coredns
kubectl logs -n kube-system -l k8s-app=kube-dns
```

**Solution:**
```bash
# Restart CoreDNS
kubectl rollout restart deployment/coredns -n kube-system

# Add custom DNS config
helm upgrade hypersdk hypersdk/hypersdk \
  --set dnsConfig.nameservers="{8.8.8.8,8.8.4.4}"
```

## Security Issues

### Q: ImagePullBackOff - Authentication required

**Error:**
```
Failed to pull image "ghcr.io/ssahani/hypersdk-hypervisord:0.2.0":
unauthorized: authentication required
```

**Solution:**
```bash
# Create image pull secret
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=$GITHUB_USER \
  --docker-password=$GITHUB_TOKEN \
  -n hypersdk

# Use in deployment
helm upgrade hypersdk hypersdk/hypersdk \
  --set imagePullSecrets[0].name=ghcr-secret
```

### Q: Secret not found

**Error:** `Error from server (NotFound): secrets "vsphere-credentials" not found`

**Solution:**
```bash
# Create required secrets
kubectl create secret generic vsphere-credentials \
  --from-literal=url=https://vcenter.example.com/sdk \
  --from-literal=username=admin \
  --from-literal=password=changeme \
  -n hypersdk

# Verify
kubectl get secrets -n hypersdk
```

### Q: Certificate verification failed

**Error:** `x509: certificate signed by unknown authority`

**Solution:**
```bash
# Option 1: Add CA certificate
kubectl create configmap ca-certificates \
  --from-file=ca.crt=/path/to/ca.crt \
  -n hypersdk

# Mount in deployment
helm upgrade hypersdk hypersdk/hypersdk \
  --set extraVolumes[0].name=ca-certs \
  --set extraVolumes[0].configMap.name=ca-certificates

# Option 2: Disable verification (INSECURE - dev only!)
kubectl patch secret vsphere-credentials -n hypersdk \
  -p '{"data":{"insecure":"MQ=="}}' # base64 of "1"
```

### Q: RBAC permission denied

**Error:** `forbidden: User "system:serviceaccount:hypersdk:hypersdk" cannot get resource`

**Solution:**
```bash
# Check current permissions
kubectl auth can-i --list --as=system:serviceaccount:hypersdk:hypersdk -n hypersdk

# Grant required permissions
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: hypersdk
  namespace: hypersdk
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: hypersdk
  namespace: hypersdk
subjects:
  - kind: ServiceAccount
    name: hypersdk
roleRef:
  kind: Role
  name: hypersdk
  apiGroup: rbac.authorization.k8s.io
EOF
```

## Performance Issues

### Q: Slow API responses

**Symptoms:** API taking >5 seconds to respond

**Diagnosis:**
```bash
# Check resource usage
kubectl top pods -n hypersdk

# Check logs for slow operations
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk | grep -i "slow\|timeout"

# Check database size
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db "SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size();"
```

**Solutions:**

**High CPU usage:**
```bash
# Increase CPU limits
helm upgrade hypersdk hypersdk/hypersdk \
  --set resources.limits.cpu=2000m
```

**High memory usage:**
```bash
# Increase memory limits
helm upgrade hypersdk hypersdk/hypersdk \
  --set resources.limits.memory=4Gi
```

**Database performance:**
```bash
# Vacuum database
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db "VACUUM;"

# Analyze tables
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db "ANALYZE;"
```

### Q: Export jobs taking too long

**Symptoms:** VM exports timing out

**Diagnosis:**
```bash
# Check concurrent worker limit
kubectl get configmap -n hypersdk hypersdk -o yaml | grep workers

# Check network bandwidth
kubectl exec -n hypersdk deploy/hypersdk -- \
  curl -o /dev/null https://speed.cloudflare.com/__down?bytes=100000000
```

**Solution:**
```bash
# Increase workers for parallel exports
helm upgrade hypersdk hypersdk/hypersdk \
  --set config.downloadWorkers=5

# Increase chunk size for faster transfers
helm upgrade hypersdk hypersdk/hypersdk \
  --set config.chunkSize=50MB
```

### Q: HPA not scaling

**Symptoms:** HPA shows `<unknown>` for metrics

**Diagnosis:**
```bash
# Check HPA status
kubectl get hpa -n hypersdk
kubectl describe hpa -n hypersdk hypersdk

# Check metrics-server
kubectl get deployment metrics-server -n kube-system
```

**Solution:**
```bash
# Install metrics-server if missing
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# For local testing (insecure)
kubectl patch deployment metrics-server -n kube-system \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"metrics-server","args":["--kubelet-insecure-tls"]}]}}}}'
```

## Backup/Recovery Issues

### Q: Backup job failing

**Error:** `backup-database-28235674: Error: command terminated with exit code 1`

**Diagnosis:**
```bash
# Check backup job logs
kubectl logs -n hypersdk job/backup-database-28235674

# Check permissions
kubectl exec -n hypersdk deploy/hypersdk -- ls -la /data
```

**Solution:**
```bash
# Fix volume permissions
kubectl exec -n hypersdk deploy/hypersdk -- \
  chown -R 1000:1000 /data

# Verify backup script
kubectl describe cronjob -n hypersdk database-backup
```

### Q: Cannot restore from backup

**Error:** `cp: cannot create regular file '/data/hypersdk.db': Read-only file system`

**Cause:** Read-only root filesystem

**Solution:**
```bash
# Disable read-only root filesystem for restore
kubectl patch deployment hypersdk -n hypersdk \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"hypersdk","securityContext":{"readOnlyRootFilesystem":false}}]}}}}'

# Restore database
kubectl cp backup.db hypersdk/hypersdk-xxx:/data/hypersdk.db

# Re-enable read-only
kubectl patch deployment hypersdk -n hypersdk \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"hypersdk","securityContext":{"readOnlyRootFilesystem":true}}]}}}}'
```

### Q: VolumeSnapshot failing

**Error:** `snapshot controller not installed`

**Solution:**
```bash
# Install snapshot controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/master/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml
```

## GitOps Issues

### Q: ArgoCD application stuck in "Progressing"

**Symptoms:** Application never reaches "Healthy" status

**Diagnosis:**
```bash
# Check application status
argocd app get hypersdk

# Check for errors
kubectl describe application -n argocd hypersdk
```

**Solution:**
```bash
# Sync manually
argocd app sync hypersdk

# Check health assessment
kubectl get application -n argocd hypersdk -o yaml | grep -A 10 health

# Add health check override if needed
kubectl patch application hypersdk -n argocd \
  --type merge \
  -p '{"spec":{"ignoreDifferences":[{"group":"apps","kind":"Deployment","jsonPointers":["/spec/replicas"]}]}}'
```

### Q: Flux HelmRelease failing

**Error:** `install retries exhausted`

**Diagnosis:**
```bash
# Check HelmRelease status
kubectl describe helmrelease -n flux-system hypersdk

# Check Helm controller logs
kubectl logs -n flux-system -l app=helm-controller
```

**Solution:**
```bash
# Increase timeout
kubectl patch helmrelease hypersdk -n flux-system \
  --type merge \
  -p '{"spec":{"timeout":"10m"}}'

# Force reconciliation
flux reconcile helmrelease hypersdk -n flux-system
```

### Q: Sealed Secret cannot be decrypted

**Error:** `no key could decrypt secret`

**Cause:** Sealed Secrets controller certificate changed

**Solution:**
```bash
# Get current public key
kubeseal --fetch-cert > pub-cert.pem

# Re-seal secrets with new cert
kubectl create secret generic vsphere-credentials \
  --from-literal=url=https://vcenter.example.com/sdk \
  --dry-run=client -o yaml | \
  kubeseal --cert pub-cert.pem -o yaml > sealed-secret.yaml

# Apply
kubectl apply -f sealed-secret.yaml
```

## Cost/Resource Issues

### Q: Running out of quota

**Error:** `exceeded quota: pods: "100"`

**Diagnosis:**
```bash
# Check current quota
kubectl describe resourcequota -n hypersdk

# Check current usage
kubectl get pods -n hypersdk | wc -l
```

**Solution:**
```bash
# Increase quota
kubectl apply -f - <<EOF
apiVersion: v1
kind: ResourceQuota
metadata:
  name: hypersdk-quota
  namespace: hypersdk
spec:
  hard:
    pods: "200"
    requests.cpu: "50"
    requests.memory: 100Gi
    limits.cpu: "100"
    limits.memory: 200Gi
EOF
```

### Q: Kubecost showing high idle costs

**Symptoms:** Idle cost is 40%+ of total cost

**Diagnosis:**
```bash
# Check actual vs requested resources
kubectl top pods -n hypersdk
kubectl describe pod -n hypersdk | grep -A 5 "Requests\|Limits"
```

**Solution:**
```bash
# Right-size requests based on actual usage
helm upgrade hypersdk hypersdk/hypersdk \
  --set resources.requests.cpu=250m \
  --set resources.requests.memory=512Mi

# Enable VPA for automatic right-sizing
kubectl apply -f - <<EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: hypersdk-vpa
  namespace: hypersdk
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hypersdk
  updatePolicy:
    updateMode: "Auto"
EOF
```

## Monitoring Issues

### Q: Prometheus not scraping metrics

**Symptoms:** No metrics in Grafana

**Diagnosis:**
```bash
# Check ServiceMonitor
kubectl get servicemonitor -n hypersdk

# Check if Prometheus Operator is watching namespace
kubectl get prometheus -n monitoring -o yaml | grep namespaceSelector

# Test metrics endpoint
kubectl port-forward -n hypersdk svc/hypersdk 8081:8081
curl http://localhost:8081/metrics
```

**Solution:**
```bash
# Add namespace label
kubectl label namespace hypersdk monitoring=enabled

# Or configure Prometheus to watch all namespaces
kubectl patch prometheus -n monitoring kube-prometheus-stack \
  --type merge \
  -p '{"spec":{"serviceMonitorNamespaceSelector":{}}}'
```

### Q: Grafana dashboard shows no data

**Symptoms:** Dashboard panels empty

**Diagnosis:**
```bash
# Check data source
kubectl port-forward -n monitoring svc/kube-prometheus-stack-grafana 3000:80

# Check Prometheus targets
kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090
# Navigate to http://localhost:9090/targets
```

**Solution:**
```bash
# Verify metrics are being collected
kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090
# Query: up{job="hypersdk"}

# Re-import dashboard with correct data source
kubectl apply -f deployments/kubernetes/monitoring/grafana-dashboard-k8s.json
```

### Q: Alerts not firing

**Symptoms:** No alerts despite obvious issues

**Diagnosis:**
```bash
# Check PrometheusRule
kubectl get prometheusrule -n hypersdk

# Check Alertmanager config
kubectl get secret -n monitoring alertmanager-kube-prometheus-stack-alertmanager \
  -o jsonpath='{.data.alertmanager\.yaml}' | base64 -d
```

**Solution:**
```bash
# Verify alert rule syntax
promtool check rules deployments/kubernetes/monitoring/prometheusrule.yaml

# Test alert manually in Prometheus
# Navigate to http://localhost:9090/alerts
```

## General Tips

### Enable Debug Logging

```bash
helm upgrade hypersdk hypersdk/hypersdk \
  --set config.logLevel=debug

kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk -f
```

### Get Full Diagnostic Information

```bash
#!/bin/bash
# diagnostic.sh - Collect all diagnostic info

NAMESPACE="hypersdk"
OUTPUT_DIR="hypersdk-diagnostics-$(date +%Y%m%d_%H%M%S)"

mkdir -p "$OUTPUT_DIR"

# Pods
kubectl get pods -n "$NAMESPACE" -o wide > "$OUTPUT_DIR/pods.txt"
kubectl describe pods -n "$NAMESPACE" > "$OUTPUT_DIR/pods-describe.txt"

# Logs
for pod in $(kubectl get pods -n "$NAMESPACE" -o name); do
  kubectl logs -n "$NAMESPACE" "$pod" > "$OUTPUT_DIR/${pod//\//-}.log" 2>&1
  kubectl logs -n "$NAMESPACE" "$pod" --previous > "$OUTPUT_DIR/${pod//\//-}-previous.log" 2>&1 || true
done

# Resources
kubectl get all -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/all-resources.yaml"
kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' > "$OUTPUT_DIR/events.txt"

# Config
kubectl get configmap -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/configmaps.yaml"
kubectl get secret -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/secrets.yaml"

# Network
kubectl get svc,endpoints,ingress,networkpolicy -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/network.yaml"

# Storage
kubectl get pvc,pv -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/storage.yaml"

# Helm
helm list -n "$NAMESPACE" > "$OUTPUT_DIR/helm-releases.txt"
helm get values hypersdk -n "$NAMESPACE" > "$OUTPUT_DIR/helm-values.yaml"
helm get manifest hypersdk -n "$NAMESPACE" > "$OUTPUT_DIR/helm-manifest.yaml"

# Compress
tar -czf "$OUTPUT_DIR.tar.gz" "$OUTPUT_DIR"
echo "Diagnostics collected: $OUTPUT_DIR.tar.gz"
```

### Check All Health Endpoints

```bash
#!/bin/bash
# health-check.sh

NAMESPACE="hypersdk"

echo "=== Pod Health ==="
kubectl get pods -n "$NAMESPACE"

echo -e "\n=== Service Endpoints ==="
kubectl get endpoints -n "$NAMESPACE"

echo -e "\n=== API Health ==="
kubectl port-forward -n "$NAMESPACE" svc/hypersdk 8080:8080 &
PF_PID=$!
sleep 3
curl -f http://localhost:8080/health || echo "API health check failed"
kill $PF_PID

echo -e "\n=== Metrics Health ==="
kubectl port-forward -n "$NAMESPACE" svc/hypersdk 8081:8081 &
PF_PID=$!
sleep 3
curl -f http://localhost:8081/metrics | head -20 || echo "Metrics check failed"
kill $PF_PID

echo -e "\n=== Resource Usage ==="
kubectl top pods -n "$NAMESPACE"

echo -e "\n=== Recent Errors ==="
kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=hypersdk --tail=50 | grep -i error || echo "No errors found"
```

## Getting Help

If you're still stuck after trying these solutions:

1. **Collect diagnostics**: Run the diagnostic script above
2. **Check documentation**: Review the relevant guide in [OPERATIONAL-EXCELLENCE.md](OPERATIONAL-EXCELLENCE.md)
3. **Search issues**: Check [GitHub Issues](https://github.com/ssahani/hypersdk/issues)
4. **Ask for help**: Create a new issue with:
   - Error message
   - Steps to reproduce
   - Diagnostic bundle
   - Environment details (K8s version, cloud provider, etc.)

## Quick Reference

| Issue Type | First Check | Common Fix |
|------------|-------------|------------|
| Pod pending | `kubectl describe pod` | Resource requests too high |
| CrashLoopBackOff | `kubectl logs --previous` | Check config, permissions |
| ImagePullBackOff | Image name, auth | Create imagePullSecret |
| Network timeout | NetworkPolicy | Disable or add ingress rule |
| Storage pending | StorageClass | Create or change class |
| High CPU/memory | `kubectl top pods` | Increase limits or scale out |
| Slow performance | Resource usage | Right-size or add workers |
| Metrics missing | ServiceMonitor | Check namespace labels |

---

**Pro tip**: Most issues can be diagnosed with `kubectl describe` and `kubectl logs`. Always check these first!
