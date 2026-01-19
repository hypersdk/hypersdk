// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pterm/pterm"

	"hypersdk/config"
	"hypersdk/logger"
	"hypersdk/providers/vsphere"
)

// Command-line flags
var (
	vmName         = flag.String("vm", "", "VM name to export (skips interactive selection)")
	providerType   = flag.String("provider", "vsphere", "Provider type (vsphere, aws, azure, gcp, hyperv)")
	outputDir      = flag.String("output", "", "Output directory (default: ./export-<vmname>)")
	format         = flag.String("format", "ovf", "Export format: ovf or ova")
	compress       = flag.Bool("compress", false, "Enable compression for OVA exports")
	verify         = flag.Bool("verify", false, "Verify export with checksum validation")
	dryRun         = flag.Bool("dry-run", false, "Preview export without actually exporting")
	batchFile      = flag.String("batch", "", "File containing list of VMs to export (one per line)")
	filter         = flag.String("filter", "", "Filter VMs by tag (format: key=value)")
	folder         = flag.String("folder", "", "Filter VMs by folder path")
	powerOff       = flag.Bool("power-off", false, "Automatically power off VM before export")
	parallel       = flag.Int("parallel", 4, "Number of parallel downloads")
	quiet          = flag.Bool("quiet", false, "Minimal output (for scripting)")
	showVersion    = flag.Bool("version", false, "Show version and exit")
)

func main() {
	// Parse flags
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Println("HyperExport v0.2.0")
		fmt.Println("Multi-cloud VM export tool")
		os.Exit(0)
	}

	// Create intro animation (skip if quiet mode)
	if !*quiet {
		showIntro()
	}

	// Load configuration
	cfg := config.FromEnvironment()

	// Override config with flags
	if *parallel > 0 {
		cfg.DownloadWorkers = *parallel
	}

	// Setup logging
	log := logger.New(cfg.LogLevel)

	// Validate required environment variables
	if cfg.VCenterURL == "" || cfg.Username == "" || cfg.Password == "" {
		log.Error("missing required environment variables",
			"required", "GOVC_URL, GOVC_USERNAME, GOVC_PASSWORD")

		pterm.Error.Println("Missing required environment variables")
		pterm.DefaultBox.WithTitle("Required Environment Variables").
			WithTitleTopCenter().
			WithBoxStyle(pterm.NewStyle(pterm.FgRed)).
			Println("export GOVC_URL=https://vcenter.example.com/sdk\n" +
				"export GOVC_USERNAME=administrator@vsphere.local\n" +
				"export GOVC_PASSWORD=your-password\n" +
				"export GOVC_INSECURE=1  # (optional, for self-signed certs)")
		os.Exit(1)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Info("received signal, shutting down", "signal", sig)
		pterm.Warning.Printfln("Received signal %v, shutting down gracefully...", sig)
		cancel()
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}()

	// Run the application
	if err := run(ctx, cfg, log); err != nil {
		log.Error("application failed", "error", err)
		pterm.Error.Printfln("Application failed: %v", err)
		os.Exit(1)
	}

	pterm.Success.Println("Application completed successfully!")
}

func showIntro() {
	// Clear screen
	pterm.DefaultCenter.Println()

	// Orange/amber color scheme (Claude-inspired)
	orange := pterm.NewStyle(pterm.FgLightRed)
	amber := pterm.NewStyle(pterm.FgYellow)

	// Show big text logo
	bigText, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("HYPER", orange),
		pterm.NewLettersFromStringWithStyle("EXPORT", amber),
	).Srender()

	pterm.DefaultCenter.Println(bigText)

	// Show subtitle
	subtitle := pterm.DefaultCenter.Sprint(pterm.LightYellow("Interactive VM Export Tool"))
	version := pterm.DefaultCenter.Sprint(pterm.LightRed("Version 1.0.0"))

	pterm.Println(subtitle)
	pterm.Println(version)
	pterm.Println()
}

