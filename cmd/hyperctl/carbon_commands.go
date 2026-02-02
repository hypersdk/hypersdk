// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pterm/pterm"
)

// CarbonStatusResponse represents the grid carbon status response
type CarbonStatusResponse struct {
	Zone             string    `json:"zone"`
	CurrentIntensity float64   `json:"current_intensity"`
	RenewablePercent float64   `json:"renewable_percent"`
	OptimalForBackup bool      `json:"optimal_for_backup"`
	NextOptimalTime  *time.Time `json:"next_optimal_time,omitempty"`
	Forecast         []Forecast `json:"forecast_next_4h"`
	Reasoning        string    `json:"reasoning"`
	Quality          string    `json:"quality"`
	Timestamp        time.Time `json:"timestamp"`
}

// Forecast represents a carbon intensity forecast
type Forecast struct {
	Time      time.Time `json:"time"`
	Intensity float64   `json:"intensity_gco2_kwh"`
	Quality   string    `json:"quality"`
}

// CarbonReportResponse represents a carbon footprint report
type CarbonReportResponse struct {
	OperationID      string    `json:"operation_id"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	Duration         float64   `json:"duration_hours"`
	DataSize         float64   `json:"data_size_gb"`
	EnergyUsed       float64   `json:"energy_kwh"`
	CarbonIntensity  float64   `json:"carbon_intensity_gco2_kwh"`
	CarbonEmissions  float64   `json:"carbon_emissions_kg_co2"`
	SavingsVsWorst   float64   `json:"savings_vs_worst_kg_co2"`
	RenewablePercent float64   `json:"renewable_percent"`
	Equivalent       string    `json:"equivalent"`
}

// ZoneInfo represents information about a carbon zone
type ZoneInfo struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Region      string  `json:"region"`
	Description string  `json:"description"`
	Typical     float64 `json:"typical_intensity"`
}

// CarbonZonesResponse represents available carbon zones
type CarbonZonesResponse struct {
	Zones []ZoneInfo `json:"zones"`
	Total int        `json:"total"`
}

// CarbonEstimateResponse represents carbon savings estimate
type CarbonEstimateResponse struct {
	CurrentIntensity float64    `json:"current_intensity_gco2_kwh"`
	CurrentEmissions float64    `json:"current_emissions_kg_co2"`
	BestIntensity    float64    `json:"best_intensity_gco2_kwh"`
	BestEmissions    float64    `json:"best_emissions_kg_co2"`
	BestTime         *time.Time `json:"best_time,omitempty"`
	SavingsKgCO2     float64    `json:"savings_kg_co2"`
	SavingsPercent   float64    `json:"savings_percent"`
	Recommendation   string     `json:"recommendation"`
	DelayMinutes     float64    `json:"delay_minutes,omitempty"`
	Forecast         []Forecast `json:"forecast"`
}

// handleCarbonStatus gets current carbon status for a zone
func handleCarbonStatus(daemonURL, zone string, threshold float64, jsonOutput bool) {
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput {
		spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("🌍 Checking grid carbon status for %s...", zone))
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"zone":      zone,
		"threshold": threshold,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to prepare request: %v", err))
		}
		os.Exit(1)
	}

	// Make API request
	resp, err := apiRequest(daemonURL+"/carbon/status", "POST", "application/json", data)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to get carbon status: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		}
		os.Exit(1)
	}

	var statusResp CarbonStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("Retrieved carbon status for %s", zone))
	}

	// Display results
	if jsonOutput {
		output, _ := json.MarshalIndent(statusResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayCarbonStatus(statusResp)
	}
}

// displayCarbonStatus displays carbon status in a nice format
func displayCarbonStatus(status CarbonStatusResponse) {
	pterm.Println()
	pterm.DefaultSection.Println("⚡ Grid Carbon Status")
	pterm.Println()

	// Main status
	data := [][]string{
		{"Metric", "Value"},
		{"Zone", status.Zone},
		{"Carbon Intensity", fmt.Sprintf("%.1f gCO2/kWh", status.CurrentIntensity)},
		{"Quality", colorizeQuality(status.Quality)},
		{"Renewable Energy", fmt.Sprintf("%.1f%%", status.RenewablePercent)},
		{"Optimal for Backup", formatBool(status.OptimalForBackup)},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()

	// Reasoning
	pterm.Info.Println("💡 " + status.Reasoning)
	pterm.Println()

	// Forecast
	if len(status.Forecast) > 0 {
		pterm.DefaultSection.Println("📊 4-Hour Forecast")
		pterm.Println()

		forecastData := [][]string{
			{"Time", "Intensity (gCO2/kWh)", "Quality"},
		}

		for _, f := range status.Forecast {
			forecastData = append(forecastData, []string{
				f.Time.Format("15:04"),
				fmt.Sprintf("%.1f", f.Intensity),
				colorizeQuality(f.Quality),
			})
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed().
			WithData(forecastData).
			Render()

		pterm.Println()
	}

	// Next optimal time
	if status.NextOptimalTime != nil {
		delay := time.Until(*status.NextOptimalTime)
		pterm.Success.Printfln("⏰ Next optimal time: %s (in %s)",
			status.NextOptimalTime.Format("15:04:05"),
			formatCarbonDuration(delay))
	}
}

// handleCarbonReport generates a carbon report for a completed job
func handleCarbonReport(daemonURL, jobID, zone string, startTime, endTime time.Time, dataSizeGB float64, jsonOutput bool) {
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput {
		spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("📊 Generating carbon report for job %s...", jobID))
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"job_id":       jobID,
		"start_time":   startTime,
		"end_time":     endTime,
		"data_size_gb": dataSizeGB,
		"zone":         zone,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to prepare request: %v", err))
		}
		os.Exit(1)
	}

	// Make API request
	resp, err := apiRequest(daemonURL+"/carbon/report", "POST", "application/json", data)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to get carbon report: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		}
		os.Exit(1)
	}

	var reportResp CarbonReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&reportResp); err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("Generated carbon report for job %s", jobID))
	}

	// Display results
	if jsonOutput {
		output, _ := json.MarshalIndent(reportResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayCarbonReport(reportResp)
	}
}

// displayCarbonReport displays carbon report in a nice format
func displayCarbonReport(report CarbonReportResponse) {
	pterm.Println()
	pterm.DefaultSection.Println("🌿 Carbon Footprint Report")
	pterm.Println()

	// Job info
	data := [][]string{
		{"Metric", "Value"},
		{"Job ID", report.OperationID},
		{"Duration", fmt.Sprintf("%.1f hours", report.Duration)},
		{"Data Size", fmt.Sprintf("%.1f GB", report.DataSize)},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()

	// Carbon metrics
	pterm.DefaultSection.Println("⚡ Energy & Emissions")
	pterm.Println()

	metricsData := [][]string{
		{"Metric", "Value"},
		{"Energy Used", fmt.Sprintf("%.3f kWh", report.EnergyUsed)},
		{"Carbon Intensity", fmt.Sprintf("%.1f gCO2/kWh", report.CarbonIntensity)},
		{"Carbon Emissions", pterm.LightGreen(fmt.Sprintf("%.3f kg CO2", report.CarbonEmissions))},
		{"Renewable Energy", fmt.Sprintf("%.1f%%", report.RenewablePercent)},
		{"Savings vs Worst Case", pterm.Green(fmt.Sprintf("%.3f kg CO2", report.SavingsVsWorst))},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(metricsData).
		Render()

	pterm.Println()

	// Equivalent
	pterm.Info.Printfln("🚗 Equivalent: %s", report.Equivalent)
	pterm.Println()

	// Show success message
	pterm.Success.Println("✅ Carbon report generated successfully!")
}

// handleCarbonZones lists available carbon zones
func handleCarbonZones(daemonURL string, jsonOutput bool) {
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput {
		spinner, _ = pterm.DefaultSpinner.Start("🌍 Fetching available carbon zones...")
	}

	// Make API request
	resp, err := apiRequest(daemonURL+"/carbon/zones", "GET", "", nil)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to get carbon zones: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		}
		os.Exit(1)
	}

	var zonesResp CarbonZonesResponse
	if err := json.NewDecoder(resp.Body).Decode(&zonesResp); err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("Found %d carbon zones", zonesResp.Total))
	}

	// Display results
	if jsonOutput {
		output, _ := json.MarshalIndent(zonesResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayCarbonZones(zonesResp.Zones)
	}
}

// displayCarbonZones displays carbon zones in a nice format
func displayCarbonZones(zones []ZoneInfo) {
	pterm.Println()
	pterm.DefaultSection.Println("🌍 Available Carbon Zones")
	pterm.Println()

	// Group by region
	regions := make(map[string][]ZoneInfo)
	for _, zone := range zones {
		regions[zone.Region] = append(regions[zone.Region], zone)
	}

	// Display each region
	for region, regionZones := range regions {
		pterm.DefaultHeader.Printfln("📍 %s", region)
		pterm.Println()

		data := [][]string{
			{"Zone ID", "Name", "Typical Intensity", "Description"},
		}

		for _, zone := range regionZones {
			data = append(data, []string{
				zone.ID,
				zone.Name,
				fmt.Sprintf("%.0f gCO2/kWh", zone.Typical),
				truncate(zone.Description, 40),
			})
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed().
			WithData(data).
			Render()

		pterm.Println()
	}

	pterm.Info.Println("💡 Use zone ID with: hyperctl carbon status -zone <ZONE_ID>")
}

// handleCarbonEstimate estimates carbon savings from delaying a backup
func handleCarbonEstimate(daemonURL, zone string, dataSizeGB, durationHours float64, jsonOutput bool) {
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput {
		spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("🔮 Estimating carbon savings for %s...", zone))
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"zone":           zone,
		"data_size_gb":   dataSizeGB,
		"duration_hours": durationHours,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to prepare request: %v", err))
		}
		os.Exit(1)
	}

	// Make API request
	resp, err := apiRequest(daemonURL+"/carbon/estimate", "POST", "application/json", data)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to get estimate: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		}
		os.Exit(1)
	}

	var estimateResp CarbonEstimateResponse
	if err := json.NewDecoder(resp.Body).Decode(&estimateResp); err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	if spinner != nil {
		spinner.Success("Generated carbon savings estimate")
	}

	// Display results
	if jsonOutput {
		output, _ := json.MarshalIndent(estimateResp, "", "  ")
		fmt.Println(string(output))
	} else {
		displayCarbonEstimate(estimateResp)
	}
}

// displayCarbonEstimate displays carbon estimate in a nice format
func displayCarbonEstimate(estimate CarbonEstimateResponse) {
	pterm.Println()
	pterm.DefaultSection.Println("🔮 Carbon Savings Estimate")
	pterm.Println()

	// Current vs Best
	data := [][]string{
		{"Scenario", "Intensity (gCO2/kWh)", "Emissions (kg CO2)"},
		{"Run Now", fmt.Sprintf("%.1f", estimate.CurrentIntensity), fmt.Sprintf("%.3f", estimate.CurrentEmissions)},
		{"Run at Best Time", pterm.Green(fmt.Sprintf("%.1f", estimate.BestIntensity)), pterm.Green(fmt.Sprintf("%.3f", estimate.BestEmissions))},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()

	// Savings
	pterm.DefaultSection.Println("💰 Potential Savings")
	pterm.Println()

	savingsData := [][]string{
		{"Metric", "Value"},
		{"CO2 Savings", pterm.LightGreen(fmt.Sprintf("%.3f kg CO2", estimate.SavingsKgCO2))},
		{"Reduction", pterm.LightGreen(fmt.Sprintf("%.1f%%", estimate.SavingsPercent))},
	}

	if estimate.BestTime != nil {
		delay := time.Until(*estimate.BestTime)
		savingsData = append(savingsData, []string{
			"Best Time",
			estimate.BestTime.Format("15:04:05"),
		})
		savingsData = append(savingsData, []string{
			"Delay Required",
			formatCarbonDuration(delay),
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(savingsData).
		Render()

	pterm.Println()

	// Recommendation
	pterm.Info.Printfln("💡 %s", estimate.Recommendation)
	pterm.Println()

	// Forecast if available
	if len(estimate.Forecast) > 0 {
		pterm.DefaultSection.Println("📊 4-Hour Forecast")
		pterm.Println()

		forecastData := [][]string{
			{"Time", "Intensity (gCO2/kWh)", "Quality"},
		}

		for _, f := range estimate.Forecast {
			forecastData = append(forecastData, []string{
				f.Time.Format("15:04"),
				fmt.Sprintf("%.1f", f.Intensity),
				colorizeQuality(f.Quality),
			})
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed().
			WithData(forecastData).
			Render()

		pterm.Println()
	}
}

// Helper functions

// colorizeQuality adds color to quality levels
func colorizeQuality(quality string) string {
	switch quality {
	case "excellent":
		return pterm.Green(quality)
	case "good":
		return pterm.LightGreen(quality)
	case "moderate":
		return pterm.Yellow(quality)
	case "poor":
		return pterm.LightRed(quality)
	case "very poor":
		return pterm.Red(quality)
	default:
		return quality
	}
}

// formatBool formats a boolean for display
func formatBool(b bool) string {
	if b {
		return pterm.Green("✓ Yes")
	}
	return pterm.Red("✗ No")
}

// formatCarbonDuration formats a duration in a human-readable way
func formatCarbonDuration(d time.Duration) string {
	if d < 0 {
		return "now"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// parseTime parses a time string in multiple formats
func parseTime(timeStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
}
