# Security Hardening & Compliance Guide for HyperSDK

Comprehensive security best practices and compliance guidance for production Kubernetes deployments.

## Overview

This guide covers security hardening across multiple layers:

- **Container Security** - Image scanning, runtime security, supply chain
- **Network Security** - NetworkPolicy, service mesh, encryption
- **Access Control** - RBAC, Pod Security Standards, admission control
- **Secrets Management** - Encryption, rotation, external stores
- **Compliance** - SOC2, HIPAA, PCI-DSS, GDPR considerations
- **Audit & Monitoring** - Security logging, intrusion detection
- **Incident Response** - Security event handling

## Security Checklist

### Pre-Deployment

- [ ] Enable Pod Security Standards (restricted profile)
- [ ] Configure NetworkPolicy for zero-trust networking
- [ ] Set up secrets encryption at rest
- [ ] Enable audit logging
- [ ] Scan container images for vulnerabilities
- [ ] Review RBAC permissions (principle of least privilege)
- [ ] Configure TLS for all external endpoints
- [ ] Set up image pull secrets for private registries
- [ ] Enable resource quotas and limits
- [ ] Configure security context constraints

### Post-Deployment

- [ ] Verify pods run as non-root
- [ ] Confirm NetworkPolicy is enforced
- [ ] Test secrets rotation procedures
- [ ] Validate audit logs are collected
- [ ] Run vulnerability scans (Trivy, Grype)
- [ ] Enable runtime security monitoring (Falco)
- [ ] Set up security alerts (OPA violations, suspicious activity)
- [ ] Perform penetration testing
- [ ] Review compliance requirements
- [ ] Document security controls

## Container Security

### Image Scanning

#### Trivy Integration

```bash
# Scan HyperSDK images
trivy image ghcr.io/ssahani/hypersdk-hypervisord:0.2.0

# Scan with severity filter
trivy image --severity HIGH,CRITICAL ghcr.io/ssahani/hypersdk-hypervisord:0.2.0

# Output to JSON for automation
trivy image -f json -o scan-results.json ghcr.io/ssahani/hypersdk-hypervisord:0.2.0

# Fail CI/CD on critical vulnerabilities
trivy image --exit-code 1 --severity CRITICAL ghcr.io/ssahani/hypersdk-hypervisord:0.2.0
```

#### Continuous Scanning

```yaml
# .github/workflows/security-scan.yml
name: Security Scan

on:
  schedule:
    - cron: '0 0 * * *'  # Daily
  push:
    branches: [main]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ghcr.io/ssahani/hypersdk-hypervisord:0.2.0
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH'

      - name: Upload Trivy results to GitHub Security
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'
```

### Image Signing & Verification

#### Cosign (Sigstore)

```bash
# Sign image with cosign
cosign sign --key cosign.key ghcr.io/ssahani/hypersdk-hypervisord:0.2.0

# Verify signature
cosign verify --key cosign.pub ghcr.io/ssahani/hypersdk-hypervisord:0.2.0

# Keyless signing (recommended)
cosign sign ghcr.io/ssahani/hypersdk-hypervisord:0.2.0

# Generate SBOM (Software Bill of Materials)
cosign attach sbom --sbom sbom.spdx ghcr.io/ssahani/hypersdk-hypervisord:0.2.0
```

#### Policy Controller (enforce signatures)

```yaml
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: hypersdk-image-policy
spec:
  images:
    - glob: "ghcr.io/ssahani/hypersdk-*"
  authorities:
    - keyless:
        url: https://fulcio.sigstore.dev
        identities:
          - issuer: https://github.com/login/oauth
            subject: https://github.com/ssahani/hypersdk/.github/workflows/*
```

### Runtime Security (Falco)