func run(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	// Handle batch mode
	if *batchFile != "" {
		return runBatchExport(ctx, cfg, log)
	}

	// Connection spinner
	var spinner *pterm.SpinnerPrinter
	if !*quiet {
		spinner, _ = pterm.DefaultSpinner.Start("Connecting to " + *providerType + "...")
	}

	log.Info("connecting to provider", "type", *providerType)

	client, err := vsphere.NewVSphereClient(ctx, cfg, log)
	if err != nil {
		if spinner != nil {
			spinner.Fail("Failed to connect to " + *providerType)
		}
		return fmt.Errorf("connect to provider: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Error("failed to close client", "error", err)
		}
	}()

	if spinner != nil {
		spinner.Success("Connected to " + *providerType + " successfully!")
	}

	// Show connection info panel (skip in quiet mode)
	if !*quiet {
		pterm.DefaultBox.WithTitle("Connection Info").
			WithTitleTopLeft().
			WithBoxStyle(pterm.NewStyle(pterm.FgCyan)).
			Printfln("vCenter: %s\nUser: %s",
				cfg.VCenterURL,
				cfg.Username)
	}

	// Determine selected VM
	var selectedVM string
	if *vmName != "" {
		// VM specified via command line
		selectedVM = *vmName
		log.Info("using VM from command line", "vm", selectedVM)
	} else {
		// Discover VMs with spinner
		if !*quiet {
			spinner, _ = pterm.DefaultSpinner.Start("Discovering virtual machines...")
		}

		log.Info("discovering VMs")
		vms, err := client.FindAllVMs(ctx)
		if err != nil {
			if spinner != nil {
				spinner.Fail("Failed to discover VMs")
			}
			return fmt.Errorf("find VMs: %w", err)
		}

		// Apply filters
		if *folder != "" {
			vms = filterByFolder(vms, *folder)
		}
		if *filter != "" {
			// Tag filtering would require API support - skip for now
			log.Info("tag filtering not yet implemented", "filter", *filter)
		}

		if spinner != nil {
			spinner.Success(fmt.Sprintf("Found %d virtual machine(s)", len(vms)))
		}
		log.Info("found VMs", "count", len(vms))

		if len(vms) == 0 {
			log.Warn("no VMs found")
			if !*quiet {
				pterm.Warning.Println("No VMs found")
			}
			return nil
		}

		// Interactive VM selection
		selectedVM, err = selectVMInteractive(vms, log)
		if err != nil {
			return fmt.Errorf("select VM: %w", err)
		}
	}

	// Get VM info with spinner
	if !*quiet {
		spinner, _ = pterm.DefaultSpinner.Start("Retrieving VM information...")
	}

	info, err := client.GetVMInfo(ctx, selectedVM)
	if err != nil {
		if spinner != nil {
			spinner.Fail("Failed to get VM info")
		}
		return fmt.Errorf("get VM info: %w", err)
	}

	if spinner != nil {
		spinner.Success("Retrieved VM information")
	}

	// Display VM info in a beautiful panel (skip in quiet mode)
	if !*quiet {
		displayVMInfo(info)
	}

	log.Info("selected VM",
		"name", info.Name,
		"powerState", info.PowerState,
		"memoryMB", info.MemoryMB,
		"cpus", info.NumCPU,
		"os", info.GuestOS)

	// Dry-run mode
	if *dryRun {
		if !*quiet {
			pterm.Info.Println("Dry-run mode: Export preview")
			pterm.DefaultSection.Println("Export Plan")
			fmt.Printf("  VM: %s\n", info.Name)
			fmt.Printf("  Format: %s\n", *format)
			fmt.Printf("  Compression: %v\n", *compress)
			fmt.Printf("  Output: %s\n", getOutputDir(info.Name))
			fmt.Printf("  Estimated Size: %s\n", formatBytes(info.Storage))
		} else {
			fmt.Printf("dry-run: %s -> %s (%s, %s)\n", info.Name, getOutputDir(info.Name), *format, formatBytes(info.Storage))
		}
		return nil
	}

	// Shutdown if needed
	if info.PowerState == "poweredOn" {
		shouldPowerOff := *powerOff

		// Interactive confirmation if not specified via flag
		if !shouldPowerOff && !*quiet {
			pterm.Info.Println("VM is currently powered on")
			result, _ := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Do you want to shutdown the VM before export?").
				WithDefaultValue(false).
				Show()
			shouldPowerOff = result
		}

		if shouldPowerOff {
			log.Warn("VM is powered on, attempting graceful shutdown")

			if !*quiet {
				spinner, _ = pterm.DefaultSpinner.Start("Shutting down VM gracefully...")
			}

			shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Minute)
			defer shutdownCancel()

			if err := client.ShutdownVM(shutdownCtx, selectedVM, 5*time.Minute); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					if spinner != nil {
						spinner.Warning("Graceful shutdown timeout, forcing power off...")
					}
					log.Warn("graceful shutdown timeout, forcing power off")

					if err := client.PowerOffVM(ctx, selectedVM); err != nil {
						if spinner != nil {
							spinner.Fail("Failed to power off VM")
						}
						return fmt.Errorf("force power off: %w", err)
					}
				} else {
					if spinner != nil {
						spinner.Fail("Failed to shutdown VM")
					}
					return fmt.Errorf("shutdown VM: %w", err)
				}
			}
			if spinner != nil {
				spinner.Success("VM powered off successfully")
			}
		} else if !*quiet {
			pterm.Warning.Println("Continuing with VM powered on (not recommended)")
		}
	}

	// Export VM
	exportDir := getOutputDir(info.Name)

	opts := vsphere.DefaultExportOptions()
	opts.OutputPath = exportDir
	opts.ParallelDownloads = cfg.DownloadWorkers
	opts.ShowIndividualProgress = cfg.LogLevel == "debug"
	opts.Format = *format
	opts.Compress = *compress

	if !*quiet {
		pterm.Info.Printfln("Starting %s export to: %s", strings.ToUpper(*format), exportDir)
	}
	log.Info("starting export", "vm", info.Name, "output", exportDir, "format", *format, "compress", *compress)

	result, err := client.ExportOVF(ctx, selectedVM, opts)
	if err != nil {
		if !*quiet {
			pterm.Error.Printfln("Export failed: %v", err)
		}
		return fmt.Errorf("export %s: %w", *format, err)
	}

	// Create OVA if requested
	if *format == "ova" {
		if !*quiet {
			spinner, _ = pterm.DefaultSpinner.Start("Packaging as OVA...")
		}
		log.Info("creating OVA archive")

		ovaPath := filepath.Join(exportDir, sanitizeForPath(info.Name)+".ova")
		compressionLevel := 6 // Default gzip compression level
		if err := vsphere.CreateOVA(exportDir, ovaPath, *compress, compressionLevel, log); err != nil {
			if spinner != nil {
				spinner.Fail("Failed to create OVA")
			}
			return fmt.Errorf("create OVA: %w", err)
		}

		if spinner != nil {
			spinner.Success("OVA created successfully")
		}
		result.OVAPath = ovaPath
		result.Format = "ova"
	}

	// Verify export if requested
	if *verify {
		if !*quiet {
			spinner, _ = pterm.DefaultSpinner.Start("Verifying export with checksums...")
		}
		log.Info("verifying export")

		checksums, err := verifyExport(result, log)
		if err != nil {
			if spinner != nil {
				spinner.Fail("Verification failed")
			}
			return fmt.Errorf("verify export: %w", err)
		}

		if spinner != nil {
			spinner.Success("Export verified successfully")
		}

		// Save checksums to file
		checksumFile := filepath.Join(exportDir, "checksums.txt")
		if err := saveChecksums(checksumFile, checksums); err != nil {
			log.Error("failed to save checksums", "error", err)
		}
	}

	// Show export summary in a fancy table (skip in quiet mode)
	if !*quiet {
		showExportSummary(info, result)
	} else {
		fmt.Printf("success: %s exported to %s (%s)\n", info.Name, exportDir, formatBytes(result.TotalSize))
	}

	log.Info("export completed successfully",
		"duration", result.Duration.Round(time.Second),
		"totalSize", formatBytes(result.TotalSize),
		"files", len(result.Files),
		"output", result.OutputDir)

	// Celebration (skip in quiet mode)
	if !*quiet {
		pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgLightGreen)).
			WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
			Println("Export Completed Successfully!")
	}

	return nil
}

