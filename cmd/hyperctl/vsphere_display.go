// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"

	"hypersdk/providers/vsphere"
)

// Display functions for vSphere data

func displayHosts(hosts []vsphere.HostInfo) {
	pterm.DefaultSection.Println("ESXi Hosts")

	data := [][]string{
		{"Name", "Datacenter", "Cluster", "State", "CPUs", "Memory (GB)", "VMs"},
	}

	for _, host := range hosts {
		data = append(data, []string{
			truncate(host.Name, 30),
			truncate(host.Datacenter, 20),
			truncate(host.Cluster, 20),
			colorizeState(host.ConnectionState),
			fmt.Sprintf("%d cores", host.CPUCores),
			fmt.Sprintf("%.1f", float64(host.MemoryMB)/1024),
			fmt.Sprintf("%d", host.NumVMs),
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func displayClusters(clusters []vsphere.ClusterInfo) {
	pterm.DefaultSection.Println("Clusters")

	data := [][]string{
		{"Name", "Hosts", "CPUs", "Memory (GB)", "DRS", "HA"},
	}

	for _, cluster := range clusters {
		data = append(data, []string{
			truncate(cluster.Name, 30),
			fmt.Sprintf("%d", cluster.NumHosts),
			fmt.Sprintf("%d cores", cluster.NumCPUCores),
			fmt.Sprintf("%.1f", float64(cluster.TotalMemory)/1024),
			boolToIcon(cluster.DRSEnabled),
			boolToIcon(cluster.HAEnabled),
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func displayMetrics(metrics *vsphere.PerformanceMetrics) {
	if metrics == nil {
		return
	}

	pterm.DefaultSection.Printfln("Performance Metrics: %s (%s)", metrics.EntityName, metrics.EntityType)

	data := [][]string{
		{"Metric", "Value"},
		{"CPU Usage", fmt.Sprintf("%.2f%%", metrics.CPUPercent)},
		{"CPU (MHz)", fmt.Sprintf("%d MHz", metrics.CPUUsageMhz)},
		{"Memory Usage", fmt.Sprintf("%.2f%%", metrics.MemoryPercent)},
		{"Memory (MB)", fmt.Sprintf("%d MB", metrics.MemoryUsageMB)},
		{"Disk Read", fmt.Sprintf("%.2f MB/s", metrics.DiskReadMBps)},
		{"Disk Write", fmt.Sprintf("%.2f MB/s", metrics.DiskWriteMBps)},
		{"Network RX", fmt.Sprintf("%.2f MB/s", metrics.NetRxMBps)},
		{"Network TX", fmt.Sprintf("%.2f MB/s", metrics.NetTxMBps)},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithBoxed().
		WithData(data).
		Render()
}

func displayHistoricalMetrics(history *vsphere.MetricsHistory) {
	if history == nil || len(history.Samples) == 0 {
		pterm.Info.Println("No historical data available")
		return
	}

	pterm.DefaultSection.Printfln("Historical Metrics: %s (%d samples)", history.EntityName, len(history.Samples))

	data := [][]string{
		{"Time", "CPU %", "Memory %", "Disk R/W (MB/s)", "Net R/W (MB/s)"},
	}

	for _, sample := range history.Samples {
		data = append(data, []string{
			sample.Timestamp.Format("15:04:05"),
			fmt.Sprintf("%.1f%%", sample.CPUPercent),
			fmt.Sprintf("%.1f%%", sample.MemoryPercent),
			fmt.Sprintf("%.1f/%.1f", sample.DiskReadMBps, sample.DiskWriteMBps),
			fmt.Sprintf("%.1f/%.1f", sample.NetRxMBps, sample.NetTxMBps),
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func displayRealtimeMetrics(metricsData map[string]interface{}) {
	// Clear screen and reposition cursor for live update
	fmt.Print("\033[H\033[2J")

	pterm.DefaultHeader.Println("Real-time Performance Metrics")
	pterm.Println()

	// Extract metrics from map
	cpuPercent, _ := metricsData["cpu_percent"].(float64)
	memPercent, _ := metricsData["memory_percent"].(float64)

	// Create progress bars
	cpuBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("CPU Usage").Start()
	cpuBar.Current = int(cpuPercent)
	cpuBar.Stop()

	memBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Memory Usage").Start()
	memBar.Current = int(memPercent)
	memBar.Stop()

	pterm.Println()

	// Detailed stats
	statsData := [][]string{
		{"Metric", "Value"},
		{"CPU Usage", fmt.Sprintf("%.2f%%", cpuPercent)},
		{"Memory Usage", fmt.Sprintf("%.2f%%", memPercent)},
		{"Disk Read", fmt.Sprintf("%.2f MB/s", metricsData["disk_read_mbps"])},
		{"Disk Write", fmt.Sprintf("%.2f MB/s", metricsData["disk_write_mbps"])},
		{"Network RX", fmt.Sprintf("%.2f MB/s", metricsData["net_rx_mbps"])},
		{"Network TX", fmt.Sprintf("%.2f MB/s", metricsData["net_tx_mbps"])},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithBoxed().
		WithData(statsData).
		Render()

	pterm.Println()
	pterm.Info.Println("Press Ctrl+C to stop")
}

func displayResourcePools(pools []vsphere.ResourcePoolInfo) {
	pterm.DefaultSection.Println("Resource Pools")

	data := [][]string{
		{"Name", "CPU (MHz)", "Memory (GB)", "VMs", "Sub-Pools"},
	}

	for _, pool := range pools {
		cpuStr := "unlimited"
		if pool.CPULimitMhz > 0 {
			cpuStr = fmt.Sprintf("%d / %d", pool.CPUReservationMhz, pool.CPULimitMhz)
		}

		memStr := "unlimited"
		if pool.MemoryLimitMB > 0 {
			memStr = fmt.Sprintf("%d / %d", pool.MemoryReservationMB/1024, pool.MemoryLimitMB/1024)
		}

		data = append(data, []string{
			truncate(pool.Name, 30),
			cpuStr,
			memStr,
			fmt.Sprintf("%d", pool.NumVMs),
			fmt.Sprintf("%d", pool.NumSubPools),
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func displayEvents(events []vsphere.VCenterEvent) {
	pterm.DefaultSection.Println("vCenter Events")

	data := [][]string{
		{"Time", "Severity", "Type", "Entity", "Message"},
	}

	for _, evt := range events {
		data = append(data, []string{
			evt.CreatedTime.Format("15:04:05"),
			colorizeSeverity(evt.Severity),
			truncate(evt.EventType, 25),
			truncate(evt.EntityName, 20),
			truncate(evt.Message, 50),
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func displayEventStream(eventData map[string]interface{}) {
	severity, _ := eventData["severity"].(string)
	timestamp := time.Now().Format("15:04:05")
	eventType, _ := eventData["event_type"].(string)
	message, _ := eventData["message"].(string)
	entityName, _ := eventData["entity_name"].(string)

	line := fmt.Sprintf("[%s] %s %s %s: %s",
		timestamp,
		colorizeSeverity(severity),
		pterm.FgCyan.Sprintf("%-25s", truncate(eventType, 25)),
		pterm.FgYellow.Sprintf("%-20s", truncate(entityName, 20)),
		truncate(message, 80))

	pterm.Println(line)
}

func displayCloneResults(results []vsphere.CloneResult) {
	pterm.DefaultSection.Println("Clone Results")

	data := [][]string{
		{"Source", "Target", "Status", "Duration", "Error"},
	}

	for _, result := range results {
		status := pterm.FgGreen.Sprint("✓ Success")
		errorMsg := "-"

		if !result.Success {
			status = pterm.FgRed.Sprint("✗ Failed")
			errorMsg = truncate(result.Error, 40)
		}

		data = append(data, []string{
			truncate(result.SourceVM, 25),
			truncate(result.TargetName, 25),
			status,
			result.Duration.String(),
			errorMsg,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

// Helper functions

func colorizeState(state string) string {
	switch state {
	case "connected", "Connected":
		return pterm.FgGreen.Sprint(state)
	case "disconnected", "Disconnected":
		return pterm.FgRed.Sprint(state)
	case "notResponding", "NotResponding":
		return pterm.FgYellow.Sprint("Not Responding")
	default:
		return state
	}
}

func colorizeSeverity(severity string) string {
	switch severity {
	case "error":
		return pterm.FgRed.Sprint("ERROR")
	case "warning":
		return pterm.FgYellow.Sprint("WARN ")
	case "info":
		return pterm.FgCyan.Sprint("INFO ")
	default:
		return severity
	}
}

func boolToIcon(b bool) string {
	if b {
		return pterm.FgGreen.Sprint("✓")
	}
	return pterm.FgRed.Sprint("✗")
}