```yaml
# Install Falco
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm install falco falcosecurity/falco \
  --namespace falco \
  --create-namespace \
  --set falcosidekick.enabled=true \
  --set falcosidekick.webui.enabled=true

---
# Custom Falco rules for HyperSDK
apiVersion: v1
kind: ConfigMap
metadata:
  name: falco-rules-hypersdk
  namespace: falco
data:
  hypersdk-rules.yaml: |
    - rule: Unauthorized Process in HyperSDK Container
      desc: Detect unauthorized process execution
      condition: >
        spawned_process and
        container.image.repository = "ghcr.io/ssahani/hypersdk-hypervisord" and
        not proc.name in (hypervisord, curl, sh)
      output: >
        Unauthorized process in HyperSDK
        (user=%user.name command=%proc.cmdline container=%container.name)
      priority: WARNING
      tags: [container, process, hypersdk]

    - rule: Sensitive File Access in HyperSDK
      desc: Detect access to sensitive files
      condition: >
        open_read and
        container.image.repository = "ghcr.io/ssahani/hypersdk-hypervisord" and
        fd.name in (/etc/shadow, /etc/passwd, /root/.ssh/id_rsa)
      output: >
        Sensitive file accessed
        (user=%user.name file=%fd.name container=%container.name)
      priority: CRITICAL
      tags: [filesystem, hypersdk]

    - rule: Outbound Connection to Suspicious IP
      desc: Detect connections to non-whitelisted IPs
      condition: >
        outbound and
        container.image.repository = "ghcr.io/ssahani/hypersdk-hypervisord" and
        not fd.sip in (vcenter_ip_range, aws_ip_range)
      output: >
        Suspicious outbound connection
        (destination=%fd.sip port=%fd.sport container=%container.name)
      priority: WARNING
      tags: [network, hypersdk]
```

## Network Security

### Zero-Trust NetworkPolicy

```yaml
# Default deny all traffic
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: hypersdk
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress

---
# Allow ingress to HyperSDK API
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-api-ingress
  namespace: hypersdk
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: hypersdk
  policyTypes:
    - Ingress
  ingress:
    # Allow from Ingress controller
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080

---
# Allow egress to vSphere/cloud providers
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-cloud-egress
  namespace: hypersdk
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: hypersdk
  policyTypes:
    - Egress
  egress:
    # DNS resolution
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53

    # vSphere API (HTTPS)
    - to:
        - ipBlock:
            cidr: 10.0.0.0/8  # Internal network
      ports:
        - protocol: TCP
          port: 443

    # AWS API (HTTPS)
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
            except:
              - 169.254.169.254/32  # Block metadata service
      ports:
        - protocol: TCP
          port: 443

---
# Allow Prometheus scraping
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-prometheus-scraping
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
              name: monitoring
        - podSelector:
            matchLabels:
              app: prometheus
      ports:
        - protocol: TCP
          port: 8081
```

### Service Mesh Security (Istio)

```yaml
# Mutual TLS enforcement
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: hypersdk-mtls
  namespace: hypersdk
spec:
  mtls:
    mode: STRICT

---
# Authorization policy
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: hypersdk-authz
  namespace: hypersdk
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: hypersdk
  action: ALLOW
  rules:
    # Allow authenticated users
    - from:
        - source:
            principals: ["cluster.local/ns/hypersdk/sa/hypersdk"]
      to:
        - operation:
            methods: ["GET", "POST"]
            paths: ["/api/v1/*"]

    # Allow Prometheus scraping
    - from:
        - source:
            namespaces: ["monitoring"]
      to:
        - operation:
            methods: ["GET"]
            paths: ["/metrics"]

---
# Request authentication (JWT)
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: hypersdk-jwt
  namespace: hypersdk
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: hypersdk
  jwtRules:
    - issuer: "https://your-identity-provider.com"
      jwksUri: "https://your-identity-provider.com/.well-known/jwks.json"
      audiences:
        - "hypersdk-api"
```

### TLS/SSL Configuration

