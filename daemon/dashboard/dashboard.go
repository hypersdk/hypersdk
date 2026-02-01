// SPDX-License-Identifier: LGPL-3.0-or-later

// Package dashboard provides a real-time web dashboard for HyperSDK
package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed templates/* static/*
var embeddedFS embed.FS

// Config holds dashboard configuration
type Config struct {
	// Enabled determines if dashboard is enabled
	Enabled bool

	// Port is the HTTP port to listen on
	Port int

	// UpdateInterval is how often to push updates
	UpdateInterval time.Duration

	// MaxClients is the maximum number of concurrent WebSocket clients
	MaxClients int
}

// DefaultConfig returns default dashboard configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		Port:           8080,
		UpdateInterval: 1 * time.Second,
		MaxClients:     100,
	}
}

// Dashboard provides real-time monitoring
type Dashboard struct {
	config    *Config
	templates *template.Template
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	broadcast chan []byte
	metrics   *Metrics
	metricsMu sync.RWMutex
	k8sDash   *K8sDashboard
}

// Metrics holds dashboard metrics
type Metrics struct {
	Timestamp         time.Time      `json:"timestamp"`
	JobsActive        int            `json:"jobs_active"`
	JobsCompleted     int            `json:"jobs_completed"`
	JobsFailed        int            `json:"jobs_failed"`
	JobsPending       int            `json:"jobs_pending"`
	QueueLength       int            `json:"queue_length"`
	HTTPRequests      int64          `json:"http_requests"`
	HTTPErrors        int64          `json:"http_errors"`
	AvgResponseTime   float64        `json:"avg_response_time"`
	MemoryUsage       int64          `json:"memory_usage"`
	CPUUsage          float64        `json:"cpu_usage"`
	Goroutines        int            `json:"goroutines"`
	ActiveConnections int            `json:"active_connections"`
	ProviderStats     map[string]int `json:"provider_stats"`
	RecentJobs        []JobInfo      `json:"recent_jobs"`
	SystemHealth      string         `json:"system_health"`
	Alerts            []Alert        `json:"alerts"`
}

// JobInfo represents job information
type JobInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Duration  float64   `json:"duration"`
	Provider  string    `json:"provider"`
	VMName    string    `json:"vm_name"`
	ErrorMsg  string    `json:"error_msg,omitempty"`
}

// Alert represents a system alert
type Alert struct {
	ID       string    `json:"id"`
	Severity string    `json:"severity"`
	Message  string    `json:"message"`
	Time     time.Time `json:"time"`
}

// NewDashboard creates a new dashboard
func NewDashboard(config *Config) (*Dashboard, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Parse embedded templates
	tmpl, err := template.ParseFS(embeddedFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Dashboard{
		config:    config,
		templates: tmpl,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte, 256),
		metrics: &Metrics{
			Timestamp:     time.Now(),
			ProviderStats: make(map[string]int),
			RecentJobs:    make([]JobInfo, 0),
			Alerts:        make([]Alert, 0),
			SystemHealth:  "healthy",
		},
	}, nil
}

// Start starts the dashboard server
func (d *Dashboard) Start(ctx context.Context) error {
	if !d.config.Enabled {
		return nil
	}

	// Start broadcast goroutine
	go d.handleBroadcast()

	// Start metrics update goroutine
	go d.updateMetrics(ctx)

	// Try to initialize Kubernetes dashboard (optional)
	k8sDash, err := NewK8sDashboard(d, "", "")
	if err == nil {
		d.k8sDash = k8sDash
		// Start K8s metrics collection
		go d.k8sDash.Start(ctx)
	}

	// Setup HTTP handlers
	mux := http.NewServeMux()

	// Serve embedded static files
	mux.Handle("/static/", http.FileServer(http.FS(embeddedFS)))

	// Dashboard pages
	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/k8s", d.handleK8s)
	mux.HandleFunc("/k8s/charts", d.handleK8sCharts)
	mux.HandleFunc("/k8s/vms", d.handleK8sVMs)

	// API endpoints
	mux.HandleFunc("/api/metrics", d.handleMetrics)
	mux.HandleFunc("/api/jobs", d.handleJobs)
	mux.HandleFunc("/api/jobs/", d.handleJobDetail)

	// WebSocket endpoint
	mux.HandleFunc("/ws", d.handleWebSocket)

	// Register Kubernetes dashboard handlers if available
	if d.k8sDash != nil {
		d.k8sDash.RegisterHandlers(mux)
	}

	addr := fmt.Sprintf(":%d", d.config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Dashboard server error: %v\n", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return server.Shutdown(context.Background())
}

// handleIndex serves the main dashboard page
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if err := d.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleK8s serves the Kubernetes dashboard page
func (d *Dashboard) handleK8s(w http.ResponseWriter, r *http.Request) {
	if err := d.templates.ExecuteTemplate(w, "k8s.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleK8sCharts serves the Kubernetes charts and analytics page
func (d *Dashboard) handleK8sCharts(w http.ResponseWriter, r *http.Request) {
	if err := d.templates.ExecuteTemplate(w, "k8s-charts.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleK8sVMs serves the Kubernetes VM management page
func (d *Dashboard) handleK8sVMs(w http.ResponseWriter, r *http.Request) {
	if err := d.templates.ExecuteTemplate(w, "k8s-vms.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleMetrics serves current metrics as JSON
func (d *Dashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	d.metricsMu.RLock()
	metrics := *d.metrics
	d.metricsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleJobs serves jobs list
func (d *Dashboard) handleJobs(w http.ResponseWriter, r *http.Request) {
	d.metricsMu.RLock()
	jobs := d.metrics.RecentJobs
	d.metricsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// handleJobDetail serves job details
func (d *Dashboard) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from path
	jobID := r.URL.Path[len("/api/jobs/"):]

	d.metricsMu.RLock()
	defer d.metricsMu.RUnlock()

	for _, job := range d.metrics.RecentJobs {
		if job.ID == jobID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(job)
			return
		}
	}

	http.NotFound(w, r)
}

// handleWebSocket handles WebSocket connections
func (d *Dashboard) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check client limit
	d.clientsMu.RLock()
	clientCount := len(d.clients)
	d.clientsMu.RUnlock()

	if clientCount >= d.config.MaxClients {
		http.Error(w, "Too many clients", http.StatusServiceUnavailable)
		return
	}

	// Upgrade connection
	conn, err := d.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	// Register client
	d.clientsMu.Lock()
	d.clients[conn] = true
	d.clientsMu.Unlock()

	// Send initial metrics
	d.metricsMu.RLock()
	data, _ := json.Marshal(d.metrics)
	d.metricsMu.RUnlock()
	conn.WriteMessage(websocket.TextMessage, data)

	// Handle client messages
	go d.handleClient(conn)
}

// handleClient handles individual client connections
func (d *Dashboard) handleClient(conn *websocket.Conn) {
	defer func() {
		d.clientsMu.Lock()
		delete(d.clients, conn)
		d.clientsMu.Unlock()
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handleBroadcast broadcasts metrics to all connected clients
func (d *Dashboard) handleBroadcast() {
	for data := range d.broadcast {
		d.clientsMu.RLock()
		for client := range d.clients {
			err := client.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				client.Close()
				delete(d.clients, client)
			}
		}
		d.clientsMu.RUnlock()
	}
}

// updateMetrics periodically updates and broadcasts metrics
func (d *Dashboard) updateMetrics(ctx context.Context) {
	ticker := time.NewTicker(d.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Collect metrics (in real implementation, this would fetch from actual sources)
			d.collectMetrics()

			// Broadcast to clients
			d.metricsMu.RLock()
			data, err := json.Marshal(d.metrics)
			d.metricsMu.RUnlock()

			if err == nil {
				select {
				case d.broadcast <- data:
				default:
					// Channel full, skip this update
				}
			}
		}
	}
}

// collectMetrics collects current metrics
func (d *Dashboard) collectMetrics() {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	// Update timestamp
	d.metrics.Timestamp = time.Now()

	// In a real implementation, these would be fetched from actual sources
	// For now, we'll use placeholder logic

	// Update active connections
	d.clientsMu.RLock()
	d.metrics.ActiveConnections = len(d.clients)
	d.clientsMu.RUnlock()
}

// UpdateJobMetrics updates job-related metrics
func (d *Dashboard) UpdateJobMetrics(active, completed, failed, pending, queueLen int) {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	d.metrics.JobsActive = active
	d.metrics.JobsCompleted = completed
	d.metrics.JobsFailed = failed
	d.metrics.JobsPending = pending
	d.metrics.QueueLength = queueLen
}

// AddJob adds a job to the recent jobs list
func (d *Dashboard) AddJob(job JobInfo) {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	// Add to beginning of list
	d.metrics.RecentJobs = append([]JobInfo{job}, d.metrics.RecentJobs...)

	// Keep only last 50 jobs
	if len(d.metrics.RecentJobs) > 50 {
		d.metrics.RecentJobs = d.metrics.RecentJobs[:50]
	}

	// Update provider stats
	if d.metrics.ProviderStats == nil {
		d.metrics.ProviderStats = make(map[string]int)
	}
	d.metrics.ProviderStats[job.Provider]++
}

// UpdateSystemMetrics updates system resource metrics
func (d *Dashboard) UpdateSystemMetrics(memoryMB int64, cpuPercent float64, goroutines int) {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	d.metrics.MemoryUsage = memoryMB
	d.metrics.CPUUsage = cpuPercent
	d.metrics.Goroutines = goroutines
}

// UpdateHTTPMetrics updates HTTP metrics
func (d *Dashboard) UpdateHTTPMetrics(requests, errors int64, avgResponseTime float64) {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	d.metrics.HTTPRequests = requests
	d.metrics.HTTPErrors = errors
	d.metrics.AvgResponseTime = avgResponseTime
}

// AddAlert adds a new alert
func (d *Dashboard) AddAlert(severity, message string) {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	alert := Alert{
		ID:       fmt.Sprintf("alert-%d", time.Now().UnixNano()),
		Severity: severity,
		Message:  message,
		Time:     time.Now(),
	}

	d.metrics.Alerts = append([]Alert{alert}, d.metrics.Alerts...)

	// Keep only last 20 alerts
	if len(d.metrics.Alerts) > 20 {
		d.metrics.Alerts = d.metrics.Alerts[:20]
	}
}

// SetSystemHealth sets the overall system health status
func (d *Dashboard) SetSystemHealth(health string) {
	d.metricsMu.Lock()
	defer d.metricsMu.Unlock()

	d.metrics.SystemHealth = health
}

// GetClientCount returns the number of connected WebSocket clients
func (d *Dashboard) GetClientCount() int {
	d.clientsMu.RLock()
	defer d.clientsMu.RUnlock()
	return len(d.clients)
}
