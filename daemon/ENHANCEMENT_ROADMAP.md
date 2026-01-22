# Daemon Enhancement Roadmap

**Last Updated:** 2026-01-23
**Version:** 0.0.1

## Overview

This document outlines potential enhancements for the `hypervisord` daemon, organized by priority and category.

---

## 🔴 High Priority (Production Critical)

### 1. Prometheus Metrics Endpoint
**Status:** Not implemented
**Effort:** Low (2-4 hours)
**Value:** Very High

**What:**
Add `/metrics` endpoint exposing Prometheus-compatible metrics.

**Benefits:**
- Integration with existing monitoring (Grafana, Prometheus)
- Standard format for metrics collection
- Historical trending and alerting
- SLA tracking

**Implementation:**
```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// In enhanced_server.go
func (es *EnhancedServer) registerMetricsEndpoint() {
    if es.config.Metrics.Enabled {
        http.Handle("/metrics", promhttp.Handler())
    }
}
```

**Metrics to Expose:**
```
# HTTP metrics
hypervisord_http_requests_total{method, path, status}
hypervisord_http_request_duration_seconds{method, path}
hypervisord_http_errors_total{method, path}

# Job metrics
hypervisord_jobs_total{status}
hypervisord_jobs_duration_seconds{provider, format}
hypervisord_queue_length

# System metrics
hypervisord_memory_bytes
hypervisord_goroutines
hypervisord_uptime_seconds
```

**Example Grafana Dashboard:**
```yaml
# grafana-dashboard.json
{
  "panels": [
    {
      "title": "HTTP Request Rate",
      "targets": [
        "rate(hypervisord_http_requests_total[5m])"
      ]
    },
    {
      "title": "Job Success Rate",
      "targets": [
        "rate(hypervisord_jobs_total{status='completed'}[5m]) / rate(hypervisord_jobs_total[5m])"
      ]
    }
  ]
}
```

---

### 2. Persistent Job Queue
**Status:** In-memory only
**Effort:** Medium (1-2 days)
**Value:** Very High

**Problem:**
- Jobs lost on daemon restart
- No recovery from crashes
- Can't survive deployments

**Solution:**
Store job queue in SQLite database.

**Implementation:**
```go
// daemon/queue/persistent_queue.go
type PersistentQueue struct {
    db    *sql.DB
    jobs  chan *models.Job
    mu    sync.RWMutex
}

func (pq *PersistentQueue) Enqueue(job *models.Job) error {
    // 1. Save to database
    _, err := pq.db.Exec(`
        INSERT INTO job_queue (id, definition, status, created_at)
        VALUES (?, ?, ?, ?)
    `, job.ID, job.Definition, job.Status, job.CreatedAt)

    // 2. Add to in-memory queue
    pq.jobs <- job
    return err
}

func (pq *PersistentQueue) LoadOnStartup() error {
    // Load pending jobs from database
    rows, err := pq.db.Query(`
        SELECT id, definition, status FROM job_queue
        WHERE status IN ('pending', 'running')
        ORDER BY created_at ASC
    `)
    // Re-queue pending jobs
    for rows.Next() {
        var job models.Job
        rows.Scan(&job.ID, &job.Definition, &job.Status)
        pq.jobs <- &job
    }
}
```

**Benefits:**
- Zero job loss
- Survive daemon restarts
- Audit trail of all jobs
- Query historical jobs

---

### 3. Rate Limiting per Client
**Status:** Not implemented
**Effort:** Low (4-6 hours)
**Value:** High

**What:**
Protect API from abuse with rate limiting.

**Implementation:**
```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

func (rl *RateLimiter) Allow(clientIP string) bool {
    rl.mu.Lock()
    limiter, exists := rl.limiters[clientIP]
    if !exists {
        limiter = rate.NewLimiter(rl.rate, rl.burst)
        rl.limiters[clientIP] = limiter
    }
    rl.mu.Unlock()

    return limiter.Allow()
}

// Middleware
func (es *EnhancedServer) rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        clientIP := getClientIP(r)
        if !es.rateLimiter.Allow(clientIP) {
            http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

**Configuration:**
```yaml
# config.yaml
security:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst: 10
    per_endpoint:
      /jobs/submit: 10  # Limit expensive operations
      /jobs/query: 100