```yaml
# Certificate management with cert-manager
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: hypersdk-tls
  namespace: hypersdk
spec:
  secretName: hypersdk-tls-secret
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - hypersdk.example.com
    - api.hypersdk.example.com
  privateKey:
    algorithm: RSA
    size: 4096
    rotationPolicy: Always

---
# Ingress with TLS
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hypersdk-ingress
  namespace: hypersdk
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/ssl-protocols: "TLSv1.2 TLSv1.3"
    nginx.ingress.kubernetes.io/ssl-ciphers: "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - hypersdk.example.com
      secretName: hypersdk-tls-secret
  rules:
    - host: hypersdk.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: hypersdk
                port:
                  number: 8080
```

## Access Control

### Pod Security Standards

```yaml
# Enforce restricted Pod Security Standard
apiVersion: v1
kind: Namespace
metadata:
  name: hypersdk
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted

---
# Pod security context (already in Helm chart)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hypersdk
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: hypersdk
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: false  # SQLite needs write
            capabilities:
              drop:
                - ALL
          volumeMounts:
            - name: data
              mountPath: /data
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir: {}
```

### RBAC Best Practices

```yaml
# Minimal ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hypersdk
  namespace: hypersdk
automountServiceAccountToken: true

---
# Minimal Role (read-only to own namespace)
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: hypersdk
  namespace: hypersdk
rules:
  # Read own ConfigMap
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch"]
    resourceNames: ["hypersdk"]

  # Read own Secrets (for credentials)
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get"]
    resourceNames:
      - "vsphere-credentials"
      - "aws-credentials"
      - "azure-credentials"
      - "gcp-credentials"

---
# RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: hypersdk
  namespace: hypersdk
subjects:
  - kind: ServiceAccount
    name: hypersdk
    namespace: hypersdk
roleRef:
  kind: Role
  name: hypersdk
  apiGroup: rbac.authorization.k8s.io
```

### OPA/Gatekeeper Policies

```yaml
# Install Gatekeeper
helm repo add gatekeeper https://open-policy-agent.github.io/gatekeeper/charts
helm install gatekeeper gatekeeper/gatekeeper -n gatekeeper-system --create-namespace

---
# Constraint Template: Block privileged containers
apiVersion: templates.gatekeeper.sh/v1beta1
kind: ConstraintTemplate
metadata:
  name: k8sblockprivileged
spec:
  crd:
    spec:
      names:
        kind: K8sBlockPrivileged
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8sblockprivileged

        violation[{"msg": msg}] {
          container := input.review.object.spec.containers[_]
          container.securityContext.privileged
          msg := sprintf("Privileged container not allowed: %v", [container.name])
        }

---
# Constraint: Enforce no privileged containers
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sBlockPrivileged
metadata:
  name: block-privileged-containers
spec:
  match:
    kinds:
      - apiGroups: [""]
        kinds: ["Pod"]
    namespaces:
      - hypersdk

---
# Constraint Template: Require resource limits
apiVersion: templates.gatekeeper.sh/v1beta1
kind: ConstraintTemplate
metadata:
  name: k8srequireresourcelimits
spec:
  crd:
    spec:
      names:
        kind: K8sRequireResourceLimits
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8srequireresourcelimits

        violation[{"msg": msg}] {
          container := input.review.object.spec.containers[_]
          not container.resources.limits.memory
          msg := sprintf("Container %v missing memory limit", [container.name])
        }

        violation[{"msg": msg}] {
          container := input.review.object.spec.containers[_]
          not container.resources.limits.cpu
          msg := sprintf("Container %v missing CPU limit", [container.name])
        }
```

## Secrets Management

### Sealed Secrets (for GitOps)

```bash
# Install Sealed Secrets controller
helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
helm install sealed-secrets sealed-secrets/sealed-secrets -n kube-system

# Create sealed secret
kubectl create secret generic vsphere-credentials \
  --from-literal=url=https://vcenter.example.com/sdk \
  --from-literal=username=admin \
  --from-literal=password=changeme \
  --dry-run=client -o yaml | \
  kubeseal -o yaml > vsphere-sealed-secret.yaml

# Commit to Git (safe!)
git add vsphere-sealed-secret.yaml
git commit -m "Add vSphere sealed secret"
```

