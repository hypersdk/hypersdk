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

	tea "github.com/charmbracelet/bubbletea"
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
	interactive    = flag.Bool("interactive", false, "Launch advanced interactive TUI mode")
	tui            = flag.Bool("tui", false, "Launch advanced interactive TUI mode (alias for -interactive)")
	validateOnly   = flag.Bool("validate-only", false, "Only run pre-export validation checks")
	resume         = flag.Bool("resume", false, "Resume interrupted export from checkpoint")
	showHistory    = flag.Bool("history", false, "Show export history")
	historyLimit   = flag.Int("history-limit", 10, "Number of recent exports to show in history")
	generateReport = flag.Bool("report", false, "Generate export statistics report")
	reportFile     = flag.String("report-file", "", "Save report to file instead of stdout")
	clearHistory   = flag.Bool("clear-history", false, "Clear export history")
	uploadTo       = flag.String("upload", "", "Upload export to cloud storage (s3://bucket/path, azure://container/path, gs://bucket/path, sftp://host/path)")
	streamUpload   = flag.Bool("stream-upload", false, "Stream export directly to cloud (no local storage)")
	keepLocal      = flag.Bool("keep-local", true, "Keep local copy after cloud upload")
	encrypt        = flag.Bool("encrypt", false, "Encrypt export files")
	encryptMethod  = flag.String("encrypt-method", "aes256", "Encryption method: aes256 or gpg")
	passphrase     = flag.String("passphrase", "", "Encryption passphrase")
	keyFile        = flag.String("keyfile", "", "Encryption key file")
	gpgRecipient   = flag.String("gpg-recipient", "", "GPG recipient email for encryption")
	profile        = flag.String("profile", "", "Use saved export profile")
	saveProfile    = flag.String("save-profile", "", "Save current settings as a profile")
	listProfiles   = flag.Bool("list-profiles", false, "List available profiles")
	deleteProfile  = flag.String("delete-profile", "", "Delete a saved profile")
	createDefaults = flag.Bool("create-default-profiles", false, "Create default profiles")

	// Artifact Manifest v1.0 options
	generateManifest    = flag.Bool("manifest", false, "Generate Artifact Manifest v1.0 for hyper2kvm")
	verifyManifestFlag  = flag.Bool("verify-manifest", false, "Verify manifest after generation")
	manifestChecksum    = flag.Bool("manifest-checksum", true, "Compute SHA-256 checksums for disks in manifest")
	manifestTargetFormat = flag.String("manifest-target", "qcow2", "Target disk format for conversion (qcow2, raw, vdi)")

	// Automatic conversion options (Phase 2)
	autoConvert         = flag.Bool("convert", false, "Automatically convert with hyper2kvm after export")
	hyper2kvmBinary     = flag.String("hyper2kvm-binary", "", "Path to hyper2kvm binary (auto-detect if empty)")
	conversionTimeout   = flag.Duration("conversion-timeout", 2*time.Hour, "Timeout for hyper2kvm conversion")
	streamConversion    = flag.Bool("stream-conversion", true, "Stream hyper2kvm output to console")
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

	// Setup logging (needed for history and profile operations)
	cfg := config.FromEnvironment()
	log := logger.New(cfg.LogLevel)

	// Handle profile operations (don't require provider connection)
	profileManager, err := NewProfileManager(log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create profile manager: %v\n", err)
		os.Exit(1)
	}

	if *createDefaults {
		if err := profileManager.CreateDefaultProfiles(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create default profiles: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Default profiles created successfully")
		fmt.Println("Profiles created:")
		fmt.Println("  - quick-export: Fast export without compression")
		fmt.Println("  - production-backup: OVA with compression and verification")
		fmt.Println("  - encrypted-backup: Encrypted backup for sensitive data")
		fmt.Println("  - cloud-backup: Backup and upload to cloud storage")
		fmt.Println("  - development: Quick export for development/testing")
		os.Exit(0)
	}

	if *listProfiles {
		profiles, err := profileManager.ListProfiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to list profiles: %v\n", err)
			os.Exit(1)
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found. Create some with --create-default-profiles or --save-profile")
			os.Exit(0)
		}

		fmt.Println("\n=== Available Export Profiles ===")
		for _, p := range profiles {
			fmt.Printf("Profile: %s\n", p.Name)
			fmt.Printf("  Description: %s\n", p.Description)
			fmt.Printf("  Format: %s", p.Format)
			if p.Compress {
				fmt.Printf(" (compressed)")
			}
			fmt.Println()
			if p.Encrypt {
				fmt.Printf("  Encryption: %s\n", p.EncryptMethod)
			}
			if p.UploadTo != "" {
				fmt.Printf("  Upload to: %s\n", p.UploadTo)
			}
			fmt.Printf("  Created: %s\n", p.Created.Format("2006-01-02 15:04:05"))
			fmt.Println()
		}
		os.Exit(0)
	}

	if *deleteProfile != "" {
		if err := profileManager.DeleteProfile(*deleteProfile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to delete profile: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Profile '%s' deleted successfully\n", *deleteProfile)
		os.Exit(0)
	}

	// Load profile if specified
	if *profile != "" {
		loadedProfile, err := profileManager.LoadProfile(*profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to load profile: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Loaded profile: %s\n", loadedProfile.Name)
		fmt.Printf("Description: %s\n", loadedProfile.Description)

		// Apply profile settings to flags (override current values)
		*format = loadedProfile.Format
		*compress = loadedProfile.Compress
		*verify = loadedProfile.Verify
		*powerOff = loadedProfile.PowerOff
		*parallel = loadedProfile.Parallel
		if loadedProfile.UploadTo != "" {
			*uploadTo = loadedProfile.UploadTo
		}
		*keepLocal = loadedProfile.KeepLocal
		*encrypt = loadedProfile.Encrypt
		*encryptMethod = loadedProfile.EncryptMethod
		if loadedProfile.GPGRecipient != "" {
			*gpgRecipient = loadedProfile.GPGRecipient
		}
		*validateOnly = loadedProfile.ValidateOnly

		// Apply manifest settings
		*generateManifest = loadedProfile.GenerateManifest
		*verifyManifestFlag = loadedProfile.VerifyManifest
		*manifestChecksum = loadedProfile.ManifestChecksum
		if loadedProfile.ManifestTargetFormat != "" {
			*manifestTargetFormat = loadedProfile.ManifestTargetFormat
		}

		// Apply conversion settings (Phase 2)
		*autoConvert = loadedProfile.AutoConvert
		if loadedProfile.Hyper2KVMBinary != "" {
			*hyper2kvmBinary = loadedProfile.Hyper2KVMBinary
		}
		*streamConversion = loadedProfile.StreamConversion
	}

	// Handle history operations (don't require provider connection)
	if *showHistory || *generateReport || *clearHistory {
		historyFile, err := GetDefaultHistoryFile()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get history file: %v\n", err)
			os.Exit(1)
		}

		history := NewExportHistory(historyFile, log)

		if *clearHistory {
			if err := history.ClearHistory(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to clear history: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Export history cleared successfully")
			os.Exit(0)
		}

		if *showHistory {
			entries, err := history.GetRecentExports(*historyLimit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to get history: %v\n", err)
				os.Exit(1)
			}

			if len(entries) == 0 {
				fmt.Println("No export history found")
				os.Exit(0)
			}

			fmt.Printf("\n=== Export History (Last %d) ===\n\n", len(entries))
			for i, entry := range entries {
				status := "✓ SUCCESS"
				if !entry.Success {
					status = "✗ FAILED"
				}

				fmt.Printf("%d. %s [%s]\n", i+1, entry.VMName, status)
				fmt.Printf("   Time: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
				fmt.Printf("   Format: %s | Size: %s | Duration: %s\n",
					entry.Format,
					formatBytes(entry.TotalSize),
					entry.Duration.Round(time.Second))
				fmt.Printf("   Output: %s\n", entry.OutputDir)

				if !entry.Success && entry.ErrorMessage != "" {
					fmt.Printf("   Error: %s\n", entry.ErrorMessage)
				}
				fmt.Println()
			}
			os.Exit(0)
		}

		if *generateReport {
			report := NewExportReport(history)

			if *reportFile != "" {
				if err := report.SaveReportToFile(*reportFile, true, 20); err != nil {
					fmt.Fprintf(os.Stderr, "Error: failed to save report: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Report saved to: %s\n", *reportFile)
			} else {
				reportText, err := report.GenerateReport(true, 20)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: failed to generate report: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(reportText)
			}
			os.Exit(0)
		}
	}

	// Create intro animation (skip if quiet mode)
	if !*quiet {
		showIntro()
	}

	// Override config with flags
	if *parallel > 0 {
		cfg.DownloadWorkers = *parallel
	}

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
	if !*quiet && !*interactive && !*tui {
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

	// Launch interactive TUI mode if requested
	if *interactive || *tui {
		return runInteractiveTUI(ctx, client, cfg, log)
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

	// Pre-export validation
	if !*quiet {
		spinner, _ = pterm.DefaultSpinner.Start("Running pre-export validation...")
	}

	preValidator := NewPreExportValidator(log)
	preReport := preValidator.ValidateExport(ctx, *info, exportDir, info.Storage)

	if spinner != nil {
		if preReport.AllPassed {
			spinner.Success("Pre-export validation passed")
		} else {
			spinner.Warning("Pre-export validation completed with issues")
		}
	}

	// Display validation results
	if !*quiet {
		displayValidationReport("Pre-Export Validation", preReport)
	}

	// Stop if validation-only mode
	if *validateOnly {
		if preReport.AllPassed {
			pterm.Success.Println("Validation completed - VM is ready for export")
			return nil
		} else {
			pterm.Error.Println("Validation failed - fix issues before exporting")
			return fmt.Errorf("validation failed")
		}
	}

	// Stop if validation failed (unless warnings only)
	if !preReport.AllPassed {
		if !*quiet {
			pterm.Error.Println("Pre-export validation failed - cannot proceed")
		}
		return fmt.Errorf("pre-export validation failed")
	}

	// Warn about validation warnings
	if preReport.HasWarnings && !*quiet {
		pterm.Warning.Println("Pre-export validation has warnings - proceeding anyway")
		if !*quiet && !*powerOff {
			result, _ := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Continue with export despite warnings?").
				WithDefaultValue(true).
				Show()
			if !result {
				return fmt.Errorf("export cancelled by user")
			}
		}
	}

	opts := vsphere.DefaultExportOptions()
	opts.OutputPath = exportDir
	opts.ParallelDownloads = cfg.DownloadWorkers
	opts.ShowIndividualProgress = cfg.LogLevel == "debug"
	opts.Format = *format
	opts.Compress = *compress

	// Artifact Manifest v1.0 options
	opts.GenerateManifest = *generateManifest
	opts.VerifyManifest = *verifyManifestFlag
	opts.ManifestComputeChecksum = *manifestChecksum
	opts.ManifestTargetFormat = *manifestTargetFormat

	// Automatic conversion options (Phase 2)
	opts.AutoConvert = *autoConvert
	opts.Hyper2KVMBinary = *hyper2kvmBinary
	opts.ConversionTimeout = *conversionTimeout
	opts.StreamConversionOutput = *streamConversion

	// If auto-convert is enabled, force manifest generation
	if opts.AutoConvert {
		opts.GenerateManifest = true
		if !*quiet {
			pterm.Info.Println("Auto-convert enabled: manifest generation forced")
		}
	}

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

	// Post-export validation
	if !*quiet {
		spinner, _ = pterm.DefaultSpinner.Start("Running post-export validation...")
	}

	postValidator := NewPostExportValidator(log)
	postReport := postValidator.ValidateExportedFiles(exportDir)

	if spinner != nil {
		if postReport.AllPassed {
			spinner.Success("Post-export validation passed")
		} else {
			spinner.Warning("Post-export validation completed with issues")
		}
	}

	// Display validation results
	if !*quiet {
		displayValidationReport("Post-Export Validation", postReport)
	}

	// Warn if post-validation failed
	if !postReport.AllPassed {
		log.Warn("post-export validation failed", "checks", len(postReport.Checks))
		if !*quiet {
			pterm.Warning.Println("Post-export validation detected issues with exported files")
		}
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

	// Encrypt export if requested
	if *encrypt {
		if !*quiet {
			spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Encrypting export with %s...", *encryptMethod))
		}
		log.Info("encrypting export", "method", *encryptMethod)

		encConfig := &EncryptionConfig{
			Method:       EncryptionMethod(*encryptMethod),
			Passphrase:   *passphrase,
			KeyFile:      *keyFile,
			GPGRecipient: *gpgRecipient,
		}

		// Check if passphrase/key is provided
		if encConfig.Passphrase == "" && encConfig.KeyFile == "" && encConfig.GPGRecipient == "" {
			if spinner != nil {
				spinner.Fail("Encryption failed: no passphrase, key file, or GPG recipient provided")
			}
			return fmt.Errorf("encryption requires passphrase, key file, or GPG recipient")
		}

		encryptor := NewEncryptor(encConfig, log)

		// Encrypt all files in export directory
		encryptedDir := exportDir + "-encrypted"
		if err := encryptor.EncryptDirectory(exportDir, encryptedDir); err != nil {
			if spinner != nil {
				spinner.Fail("Encryption failed")
			}
			return fmt.Errorf("encrypt export: %w", err)
		}

		if spinner != nil {
			spinner.Success("Export encrypted successfully")
		}

		// Replace export directory with encrypted directory
		if err := os.RemoveAll(exportDir); err != nil {
			log.Warn("failed to remove unencrypted export", "error", err)
		}
		if err := os.Rename(encryptedDir, exportDir); err != nil {
			return fmt.Errorf("rename encrypted directory: %w", err)
		}

		log.Info("export encrypted successfully", "method", *encryptMethod)
	}

	// Upload to cloud storage if requested
	if *uploadTo != "" {
		if !*quiet {
			spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Uploading to %s...", *uploadTo))
		}
		log.Info("uploading export to cloud", "destination", *uploadTo)

		cloudStorage, err := NewCloudStorage(*uploadTo, log)
		if err != nil {
			if spinner != nil {
				spinner.Fail("Failed to initialize cloud storage")
			}
			log.Error("failed to create cloud storage client", "error", err)
			if !*quiet {
				pterm.Error.Printfln("Cloud upload failed: %v", err)
			}
		} else {
			defer cloudStorage.Close()

			// Upload the export directory
			remotePath := sanitizeForPath(info.Name)
			if err := UploadDirectory(ctx, cloudStorage, exportDir, remotePath, log); err != nil {
				if spinner != nil {
					spinner.Fail("Upload failed")
				}
				log.Error("failed to upload export", "error", err)
				if !*quiet {
					pterm.Error.Printfln("Cloud upload failed: %v", err)
				}
			} else {
				if spinner != nil {
					spinner.Success(fmt.Sprintf("Uploaded to %s successfully", *uploadTo))
				}
				log.Info("export uploaded to cloud successfully", "destination", *uploadTo)

				// Delete local copy if requested
				if !*keepLocal {
					if !*quiet {
						spinner, _ = pterm.DefaultSpinner.Start("Removing local copy...")
					}
					log.Info("removing local export", "path", exportDir)

					if err := os.RemoveAll(exportDir); err != nil {
						if spinner != nil {
							spinner.Warning("Failed to remove local copy")
						}
						log.Warn("failed to remove local export", "error", err)
					} else {
						if spinner != nil {
							spinner.Success("Local copy removed")
						}
					}
				}
			}
		}
	}

	// Record export in history
	historyFile, err := GetDefaultHistoryFile()
	if err == nil {
		history := NewExportHistory(historyFile, log)
		historyEntry := ExportHistoryEntry{
			Timestamp:  time.Now(),
			VMName:     info.Name,
			VMPath:     selectedVM,
			Provider:   *providerType,
			Format:     result.Format,
			OutputDir:  result.OutputDir,
			TotalSize:  result.TotalSize,
			Duration:   result.Duration,
			FilesCount: len(result.Files),
			Success:    true,
			Compressed: *compress,
			Verified:   *verify,
			Metadata: map[string]string{
				"uploaded_to": *uploadTo,
			},
		}
		if err := history.RecordExport(historyEntry); err != nil {
			log.Warn("failed to record export in history", "error", err)
		}
	}

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

	// Add manifest path if generated
	if result.ManifestPath != "" {
		data = append(data, []string{"Artifact Manifest", result.ManifestPath})
	}

	// Add conversion results if present (Phase 2)
	if result.ConversionResult != nil {
		convStatus := pterm.Red("FAILED")
		if result.ConversionResult.Success {
			convStatus = pterm.Green("SUCCESS")
		}
		data = append(data, []string{"Conversion Status", convStatus})

		if result.ConversionResult.Success {
			data = append(data, []string{"Converted Files", fmt.Sprintf("%d", len(result.ConversionResult.ConvertedFiles))})
			data = append(data, []string{"Conversion Duration", result.ConversionResult.Duration.Round(time.Second).String()})
			if result.ConversionResult.ReportPath != "" {
				data = append(data, []string{"Conversion Report", result.ConversionResult.ReportPath})
			}
		} else if result.ConversionResult.Error != "" {
			data = append(data, []string{"Conversion Error", result.ConversionResult.Error})
		}
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

	// Show converted files if present (Phase 2)
	if result.ConversionResult != nil && result.ConversionResult.Success && len(result.ConversionResult.ConvertedFiles) > 0 {
		pterm.DefaultSection.Println("Converted Files (KVM-Ready)")
		fileList := pterm.DefaultBulletList
		items := make([]pterm.BulletListItem, 0, len(result.ConversionResult.ConvertedFiles))
		for _, file := range result.ConversionResult.ConvertedFiles {
			items = append(items, pterm.BulletListItem{
				Level: 0,
				Text:  pterm.Green(file),
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

// displayValidationReport displays validation results in a nice table
func displayValidationReport(title string, report *ValidationReport) {
	pterm.DefaultSection.Println(title)

	// Create table data
	data := pterm.TableData{
		{"Check", "Status", "Details"},
	}

	for _, check := range report.Checks {
		var status string
		var statusColor pterm.Color

		if !check.Passed {
			status = "✗ FAIL"
			statusColor = pterm.FgRed
		} else if check.Warning {
			status = "⚠ WARN"
			statusColor = pterm.FgYellow
		} else {
			status = "✓ PASS"
			statusColor = pterm.FgGreen
		}

		data = append(data, []string{
			check.Name,
			pterm.NewStyle(statusColor).Sprint(status),
			check.Message,
		})
	}

	// Render table
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	// Summary
	if report.AllPassed {
		if report.HasWarnings {
			pterm.Info.Printfln("Validation passed with %d warnings", countWarnings(report))
		} else {
			pterm.Success.Println("All validation checks passed")
		}
	} else {
		failedCount := countFailed(report)
		pterm.Error.Printfln("%d validation check(s) failed", failedCount)
	}

	fmt.Println()
}

// countWarnings counts the number of warnings in a validation report
func countWarnings(report *ValidationReport) int {
	count := 0
	for _, check := range report.Checks {
		if check.Warning {
			count++
		}
	}
	return count
}

// countFailed counts the number of failed checks in a validation report
func countFailed(report *ValidationReport) int {
	count := 0
	for _, check := range report.Checks {
		if !check.Passed {
			count++
		}
	}
	return count
}

// runInteractiveTUI launches the advanced interactive TUI mode
func runInteractiveTUI(ctx context.Context, client *vsphere.VSphereClient, cfg *config.Config, log logger.Logger) error {
	// Get output directory
	outputDirPath := *outputDir
	if outputDirPath == "" {
		outputDirPath = "./exports"
	}

	// Create initial model
	m := tuiModel{
		vms:            []tuiVMItem{},
		filteredVMs:    []tuiVMItem{},
		cursor:         0,
		phase:          "loading",
		sortMode:       "name",
		client:         client,
		outputDir:      outputDirPath,
		log:            log,
		ctx:            ctx,
	}

	// Run the TUI program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