func selectVMInteractive(vms []string, log logger.Logger) (string, error) {
	if len(vms) == 1 {
		log.Info("auto-selecting only VM", "vm", vms[0])
		pterm.Info.Println("Auto-selecting the only available VM")
		return vms[0], nil
	}

	// Extract VM names for display
	vmNames := make([]string, len(vms))
	for i, vm := range vms {
		parts := strings.Split(vm, "/")
		vmNames[i] = parts[len(parts)-1]
	}

	// Interactive select with search
	selectedName, err := pterm.DefaultInteractiveSelect.
		WithOptions(vmNames).
		WithDefaultText("Select a VM to export").
		WithFilter(true).
		Show()

	if err != nil {
		return "", err
	}

	// Find the full path
	for i, name := range vmNames {
		if name == selectedName {
			pterm.Success.Printfln("Selected: %s", selectedName)
			return vms[i], nil
		}
	}

	return "", fmt.Errorf("VM not found")
}

func displayVMInfo(info *vsphere.VMInfo) {
	// Create table data
	data := pterm.TableData{
		{"Property", "Value"},
		{"Name", info.Name},
		{"Power State", getPowerStateIcon(info.PowerState) + " " + info.PowerState},
		{"Guest OS", info.GuestOS},
		{"Memory", fmt.Sprintf("%d MB", info.MemoryMB)},
		{"CPUs", fmt.Sprintf("%d", info.NumCPU)},
		{"Storage", formatBytes(info.Storage)},
	}

	// Render table
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func showExportSummary(info *vsphere.VMInfo, result *vsphere.ExportResult) {
	// Create summary table
	data := pterm.TableData{
		{"Metric", "Value"},
		{"VM Name", info.Name},
		{"Duration", result.Duration.Round(time.Second).String()},
		{"Total Size", formatBytes(result.TotalSize)},
		{"Files Exported", fmt.Sprintf("%d", len(result.Files))},
		{"Output Directory", result.OutputDir},
	}

	pterm.DefaultSection.Println("Export Summary")
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	// Show file list in a panel if there are files
	if len(result.Files) > 0 && len(result.Files) <= 10 {
		pterm.DefaultSection.Println("Exported Files")
		fileList := pterm.DefaultBulletList
		items := make([]pterm.BulletListItem, 0, len(result.Files))
		for _, file := range result.Files {
			items = append(items, pterm.BulletListItem{
				Level: 0,
				Text:  file,
			})
		}
		fileList.WithItems(items).Render()
	}
}

func getPowerStateIcon(state string) string {
	switch state {
	case "poweredOn":
		return pterm.Green("●")
	case "poweredOff":
		return pterm.Red("●")
	case "suspended":
		return pterm.Yellow("●")
	default:
		return pterm.Gray("●")
	}
}

func sanitizeForPath(name string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '|', '?', '*', '/', '\\':
			return '_'
		default:
			return r
		}
	}, name)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func getOutputDir(vmName string) string {
	if *outputDir != "" {
		return *outputDir
	}
	return "./export-" + sanitizeForPath(vmName)
}

