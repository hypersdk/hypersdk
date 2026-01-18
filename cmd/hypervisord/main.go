// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pterm/pterm"

	"hypersdk/config"
	"hypersdk/daemon/api"
	"hypersdk/daemon/capabilities"
	"hypersdk/daemon/jobs"
	"hypersdk/logger"
)

const (
	defaultAddr = "localhost:8080"
	version     = "0.0.1"
)

func main() {
	// Parse flags
	configFile := flag.String("config", "", "Path to config file (YAML)")
	addr := flag.String("addr", "", "API server address (overrides config file)")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error)")
	disableWeb := flag.Bool("disable-web", false, "Disable web dashboard (API-only mode)")
	versionFlag := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("hypervisord version %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error

	if *configFile != "" {
		// Load from file
		cfg, err = config.FromFile(*configFile)
		if err != nil {
			pterm.Error.Printfln("Failed to load config file: %v", err)
			os.Exit(1)
		}
		// Merge with environment variables (env takes precedence)
		cfg = cfg.MergeWithEnv()
		pterm.Info.Printfln("Loaded configuration from: %s", *configFile)
	} else {
		// Load from environment
		cfg = config.FromEnvironment()
	}

	// Override with flags if specified
	if *addr != "" {
		cfg.DaemonAddr = *addr
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	// Use defaults if still empty
	if cfg.DaemonAddr == "" {
		cfg.DaemonAddr = defaultAddr
	}

	// Show banner
	showBanner()

	// Setup logging
	log := logger.New(cfg.LogLevel)

	pterm.Info.Printfln("Starting hypervisord daemon v%s", version)
	pterm.Info.Printfln("API server will listen on: %s", cfg.DaemonAddr)

	// Create capability detector
	pterm.Info.Println("Detecting available export capabilities...")
	detector := capabilities.NewDetector(log)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := detector.Detect(ctx); err != nil {
		pterm.Warning.Printfln("Failed to detect capabilities: %v", err)
	}

	// Display detected capabilities
	showCapabilities(detector)

	// Create job manager with capability detector
	manager := jobs.NewManager(log, detector)

	// Create API config
	apiConfig := &api.Config{}
	apiConfig.Metrics.Enabled = false
	apiConfig.Security.EnableAuth = false // Disable auth for local development
	apiConfig.Web.Disabled = *disableWeb  // Set web dashboard state from CLI flag

	// Create Enhanced API server with Phase 2 features
	server, err := api.NewEnhancedServer(manager, detector, log, cfg.DaemonAddr, apiConfig)
	if err != nil {
		pterm.Error.Printfln("Failed to create server: %v", err)
		os.Exit(1)
	}

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errCh <- err
		}
	}()

	pterm.Success.Printfln("Daemon started successfully")
	pterm.Info.Println("Waiting for jobs... (Press Ctrl+C to stop)")

	// Show API endpoints
	showEndpoints(cfg.DaemonAddr)

	// Wait for signal or error
	select {
	case sig := <-sigCh:
		pterm.Warning.Printfln("Received signal: %v", sig)
		pterm.Info.Println("Shutting down gracefully...")

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			pterm.Error.Printfln("Server shutdown error: %v", err)
		}

		manager.Shutdown()
		pterm.Success.Println("Daemon stopped")

	case err := <-errCh:
		pterm.Error.Printfln("Server error: %v", err)
		os.Exit(1)
	}
}

func showBanner() {
	pterm.DefaultCenter.Println()

	// Orange/amber color scheme (Claude-inspired)
	orange := pterm.NewStyle(pterm.FgLightRed)
	amber := pterm.NewStyle(pterm.FgYellow)

	bigText, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("HYPER", orange),
		pterm.NewLettersFromStringWithStyle("VISOR", amber),
		pterm.NewLettersFromStringWithStyle("D", orange),
	).Srender()

	pterm.DefaultCenter.Println(bigText)

	subtitle := pterm.DefaultCenter.Sprint(pterm.LightYellow("Multi-Cloud VM Export Daemon"))
	pterm.Println(subtitle)
	pterm.Println()
}

func showCapabilities(detector *capabilities.Detector) {
	caps := detector.GetCapabilities()
	defaultMethod := detector.GetDefaultMethod()

	capData := [][]string{
		{"Method", "Available", "Priority", "Path"},
	}

	// Priority order: ctl, govc, ovftool, web
	methods := []capabilities.ExportMethod{
		capabilities.ExportMethodCTL,
		capabilities.ExportMethodGovc,
		capabilities.ExportMethodOvftool,
		capabilities.ExportMethodWeb,
	}

	for _, method := range methods {
		if cap, ok := caps[method]; ok {
			available := "✗"
			if cap.Available {
				available = "✓"
			}
			isDefault := ""
			if method == defaultMethod {
				isDefault = " (default)"
			}
			capData = append(capData, []string{
				string(method) + isDefault,
				available,
				fmt.Sprintf("%d", cap.Priority),
				cap.Path,
			})
		}
	}

	pterm.DefaultSection.Println("Export Capabilities")
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(capData).
		Render()

	pterm.Success.Printfln("Default export method: %s", defaultMethod)
	pterm.Println()
}

func showEndpoints(addr string) {
	baseURL := fmt.Sprintf("http://%s", addr)

	endpoints := [][]string{
		{"Endpoint", "Method", "Description"},
		{baseURL + "/", "GET", "Web Dashboard (redirect)"},
		{baseURL + "/web/dashboard/", "GET", "Web Dashboard UI"},
		{baseURL + "/ws", "WS", "WebSocket (real-time updates)"},
		{baseURL + "/health", "GET", "Health check"},
		{baseURL + "/status", "GET", "Daemon status"},
		{baseURL + "/capabilities", "GET", "Export capabilities"},
		{baseURL + "/jobs/submit", "POST", "Submit job(s) (JSON/YAML)"},
		{baseURL + "/jobs/query", "POST", "Query jobs"},
		{baseURL + "/jobs/{id}", "GET", "Get specific job"},
		{baseURL + "/jobs/cancel", "POST", "Cancel job(s)"},
		{baseURL + "/schedules", "GET/POST", "Manage schedules"},
		{baseURL + "/webhooks", "GET/POST", "Manage webhooks"},
		{baseURL + "/vms/list", "GET", "List discovered VMs"},
	}

	pterm.DefaultSection.Println("Available API Endpoints")
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(endpoints).
		Render()

	pterm.Info.Printfln("\n📊 Open dashboard in browser: %s/web/dashboard/", baseURL)
}
