// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
)

// DaemonInstance represents a hyper2kvm daemon instance
type DaemonInstance struct {
	Name      string `json:"name"`
	Service   string `json:"service"`
	Active    bool   `json:"active"`
	Status    string `json:"status"`
	MainPID   string `json:"main_pid,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
	WatchDir  string `json:"watch_dir"`
	OutputDir string `json:"output_dir"`
}

// handleDaemonStatus shows the status of hyper2kvm daemon instances
func handleDaemonStatus(daemonURL, instance string, jsonOutput bool) {
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput {
		spinner, _ = pterm.DefaultSpinner.Start("🔍 Checking hyper2kvm daemon status...")
	}

	// Determine which service(s) to check
	var services []string
	if instance != "" {
		// Specific instance
		services = append(services, fmt.Sprintf("hyper2kvm@%s.service", instance))
	} else {
		// Check both default and list templated instances
		services = append(services, "hyper2kvm.service")

		// List all hyper2kvm@ instances
		cmd := exec.Command("systemctl", "list-units", "--all", "--type=service", "--no-pager", "hyper2kvm@*")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "hyper2kvm@") && strings.Contains(line, ".service") {
					fields := strings.Fields(line)
					if len(fields) > 0 {
						serviceName := fields[0]
						if !contains(services, serviceName) {
							services = append(services, serviceName)
						}
					}
				}
			}
		}
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("Found %d daemon instance(s)", len(services)))
	}

	// Get status for each service
	instances := make([]DaemonInstance, 0)
	for _, service := range services {
		instance := getDaemonInstanceStatus(service)
		if instance != nil {
			instances = append(instances, *instance)
		}
	}

	if len(instances) == 0 {
		if jsonOutput {
			fmt.Println("{\"instances\": [], \"total\": 0, \"message\": \"No hyper2kvm daemon instances found\"}")
		} else {
			pterm.Warning.Println("No hyper2kvm daemon instances found")
			pterm.Println()
			pterm.Info.Println("💡 To start a daemon instance:")
			pterm.Println("   sudo systemctl start hyper2kvm.service              # Default instance")
			pterm.Println("   sudo systemctl start hyper2kvm@vsphere.service      # Named instance")
		}
		return
	}

	// Display results
	if jsonOutput {
		output := map[string]interface{}{
			"instances": instances,
			"total":     len(instances),
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		displayDaemonInstances(instances)
	}
}

// handleDaemonList lists all hyper2kvm daemon instances
func handleDaemonList(jsonOutput bool) {
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput {
		spinner, _ = pterm.DefaultSpinner.Start("🔍 Listing hyper2kvm daemon instances...")
	}

	// List all hyper2kvm services
	cmd := exec.Command("systemctl", "list-units", "--all", "--type=service", "--no-pager", "hyper2kvm*")
	output, err := cmd.Output()
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to list daemon instances: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Parse systemctl output
	var instances []DaemonInstance
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "hyper2kvm") && strings.Contains(line, ".service") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				serviceName := fields[0]
				instance := getDaemonInstanceStatus(serviceName)
				if instance != nil {
					instances = append(instances, *instance)
				}
			}
		}
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("Found %d daemon instance(s)", len(instances)))
	}

	if len(instances) == 0 {
		if jsonOutput {
			fmt.Println("{\"instances\": [], \"total\": 0}")
		} else {
			pterm.Info.Println("No hyper2kvm daemon instances found")
			pterm.Println()
			pterm.Info.Println("💡 To create a daemon instance:")
			pterm.Println("   sudo systemctl start hyper2kvm.service              # Default instance")
			pterm.Println("   sudo systemctl start hyper2kvm@vsphere.service      # Named instance")
		}
		return
	}

	// Display results
	if jsonOutput {
		output := map[string]interface{}{
			"instances": instances,
			"total":     len(instances),
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		displayDaemonInstances(instances)
	}
}

// getDaemonInstanceStatus retrieves status for a specific daemon instance
func getDaemonInstanceStatus(serviceName string) *DaemonInstance {
	// Check if service is active
	cmd := exec.Command("systemctl", "is-active", serviceName)
	activeOutput, _ := cmd.Output()
	active := strings.TrimSpace(string(activeOutput)) == "active"

	// Get detailed status
	cmd = exec.Command("systemctl", "status", serviceName, "--no-pager")
	statusOutput, _ := cmd.Output()

	status := strings.TrimSpace(string(activeOutput))
	mainPID := ""
	uptime := ""

	// Parse status output for details
	statusLines := strings.Split(string(statusOutput), "\n")
	for _, line := range statusLines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Main PID:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				mainPID = parts[2]
			}
		}
		if strings.Contains(line, "Active:") {
			// Extract uptime from active line
			// Example: "Active: active (running) since Fri 2024-01-19 10:30:00 UTC; 2h 15min ago"
			if idx := strings.Index(line, "since"); idx != -1 {
				remainder := line[idx+6:]
				if idx2 := strings.Index(remainder, ";"); idx2 != -1 {
					uptime = strings.TrimSpace(remainder[idx2+1:])
				}
			}
		}
	}

	// Extract instance name
	instanceName := "default"
	if strings.Contains(serviceName, "@") {
		parts := strings.Split(serviceName, "@")
		if len(parts) >= 2 {
			instanceName = strings.TrimSuffix(parts[1], ".service")
		}
	}

	// Get watch and output directories (these would come from service config)
	// For now, use defaults
	watchDir := "/var/lib/hyper2kvm/queue"
	outputDir := "/var/lib/hyper2kvm/output"
	if instanceName != "default" {
		watchDir = fmt.Sprintf("/var/lib/hyper2kvm/%s-queue", instanceName)
		outputDir = fmt.Sprintf("/var/lib/hyper2kvm/%s-output", instanceName)
	}

	return &DaemonInstance{
		Name:      instanceName,
		Service:   serviceName,
		Active:    active,
		Status:    status,
		MainPID:   mainPID,
		Uptime:    uptime,
		WatchDir:  watchDir,
		OutputDir: outputDir,
	}
}

// displayDaemonInstances displays daemon instances in a table
func displayDaemonInstances(instances []DaemonInstance) {
	pterm.Println()
	pterm.DefaultSection.Println("🔧 hyper2kvm Daemon Instances")
	pterm.Println()

	// Build table data
	data := [][]string{
		{"Instance", "Service", "Status", "PID", "Uptime"},
	}

	for _, inst := range instances {
		statusStr := inst.Status
		if inst.Active {
			statusStr = pterm.Green("active")
		} else {
			statusStr = pterm.Gray(inst.Status)
		}

		pid := inst.MainPID
		if pid == "" {
			pid = "-"
		}

		uptime := inst.Uptime
		if uptime == "" {
			uptime = "-"
		}

		data = append(data, []string{
			inst.Name,
			inst.Service,
			statusStr,
			pid,
			uptime,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()

	// Show detailed configuration
	pterm.DefaultSection.Println("📂 Configuration")
	pterm.Println()

	configData := [][]string{
		{"Instance", "Watch Directory", "Output Directory"},
	}

	for _, inst := range instances {
		configData = append(configData, []string{
			inst.Name,
			inst.WatchDir,
			inst.OutputDir,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(configData).
		Render()

	pterm.Println()

	// Show helpful commands
	pterm.Info.Println("💡 Useful commands:")
	pterm.Println("   sudo systemctl status hyper2kvm.service              # Check status")
	pterm.Println("   sudo systemctl start hyper2kvm@instance.service      # Start instance")
	pterm.Println("   sudo systemctl stop hyper2kvm@instance.service       # Stop instance")
	pterm.Println("   sudo journalctl -u hyper2kvm.service -f              # View logs")
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
