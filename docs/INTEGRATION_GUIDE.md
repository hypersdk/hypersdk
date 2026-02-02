# HyperSDK Integration Guide

Complete guide for integrating HyperSDK with popular tools and platforms.

## 📋 Table of Contents

- [CI/CD Integration](#cicd-integration)
- [Configuration Management](#configuration-management)
- [Monitoring & Alerting](#monitoring--alerting)
- [Container Orchestration](#container-orchestration)
- [Cloud Platforms](#cloud-platforms)

---

## 🔄 CI/CD Integration

### Jenkins Pipeline

**Jenkinsfile** for automated VM exports:

```groovy
pipeline {
    agent any

    environment {
        HYPERSDK_API = 'http://hypersdk-server:8080'
        VM_PATH = '/datacenter/vm/build-agent-01'
        OUTPUT_PATH = '/exports/jenkins-builds'
    }

    stages {
        stage('Prepare') {
            steps {
                script {
                    // Check HyperSDK health
                    sh """
                        curl -f ${HYPERSDK_API}/health || exit 1
                    """
                }
            }
        }

        stage('Export VM') {
            steps {
                script {
                    // Submit export job
                    def response = sh(
                        script: """
                            curl -s -X POST ${HYPERSDK_API}/jobs/submit \\
                                -H 'Content-Type: application/json' \\
                                -d '{
                                    "vm_path": "${VM_PATH}",
                                    "output_path": "${OUTPUT_PATH}",
                                    "format": "ova",
                                    "compression": true
                                }'
                        """,
                        returnStdout: true
                    ).trim()

                    def json = readJSON text: response
                    env.JOB_ID = json.job_ids[0]

                    echo "Export job submitted: ${env.JOB_ID}"
                }
            }
        }

        stage('Monitor Progress') {
            steps {
                script {
                    // Poll for completion
                    timeout(time: 2, unit: 'HOURS') {
                        waitUntil {
                            def status = sh(
                                script: """
                                    curl -s ${HYPERSDK_API}/jobs/${env.JOB_ID} | \\
                                    jq -r '.status'
                                """,
                                returnStdout: true
                            ).trim()

                            echo "Job status: ${status}"

                            if (status == 'failed') {
                                error("Export job failed")
                            }

                            return status == 'completed'
                        }
                    }
                }
            }
        }

        stage('Verify Export') {
            steps {
                sh """
                    # Verify export exists
                    test -f ${OUTPUT_PATH}/*.ovf || exit 1
                    echo "✅ Export completed successfully"
                """
            }
        }
    }

    post {
        success {
            echo '✅ VM export pipeline completed successfully'
        }
        failure {
            echo '❌ VM export pipeline failed'
        }
    }
}
```

**Shared Library** (`vars/hypersdkExport.groovy`):

```groovy
def call(Map config) {
    def apiUrl = config.apiUrl ?: 'http://localhost:8080'
    def vmPath = config.vmPath
    def outputPath = config.outputPath
    def format = config.format ?: 'ova'

    // Submit job
    def response = sh(
        script: """
            curl -s -X POST ${apiUrl}/jobs/submit \\
                -H 'Content-Type: application/json' \\
                -d '{
                    "vm_path": "${vmPath}",
                    "output_path": "${outputPath}",
                    "format": "${format}",
                    "compression": true,
                    "verify": true
                }'
        """,
        returnStdout: true
    )

    def json = readJSON text: response
    def jobId = json.job_ids[0]

    // Wait for completion
    timeout(time: 3, unit: 'HOURS') {
        waitUntil {
            def job = sh(
                script: "curl -s ${apiUrl}/jobs/${jobId}",
                returnStdout: true
            )
            def jobJson = readJSON text: job
            return jobJson.status == 'completed' || jobJson.status == 'failed'
        }
    }

    return jobId
}
```

**Usage**:
```groovy
// In Jenkinsfile
hypersdkExport(
    vmPath: '/datacenter/vm/my-vm',
    outputPath: '/exports/build-${BUILD_NUMBER}'
)
```

---

### GitLab CI/CD

**.gitlab-ci.yml** for VM backup automation:

```yaml
variables:
  HYPERSDK_API: "http://hypersdk-server:8080"
  VM_PATH: "/datacenter/vm/gitlab-runner"
  OUTPUT_PATH: "/exports/gitlab-ci"

stages:
  - health-check
  - export
  - verify
  - cleanup

health_check:
  stage: health-check
  script:
    - curl -f $HYPERSDK_API/health
  only:
    - schedules

export_vm:
  stage: export
  script:
    - |
      RESPONSE=$(curl -s -X POST $HYPERSDK_API/jobs/submit \
        -H 'Content-Type: application/json' \
        -d "{
          \"vm_path\": \"$VM_PATH\",
          \"output_path\": \"$OUTPUT_PATH/$CI_PIPELINE_ID\",
          \"format\": \"ova\",
          \"compression\": true
        }")
      export JOB_ID=$(echo $RESPONSE | jq -r '.job_ids[0]')
      echo $JOB_ID > job_id.txt
      echo "Export job: $JOB_ID"
  artifacts:
    paths:
      - job_id.txt
    expire_in: 1 day
  only:
    - schedules

monitor_export:
  stage: export
  dependencies:
    - export_vm
  script:
    - |
      JOB_ID=$(cat job_id.txt)
      echo "Monitoring job: $JOB_ID"

      while true; do
        STATUS=$(curl -s $HYPERSDK_API/jobs/$JOB_ID | jq -r '.status')
        echo "Status: $STATUS"

        if [ "$STATUS" = "completed" ]; then
          echo "✅ Export completed"
          exit 0
        elif [ "$STATUS" = "failed" ]; then
          echo "❌ Export failed"
          exit 1
        fi

        sleep 30
      done
  timeout: 3h
  only:
    - schedules

verify_export:
  stage: verify
  dependencies:
    - export_vm
  script:
    - |
      JOB_ID=$(cat job_id.txt)
      JOB=$(curl -s $HYPERSDK_API/jobs/$JOB_ID)
      OVF_PATH=$(echo $JOB | jq -r '.result.ovf_path')
      echo "Verifying: $OVF_PATH"
      test -f "$OVF_PATH" || exit 1
      echo "✅ Export verified"
  only:
    - schedules

cleanup_old_exports:
  stage: cleanup
  script:
    - find /exports/gitlab-ci -type d -mtime +7 -exec rm -rf {} +
  only:
    - schedules
  when: always
```

**Scheduled Pipeline**:
```yaml
# In GitLab UI: CI/CD > Schedules
# Or via API:
curl -X POST "https://gitlab.com/api/v4/projects/:id/pipeline_schedules" \
  --header "PRIVATE-TOKEN: <token>" \
  --data "description=Daily VM Backup" \
  --data "ref=main" \
  --data "cron=0 2 * * *"
```

---

### GitHub Actions

**.github/workflows/vm-export.yml**:

```yaml
name: VM Export

on:
  schedule:
    - cron: '0 2 * * 0'  # Weekly on Sunday at 2 AM
  workflow_dispatch:     # Manual trigger

env:
  HYPERSDK_API: http://hypersdk-server:8080
  VM_PATH: /datacenter/vm/github-actions-runner
  OUTPUT_PATH: /exports/github-actions

jobs:
  export:
    runs-on: ubuntu-latest
    timeout-minutes: 180

    steps:
      - name: Check HyperSDK Health
        run: |
          curl -f $HYPERSDK_API/health

      - name: Submit Export Job
        id: submit
        run: |
          RESPONSE=$(curl -s -X POST $HYPERSDK_API/jobs/submit \
            -H 'Content-Type: application/json' \
            -d "{
              \"vm_path\": \"$VM_PATH\",
              \"output_path\": \"$OUTPUT_PATH/${{ github.run_id }}\",
              \"format\": \"ova\",
              \"compression\": true
            }")

          JOB_ID=$(echo $RESPONSE | jq -r '.job_ids[0]')
          echo "job_id=$JOB_ID" >> $GITHUB_OUTPUT
          echo "Export job submitted: $JOB_ID"

      - name: Monitor Export Progress
        run: |
          JOB_ID=${{ steps.submit.outputs.job_id }}

          while true; do
            JOB=$(curl -s $HYPERSDK_API/jobs/$JOB_ID)
            STATUS=$(echo $JOB | jq -r '.status')
            PROGRESS=$(echo $JOB | jq -r '.progress.percent_complete // 0')

            echo "Status: $STATUS, Progress: ${PROGRESS}%"

            if [ "$STATUS" = "completed" ]; then
              echo "✅ Export completed"
              exit 0
            elif [ "$STATUS" = "failed" ]; then
              ERROR=$(echo $JOB | jq -r '.error')
              echo "❌ Export failed: $ERROR"
              exit 1
            fi

            sleep 30
          done

      - name: Verify Export
        run: |
          JOB=$(curl -s $HYPERSDK_API/jobs/${{ steps.submit.outputs.job_id }})
          OVF_PATH=$(echo $JOB | jq -r '.result.ovf_path')

          if [ -f "$OVF_PATH" ]; then
            echo "✅ Export verified: $OVF_PATH"
          else
            echo "❌ Export file not found"
            exit 1
          fi

      - name: Notify on Failure
        if: failure()
        uses: actions/github-script@v6
        with:
          script: |
            github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: 'VM Export Failed',
              body: `VM export failed in workflow run ${context.runId}`
            })
```

---

## ⚙️ Configuration Management

### Ansible Playbook

**playbook.yml** for HyperSDK deployment and VM export:

```yaml
---
- name: Deploy HyperSDK and Export VMs
  hosts: backup_servers
  become: yes
  vars:
    hypersdk_version: "latest"
    hypersdk_port: 8080
    vcenter_url: "https://vcenter.example.com/sdk"
    vcenter_username: "admin@vsphere.local"
    export_path: "/mnt/backups"

  tasks:
    - name: Install Docker
      package:
        name: docker
        state: present

    - name: Start Docker service
      service:
        name: docker
        state: started
        enabled: yes

    - name: Pull HyperSDK image
      docker_image:
        name: "hypersdk/hypervisord"
        tag: "{{ hypersdk_version }}"
        source: pull

    - name: Create export directory
      file:
        path: "{{ export_path }}"
        state: directory
        mode: '0755'

    - name: Run HyperSDK container
      docker_container:
        name: hypervisord
        image: "hypersdk/hypervisord:{{ hypersdk_version }}"
        state: started
        restart_policy: always
        ports:
          - "{{ hypersdk_port }}:8080"
        volumes:
          - "{{ export_path }}:/exports"
        env:
          GOVC_URL: "{{ vcenter_url }}"
          GOVC_USERNAME: "{{ vcenter_username }}"
          GOVC_PASSWORD: "{{ vcenter_password }}"
          GOVC_INSECURE: "1"

    - name: Wait for HyperSDK to be ready
      uri:
        url: "http://localhost:{{ hypersdk_port }}/health"
        status_code: 200
      register: result
      until: result.status == 200
      retries: 30
      delay: 2

    - name: Export VMs
      uri:
        url: "http://localhost:{{ hypersdk_port }}/jobs/submit"
        method: POST
        body_format: json
        body:
          vm_path: "{{ item.path }}"
          output_path: "{{ export_path }}/{{ item.name }}"
          format: "ova"
          compression: true
      loop: "{{ vms_to_export }}"
      register: export_jobs

    - name: Display export job IDs
      debug:
        msg: "Exported {{ item.item.name }}: {{ item.json.job_ids[0] }}"
      loop: "{{ export_jobs.results }}"
```

**inventory.ini**:
```ini
[backup_servers]
backup-01 ansible_host=192.168.1.10
backup-02 ansible_host=192.168.1.11

[backup_servers:vars]
ansible_user=admin
ansible_python_interpreter=/usr/bin/python3
```

**vars.yml**:
```yaml
vms_to_export:
  - name: web-server-01
    path: /datacenter/vm/production/web-01
  - name: database-01
    path: /datacenter/vm/production/db-01
  - name: app-server-01
    path: /datacenter/vm/production/app-01
```

**Run**:
```bash
ansible-playbook -i inventory.ini playbook.yml \
  -e @vars.yml \
  -e vcenter_password='secret'
```

---

### Terraform

**main.tf** for HyperSDK infrastructure:

```hcl
terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {
  host = "unix:///var/run/docker.sock"
}

resource "docker_image" "hypervisord" {
  name         = "hypersdk/hypervisord:latest"
  keep_locally = false
}

resource "docker_volume" "exports" {
  name = "hypersdk-exports"
}

resource "docker_container" "hypervisord" {
  name  = "hypervisord"
  image = docker_image.hypervisord.image_id

  restart = "always"

  ports {
    internal = 8080
    external = 8080
  }

  volumes {
    volume_name    = docker_volume.exports.name
    container_path = "/exports"
  }

  env = [
    "GOVC_URL=${var.vcenter_url}",
    "GOVC_USERNAME=${var.vcenter_username}",
    "GOVC_PASSWORD=${var.vcenter_password}",
    "GOVC_INSECURE=1"
  ]

  healthcheck {
    test     = ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval = "30s"
    timeout  = "3s"
    retries  = 3
  }
}

resource "null_resource" "export_vms" {
  depends_on = [docker_container.hypervisord]

  provisioner "local-exec" {
    command = <<-EOT
      for vm in ${join(" ", var.vms_to_export)}; do
        curl -X POST http://localhost:8080/jobs/submit \
          -H 'Content-Type: application/json' \
          -d "{\"vm_path\": \"$vm\", \"output_path\": \"/exports\"}"
      done
    EOT
  }
}

output "api_url" {
  value = "http://localhost:8080"
}

output "export_volume" {
  value = docker_volume.exports.name
}
```

**variables.tf**:
```hcl
variable "vcenter_url" {
  description = "vCenter SDK URL"
  type        = string
}

variable "vcenter_username" {
  description = "vCenter username"
  type        = string
}

variable "vcenter_password" {
  description = "vCenter password"
  type        = string
  sensitive   = true
}

variable "vms_to_export" {
  description = "List of VM paths to export"
  type        = list(string)
  default     = []
}
```

**Usage**:
```bash
terraform init
terraform plan \
  -var="vcenter_url=https://vcenter.example.com/sdk" \
  -var="vcenter_username=admin@vsphere.local" \
  -var="vcenter_password=secret"
terraform apply
```

---

## 📊 Monitoring & Alerting

### Prometheus Monitoring

**prometheus.yml**:
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'hypersdk'
    static_configs:
      - targets: ['hypersdk-server:8080']
    metrics_path: '/metrics'  # Future feature
```

**Custom Exporter** (Python script):
```python
#!/usr/bin/env python3
"""
HyperSDK Prometheus Exporter

Exports HyperSDK metrics to Prometheus format.
"""

from prometheus_client import start_http_server, Gauge, Counter
from hypersdk import HyperSDK
import time

# Metrics
jobs_total = Counter('hypersdk_jobs_total', 'Total jobs submitted')
jobs_completed = Counter('hypersdk_jobs_completed', 'Total jobs completed')
jobs_failed = Counter('hypersdk_jobs_failed', 'Total jobs failed')
jobs_running = Gauge('hypersdk_jobs_running', 'Currently running jobs')
export_size_bytes = Gauge('hypersdk_export_size_bytes', 'Total bytes exported')

def collect_metrics(client):
    """Collect metrics from HyperSDK"""
    jobs = client.list_jobs(all=True)

    running_count = 0
    total_size = 0

    for job in jobs:
        if job['status'] == 'running':
            running_count += 1
        elif job['status'] == 'completed':
            jobs_completed.inc()
            if 'result' in job:
                total_size += job['result'].get('total_size_bytes', 0)
        elif job['status'] == 'failed':
            jobs_failed.inc()

    jobs_running.set(running_count)
    export_size_bytes.set(total_size)

if __name__ == '__main__':
    # Start Prometheus metrics server
    start_http_server(9090)

    # Connect to HyperSDK
    client = HyperSDK('http://localhost:8080')

    # Collect metrics every 30 seconds
    while True:
        try:
            collect_metrics(client)
        except Exception as e:
            print(f"Error collecting metrics: {e}")
        time.sleep(30)
```

---

### Grafana Dashboard

**dashboard.json** (import to Grafana):
```json
{
  "dashboard": {
    "title": "HyperSDK Monitoring",
    "panels": [
      {
        "title": "Running Jobs",
        "targets": [
          {
            "expr": "hypersdk_jobs_running"
          }
        ],
        "type": "stat"
      },
      {
        "title": "Job Success Rate",
        "targets": [
          {
            "expr": "rate(hypersdk_jobs_completed[5m]) / rate(hypersdk_jobs_total[5m])"
          }
        ],
        "type": "gauge"
      },
      {
        "title": "Total Data Exported",
        "targets": [
          {
            "expr": "hypersdk_export_size_bytes"
          }
        ],
        "type": "stat",
        "format": "bytes"
      }
    ]
  }
}
```

---

## 🐳 Container Orchestration

### Docker Compose

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  hypervisord:
    image: hypersdk/hypervisord:latest
    container_name: hypervisord
    restart: always
    ports:
      - "8080:8080"
    volumes:
      - exports:/exports
      - ./config.yaml:/etc/hypervisord/config.yaml:ro
    environment:
      - GOVC_URL=${GOVC_URL}
      - GOVC_USERNAME=${GOVC_USERNAME}
      - GOVC_PASSWORD=${GOVC_PASSWORD}
      - GOVC_INSECURE=1
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
    networks:
      - hypersdk-net

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - hypersdk-net

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    networks:
      - hypersdk-net

volumes:
  exports:
  prometheus-data:
  grafana-data:

networks:
  hypersdk-net:
    driver: bridge
```

**.env**:
```bash
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=admin@vsphere.local
GOVC_PASSWORD=your-password
```

**Run**:
```bash
docker-compose up -d
```

---

### Kubernetes

**deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hypervisord
  labels:
    app: hypervisord
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hypervisord
  template:
    metadata:
      labels:
        app: hypervisord
    spec:
      containers:
      - name: hypervisord
        image: hypersdk/hypervisord:latest
        ports:
        - containerPort: 8080
        env:
        - name: GOVC_URL
          valueFrom:
            secretKeyRef:
              name: vcenter-credentials
              key: url
        - name: GOVC_USERNAME
          valueFrom:
            secretKeyRef:
              name: vcenter-credentials
              key: username
        - name: GOVC_PASSWORD
          valueFrom:
            secretKeyRef:
              name: vcenter-credentials
              key: password
        volumeMounts:
        - name: exports
          mountPath: /exports
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
      volumes:
      - name: exports
        persistentVolumeClaim:
          claimName: hypersdk-exports-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: hypervisord
spec:
  selector:
    app: hypervisord
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-exports-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
```

**Create secret**:
```bash
kubectl create secret generic vcenter-credentials \
  --from-literal=url='https://vcenter.example.com/sdk' \
  --from-literal=username='admin@vsphere.local' \
  --from-literal=password='your-password'
```

---

## ☁️ Cloud Platforms

### AWS ECS

**task-definition.json**:
```json
{
  "family": "hypervisord",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "2048",
  "memory": "4096",
  "containerDefinitions": [
    {
      "name": "hypervisord",
      "image": "hypersdk/hypervisord:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "GOVC_URL",
          "value": "https://vcenter.example.com/sdk"
        },
        {
          "name": "GOVC_USERNAME",
          "value": "admin@vsphere.local"
        }
      ],
      "secrets": [
        {
          "name": "GOVC_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:vcenter-password"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/hypervisord",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "mountPoints": [
        {
          "sourceVolume": "exports",
          "containerPath": "/exports"
        }
      ]
    }
  ],
  "volumes": [
    {
      "name": "exports",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-12345678",
        "transitEncryption": "ENABLED"
      }
    }
  ]
}
```

---

## 📚 Additional Resources

- [Quick Start Guide](QUICK_START.md)
- [API Reference](API_ENDPOINTS.md)
- [Examples](../examples/)
- [FAQ](FAQ.md)

---

*Last Updated: 2026-02-04*
*For more integration examples, see the [examples directory](../examples/)*
