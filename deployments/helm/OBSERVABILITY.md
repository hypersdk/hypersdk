# Observability Stack Integration for HyperSDK

Complete guide for implementing comprehensive observability with logs, metrics, and traces.

## Overview

The three pillars of observability:
- **Metrics** - Time-series data (Prometheus)
- **Logs** - Event records (Loki/ELK)
- **Traces** - Request flows (Jaeger/Tempo)

## Metrics with Prometheus

### Prerequisites

```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace
```

### ServiceMonitor for HyperSDK

Already included in Helm chart, enabled with:

```yaml
# values.yaml
monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s
    labels:
      release: prometheus
```

### Custom Metrics

```yaml
# monitoring/hypersdk-prometheusrule.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: hypersdk
  namespace: hypersdk
  labels:
    release: prometheus
spec:
  groups:
    - name: hypersdk.rules
      interval: 30s
      rules:
        # Job completion rate
        - record: hypersdk:job_completion_rate:5m
          expr: |
            rate(hypersdk_jobs_completed_total[5m])

        # Job failure rate
        - record: hypersdk:job_failure_rate:5m
          expr: |
            rate(hypersdk_jobs_failed_total[5m])

        # Average export duration
        - record: hypersdk:export_duration_seconds:avg
          expr: |
            avg(hypersdk_export_duration_seconds)

        # Active exports by provider
        - record: hypersdk:active_exports:by_provider
          expr: |
            sum(hypersdk_active_jobs) by (provider)

    - name: hypersdk.alerts
      interval: 30s
      rules:
        # High error rate
        - alert: HyperSDKHighErrorRate
          expr: |
            rate(hypersdk_http_requests_total{status=~"5.."}[5m]) > 0.05
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High error rate detected"
            description: "Error rate is {{ $value }} (threshold: 0.05)"

        # Job failures
        - alert: HyperSDKJobFailures
          expr: |
            rate(hypersdk_jobs_failed_total[5m]) > 0.1
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "High job failure rate"
            description: "Job failure rate: {{ $value }}/min"

        # Pod down
        - alert: HyperSDKPodDown
          expr: |
            up{job="hypersdk"} == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "HyperSDK pod is down"
            description: "Pod {{ $labels.pod }} is down"

        # High memory usage
        - alert: HyperSDKHighMemory
          expr: |
            container_memory_usage_bytes{pod=~"hypersdk-.*"} /
            container_spec_memory_limit_bytes{pod=~"hypersdk-.*"} > 0.9
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High memory usage"
            description: "Memory usage: {{ $value | humanizePercentage }}"

        # Database connection issues
        - alert: HyperSDKDatabaseErrors
          expr: |
            rate(hypersdk_database_errors_total[5m]) > 0.01
          for: 2m
          labels:
            severity: warning
          annotations:
            summary: "Database connection errors"
            description: "Error rate: {{ $value }}/s"
```

### Grafana Dashboards

#### Main Dashboard

```json
{
  "dashboard": {
    "title": "HyperSDK Overview",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [{
          "expr": "rate(hypersdk_http_requests_total[5m])"
        }]
      },
      {
        "title": "Error Rate",
        "targets": [{
          "expr": "rate(hypersdk_http_requests_total{status=~\"5..\"}[5m])"
        }]
      },
      {
        "title": "Active Jobs",
        "targets": [{
          "expr": "hypersdk_active_jobs"
        }]
      },
      {
        "title": "Export Duration (P95)",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(hypersdk_export_duration_seconds_bucket[5m]))"
        }]
      }
    ]
  }
}
```

See `deployments/kubernetes/monitoring/grafana-dashboard-k8s.json` for complete dashboard.

## Logs with Loki

### Install Loki Stack

```bash
# Install Loki with Promtail
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack \
  --namespace monitoring \
  --set grafana.enabled=false \
  --set prometheus.enabled=false \
  --set promtail.enabled=true
```

### Configure Log Collection