```yaml
# Sealed Secret (encrypted, safe for Git)
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: vsphere-credentials
  namespace: hypersdk
spec:
  encryptedData:
    url: AgBq7Xy... # encrypted
    username: AgCdE... # encrypted
    password: AgAkP... # encrypted
  template:
    metadata:
      name: vsphere-credentials
      namespace: hypersdk
```

### External Secrets Operator

```yaml
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets --create-namespace

---
# AWS Secrets Manager backend
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets
  namespace: hypersdk
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: hypersdk

---
# External Secret (synced from AWS)
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: vsphere-credentials
  namespace: hypersdk
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets
    kind: SecretStore
  target:
    name: vsphere-credentials
    creationPolicy: Owner
  data:
    - secretKey: url
      remoteRef:
        key: hypersdk/vsphere
        property: url
    - secretKey: username
      remoteRef:
        key: hypersdk/vsphere
        property: username
    - secretKey: password
      remoteRef:
        key: hypersdk/vsphere
        property: password
```

### HashiCorp Vault Integration

```yaml
# Install Vault
helm repo add hashicorp https://helm.releases.hashicorp.com
helm install vault hashicorp/vault \
  --namespace vault \
  --create-namespace \
  --set "server.ha.enabled=true"

---
# Vault authentication
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hypersdk
  namespace: hypersdk

---
# Vault SecretProviderClass (CSI driver)
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: hypersdk-vault-secrets
  namespace: hypersdk
spec:
  provider: vault
  parameters:
    vaultAddress: "http://vault.vault:8200"
    roleName: "hypersdk"
    objects: |
      - objectName: "vsphere-url"
        secretPath: "secret/data/hypersdk/vsphere"
        secretKey: "url"
      - objectName: "vsphere-username"
        secretPath: "secret/data/hypersdk/vsphere"
        secretKey: "username"
      - objectName: "vsphere-password"
        secretPath: "secret/data/hypersdk/vsphere"
        secretKey: "password"
  secretObjects:
    - secretName: vsphere-credentials
      type: Opaque
      data:
        - objectName: vsphere-url
          key: url
        - objectName: vsphere-username
          key: username
        - objectName: vsphere-password
          key: password

---
# Mount secrets in deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hypersdk
spec:
  template:
    spec:
      serviceAccountName: hypersdk
      containers:
        - name: hypersdk
          volumeMounts:
            - name: secrets-store
              mountPath: "/mnt/secrets"
              readOnly: true
      volumes:
        - name: secrets-store
          csi:
            driver: secrets-store.csi.k8s.io
            readOnly: true
            volumeAttributes:
              secretProviderClass: "hypersdk-vault-secrets"
```

### Secrets Rotation

```bash
# Automated rotation script
#!/bin/bash
# rotate-vsphere-credentials.sh

set -e

NAMESPACE="hypersdk"
SECRET_NAME="vsphere-credentials"
VAULT_PATH="secret/hypersdk/vsphere"

# Generate new password
NEW_PASSWORD=$(openssl rand -base64 32)

# Update password in vCenter
vcenter-cli update-password --username admin --new-password "$NEW_PASSWORD"

# Update Vault
vault kv put "$VAULT_PATH" \
  url="https://vcenter.example.com/sdk" \
  username="admin" \
  password="$NEW_PASSWORD"

# External Secrets Operator will sync automatically
echo "Credentials rotated successfully"

# Restart pods to pick up new credentials
kubectl rollout restart deployment/hypersdk -n "$NAMESPACE"
```

## Audit Logging

### Enable Kubernetes Audit Logs

