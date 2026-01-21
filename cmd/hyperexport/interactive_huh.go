// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"

	"hypersdk/config"
	"hypersdk/logger"
	"hypersdk/providers/vsphere"
)

// Export templates
type exportTemplate struct {
	name        string
	description string
	format      string
	compress    bool
	verify      bool
}

var templates = []exportTemplate{
	{
		name:        "Quick Export",
		description: "Fast export without compression (OVF)",
		format:      "ovf",
		compress:    false,
		verify:      false,
	},
	{
		name:        "Production Backup",
		description: "OVA with compression and verification",
		format:      "ova",
		compress:    true,
		verify:      true,
	},
	{
		name:        "Development",
		description: "OVF for fast development cycles",
		format:      "ovf",
		compress:    false,
		verify:      false,
	},
	{
		name:        "Archive",
		description: "Compressed OVA for long-term storage",
		format:      "ova",
		compress:    true,
		verify:      true,
	},
}

// Orange theme colors
var (
	orangePrimary   = lipgloss.Color("#FF9E64") // Vibrant peach/orange
	orangeSecondary = lipgloss.Color("#E0AF68") // Golden amber
	orangeDark      = lipgloss.Color("#D35400") // Deep orange
)

// runInteractiveHuh runs the new huh-based interactive TUI
func runInteractiveHuh(ctx context.Context, client *vsphere.VSphereClient, cfg *config.Config, log logger.Logger) error {
	// Set orange theme for huh
	theme := huh.ThemeBase()
	theme.Focused.Base = theme.Focused.Base.BorderForeground(orangePrimary)
	theme.Focused.Title = theme.Focused.Title.Foreground(orangePrimary).Bold(true)
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(orangePrimary)
	theme.Focused.MultiSelectSelector = theme.Focused.MultiSelectSelector.Foreground(orangePrimary)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(orangePrimary)

	// Print banner
	printBanner()

	// Get output directory
	outputDirPath := *outputDir
	if outputDirPath == "" {
		outputDirPath = "./exports"
	}

	// Step 1: Load VMs
	spinner := newOrangeSpinner("Loading VMs from vSphere...")
	vms, err := client.ListVMs(ctx)
	spinner.Stop()
	if err != nil {
		return fmt.Errorf("failed to load VMs: %w", err)
	}

	if len(vms) == 0 {
		pterm.Warning.Println("No VMs found")
		return nil
	}

	pterm.Success.Printf("Found %d VMs\n\n", len(vms))

	// Step 2: VM Selection with search/filter
	selectedVMs, err := selectVMs(vms, theme)
	if err != nil {
		return err
	}

	// Step 3: Export configuration
	exportConfig, err := configureExport(outputDirPath, theme)
	if err != nil {
		return err
	}

	// Step 4: Confirm and execute
	if err := confirmAndExecute(ctx, client, selectedVMs, exportConfig, log, theme); err != nil {
		return err
	}

	return nil
}

// selectVMs presents a multi-select interface for choosing VMs
func selectVMs(vms []vsphere.VMInfo, theme *huh.Theme) ([]vsphere.VMInfo, error) {
	// Sort VMs by name
	sort.Slice(vms, func(i, j int) bool {
		return strings.ToLower(vms[i].Name) < strings.ToLower(vms[j].Name)
	})

	// Create options for multi-select
	options := make([]huh.Option[string], len(vms))
	vmMap := make(map[string]vsphere.VMInfo)

	for i, vm := range vms {
		// Format VM info
		powerIcon := "○"
		if vm.PowerState == "poweredOn" {
			powerIcon = "⚡"
		}

		label := fmt.Sprintf("%s %-30s │ %2d CPU │ %4.0f GB RAM │ %s",
			powerIcon,
			truncate(vm.Name, 30),
			vm.NumCPU,
			float64(vm.MemoryMB)/1024,
			formatBytes(vm.Storage),
		)

		options[i] = huh.NewOption(label, vm.Path)
		vmMap[vm.Path] = vm
	}

	var selectedPaths []string

	// Loop until at least one VM is selected
	for {
		// Create the form with multi-select and orange theme
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select VMs to Export").
					Description("Use arrow keys to navigate, space to select, enter to confirm").
					Options(options...).
					Value(&selectedPaths).
					Height(15).
					Filterable(true),
			),
		).WithTheme(theme)

		if err := form.Run(); err != nil {
			return nil, err
		}

		// Check if at least one VM is selected
		if len(selectedPaths) == 0 {
			pterm.Warning.Println("Please select at least one VM to export")
			pterm.Println()
			continue
		}

		// Valid selection - break out of loop
		break
	}

	// Map selected paths back to VM objects
	selected := make([]vsphere.VMInfo, 0, len(selectedPaths))
	for _, path := range selectedPaths {
		if vm, ok := vmMap[path]; ok {
			selected = append(selected, vm)
		}
	}

	return selected, nil
}

