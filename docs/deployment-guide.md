# HyperSDK Deployment Guide

Complete guide for deploying HyperSDK across different platforms and environments.

## Quick Start

Get HyperSDK running in under 5 minutes:

### Docker/Podman (Development)

```bash
# Build and start
./deployments/scripts/build-images.sh --builder podman
cd deployments/docker && podman compose up -d

# Access
open http://localhost:8080/web/dashboard/
```

### Kubernetes (Production)

```bash
# Configure and deploy
cp deployments/kubernetes/base/secrets.yaml.example \
   deployments/kubernetes/overlays/development/secrets.yaml
./deployments/scripts/deploy-k8s.sh development

# Access
kubectl port-forward -n hypersdk svc/hypervisord 8080:8080
```

## Deployment Methods

HyperSDK supports three deployment methods:

1. **Docker/Podman** - Best for development and testing
2. **Kubernetes** - Best for production with HA and scaling
3. **Systemd** - Best for traditional Linux deployments

See detailed guides:
- [Docker/Podman Guide](../deployments/docker/README.md)
- [Kubernetes Guide](../deployments/kubernetes/README.md)
- [Systemd Guide](../systemd/README.md)

## Key Features

All deployment methods include:
- Security hardening (non-root, RBAC, NetworkPolicy)
- Prometheus metrics and Grafana dashboards
- Health checks and auto-recovery
- Volume management for persistence
- Multi-cloud provider support
- Complete documentation

For detailed deployment instructions, see the comprehensive guides in the `deployments/` directory.