```yaml
# API server audit policy
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  # Log all requests to HyperSDK resources
  - level: RequestResponse
    namespaces: ["hypersdk"]
    verbs: ["create", "update", "patch", "delete"]
    resources:
      - group: ""
        resources: ["secrets", "configmaps", "pods"]

  # Log all authentication events
  - level: Metadata
    omitStages:
      - RequestReceived
    userGroups: ["system:authenticated"]

  # Log all access to sensitive endpoints
  - level: Request
    verbs: ["get", "list"]
    namespaces: ["hypersdk"]
    resources:
      - group: ""
        resources: ["secrets"]
```

### Application Audit Logging

```yaml
# Configure HyperSDK to log security events
apiVersion: v1
kind: ConfigMap
metadata:
  name: hypersdk
  namespace: hypersdk
data:
  config.yaml: |
    audit:
      enabled: true
      log_level: info
      events:
        - authentication_success
        - authentication_failure
        - authorization_failure
        - export_started
        - export_completed
        - export_failed
        - credential_accessed
        - configuration_changed
      output:
        - type: stdout
          format: json
        - type: syslog
          server: syslog.example.com:514
          protocol: tcp
```

### Audit Log Analysis (Falco)

```yaml
# Falco rules for audit log analysis
- rule: Unauthorized Secret Access
  desc: Detect access to secrets by unauthorized users
  condition: >
    kevt and
    ka.verb = "get" and
    ka.target.resource = "secrets" and
    ka.target.namespace = "hypersdk" and
    not ka.user.name in (system:serviceaccount:hypersdk:hypersdk)
  output: >
    Unauthorized secret access
    (user=%ka.user.name secret=%ka.target.name namespace=%ka.target.namespace)
  priority: CRITICAL
  source: k8s_audit

- rule: Multiple Failed Login Attempts
  desc: Detect brute force attacks
  condition: >
    kevt and
    ka.verb = "create" and
    ka.target.resource = "tokenreviews" and
    ka.response.code >= 400
  output: >
    Failed authentication attempt
    (user=%ka.user.name source_ip=%ka.source.ip)
  priority: WARNING
  source: k8s_audit
```

## Compliance

### SOC 2 Controls

#### Access Control (CC6.1, CC6.2)

```yaml
# Implement RBAC with least privilege
# See RBAC section above

# Audit access to sensitive resources
# See Audit Logging section above

# Implement MFA for cluster access
# Use OIDC provider with MFA (Okta, Auth0, etc.)
```

#### Logical and Physical Security (CC6.6, CC6.7)

```yaml
# NetworkPolicy for network segmentation
# See Network Security section above

# Encryption in transit (TLS)
# See TLS/SSL Configuration section above

# Encryption at rest
apiVersion: v1
kind: EncryptionConfiguration
resources:
  - resources:
      - secrets
    providers:
      - aescbc:
          keys:
            - name: key1
              secret: <base64-encoded-32-byte-key>
      - identity: {}
```

#### Change Management (CC8.1)

```yaml
# Use GitOps for all changes
# See GitOps guide (GITOPS.md)

# Require PR reviews
# GitHub branch protection rules

# Audit all changes
# Git commit history + K8s audit logs
```

### HIPAA Compliance

#### Encryption Requirements

```yaml
# Encrypt data in transit (already covered)
# Encrypt data at rest
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-data
  namespace: hypersdk
spec:
  storageClassName: encrypted-ssd  # Use encrypted storage class
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

#### Access Controls

```yaml
# Implement role-based access
# Audit all PHI access
# Automatic session timeout (configure in ingress)
nginx.ingress.kubernetes.io/session-cookie-max-age: "1800"  # 30 minutes
```

#### Audit Controls

```yaml
# Comprehensive audit logging (already covered)
# Retain logs for 6 years
# See retention policies in observability guide
```

### PCI-DSS Requirements

#### Requirement 1: Firewall Configuration

```yaml
# NetworkPolicy for network segmentation (already covered)
# Ingress controls (already covered)
```

#### Requirement 2: No Default Passwords

```yaml
# Use secure secret generation
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
  namespace: hypersdk
type: Opaque
stringData:
  password: $(openssl rand -base64 32)
