# Docker Quick Start Guide

Get HyperSDK running with Docker in under 5 minutes.

## Prerequisites

- Docker 20.10+ or Podman 4.0+
- 4GB RAM
- Internet connection to pull images

## Option 1: Use Pre-built Images (Fastest)

Pull and run pre-built images from GitHub Container Registry:

```bash
# Pull the hypervisord image
docker pull ghcr.io/ssahani/hypersdk-hypervisord:latest

# Create volumes for data persistence
docker volume create hypersdk-data
docker volume create hypersdk-exports

# Run hypervisord
docker run -d \
  --name hypersdk \
  -p 8080:8080 \
  -p 8081:8081 \
  -e LOG_LEVEL=info \
  -e GOVC_URL=https://vcenter.example.com/sdk \
  -e GOVC_USERNAME=administrator@vsphere.local \
  -e GOVC_PASSWORD=your-password \
  -e GOVC_INSECURE=1 \
  -v hypersdk-data:/data \
  -v hypersdk-exports:/exports \
  ghcr.io/ssahani/hypersdk-hypervisord:latest

# Verify it's running
docker ps | grep hypersdk

# Check logs
docker logs -f hypersdk
```

**Access Services:**
- Web Dashboard: http://localhost:8080/web/dashboard/
- API Health: http://localhost:8080/health
- Metrics: http://localhost:8081/metrics

## Option 2: Use Docker Compose (Recommended)

Get the full stack with monitoring:

```bash
# Clone repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk/deployments/docker

# Create environment file
cp .env.example .env

# Edit with your credentials
vim .env
# Update GOVC_URL, GOVC_USERNAME, GOVC_PASSWORD

# Start all services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f hypervisord
```

**Access Services:**
- Dashboard: http://localhost:8080/web/dashboard/
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090

## Option 3: Build from Source

Build your own images:

```bash
# Clone repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Build all images
./deployments/scripts/build-images.sh --builder docker

# Start services
cd deployments/docker
docker compose up -d
```

## Quick Test

Submit a test export job:

```bash
# Check daemon status
curl http://localhost:8080/status

# List available VMs (if vSphere is configured)
curl http://localhost:8080/vms/list

# Submit export job
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm": "/datacenter/vm/test-vm",
    "output": "/exports/test-vm",
    "format": "ova",
    "compress": true
  }'

# Check job status
curl http://localhost:8080/jobs/query
```

## Using CLI Tools

Run CLI tools from containers:

```bash
# Using hyperexport
docker run --rm \
  -e GOVC_URL=$GOVC_URL \
  -e GOVC_USERNAME=$GOVC_USERNAME \
  -e GOVC_PASSWORD=$GOVC_PASSWORD \
  -v ./exports:/exports \
  ghcr.io/ssahani/hypersdk-hyperexport:latest \
  -vm "/datacenter/vm/my-vm" \
  -output /exports/my-vm

# Using hyperctl to query daemon
docker run --rm \
  -e DAEMON_URL=http://host.docker.internal:8080 \
  ghcr.io/ssahani/hypersdk-hyperctl:latest \
  status
```

## Stopping and Cleanup

```bash
# Stop single container
docker stop hypersdk
docker rm hypersdk

# Stop compose stack
docker compose down

# Remove volumes (WARNING: deletes all data)
docker compose down -v

# Remove images
docker rmi ghcr.io/ssahani/hypersdk-hypervisord:latest
docker rmi ghcr.io/ssahani/hypersdk-hyperexport:latest
docker rmi ghcr.io/ssahani/hypersdk-hyperctl:latest
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker logs hypersdk

# Common issues:
# - Missing credentials (GOVC_URL, GOVC_USERNAME, GOVC_PASSWORD)
# - Port already in use
# - Volume permission issues
```

### Cannot connect to vCenter

```bash
# Test connection from container
docker exec hypersdk curl -k $GOVC_URL

# Verify credentials
docker exec hypersdk env | grep GOVC
```

### Check container health

```bash
# View health status
docker inspect hypersdk | grep -A 10 Health

# Check health endpoint
curl http://localhost:8080/health
```

## Next Steps

- [Full Docker/Podman Guide](../deployments/docker/README.md)
- [Kubernetes Deployment](../deployments/kubernetes/README.md)
- [API Documentation](../docs/api/README.md)
- [User Guides](../docs/user-guides/)

## Support

For issues:
- GitHub Issues: https://github.com/ssahani/hypersdk/issues
- Documentation: https://github.com/ssahani/hypersdk/tree/main/docs
