# Deployment Scripts

Automation scripts for deploying and managing HyperSDK across different environments.

## Available Scripts

### build-images.sh

Build all HyperSDK container images.

**Usage:**
```bash
./build-images.sh [OPTIONS]
```

**Options:**
- `--builder BUILDER` - Container builder: docker or podman (default: docker)
- `--version VERSION` - Version tag (default: from git describe)
- `--registry REGISTRY` - Container registry URL
- `--push` - Push images to registry after building
- `--no-cache` - Build without cache

**Examples:**
```bash
# Build with default settings
./build-images.sh

# Build with specific version
./build-images.sh --version v0.2.0

# Build and push to registry
./build-images.sh --version v0.2.0 --registry ghcr.io/ssahani --push

# Build with Podman
./build-images.sh --builder podman
```

### deploy-k8s.sh

Deploy HyperSDK to Kubernetes cluster using Kustomize.

**Usage:**
```bash
./deploy-k8s.sh [ENVIRONMENT]
```

**Environments:**
- `development` - Development environment with lower resources
- `staging` - Staging environment with ingress and autoscaling
- `production` - Production environment with full features

**Examples:**
```bash
# Deploy to development
./deploy-k8s.sh development

# Deploy to production
./deploy-k8s.sh production
```

**Prerequisites:**
- kubectl configured with cluster access
- Secrets configured in overlay directory

### deploy-k3d.sh

Create k3d cluster and deploy HyperSDK automatically.

**Usage:**
```bash
./deploy-k3d.sh [OPTIONS]
```

**Options:**
- `-n, --name NAME` - Cluster name (default: hypersdk)
- `-a, --agents NUM` - Number of agent nodes (default: 2)
- `-e, --env ENV` - Environment: dev, staging, prod (default: dev)
- `-m, --monitoring` - Deploy monitoring stack
- `-r, --registry` - Create local registry
- `-p, --port PORT` - LoadBalancer port (default: 8080)
- `-v, --volume PATH` - Mount host volume for exports
- `-d, --delete` - Delete existing cluster first
- `-h, --help` - Show help message

**Examples:**
```bash
# Quick start with defaults
./deploy-k3d.sh

# Create cluster with monitoring
./deploy-k3d.sh --monitoring

# Create cluster with custom name and 3 agents
./deploy-k3d.sh --name test --agents 3

# Create cluster with local registry
./deploy-k3d.sh --registry

# Delete and recreate with volume mount
./deploy-k3d.sh --delete --volume /data/exports

# Deploy staging environment
./deploy-k3d.sh --env staging --monitoring
```

**What it does:**
1. Checks dependencies (k3d, kubectl, docker)
2. Creates k3d cluster with specified configuration
3. Deploys HyperSDK to the cluster
4. Creates secrets (with placeholder values)
5. Waits for deployment to be ready
6. Shows access information and quick reference commands

**Output:**
- Cluster kubeconfig path
- LoadBalancer IP and access URLs
- Quick reference commands
- Pod status and logs

### test-k3d.sh

Validate HyperSDK deployment on k3d cluster.

**Usage:**
```bash
./test-k3d.sh [OPTIONS]
```

**Options:**
- `-n, --name NAME` - Cluster name (default: hypersdk)
- `-o, --output FILE` - Save test results to file
- `-v, --verbose` - Verbose output
- `-h, --help` - Show help message

**Examples:**
```bash
# Run all tests
./test-k3d.sh

# Test specific cluster
./test-k3d.sh --name my-cluster

# Run tests with verbose output and save results
./test-k3d.sh --verbose --output results.md
```

**Tests performed:**
1. Cluster exists and is accessible
2. Namespace exists
3. Deployment exists
4. Pod is running and ready
5. Services are created
6. PVCs are bound
7. ConfigMap and Secrets exist
8. RBAC is configured
9. Health endpoint responds
10. Status endpoint returns valid JSON
11. Capabilities endpoint shows web available

**Output:**
- Test results summary (passed/failed/total)
- Success rate percentage
- Detailed test output
- Cluster information (if verbose)
- Access URLs
- Optional: Test results saved to file

### health-check.sh

Perform health checks across different deployment environments.

**Usage:**
```bash
./health-check.sh [ENVIRONMENT] [OPTIONS]
```

**Environments:**
- `docker` - Check Docker Compose deployment
- `podman` - Check Podman deployment
- `kubernetes` - Check Kubernetes deployment

**Options:**
- `--namespace NAMESPACE` - Kubernetes namespace (default: hypersdk)
- `--timeout SECONDS` - Timeout for checks (default: 30)

**Examples:**
```bash
# Check Docker deployment
./health-check.sh docker

# Check Kubernetes deployment
./health-check.sh kubernetes --namespace hypersdk

# Check with custom timeout
./health-check.sh kubernetes --timeout 60
```

## Complete Workflows

### Local Development with k3d

```bash
# 1. Create k3d cluster and deploy
./deploy-k3d.sh --name dev --agents 2 --volume /tmp/exports

# 2. Update credentials
kubectl edit secret vsphere-credentials -n hypersdk

# 3. Run tests
./test-k3d.sh --name dev --verbose

# 4. Build and test local changes
cd ../..
./deployments/scripts/build-images.sh --builder docker
k3d image import hypersdk/hypervisord:latest -c dev
kubectl rollout restart deployment/hypervisord -n hypersdk

# 5. Run tests again
./deployments/scripts/test-k3d.sh --name dev

# 6. Cleanup when done
k3d cluster delete dev
```