```

#### Requirement 3: Protect Stored Data

```yaml
# Encrypt PVCs (already covered)
# Encrypt secrets at rest (already covered)
# Mask sensitive data in logs

apiVersion: v1
kind: ConfigMap
metadata:
  name: hypersdk
data:
  config.yaml: |
    logging:
      mask_sensitive_fields: true
      masked_fields:
        - password
        - api_key
        - token
        - secret
```

#### Requirement 10: Track and Monitor

```yaml
# Audit logging (already covered)
# Security monitoring (Falco, already covered)
# Alert on security events (already covered)
```

### GDPR Considerations

#### Right to Erasure

```bash
# Script to delete user data
#!/bin/bash
USER_ID=$1

# Delete from database
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db \
  "DELETE FROM jobs WHERE user_id='$USER_ID'; \
   DELETE FROM exports WHERE user_id='$USER_ID';"

# Delete exported files
kubectl exec -n hypersdk deploy/hypersdk -- \
  find /exports -user "$USER_ID" -delete

# Audit log entry
echo "User data deleted: $USER_ID" | \
  kubectl exec -i -n hypersdk deploy/hypersdk -- \
  logger -t gdpr-deletion
```

#### Data Minimization

```yaml
# Configure minimal data retention
apiVersion: v1
kind: ConfigMap
metadata:
  name: hypersdk
data:
  config.yaml: |
    data_retention:
      completed_jobs: 30d
      failed_jobs: 90d
      exports: 7d
      audit_logs: 90d
```

## Security Monitoring

### Prometheus Security Metrics

```yaml
# PrometheusRule for security alerts
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: hypersdk-security-alerts
  namespace: hypersdk
spec:
  groups:
    - name: hypersdk.security
      interval: 30s
      rules:
        - alert: HighAuthenticationFailureRate
          expr: |
            rate(hypersdk_auth_failures_total[5m]) > 0.1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High authentication failure rate"
            description: "Authentication failure rate: {{ $value }}/s"

        - alert: UnauthorizedAccess
          expr: |
            rate(hypersdk_authorization_failures_total[5m]) > 0
          for: 1m
          labels:
            severity: critical
          annotations:
            summary: "Unauthorized access attempt detected"

        - alert: SuspiciousAPIActivity
          expr: |
            rate(hypersdk_api_requests_total{status="403"}[5m]) > 1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Suspicious API activity (multiple 403s)"

        - alert: VulnerabilityDetected
          expr: |
            hypersdk_vulnerabilities_critical > 0
          for: 1m
          labels:
            severity: critical
          annotations:
            summary: "Critical vulnerability detected in container"
```

### SIEM Integration

```yaml
# Fluentd configuration for SIEM
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
  namespace: logging
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/containers/hypersdk-*.log
      pos_file /var/log/fluentd-hypersdk.pos
      tag hypersdk
      <parse>
        @type json
      </parse>
    </source>

    <filter hypersdk>
      @type grep
      <regexp>
        key level
        pattern /(ERROR|WARNING|SECURITY)/
      </regexp>
    </filter>

    <match hypersdk>
      @type syslog
      host siem.example.com
      port 514
      protocol tcp
      <format>
        @type json
      </format>
    </match>
```

## Incident Response

### Security Incident Playbook

#### Phase 1: Detection

```bash
# Check Falco alerts
kubectl logs -n falco -l app=falco | grep -i "priority: CRITICAL"

# Check Prometheus alerts
curl http://alertmanager:9093/api/v2/alerts | jq '.[] | select(.labels.severity=="critical")'

# Check audit logs
kubectl logs -n kube-system kube-apiserver-* | grep -i "forbidden\|unauthorized"
```

#### Phase 2: Containment

```bash
# Isolate affected pod with NetworkPolicy
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: isolate-compromised-pod
  namespace: hypersdk
spec:
  podSelector:
    matchLabels:
      pod: compromised-pod-name
  policyTypes:
    - Ingress
    - Egress
EOF