```yaml
# HyperSDK logs are automatically collected by Promtail
# Configure additional parsing:

# loki/promtail-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: promtail-config
  namespace: monitoring
data:
  promtail.yaml: |
    server:
      http_listen_port: 3101

    clients:
      - url: http://loki:3100/loki/api/v1/push

    positions:
      filename: /tmp/positions.yaml

    scrape_configs:
      - job_name: kubernetes-pods
        kubernetes_sd_configs:
          - role: pod

        pipeline_stages:
          # Parse JSON logs
          - json:
              expressions:
                level: level
                msg: msg
                timestamp: timestamp

          # Extract log level
          - labels:
              level:

          # Parse structured logs
          - match:
              selector: '{app="hypersdk"}'
              stages:
                - json:
                    expressions:
                      job_id: job_id
                      provider: provider
                      vm_name: vm_name
                - labels:
                    job_id:
                    provider:

        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_label_app]
            target_label: app
          - source_labels: [__meta_kubernetes_namespace]
            target_label: namespace
          - source_labels: [__meta_kubernetes_pod_name]
            target_label: pod
```

### Log Queries (LogQL)

```logql
# All HyperSDK logs
{app="hypersdk"}

# Error logs only
{app="hypersdk"} |= "level=error"

# Logs for specific job
{app="hypersdk"} | json | job_id="12345"

# Failed export attempts
{app="hypersdk"} |= "export failed"

# Rate of errors
rate({app="hypersdk"} |= "error" [5m])

# Top error messages
topk(10, sum by (msg) (rate({app="hypersdk"} |= "error" [5m])))
```

### Grafana Loki Dashboard

```json
{
  "dashboard": {
    "title": "HyperSDK Logs",
    "panels": [
      {
        "title": "Log Volume",
        "targets": [{
          "expr": "sum(rate({app=\"hypersdk\"}[1m]))"
        }]
      },
      {
        "title": "Error Rate",
        "targets": [{
          "expr": "sum(rate({app=\"hypersdk\"} |= \"error\" [5m]))"
        }]
      },
      {
        "title": "Recent Logs",
        "targets": [{
          "expr": "{app=\"hypersdk\"}"
        }]
      }
    ]
  }
}
```

## Logs with ELK Stack

### Install ELK

```bash
# Install Elasticsearch
helm repo add elastic https://helm.elastic.co
helm install elasticsearch elastic/elasticsearch \
  --namespace logging \
  --create-namespace \
  --set replicas=3

# Install Kibana
helm install kibana elastic/kibana \
  --namespace logging \
  --set elasticsearchHosts=http://elasticsearch-master:9200

# Install Filebeat
helm install filebeat elastic/filebeat \
  --namespace logging
```

### Filebeat Configuration

```yaml
# filebeat/filebeat-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: filebeat-config
  namespace: logging
data:
  filebeat.yml: |
    filebeat.inputs:
      - type: container
        paths:
          - /var/log/containers/*hypersdk*.log
        processors:
          - add_kubernetes_metadata:
              in_cluster: true
          - decode_json_fields:
              fields: ["message"]
              target: ""
              overwrite_keys: true

    output.elasticsearch:
      hosts: ['elasticsearch-master:9200']
      index: "hypersdk-%{+yyyy.MM.dd}"

    setup.template.name: "hypersdk"
    setup.template.pattern: "hypersdk-*"
    setup.ilm.enabled: false
```

### Kibana Index Pattern

```bash
# Create index pattern
curl -X POST "http://kibana:5601/api/saved_objects/index-pattern/hypersdk-*" \
  -H 'kbn-xsrf: true' \
  -H 'Content-Type: application/json' \
  -d '{
    "attributes": {
      "title": "hypersdk-*",
      "timeFieldName": "@timestamp"
    }
  }'
```

## Distributed Tracing with Jaeger

### Install Jaeger