func filterByFolder(vms []string, folder string) []string {
	filtered := make([]string, 0)
	for _, vm := range vms {
		if strings.Contains(vm, folder) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

func runBatchExport(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	// Read VM list from file
	data, err := os.ReadFile(*batchFile)
	if err != nil {
		return fmt.Errorf("read batch file: %w", err)
	}

	vmList := strings.Split(string(data), "\n")
	validVMs := make([]string, 0)
	for _, vm := range vmList {
		vm = strings.TrimSpace(vm)
		if vm != "" && !strings.HasPrefix(vm, "#") {
			validVMs = append(validVMs, vm)
		}
	}

	if len(validVMs) == 0 {
		return fmt.Errorf("no valid VMs in batch file")
	}

	log.Info("batch export", "count", len(validVMs))
	if !*quiet {
		pterm.Info.Printfln("Batch export: %d VMs", len(validVMs))
	}

	// Connect once for all exports
	client, err := vsphere.NewVSphereClient(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("connect to vSphere: %w", err)
	}
	defer client.Close()

	successCount := 0
	failureCount := 0

	for i, vmPath := range validVMs {
		if !*quiet {
			pterm.DefaultSection.Printfln("Exporting VM %d/%d: %s", i+1, len(validVMs), vmPath)
		}

		// Temporarily set vmName flag
		originalVMName := *vmName
		*vmName = vmPath

		// Run export
		if err := run(ctx, cfg, log); err != nil {
			log.Error("batch export failed", "vm", vmPath, "error", err)
			if !*quiet {
				pterm.Error.Printfln("Failed to export %s: %v", vmPath, err)
			}
			failureCount++
		} else {
			successCount++
		}

		// Restore original vmName
		*vmName = originalVMName
	}

	if !*quiet {
		pterm.DefaultSection.Println("Batch Export Summary")
		pterm.Info.Printfln("Total: %d | Success: %d | Failed: %d", len(validVMs), successCount, failureCount)
	} else {
		fmt.Printf("batch-summary: total=%d success=%d failed=%d\n", len(validVMs), successCount, failureCount)
	}

	if failureCount > 0 {
		return fmt.Errorf("%d exports failed", failureCount)
	}

	return nil
}

func verifyExport(result *vsphere.ExportResult, log logger.Logger) (map[string]string, error) {
	checksums := make(map[string]string)

	files := result.Files
	if result.OVAPath != "" {
		// Verify OVA file instead of individual files
		files = []string{result.OVAPath}
	}

	for _, filePath := range files {
		hash, err := calculateSHA256(filePath)
		if err != nil {
			log.Error("failed to calculate checksum", "file", filePath, "error", err)
			return nil, fmt.Errorf("checksum %s: %w", filePath, err)
		}
		checksums[filepath.Base(filePath)] = hash
		log.Info("calculated checksum", "file", filepath.Base(filePath), "sha256", hash)
	}

	return checksums, nil
}

func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func saveChecksums(filename string, checksums map[string]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for name, hash := range checksums {
		fmt.Fprintf(file, "%s  %s\n", hash, name)
	}

	return nil
}