// exportConfiguration holds export settings
type exportConfiguration struct {
	outputDir    string
	templateName string
	format       string
	compress     bool
	verify       bool
	parallel     int
	cloudUpload  bool
	parallelStr  string // for form input
	customizeStr string // for form confirmation
}

// configureExport presents export configuration options
func configureExport(defaultOutputDir string, theme *huh.Theme) (*exportConfiguration, error) {
	config := &exportConfiguration{
		outputDir:   defaultOutputDir,
		parallel:    4,
		parallelStr: "4",
	}

	// Template selection
	templateOptions := make([]huh.Option[string], len(templates))
	for i, t := range templates {
		templateOptions[i] = huh.NewOption(
			fmt.Sprintf("%s - %s", t.name, t.description),
			t.name,
		)
	}

	var templateName string
	var customFormat string
	var compress bool
	var verify bool
	var customize bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Export Template").
				Description("Choose a predefined export configuration").
				Options(templateOptions...).
				Value(&templateName),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Output Directory").
				Description("Where to save exported VMs").
				Value(&config.outputDir).
				Placeholder(defaultOutputDir),

			huh.NewInput().
				Title("Parallel Downloads").
				Description("Number of concurrent file downloads (1-8)").
				Value(&config.parallelStr).
				Placeholder("4").
				Validate(func(s string) error {
					var num int
					if _, err := fmt.Sscanf(s, "%d", &num); err != nil {
						return fmt.Errorf("must be a number")
					}
					if num < 1 || num > 8 {
						return fmt.Errorf("must be between 1 and 8")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Advanced Options").
				Description("Do you want to customize the export format?").
				Affirmative("Yes, customize").
				Negative("No, use template").
				Value(&customize),
		).WithHideFunc(func() bool {
			return templateName == ""
		}),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Export Format").
				Options(
					huh.NewOption("OVF (Open Virtualization Format)", "ovf"),
					huh.NewOption("OVA (Open Virtualization Archive)", "ova"),
				).
				Value(&customFormat),

			huh.NewConfirm().
				Title("Enable Compression").
				Description("Compress the export (recommended for OVA)").
				Value(&compress),

			huh.NewConfirm().
				Title("Enable Verification").
				Description("Verify export integrity with checksums").
				Value(&verify),
		).WithHideFunc(func() bool {
			return !customize
		}),
	).WithTheme(theme)

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Convert parallel string to int
	if config.parallelStr != "" {
		if _, err := fmt.Sscanf(config.parallelStr, "%d", &config.parallel); err != nil {
			return nil, fmt.Errorf("invalid parallel downloads value: %w", err)
		}
	}

	// Apply template or custom settings
	if customize && customFormat != "" {
		// Custom settings
		config.format = customFormat
		config.compress = compress
		config.verify = verify
		config.templateName = "Custom"
	} else {
		// Apply template
		for _, t := range templates {
			if t.name == templateName {
				config.format = t.format
				config.compress = t.compress
				config.verify = t.verify
				config.templateName = t.name
				break
			}
		}
	}

	return config, nil
}