```bash
# Install Jaeger Operator
kubectl create namespace observability
kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/latest/download/jaeger-operator.yaml -n observability

# Create Jaeger instance
kubectl apply -f - <<EOF
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: hypersdk-jaeger
  namespace: observability
spec:
  strategy: production
  storage:
    type: elasticsearch
    options:
      es:
        server-urls: http://elasticsearch:9200
        index-prefix: jaeger
  ingress:
    enabled: true
EOF
```

### Configure HyperSDK for Tracing

```yaml
# Add Jaeger agent sidecar
spec:
  template:
    spec:
      containers:
        - name: hypersdk
          env:
            - name: JAEGER_AGENT_HOST
              value: "localhost"
            - name: JAEGER_AGENT_PORT
              value: "6831"
            - name: JAEGER_SAMPLER_TYPE
              value: "probabilistic"
            - name: JAEGER_SAMPLER_PARAM
              value: "0.1"  # 10% sampling

        - name: jaeger-agent
          image: jaegertracing/jaeger-agent:latest
          ports:
            - containerPort: 6831
              protocol: UDP
            - containerPort: 5778
              protocol: HTTP
          env:
            - name: REPORTER_GRPC_HOST_PORT
              value: "hypersdk-jaeger-collector:14250"
```

### Tracing Queries

```
# Find slow requests
duration > 1s

# Trace specific operation
operation="vm.export"

# Errors only
error=true

# By tag
tags.provider="vsphere" AND tags.vm_name="test-vm"
```

## Distributed Tracing with Tempo

### Install Tempo

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install tempo grafana/tempo \
  --namespace monitoring \
  --set tempo.storage.trace.backend=s3 \
  --set tempo.storage.trace.s3.bucket=traces \
  --set tempo.storage.trace.s3.endpoint=minio:9000
```

### OpenTelemetry Collector

```yaml
# otel/otel-collector.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
  namespace: monitoring
data:
  config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
          http:

    processors:
      batch:
        timeout: 10s
        send_batch_size: 1024

      attributes:
        actions:
          - key: service.name
            value: hypersdk
            action: insert

    exporters:
      tempo:
        endpoint: tempo:4317
        insecure: true

      prometheus:
        endpoint: "0.0.0.0:8889"

      loki:
        endpoint: http://loki:3100/loki/api/v1/push

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch, attributes]
          exporters: [tempo]

        metrics:
          receivers: [otlp]
          processors: [batch]
          exporters: [prometheus]

        logs:
          receivers: [otlp]
          processors: [batch]
          exporters: [loki]
```

## Complete Observability Stack

### All-in-One Installation

```yaml
# observability/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: monitoring

resources:
  # Metrics
  - prometheus/
  - grafana/

  # Logs
  - loki/
  - promtail/

  # Traces
  - tempo/
  - otel-collector/

  # Integration
  - servicemonitor.yaml
  - dashboards/
```

### Unified Grafana Configuration

```yaml
# grafana/datasources.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
  namespace: monitoring
data:
  datasources.yaml: |
    apiVersion: 1
    datasources:
      # Metrics
      - name: Prometheus
        type: prometheus
        url: http://prometheus:9090
        access: proxy
        isDefault: true

      # Logs
      - name: Loki
        type: loki
        url: http://loki:3100
        access: proxy
        jsonData:
          derivedFields:
            - datasourceUid: tempo
              matcherRegex: "trace_id=(\\w+)"
              name: TraceID
              url: "$${__value.raw}"

      # Traces
      - name: Tempo
        type: tempo
        url: http://tempo:3100
        access: proxy
        jsonData:
          tracesToLogs:
            datasourceUid: loki
            filterByTraceID: true
          serviceMap:
            datasourceUid: prometheus
```

### Correlation Example

```
User Request → Trace ID
  ↓
Logs (filtered by trace_id) ← Metrics (time range)
  ↓
Root Cause Analysis
```

## Alerting

### AlertManager Configuration

```yaml
# alertmanager/alertmanager-config.yaml
apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-config
  namespace: monitoring
