# Security Best Practices

## Table of Contents

1. [Credential Management](#credential-management)
2. [Network Security](#network-security)
3. [Access Control](#access-control)
4. [Data Protection](#data-protection)
5. [Audit Logging](#audit-logging)
6. [System Hardening](#system-hardening)
7. [Compliance](#compliance)

## Credential Management

### Never Hardcode Credentials

**Bad**:
```yaml
vsphere:
  username: "administrator@vsphere.local"
  password: "MyPassword123"  # NEVER DO THIS
```

**Good**:
```yaml
vsphere:
  username: "${VCENTER_USERNAME}"
  password: "${VCENTER_PASSWORD}"
```

### Use Environment Variables

```bash
# Set in systemd service file
sudo systemctl edit hypervisord

[Service]
Environment="VCENTER_USERNAME=admin@vsphere.local"
Environment="VCENTER_PASSWORD=SecurePass123"
Environment="AWS_ACCESS_KEY_ID=AKIA..."
Environment="AWS_SECRET_ACCESS_KEY=..."

# Or use EnvironmentFile
sudo mkdir -p /etc/hypervisord
sudo vim /etc/hypervisord/credentials.env

VCENTER_USERNAME=admin@vsphere.local
VCENTER_PASSWORD=SecurePass123

# In service file
[Service]
EnvironmentFile=/etc/hypervisord/credentials.env
```

### Secure Credential Storage

```bash
# Use restrictive permissions
sudo chmod 600 /etc/hypervisord/config.yaml
sudo chmod 600 /etc/hypervisord/credentials.env
sudo chown root:root /etc/hypervisord/*.yaml
sudo chown root:root /etc/hypervisord/*.env
```

### Use Secrets Management

```bash
# HashiCorp Vault integration
export VAULT_ADDR='https://vault.example.com'
export VAULT_TOKEN='s.xxxxx'

# Retrieve secrets
export VCENTER_PASSWORD=$(vault kv get -field=password secret/vcenter)
export AWS_SECRET_ACCESS_KEY=$(vault kv get -field=secret secret/aws)

# AWS Secrets Manager
export VCENTER_PASSWORD=$(aws secretsmanager get-secret-value \
  --secret-id vcenter/password \
  --query SecretString \
  --output text)

# Azure Key Vault
export VCENTER_PASSWORD=$(az keyvault secret show \
  --vault-name my-vault \
  --name vcenter-password \
  --query value \
  --output tsv)
```

### Rotate Credentials Regularly

```bash
# Automate credential rotation
# Create script: /usr/local/bin/rotate-hypersdk-creds.sh

#!/bin/bash
# Update password in vCenter
# Update credentials.env file
sudo sed -i "s/VCENTER_PASSWORD=.*/VCENTER_PASSWORD=$NEW_PASSWORD/" \
  /etc/hypervisord/credentials.env

# Restart daemon
sudo systemctl restart hypervisord
```

## Network Security

### Use HTTPS/TLS

```yaml
web:
  enabled: true
  tls_enabled: true
  tls_cert: "/etc/hypersdk/tls/fullchain.pem"
  tls_key: "/etc/hypersdk/tls/privkey.pem"
  tls_min_version: "1.3"
  tls_ciphers:
    - "TLS_AES_256_GCM_SHA384"
    - "TLS_CHACHA20_POLY1305_SHA256"
    - "TLS_AES_128_GCM_SHA256"
```

### Generate Self-Signed Certificate

```bash
# Generate certificate
sudo mkdir -p /etc/hypersdk/tls
cd /etc/hypersdk/tls

sudo openssl req -x509 -nodes -days 365 -newkey rsa:4096 \
  -keyout privkey.pem \
  -out fullchain.pem \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=hypersdk.example.com"

sudo chmod 600 privkey.pem
sudo chmod 644 fullchain.pem
```

### Use Let's Encrypt

```bash
# Install certbot
sudo dnf install certbot

# Get certificate
sudo certbot certonly --standalone \
  -d hypersdk.example.com \
  --email admin@example.com

# Link certificates
sudo ln -s /etc/letsencrypt/live/hypersdk.example.com/fullchain.pem \
  /etc/hypersdk/tls/fullchain.pem
sudo ln -s /etc/letsencrypt/live/hypersdk.example.com/privkey.pem \
  /etc/hypersdk/tls/privkey.pem

# Auto-renewal
sudo systemctl enable --now certbot-renew.timer
```

### Firewall Configuration

```bash
# Fedora/RHEL/CentOS
sudo firewall-cmd --permanent --zone=public --add-service=https
sudo firewall-cmd --permanent --zone=public --add-port=8443/tcp
sudo firewall-cmd --reload

# Deny by default, allow specific IPs
sudo firewall-cmd --permanent --zone=public --remove-service=https
sudo firewall-cmd --permanent --zone=public --add-rich-rule='
  rule family="ipv4"
  source address="192.168.1.0/24"
  port protocol="tcp" port="8443" accept'

# Ubuntu/Debian (UFW)
sudo ufw allow from 192.168.1.0/24 to any port 8443

# iptables
sudo iptables -A INPUT -p tcp -s 192.168.1.0/24 --dport 8443 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8443 -j DROP
```

### Restrict API Access

```yaml
daemon:
  addr: "127.0.0.1:8080"  # Only localhost
  # Or specific interface
  addr: "192.168.1.100:8080"
```

### Use Reverse Proxy

```nginx
# nginx configuration
server {
    listen 443 ssl http2;
    server_name hypersdk.example.com;

    ssl_certificate /etc/letsencrypt/live/hypersdk.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/hypersdk.example.com/privkey.pem;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20;

    # Proxy to HyperSDK
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket support
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Access Control

### Implement Authentication

```yaml
# Example: API key authentication
daemon:
  auth_enabled: true
  api_keys:
    - name: "automation"
      key: "${API_KEY_AUTOMATION}"
      permissions: ["read", "write"]

    - name: "monitoring"
      key: "${API_KEY_MONITORING}"
      permissions: ["read"]
```

### Use RBAC (Role-Based Access Control)

```yaml
# Define roles
roles:
  - name: "admin"
    permissions: ["*"]

  - name: "operator"
    permissions: ["jobs.submit", "jobs.query", "jobs.cancel"]

  - name: "viewer"
    permissions: ["jobs.query", "status"]

# Assign roles to users
users:
  - username: "admin"
    role: "admin"

  - username: "ops-team"
    role: "operator"
```

### Limit vSphere Permissions

Create dedicated service account with minimal permissions:

```
vSphere Permissions Required:
- Virtual Machine.Provisioning.Allow disk access
- Virtual Machine.Provisioning.Allow read-only disk access
- Virtual Machine.State.Create snapshot
- Datastore.Browse datastore
- Datastore.Low level file operations
```

### Use Read-Only Mode

```yaml
export:
  read_only: true  # Prevent exports from modifying source VMs
```

## Data Protection

### Encryption at Rest

```bash
# Encrypt export directory
sudo cryptsetup luksFormat /dev/sdb1
sudo cryptsetup luksOpen /dev/sdb1 exports
sudo mkfs.ext4 /dev/mapper/exports
sudo mount /dev/mapper/exports /exports

# Auto-mount with key file
sudo dd if=/dev/urandom of=/root/exports.key bs=1024 count=4
sudo chmod 600 /root/exports.key
sudo cryptsetup luksAddKey /dev/sdb1 /root/exports.key

# Add to /etc/crypttab
exports /dev/sdb1 /root/exports.key luks

# Add to /etc/fstab
/dev/mapper/exports /exports ext4 defaults 0 2
```

### Encryption in Transit

```yaml
vsphere:
  insecure: false  # Enforce SSL/TLS

web:
  tls_enabled: true
```

### Encrypt Exports

```bash
# Encrypt OVA with GPG
gpg --encrypt --recipient admin@example.com vm.ova

# Decrypt
gpg --decrypt vm.ova.gpg > vm.ova

# Or use openssl
openssl enc -aes-256-cbc -salt -in vm.ova -out vm.ova.enc
openssl enc -aes-256-cbc -d -in vm.ova.enc -out vm.ova
```

### Secure Deletion

```bash
# Securely delete exports after migration
shred -vfz -n 3 /exports/vm.ova

# Or use wipe
wipe -rf /exports/old-vms/
```

## Audit Logging

### Enable Audit Logging

```yaml
logging:
  audit_enabled: true
  audit_file: "/var/log/hypersdk/audit.log"
  audit_level: "info"
  audit_format: "json"
```

### Log All API Requests

```yaml
daemon:
  log_requests: true
  log_request_body: false  # Don't log sensitive data
  log_response_body: false
```

### Centralized Logging

```bash
# Forward logs to syslog
rsyslog configuration:

# /etc/rsyslog.d/hypersdk.conf
if $programname == 'hypervisord' then {
  action(type="omfwd" target="syslog.example.com" port="514" protocol="tcp")
  stop
}

# Or use journald forwarding
sudo systemctl edit systemd-journald

[Journal]
ForwardToSyslog=yes
```

### Monitor for Security Events

```bash
# Create alerts for suspicious activity
grep "authentication failed" /var/log/hypersdk/audit.log

# Monitor for unauthorized API access
grep "401 Unauthorized" /var/log/hypersdk/access.log

# Alert on configuration changes
grep "config changed" /var/log/hypersdk/audit.log
```

## System Hardening

### SELinux

```bash
# Ensure SELinux is enforcing
sudo setenforce 1
sudo sed -i 's/SELINUX=.*/SELINUX=enforcing/' /etc/selinux/config

# Create custom policy if needed
sudo ausearch -m avc -ts recent | audit2allow -M hypersdk
sudo semodule -i hypersdk.pp
```

### Systemd Security

```ini
[Service]
# No new privileges
NoNewPrivileges=true

# Private temp
PrivateTmp=true

# Protect system
ProtectSystem=strict
ReadWritePaths=/var/lib/hypersdk /var/log/hypersdk /exports

# Protect home
ProtectHome=true

# Protect kernel
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Restrict namespaces
RestrictNamespaces=true

# Restrict realtime
RestrictRealtime=true

# Capabilities
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

# System call filter
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources
SystemCallErrorNumber=EPERM
```

### Disable Unnecessary Services

```bash
# Disable web dashboard if not needed
systemctl stop hypersdk-web
systemctl disable hypersdk-web

# Or in config
web:
  enabled: false
```

### Regular Updates

```bash
# Enable automatic security updates
sudo dnf install dnf-automatic
sudo systemctl enable --now dnf-automatic.timer

# For HyperSDK
sudo dnf update hypersdk
```

## Compliance

### PCI-DSS Compliance

- Encrypt cardholder data at rest and in transit
- Implement strong access control measures
- Regularly monitor and test networks
- Maintain audit trails

```yaml
# Strong encryption
web:
  tls_min_version: "1.2"

# Audit logging
logging:
  audit_enabled: true

# Access control
daemon:
  auth_enabled: true
```

### HIPAA Compliance

- Encrypt PHI data
- Implement access controls
- Audit logs for all PHI access
- Secure communication channels

### GDPR Compliance

- Data minimization
- Encryption
- Right to erasure
- Audit trails

```bash
# Implement data retention policy
find /exports -type f -mtime +90 -delete

# Secure deletion
shred -vfz /exports/vm-with-pii.ova
```

## Security Checklist

- [ ] Credentials stored securely (not in config files)
- [ ] TLS/HTTPS enabled for web dashboard
- [ ] Firewall configured to restrict access
- [ ] SELinux/AppArmor enforcing
- [ ] Systemd service hardened
- [ ] Audit logging enabled
- [ ] Regular security updates applied
- [ ] Exports encrypted if containing sensitive data
- [ ] Old exports securely deleted
- [ ] Access control implemented
- [ ] Network segmentation in place
- [ ] Intrusion detection configured
- [ ] Security monitoring active
- [ ] Incident response plan documented

## Security Incident Response

### If Credentials Compromised

1. **Immediately rotate credentials**
2. **Check audit logs for unauthorized access**
3. **Revoke old credentials**
4. **Investigate breach scope**
5. **Update security policies**

### If System Compromised

1. **Isolate affected system**
2. **Preserve evidence**
3. **Analyze logs**
4. **Rebuild from known-good state**
5. **Update security measures**

## See Also

- [Configuration Reference](configuration-reference.md)
- [Installation Guide](installation-guide.md)
- [Audit Logging Reference](audit-logging.md)
- [Compliance Guide](compliance-guide.md)