```

---

### 4. Health Check Improvements
**Status:** Basic `/health` endpoint exists
**Effort:** Low (2-3 hours)
**Value:** High

**Current:**
```go
// Just returns 200 OK
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}
```

**Enhanced:**
```go
type HealthStatus struct {
    Status      string            `json:"status"`      // "healthy", "degraded", "unhealthy"
    Version     string            `json:"version"`
    Uptime      float64           `json:"uptime_seconds"`
    Checks      map[string]Check  `json:"checks"`
    Timestamp   string            `json:"timestamp"`
}

type Check struct {
    Status  string  `json:"status"`
    Message string  `json:"message,omitempty"`
    Latency float64 `json:"latency_ms,omitempty"`
}

func (es *EnhancedServer) handleHealth(w http.ResponseWriter, r *http.Request) {
    status := HealthStatus{
        Status:    "healthy",
        Version:   "0.0.1",
        Uptime:    es.systemMetrics.GetUptime(),
        Timestamp: time.Now().Format(time.RFC3339),
        Checks:    make(map[string]Check),
    }

    // Check database
    if es.store != nil {
        start := time.Now()
        err := es.store.Ping()
        latency := time.Since(start).Milliseconds()

        if err != nil {
            status.Checks["database"] = Check{
                Status:  "unhealthy",
                Message: err.Error(),
                Latency: float64(latency),
            }
            status.Status = "degraded"
        } else {
            status.Checks["database"] = Check{
                Status:  "healthy",
                Latency: float64(latency),
            }
        }
    }

    // Check job queue
    queueDepth := es.manager.QueueDepth()
    if queueDepth > 100 {
        status.Checks["queue"] = Check{
            Status:  "degraded",
            Message: fmt.Sprintf("queue depth high: %d", queueDepth),
        }
        status.Status = "degraded"
    } else {
        status.Checks["queue"] = Check{Status: "healthy"}
    }

    // Check memory
    memUsage := es.systemMetrics.GetMemoryUsage()
    if memUsage > 1024*1024*1024 { // > 1GB
        status.Checks["memory"] = Check{
            Status:  "warning",
            Message: fmt.Sprintf("high memory usage: %d MB", memUsage/1024/1024),
        }
    } else {
        status.Checks["memory"] = Check{Status: "healthy"}
    }

    // Set HTTP status based on health
    httpStatus := http.StatusOK
    if status.Status == "unhealthy" {
        httpStatus = http.StatusServiceUnavailable
    } else if status.Status == "degraded" {
        httpStatus = http.StatusOK // Still operational
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(status)
}
```

**Usage:**
```bash
# Kubernetes liveness probe
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

# Monitor
curl http://localhost:8080/health | jq
{
  "status": "healthy",
  "version": "0.0.1",
  "uptime_seconds": 3600.5,
  "checks": {
    "database": {"status": "healthy", "latency_ms": 1.2},
    "queue": {"status": "healthy"},
    "memory": {"status": "healthy"}
  }
}
```

---

### 5. Job Cancellation Tracking
**Status:** TODO in websocket.go:526
**Effort:** Low (1-2 hours)
**Value:** Medium

**What:**
Track cancelled jobs separately from failed jobs.

**Implementation:**
```go
// In daemon/jobs/manager.go
type JobManager struct {
    // ... existing fields
    cancelledJobs atomic.Int64
}

func (m *JobManager) CancelJob(jobID string) error {
    job, exists := m.jobs[jobID]
    if !exists {
        return fmt.Errorf("job not found")
    }

    // Cancel context
    if job.cancelFunc != nil {
        job.cancelFunc()
    }

    // Update status
    job.Status = models.JobStatusCancelled
    job.CompletedAt = timePtr(time.Now())

    // Increment counter
    m.cancelledJobs.Add(1)

    return nil
}

func (m *JobManager) GetStatus() JobStatus {
    return JobStatus{
        TotalJobs:      m.totalJobs.Load(),
        RunningJobs:    m.runningJobs.Load(),
        CompletedJobs:  m.completedJobs.Load(),
        FailedJobs:     m.failedJobs.Load(),
        CancelledJobs:  m.cancelledJobs.Load(), // ✓ Now tracked
    }
}
```

---

## 🟡 Medium Priority (Valuable Enhancements)

### 6. Distributed Tracing (OpenTelemetry)
**Status:** Not implemented
**Effort:** Medium (2-3 days)
**Value:** High (for debugging)

**What:**
Add OpenTelemetry instrumentation for distributed tracing.

**Benefits:**
- Trace requests across microservices
- Identify performance bottlenecks
- Debug complex failures
- Monitor dependencies

**Implementation:**
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (es *EnhancedServer) traceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, span := otel.Tracer("hypervisord").Start(r.Context(), r.URL.Path)
        defer span.End()

        span.SetAttributes(
            attribute.String("http.method", r.Method),
            attribute.String("http.path", r.URL.Path),
        )

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Trace job execution
func (m *JobManager) SubmitJob(definition models.JobDefinition) (string, error) {
    ctx, span := otel.Tracer("jobs").Start(context.Background(), "SubmitJob")
    defer span.End()

    span.SetAttributes(
        attribute.String("vm", definition.VMPath),
        attribute.String("format", definition.Format),
    )

    // ... job logic
}
```

**Jaeger Integration:**
```bash
# Run Jaeger
docker run -d -p 16686:16686 -p 4318:4318 jaegertracing/all-in-one:latest

# Configure daemon
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
./hypervisord

# View traces
open http://localhost:16686
```

---

### 7. Job Prioritization
**Status:** FIFO queue only
**Effort:** Medium (1-2 days)
**Value:** Medium-High

**What:**
Priority queue for urgent exports.

**Implementation:**
```go
type PriorityQueue struct {
    queues map[Priority]*list.List
    mu     sync.Mutex
}

const (
    PriorityLow Priority = iota
    PriorityNormal
    PriorityHigh
    PriorityCritical
)

func (pq *PriorityQueue) Enqueue(job *models.Job, priority Priority) {
    pq.mu.Lock()
    defer pq.mu.Unlock()

    queue := pq.queues[priority]
    queue.PushBack(job)
}

func (pq *PriorityQueue) Dequeue() *models.Job {
    pq.mu.Lock()
    defer pq.mu.Unlock()

    // Try critical first, then high, normal, low
    for _, priority := range []Priority{PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow} {
        queue := pq.queues[priority]
        if queue.Len() > 0 {
            elem := queue.Front()
            queue.Remove(elem)
            return elem.Value.(*models.Job)
        }
    }
    return nil
}
```

**API Usage:**
```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "/DC/vm/production-db",
    "priority": "critical",
    "format": "ova"
  }'
```

---

### 8. Job Templates & Profiles
**Status:** Not implemented
**Effort:** Medium (1-2 days)
**Value:** Medium

**What:**
Reusable export configurations.

**Example:**
```yaml
# /etc/hypersdk/profiles/prod-backup.yaml
name: "Production Backup"
description: "Standard production VM backup"
defaults:
  format: "ova"
  compress: true
  compression_level: 6
  bandwidth_limit: 50MB
  enable_checkpoints: true
  verify_checksum: true
  generate_manifest: true
```

**Usage:**
```bash
# Use profile
hyperexport vsphere export \
  --vm /DC/vm/web-01 \
  --profile prod-backup \
  --output /backups

# Daemon API
curl -X POST http://localhost:8080/jobs/submit \
  -d '{"vm_path": "/DC/vm/web-01", "profile": "prod-backup"}'
```

---

### 9. Multi-Tenancy
**Status:** Not implemented
**Effort:** High (3-5 days)
**Value:** High (for SaaS)

**What:**
Support multiple organizations/teams with isolation.

**Features:**
- Tenant-based authentication
- Resource quotas per tenant
- Isolated job queues
- Per-tenant billing/metrics

**Implementation:**
```go
type Tenant struct {
    ID          string
    Name        string
    Quota       TenantQuota
    APIKeys     []string
    Permissions []Permission
}

type TenantQuota struct {
    MaxConcurrentJobs int
    MaxStorageGB      int
    MaxBandwidthMBps  int
}

// Middleware to extract tenant
func (es *EnhancedServer) tenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        tenant, err := es.authMgr.GetTenantByAPIKey(apiKey)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), "tenant", tenant)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Check quota before job submission
func (m *JobManager) SubmitJobForTenant(tenant *Tenant, def models.JobDefinition) error {
    runningJobs := m.GetRunningJobsForTenant(tenant.ID)
    if runningJobs >= tenant.Quota.MaxConcurrentJobs {
        return fmt.Errorf("quota exceeded: max %d concurrent jobs", tenant.Quota.MaxConcurrentJobs)
    }
    // ... submit job
}
```

---

### 10. Webhook Retry Logic Enhancement
**Status:** Basic retry exists
**Effort:** Low-Medium (4-6 hours)
**Value:** Medium

**Current:**
Simple retry with exponential backoff.

**Enhanced:**
```go
type WebhookRetryPolicy struct {
    MaxRetries     int
    InitialDelay   time.Duration
    MaxDelay       time.Duration
    BackoffFactor  float64
    RetryOnStatus  []int  // e.g., [429, 500, 502, 503, 504]
    CircuitBreaker CircuitBreakerConfig
}

type CircuitBreakerConfig struct {
    Enabled           bool
    FailureThreshold  int
    SuccessThreshold  int
    Timeout           time.Duration
}

// Circuit breaker pattern
type CircuitBreaker struct {
    state        State  // Closed, Open, HalfOpen
    failures     int
    successes    int
    lastFailTime time.Time
    mu           sync.Mutex
}

func (wb *WebhookManager) sendWithRetry(webhook Webhook, event Event) error {
    // Check circuit breaker
    if webhook.circuitBreaker.IsOpen() {
        return fmt.Errorf("circuit breaker open for %s", webhook.URL)
    }

    var lastErr error
    for attempt := 0; attempt <= webhook.Retry.MaxRetries; attempt++ {
        err := wb.send(webhook, event)
        if err == nil {
            webhook.circuitBreaker.RecordSuccess()
            return nil
        }

        lastErr = err
        webhook.circuitBreaker.RecordFailure()

        // Check if should retry
        if !shouldRetry(err, webhook.Retry.RetryOnStatus) {
            break
        }

        // Exponential backoff
        delay := calculateBackoff(attempt, webhook.Retry)
        time.Sleep(delay)
    }

    return fmt.Errorf("webhook failed after %d retries: %w", webhook.Retry.MaxRetries, lastErr)
}
```

---

## 🟢 Low Priority (Nice to Have)

### 11. GraphQL API
**Status:** Not implemented
**Effort:** High (5-7 days)
**Value:** Low-Medium

**What:**
Alternative to REST API using GraphQL.

**Benefits:**
- Flexible queries
- Reduce over-fetching
- Better for complex UIs
- Self-documenting schema

**Example Schema:**
```graphql
type Query {
  jobs(status: JobStatus, limit: Int): [Job!]!
  job(id: ID!): Job
  vms(provider: Provider): [VM!]!
}

type Mutation {
  submitJob(input: JobInput!): Job!
  cancelJob(id: ID!): Job!
}

type Subscription {
  jobUpdates(id: ID!): Job!
}

type Job {
  id: ID!
  vmPath: String!
  status: JobStatus!
  progress: Float
  startedAt: DateTime
  completedAt: DateTime
  error: String
}
```

---

### 12. gRPC Support
**Status:** Not implemented
**Effort:** Medium-High (3-4 days)
**Value:** Medium (for performance)

**What:**
Binary protocol alternative to HTTP/JSON.

**Benefits:**
- Better performance (binary encoding)
- Streaming support
- Strong typing
- Code generation

**Implementation:**
```protobuf
// hypervisord.proto
service HypervisordService {
  rpc SubmitJob(JobRequest) returns (JobResponse);
  rpc GetJobStatus(JobStatusRequest) returns (JobStatusResponse);
  rpc StreamJobUpdates(JobID) returns (stream JobUpdate);
  rpc ListJobs(ListJobsRequest) returns (stream Job);
}

message JobRequest {
  string vm_path = 1;
  string format = 2;
  string output_path = 3;
  bool compress = 4;
}
```

---

### 13. Job Chaining & Workflows
**Status:** Not implemented
**Effort:** High (5-7 days)
**Value:** Medium

**What:**
Define dependent jobs (workflows).

**Example:**
```yaml
# workflow.yaml
name: "Backup and Convert Workflow"
jobs:
  - id: export
    type: export
    vm_path: /DC/vm/web-01
    format: ova

  - id: convert
    type: hyper2kvm
    depends_on: export
    input: $export.output
    target_format: qcow2

  - id: upload
    type: upload
    depends_on: convert
    input: $convert.output
    destination: s3://backups/web-01/

  - id: notify
    type: webhook
    depends_on: upload
    url: https://hooks.slack.com/...
    message: "Backup completed: web-01"
```

---

### 14. Auto-Scaling Worker Pool
**Status:** Fixed worker count
**Effort:** Medium (2-3 days)
**Value:** Low-Medium

**What:**
Dynamically adjust worker pool size based on queue depth.

**Implementation:**
```go
type AutoScaler struct {
    minWorkers int
    maxWorkers int
    current    int
    mu         sync.Mutex
}

func (as *AutoScaler) Adjust(queueDepth int, cpuUsage float64) {
    as.mu.Lock()
    defer as.mu.Unlock()

    // Scale up if queue is growing
    if queueDepth > as.current*2 && as.current < as.maxWorkers {
        as.current++
        go startWorker()
    }

    // Scale down if idle
    if queueDepth == 0 && cpuUsage < 20 && as.current > as.minWorkers {
        as.current--
        stopWorker()
    }
}
```

---

### 15. Audit Logging
**Status:** Basic logging exists
**Effort:** Medium (2-3 days)
**Value:** Medium (compliance)

**What:**
Comprehensive audit trail.

**Format:**
```json
{
  "timestamp": "2026-01-23T20:00:00Z",
  "event": "job.submitted",
  "actor": {
    "type": "user",
    "id": "admin",
    "ip": "192.168.1.100"
  },
  "resource": {
    "type": "job",
    "id": "job-12345",
    "vm": "/DC/vm/web-01"
  },
  "action": "create",
  "result": "success",
  "metadata": {
    "format": "ova",
    "size_gb": 50
  }
}
```

---

## Implementation Priority

### Sprint 1 (Week 1) - Production Critical
1. ✅ **System Metrics** (DONE)
2. **Prometheus Endpoint** (2-4 hours)
3. **Health Check Enhancement** (2-3 hours)
4. **Rate Limiting** (4-6 hours)

### Sprint 2 (Week 2) - Reliability
1. **Persistent Job Queue** (1-2 days)
2. **Job Cancellation Tracking** (1-2 hours)
3. **Webhook Retry Enhancement** (4-6 hours)

### Sprint 3 (Week 3) - Performance & Observability
1. **OpenTelemetry Tracing** (2-3 days)
2. **Job Prioritization** (1-2 days)

### Sprint 4 (Week 4) - Advanced Features
1. **Job Templates** (1-2 days)
2. **Audit Logging** (2-3 days)

---

## Quick Wins (< 1 Day Each)

These can be done immediately for quick value:

1. ✅ **System Metrics** - DONE
2. **Prometheus Endpoint** - 4 hours
3. **Job Cancellation Tracking** - 2 hours
4. **Health Check Enhancement** - 3 hours
5. **Rate Limiting** - 6 hours

**Total:** ~2 days for significant improvements

---

## Metrics to Track

After implementing enhancements, track:

```
# Reliability
- Job success rate (target: >99%)
- Mean time to recovery (MTTR)
- Number of retries per job

# Performance
- P50/P95/P99 response times
- Queue depth over time
- Worker utilization

# Availability
- Uptime percentage (target: 99.9%)
- Health check failure rate
- Circuit breaker trips
```

---

## Conclusion

**Recommended Next Steps:**

1. **This Week:** Prometheus endpoint + Health checks (1 day)
2. **Next Week:** Persistent queue (2 days)
3. **Month 1:** Complete Sprint 1-2 (production-ready)
4. **Month 2-3:** Advanced features based on usage data

**Expected ROI:**
- 50% reduction in manual intervention (health checks, alerts)
- 99.9% job success rate (persistent queue + retry)
- 10x better debugging (tracing + metrics)
- Zero data loss (persistent queue)