stringData:
  alertmanager.yaml: |
    global:
      resolve_timeout: 5m

    route:
      group_by: ['alertname', 'cluster']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 12h
      receiver: 'slack'

      routes:
        # Critical alerts
        - match:
            severity: critical
          receiver: 'pagerduty'
          continue: true

        # Warning alerts
        - match:
            severity: warning
          receiver: 'slack'

    receivers:
      - name: 'slack'
        slack_configs:
          - api_url: 'https://hooks.slack.com/services/xxx'
            channel: '#hypersdk-alerts'
            title: 'HyperSDK Alert'
            text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

      - name: 'pagerduty'
        pagerduty_configs:
          - service_key: 'xxx'
            description: '{{ .CommonAnnotations.summary }}'
```

## Best Practices

### 1. Structured Logging

```go
// Use structured logging format
log.Info("Export completed",
    "job_id", jobID,
    "provider", "vsphere",
    "vm_name", vmName,
    "duration_seconds", duration.Seconds(),
    "bytes_transferred", bytesTransferred)
```

### 2. Metric Naming

```
# Counter
hypersdk_exports_total{provider="vsphere",status="success"}

# Gauge
hypersdk_active_jobs{provider="vsphere"}

# Histogram
hypersdk_export_duration_seconds{provider="vsphere"}

# Summary
hypersdk_request_size_bytes{method="POST",endpoint="/api/v1/export"}
```

### 3. Sampling Strategy

```yaml
# High-frequency operations: 1% sampling
sampler:
  type: probabilistic
  param: 0.01

# Critical paths: 100% sampling
sampler:
  type: const
  param: 1
```

### 4. Log Levels

```
DEBUG: Development only
INFO:  Normal operations
WARN:  Unusual but handled
ERROR: Operation failed
FATAL: Service down
```

### 5. Retention Policies

```yaml
# Metrics: 15 days
prometheus:
  retention: 15d

# Logs: 7 days
loki:
  retention_period: 168h

# Traces: 3 days
tempo:
  retention: 72h
```

## Visualization Examples

### Service Map (Grafana)

```
External Client
      ↓
   Ingress
      ↓
  HyperSDK API
      ↓
    ┌─────┼─────┐
    ↓     ↓     ↓
  SQLite vSphere AWS
```

### Request Flow Trace

```
Request: POST /api/v1/export
  │
  ├─ Authentication (2ms)
  ├─ Validation (5ms)
  ├─ Queue Job (3ms)
  ├─ Connect to vSphere (120ms)
  ├─ Export VM (45s)
  │   ├─ Download VMDK (40s)
  │   ├─ Convert format (3s)
  │   └─ Upload to storage (2s)
  └─ Update database (8ms)

Total: 45.138s
```

## Troubleshooting with Observability

### Scenario: High Latency

1. **Check Metrics**: Identify slow endpoints
   ```promql
   histogram_quantile(0.95, rate(hypersdk_request_duration_seconds_bucket[5m]))
   ```

2. **Check Traces**: Find slow operations
   ```
   duration > 5s
   ```

3. **Check Logs**: Look for errors
   ```logql
   {app="hypersdk"} |= "timeout"
   ```

### Scenario: Export Failures

1. **Metrics**: Check failure rate
   ```promql
   rate(hypersdk_jobs_failed_total[5m])
   ```

2. **Logs**: Find error messages
   ```logql
   {app="hypersdk"} |= "export failed" | json
   ```

3. **Traces**: Analyze failed requests
   ```
   error=true AND operation="vm.export"
   ```

## Summary

Complete observability enables:

✅ **Proactive Monitoring** - Detect issues before users
✅ **Fast Troubleshooting** - Root cause in minutes
✅ **Performance Optimization** - Identify bottlenecks
✅ **Capacity Planning** - Predict resource needs
✅ **SLA Compliance** - Track and prove uptime
✅ **Business Insights** - Understand usage patterns

Stack components:
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Visualization and dashboards
- **Loki/ELK**: Log aggregation and search
- **Jaeger/Tempo**: Distributed tracing
- **OpenTelemetry**: Unified instrumentation
