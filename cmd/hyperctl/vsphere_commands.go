// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pterm/pterm"

	"hypersdk/daemon/models"
	"hypersdk/providers/vsphere"
)

// Host/Cluster Management Commands

func handleHost(daemonURL string, operation, pattern string, jsonOutput bool) {
	switch operation {
	case "list":
		handleListHosts(daemonURL, pattern, jsonOutput)
	case "info":
		handleListHosts(daemonURL, pattern, jsonOutput)
	default:
		pterm.Error.Printfln("Unknown operation: %s", operation)
		os.Exit(1)
	}
}

func handleListHosts(daemonURL, pattern string, jsonOutput bool) {
	spinner, _ := pterm.DefaultSpinner.Start("Listing ESXi hosts...")

	url := fmt.Sprintf("%s/vsphere/hosts?pattern=%s", daemonURL, pattern)
	resp, err := apiRequest(url, "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var hostResp models.HostInfoResponse
	json.Unmarshal(body, &hostResp)

	spinner.Success(fmt.Sprintf("Found %d hosts", hostResp.Total))

	if jsonOutput {
		output, _ := json.MarshalIndent(hostResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayHosts(hostResp.Hosts)
	}
}

func handleCluster(daemonURL string, operation, pattern string, jsonOutput bool) {
	switch operation {
	case "list":
		handleListClusters(daemonURL, pattern, jsonOutput)
	case "info":
		handleListClusters(daemonURL, pattern, jsonOutput)
	default:
		pterm.Error.Printfln("Unknown operation: %s", operation)
		os.Exit(1)
	}
}

func handleListClusters(daemonURL, pattern string, jsonOutput bool) {
	spinner, _ := pterm.DefaultSpinner.Start("Listing clusters...")

	url := fmt.Sprintf("%s/vsphere/clusters?pattern=%s", daemonURL, pattern)
	resp, err := apiRequest(url, "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var clusterResp models.ClusterInfoResponse
	json.Unmarshal(body, &clusterResp)

	spinner.Success(fmt.Sprintf("Found %d clusters", clusterResp.Total))

	if jsonOutput {
		output, _ := json.MarshalIndent(clusterResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayClusters(clusterResp.Clusters)
	}
}

// Performance Metrics Commands

func handleMetrics(daemonURL, entity, entityType string, realtime bool, start, end, interval string, watch, jsonOutput bool) {
	if watch {
		handleMetricsWatch(daemonURL, entity, entityType)
		return
	}

	if realtime {
		handleRealtimeMetrics(daemonURL, entity, entityType, jsonOutput)
	} else {
		handleHistoricalMetrics(daemonURL, entity, entityType, start, end, interval, jsonOutput)
	}
}

func handleRealtimeMetrics(daemonURL, entity, entityType string, jsonOutput bool) {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Fetching realtime metrics for %s...", entity))

	url := fmt.Sprintf("%s/vsphere/metrics?entity=%s&type=%s&realtime=true", daemonURL, entity, entityType)
	resp, err := apiRequest(url, "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var metricsResp models.MetricsResponse
	json.Unmarshal(body, &metricsResp)

	spinner.Success("Metrics retrieved")

	if jsonOutput {
		output, _ := json.MarshalIndent(metricsResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayMetrics(metricsResp.Current)
	}
}

func handleHistoricalMetrics(daemonURL, entity, entityType, start, end, interval string, jsonOutput bool) {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Fetching historical metrics for %s...", entity))

	url := fmt.Sprintf("%s/vsphere/metrics?entity=%s&type=%s&interval=%s", daemonURL, entity, entityType, interval)
	if start != "" {
		url += "&start=" + start
	}
	if end != "" {
		url += "&end=" + end
	}

	resp, err := apiRequest(url, "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var metricsResp models.MetricsResponse
	json.Unmarshal(body, &metricsResp)

	spinner.Success(fmt.Sprintf("Retrieved %d samples", len(metricsResp.History.Samples)))

	if jsonOutput {
		output, _ := json.MarshalIndent(metricsResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayHistoricalMetrics(metricsResp.History)
	}
}

func handleMetricsWatch(daemonURL, entity, entityType string) {
	wsURL := strings.Replace(daemonURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/vsphere/metrics/stream?entity=%s&type=%s", wsURL, entity, entityType)

	pterm.Info.Printfln("Connecting to metrics stream for %s...", entity)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		pterm.Error.Printfln("WebSocket connection failed: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	pterm.Success.Printfln("Connected! Streaming metrics (Ctrl+C to stop)...")
	pterm.Println()

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			pterm.Error.Printfln("Connection error: %v", err)
			break
		}

		if msg.Type == "metrics" {
			displayRealtimeMetrics(msg.Data)
		} else if msg.Type == "error" {
			pterm.Error.Printfln("Error: %s", msg.Data["message"])
			break
		}
	}
}

// Resource Pool Commands

func handlePool(daemonURL, operation, name, parent string, cpuReserve, cpuLimit, memReserve, memLimit int64, jsonOutput bool) {
	switch operation {
	case "list":
		handleListPools(daemonURL, name, jsonOutput)
	case "create":
		handleCreatePool(daemonURL, name, parent, cpuReserve, cpuLimit, memReserve, memLimit)
	case "update":
		handleUpdatePool(daemonURL, name, cpuReserve, cpuLimit, memReserve, memLimit)
	case "delete":
		handleDeletePool(daemonURL, name)
	default:
		pterm.Error.Printfln("Unknown operation: %s", operation)
		os.Exit(1)
	}
}

func handleListPools(daemonURL, pattern string, jsonOutput bool) {
	spinner, _ := pterm.DefaultSpinner.Start("Listing resource pools...")

	url := fmt.Sprintf("%s/vsphere/pools", daemonURL)
	if pattern != "" {
		url += "?pattern=" + pattern
	}

	resp, err := apiRequest(url, "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var poolResp models.ResourcePoolResponse
	json.Unmarshal(body, &poolResp)

	spinner.Success(fmt.Sprintf("Found %d resource pools", poolResp.Total))

	if jsonOutput {
		output, _ := json.MarshalIndent(poolResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayResourcePools(poolResp.Pools)
	}
}

func handleCreatePool(daemonURL, name, parent string, cpuReserve, cpuLimit, memReserve, memLimit int64) {
	if name == "" || parent == "" {
		pterm.Error.Println("Pool name and parent required")
		os.Exit(1)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Creating resource pool %s...", name))

	config := vsphere.ResourcePoolConfig{
		Name:                name,
		ParentPath:          parent,
		CPUReservationMhz:   cpuReserve,
		CPULimitMhz:         cpuLimit,
		CPUExpandable:       true,
		CPUShares:           "normal",
		MemoryReservationMB: memReserve,
		MemoryLimitMB:       memLimit,
		MemoryExpandable:    true,
		MemoryShares:        "normal",
	}

	reqBody, _ := json.Marshal(models.CreateResourcePoolRequest{Config: config})
	resp, err := apiRequest(daemonURL+"/vsphere/pools", "POST", "application/json", reqBody)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	spinner.Success(fmt.Sprintf("Resource pool %s created successfully", name))
}

func handleUpdatePool(daemonURL, name string, cpuReserve, cpuLimit, memReserve, memLimit int64) {
	if name == "" {
		pterm.Error.Println("Pool name required")
		os.Exit(1)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Updating resource pool %s...", name))

	config := vsphere.ResourcePoolConfig{
		Name:                name,
		CPUReservationMhz:   cpuReserve,
		CPULimitMhz:         cpuLimit,
		MemoryReservationMB: memReserve,
		MemoryLimitMB:       memLimit,
	}

	reqBody, _ := json.Marshal(models.UpdateResourcePoolRequest{Config: config})
	url := fmt.Sprintf("%s/vsphere/pools/%s", daemonURL, name)
	resp, err := apiRequest(url, "PUT", "application/json", reqBody)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	spinner.Success(fmt.Sprintf("Resource pool %s updated successfully", name))
}

func handleDeletePool(daemonURL, name string) {
	if name == "" {
		pterm.Error.Println("Pool name required")
		os.Exit(1)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Deleting resource pool %s...", name))

	url := fmt.Sprintf("%s/vsphere/pools/%s", daemonURL, name)
	resp, err := apiRequest(url, "DELETE", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	spinner.Success(fmt.Sprintf("Resource pool %s deleted successfully", name))
}

// Event Monitoring Commands

func handleEvents(daemonURL, since string, eventTypes []string, follow, jsonOutput bool) {
	if follow {
		handleEventsFollow(daemonURL, eventTypes)
		return
	}

	spinner, _ := pterm.DefaultSpinner.Start("Fetching events...")

	url := fmt.Sprintf("%s/vsphere/events", daemonURL)
	if since != "" {
		// Parse duration like "1h", "24h", "7d"
		duration, err := parseDuration(since)
		if err == nil {
			sinceTime := time.Now().Add(-duration)
			url += "?since=" + sinceTime.Format(time.RFC3339)
		}
	}
	if len(eventTypes) > 0 {
		url += "&types=" + strings.Join(eventTypes, ",")
	}

	resp, err := apiRequest(url, "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var eventResp models.EventResponse
	json.Unmarshal(body, &eventResp)

	spinner.Success(fmt.Sprintf("Found %d events", eventResp.Total))

	if jsonOutput {
		output, _ := json.MarshalIndent(eventResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayEvents(eventResp.Events)
	}
}

func handleEventsFollow(daemonURL string, eventTypes []string) {
	wsURL := strings.Replace(daemonURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/vsphere/events/stream"
	if len(eventTypes) > 0 {
		wsURL += "?types=" + strings.Join(eventTypes, ",")
	}

	pterm.Info.Println("Connecting to event stream...")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		pterm.Error.Printfln("WebSocket connection failed: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	pterm.Success.Println("Connected! Streaming events (Ctrl+C to stop)...")
	pterm.Println()

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			pterm.Error.Printfln("Connection error: %v", err)
			break
		}

		if msg.Type == "event" {
			displayEventStream(msg.Data)
		} else if msg.Type == "error" {
			pterm.Error.Printfln("Error: %s", msg.Data["message"])
			break
		}
	}
}

// VM Cloning Commands

func handleClone(daemonURL string, spec vsphere.CloneSpec, bulkFile string) {
	if bulkFile != "" {
		handleBulkClone(daemonURL, bulkFile)
		return
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Cloning %s -> %s...", spec.SourceVM, spec.TargetName))

	reqBody, _ := json.Marshal(models.CloneVMRequest{Spec: spec})
	resp, err := apiRequest(daemonURL+"/vsphere/clone", "POST", "application/json", reqBody)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var cloneResp models.CloneVMResponse
	json.Unmarshal(body, &cloneResp)

	if cloneResp.Result.Success {
		spinner.Success(fmt.Sprintf("Cloned successfully in %v", cloneResp.Result.Duration))
		showSuccessArt()
		pterm.Info.Printfln("Source: %s", cloneResp.Result.SourceVM)
		pterm.Info.Printfln("Target: %s (%s)", cloneResp.Result.TargetName, cloneResp.Result.TargetPath)
	} else {
		spinner.Fail(fmt.Sprintf("Clone failed: %s", cloneResp.Result.Error))
		os.Exit(1)
	}
}

func handleBulkClone(daemonURL, bulkFile string) {
	// Read bulk clone specs from YAML/JSON file
	data, err := os.ReadFile(bulkFile)
	if err != nil {
		pterm.Error.Printfln("Failed to read file: %v", err)
		os.Exit(1)
	}

	var bulkReq models.BulkCloneRequest
	if err := json.Unmarshal(data, &bulkReq); err != nil {
		pterm.Error.Printfln("Failed to parse file: %v", err)
		os.Exit(1)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Cloning %d VMs...", len(bulkReq.Specs)))

	reqBody, _ := json.Marshal(bulkReq)
	resp, err := apiRequest(daemonURL+"/vsphere/clone/bulk", "POST", "application/json", reqBody)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var bulkResp models.BulkCloneResponse
	json.Unmarshal(body, &bulkResp)

	spinner.Success(fmt.Sprintf("Bulk clone completed in %v", bulkResp.Duration))
	pterm.Info.Printfln("Success: %d, Failed: %d", bulkResp.Success, bulkResp.Failed)

	// Display results
	displayCloneResults(bulkResp.Results)
}

// Helper function to parse duration strings like "1h", "24h", "7d"
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var d int
		fmt.Sscanf(days, "%d", &d)
		return time.Duration(d) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// WebSocket message type
type WSMessage struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}