// confirmAndExecute shows summary and executes the export
func confirmAndExecute(ctx context.Context, client *vsphere.VSphereClient, vms []vsphere.VMInfo, cfg *exportConfiguration, log logger.Logger, theme *huh.Theme) error {
	// Calculate totals
	var totalCPU int32
	var totalMemoryMB int32
	var totalStorage int64

	for _, vm := range vms {
		totalCPU += vm.NumCPU
		totalMemoryMB += vm.MemoryMB
		totalStorage += vm.Storage
	}

	// Build summary with pterm for better terminal compatibility
	colorOrange := pterm.NewRGB(211, 84, 0)

	pterm.DefaultSection.Println("Export Summary")
	pterm.Println()

	// VM summary
	pterm.Printf("  %s  %d\n", colorOrange.Sprint("VMs Selected:     "), len(vms))
	pterm.Printf("  %s  %d\n", colorOrange.Sprint("Total CPUs:       "), totalCPU)
	pterm.Printf("  %s  %.1f GB\n", colorOrange.Sprint("Total Memory:     "), float64(totalMemoryMB)/1024)
	pterm.Printf("  %s  %s\n", colorOrange.Sprint("Total Storage:    "), formatBytes(totalStorage))
	pterm.Println()

	// Configuration
	pterm.Printf("  %s  %s\n", colorOrange.Sprint("Template:         "), cfg.templateName)
	pterm.Printf("  %s  %s\n", colorOrange.Sprint("Format:           "), strings.ToUpper(cfg.format))
	pterm.Printf("  %s  %v\n", colorOrange.Sprint("Compression:      "), cfg.compress)
	pterm.Printf("  %s  %v\n", colorOrange.Sprint("Verification:     "), cfg.verify)
	pterm.Printf("  %s  %d\n", colorOrange.Sprint("Parallel:         "), cfg.parallel)
	pterm.Printf("  %s  %s\n", colorOrange.Sprint("Output Directory: "), cfg.outputDir)
	pterm.Println()

	// Confirmation
	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Proceed with Export?").
				Description("This will export the selected VMs").
				Affirmative("Yes, export!").
				Negative("Cancel").
				Value(&confirm),
		),
	).WithTheme(theme)

	if err := confirmForm.Run(); err != nil {
		return err
	}

	if !confirm {
		pterm.Info.Println("Export cancelled")
		return nil
	}

	// Execute exports
	pterm.Info.Println("\nStarting export...")
	fmt.Println()

	for i, vm := range vms {
		pterm.DefaultSection.Printf("Exporting VM %d/%d: %s", i+1, len(vms), vm.Name)

		sanitized := sanitizeFilename(vm.Name)
		vmOutputDir := filepath.Join(cfg.outputDir, sanitized)

		// Validate path to prevent directory traversal
		absOutputDir, err := filepath.Abs(cfg.outputDir)
		if err != nil {
			return fmt.Errorf("resolve output directory path: %w", err)
		}
		absVMDir, err := filepath.Abs(vmOutputDir)
		if err != nil {
			return fmt.Errorf("resolve VM output directory path: %w", err)
		}
		if !strings.HasPrefix(absVMDir, absOutputDir+string(filepath.Separator)) {
			return fmt.Errorf("security: invalid VM name would escape output directory")
		}

		if err := os.MkdirAll(vmOutputDir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}

		// Create export options
		opts := vsphere.ExportOptions{
			Format:            cfg.format,
			OutputPath:        vmOutputDir,
			ParallelDownloads: cfg.parallel,
			ValidateChecksum:  cfg.verify,
			Compress:          cfg.compress,
			CleanupOVF:        cfg.format == "ova", // Clean up OVF files after creating OVA
		}

		// Export the VM
		spinner := newOrangeSpinner(fmt.Sprintf("Exporting %s...", vm.Name))

		startTime := time.Now()
		result, err := client.ExportOVF(ctx, vm.Path, opts)
		duration := time.Since(startTime)

		spinner.Stop()

		if err != nil {
			pterm.Error.Printf("Failed to export %s: %v\n", vm.Name, err)

			// Ask if user wants to continue
			var continueExport bool
			continueForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Continue with remaining VMs?").
						Value(&continueExport),
				),
			).WithTheme(theme)
			if err := continueForm.Run(); err != nil || !continueExport {
				return fmt.Errorf("export aborted")
			}
			continue
		}

		exportPath := result.OVFPath
		if result.Format == "ova" {
			exportPath = result.OVAPath
		}

		pterm.Success.Printf("Exported %s to %s (took %s)\n", vm.Name, exportPath, duration.Round(time.Second))
		log.Info("VM exported successfully",
			"vm", vm.Name,
			"path", exportPath,
			"format", result.Format,
			"size", formatBytes(result.TotalSize),
			"duration", duration.String(),
		)
		fmt.Println()
	}

	// Final summary
	pterm.Success.Printf("\n✓ Successfully exported %d VMs to %s\n", len(vms), cfg.outputDir)

	return nil
}

// Helper functions
func printBanner() {
	// Orange colors
	colorOrange := pterm.NewRGB(211, 84, 0)  // Deep orange #D35400
	styleOrange := pterm.NewStyle(pterm.FgLightRed) // Background style

	pterm.Println()
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		pterm.DefaultHeader.WithFullWidth(false).
			WithBackgroundStyle(styleOrange).
			WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
			WithMargin(4).
			Sprint("    HyperSDK    "),
	)
	pterm.Println()

	// Subtitle and version with orange color
	subtitle := pterm.DefaultCenter.Sprint(
		pterm.NewStyle(pterm.FgYellow).Sprint("Interactive VM export tool"))
	version := pterm.DefaultCenter.Sprint(
		colorOrange.Sprint("Version 1.0.0"))

	pterm.Println(subtitle)
	pterm.Println(version)
	pterm.Println()

	pterm.Info.Println("Use arrow keys to navigate, space to select, enter to confirm")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func sanitizeFilename(name string) string {
	// Replace invalid filename characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