# Scale down deployment
kubectl scale deployment/hypersdk --replicas=0 -n hypersdk

# Take memory dump for forensics
kubectl exec -n hypersdk POD_NAME -- gcore $(pgrep hypervisord)
```

#### Phase 3: Investigation

```bash
# Collect pod logs
kubectl logs -n hypersdk POD_NAME --previous > incident-logs.txt

# Export pod events
kubectl get events -n hypersdk --field-selector involvedObject.name=POD_NAME -o json > incident-events.json

# Collect network traffic (if using service mesh)
kubectl exec -n hypersdk POD_NAME -c istio-proxy -- \
  curl localhost:15000/config_dump > envoy-config.json
```

#### Phase 4: Eradication

```bash
# Delete compromised resources
kubectl delete pod -n hypersdk POD_NAME

# Rotate all credentials
./rotate-all-credentials.sh

# Update to patched image
kubectl set image deployment/hypersdk -n hypersdk \
  hypersdk=ghcr.io/ssahani/hypersdk-hypervisord:0.2.1-security-patch
```

#### Phase 5: Recovery

```bash
# Restore from clean backup
helm rollback hypersdk -n hypersdk

# Verify integrity
./verify-deployment.sh

# Monitor closely
kubectl logs -n hypersdk -l app=hypersdk -f
```

#### Phase 6: Post-Incident

```bash
# Document incident
# Update security controls
# Conduct root cause analysis
# Update runbooks
```

## Security Testing

### Penetration Testing

```bash
# Port scanning (from outside cluster)
nmap -sV hypersdk.example.com

# Web vulnerability scanning
nikto -h https://hypersdk.example.com

# API fuzzing
ffuf -w api-wordlist.txt -u https://hypersdk.example.com/api/FUZZ

# SQL injection testing (if applicable)
sqlmap -u "https://hypersdk.example.com/api/v1/export?id=1"
```

### Kubernetes Security Scanning

```bash
# Scan with kube-bench (CIS benchmark)
kubectl apply -f https://raw.githubusercontent.com/aquasecurity/kube-bench/main/job.yaml
kubectl logs -f job/kube-bench

# Scan with kube-hunter
kubectl apply -f https://raw.githubusercontent.com/aquasecurity/kube-hunter/main/job.yaml
kubectl logs -f job/kube-hunter

# Scan manifests with kubesec
kubesec scan deployments/helm/hypersdk/templates/deployment.yaml
```

## Summary

### Security Checklist (Quick Reference)

✅ **Container Security**
- [ ] Image scanning (Trivy)
- [ ] Image signing (Cosign)
- [ ] Runtime security (Falco)
- [ ] Non-root containers
- [ ] Read-only root filesystem (where possible)

✅ **Network Security**
- [ ] NetworkPolicy (zero-trust)
- [ ] TLS everywhere
- [ ] Service mesh (mTLS)
- [ ] Ingress controls

✅ **Access Control**
- [ ] Pod Security Standards (restricted)
- [ ] RBAC (least privilege)
- [ ] OPA/Gatekeeper policies
- [ ] Service accounts (no automounting where not needed)

✅ **Secrets Management**
- [ ] Secrets encryption at rest
- [ ] External secrets (Vault/AWS/Azure)
- [ ] Sealed Secrets for GitOps
- [ ] Regular rotation

✅ **Compliance**
- [ ] SOC 2 controls
- [ ] HIPAA (if applicable)
- [ ] PCI-DSS (if applicable)
- [ ] GDPR (if applicable)

✅ **Monitoring**
- [ ] Audit logging
- [ ] Security metrics
- [ ] SIEM integration
- [ ] Alert rules

✅ **Incident Response**
- [ ] Runbooks documented
- [ ] Forensics capability
- [ ] Backup procedures
- [ ] Communication plan

### Resources

- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [OWASP Kubernetes Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Kubernetes_Security_Cheat_Sheet.html)

---

**Security is a continuous process, not a one-time configuration. Regularly review and update security controls.**