### CI/CD Testing

```bash
# Create ephemeral cluster for testing
./deploy-k3d.sh --name ci-test --agents 2

# Run test suite
./test-k3d.sh --name ci-test --output ci-results.md

# Check exit code
if [ $? -eq 0 ]; then
  echo "Tests passed"
else
  echo "Tests failed"
  exit 1
fi

# Cleanup
k3d cluster delete ci-test
```

### Production Kubernetes Deployment

```bash
# 1. Build and push images
./build-images.sh --version v1.0.0 --registry ghcr.io/myorg --push

# 2. Update image tags in kustomization
cd ../kubernetes/overlays/production
kustomize edit set image hypersdk/hypervisord=ghcr.io/myorg/hypervisord:v1.0.0

# 3. Deploy to production
cd ../../..
./scripts/deploy-k8s.sh production

# 4. Monitor rollout
kubectl rollout status deployment/hypervisord -n hypersdk

# 5. Run health checks
./scripts/health-check.sh kubernetes --namespace hypersdk

# 6. Verify with smoke tests
kubectl run -it --rm test --image=curlimages/curl --restart=Never \
  -- curl -f http://hypervisord.hypersdk.svc.cluster.local:8080/health
```

### Multi-Environment Testing

```bash
# Test all environments in k3d
for env in development staging production; do
  echo "Testing $env environment..."

  # Create cluster
  ./deploy-k3d.sh --name "test-$env" --env "$env" --agents 2

  # Run tests
  ./test-k3d.sh --name "test-$env" --output "results-$env.md"

  # Cleanup
  k3d cluster delete "test-$env"
done

# Compare results
cat results-*.md
```

## Prerequisites

### All Scripts

- `bash` 4.0+
- `git` (for version detection)

### build-images.sh

- `docker` 20.10+ or `podman` 4.0+
- Go 1.24+ (if building from source)

### deploy-k8s.sh

- `kubectl` 1.24+
- Configured kubeconfig with cluster access
- `kustomize` (built into kubectl 1.14+)

### deploy-k3d.sh

- `k3d` 5.0+
- `kubectl` 1.24+
- `docker` 20.10+
- `jq` (for JSON parsing)

### test-k3d.sh

- `k3d` 5.0+
- `kubectl` 1.24+
- `curl`
- `jq`

### health-check.sh

- `curl`
- `jq`
- For Docker: `docker` or `podman`
- For Kubernetes: `kubectl`

## Installation

### Install k3d

```bash
# Linux/macOS
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# macOS with Homebrew
brew install k3d

# Windows with Chocolatey
choco install k3d
```

### Install kubectl

```bash
# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# macOS with Homebrew
brew install kubectl

# Windows with Chocolatey
choco install kubernetes-cli
```

### Install jq

```bash
# Linux
sudo apt-get install jq  # Debian/Ubuntu
sudo yum install jq      # RHEL/CentOS

# macOS
brew install jq

# Windows
choco install jq
```

## Environment Variables

Scripts support these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `BUILDER` | Container builder (docker/podman) | docker |
| `REGISTRY` | Container registry URL | - |
| `VERSION` | Image version tag | git describe |
| `KUBECONFIG` | Kubernetes config file | ~/.kube/config |
| `NAMESPACE` | Kubernetes namespace | hypersdk |
| `TIMEOUT` | Operation timeout (seconds) | 30 |

**Example:**
```bash
export BUILDER=podman
export REGISTRY=ghcr.io/myorg
./build-images.sh --version v1.0.0 --push
```

## Troubleshooting

### k3d cluster creation fails

```bash
# Check Docker is running
docker ps

# Check port availability
lsof -i :8080
lsof -i :6443

# Delete conflicting resources
k3d cluster delete hypersdk
docker network prune
```

### kubectl cannot connect

```bash
# Refresh kubeconfig
export KUBECONFIG="$(k3d kubeconfig write hypersdk)"
kubectl cluster-info

# Check cluster status
k3d cluster list
k3d cluster start hypersdk  # if stopped
```

### Image pull failures

```bash
# Import image to k3d
docker pull ghcr.io/ssahani/hypersdk-hypervisord:latest
k3d image import ghcr.io/ssahani/hypersdk-hypervisord:latest -c hypersdk

# Or build locally
./build-images.sh
k3d image import hypersdk/hypervisord:latest -c hypersdk
```

### Tests fail

```bash
# Check pod status
kubectl get pods -n hypersdk
kubectl describe pod -n hypersdk -l app=hypervisord

# Check logs
kubectl logs -n hypersdk -l app=hypervisord --tail=100

# Check events
kubectl get events -n hypersdk --sort-by='.lastTimestamp'

# Run verbose tests
./test-k3d.sh --verbose
```

## Contributing

When adding new scripts:

1. Add shebang: `#!/usr/bin/env bash`
2. Use `set -e` for error handling
3. Add help text with usage examples
4. Include colorized output
5. Add to this README
6. Make executable: `chmod +x script.sh`
7. Test on Linux and macOS

## License

All scripts are licensed under LGPL-3.0-or-later, same as HyperSDK.
