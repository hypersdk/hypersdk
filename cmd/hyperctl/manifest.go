// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

// ManifestPipeline represents the pipeline configuration
type ManifestPipeline struct {
	Load struct {
		SourceType string `json:"source_type" yaml:"source_type"`
		SourcePath string `json:"source_path" yaml:"source_path"`
	} `json:"load" yaml:"load"`

	Inspect struct {
		Enabled  bool `json:"enabled" yaml:"enabled"`
		DetectOS bool `json:"detect_os" yaml:"detect_os"`
	} `json:"inspect" yaml:"inspect"`

	Fix struct {
		Fstab struct {
			Enabled bool   `json:"enabled" yaml:"enabled"`
			Mode    string `json:"mode" yaml:"mode"`
		} `json:"fstab" yaml:"fstab"`

		Grub struct {
			Enabled bool `json:"enabled" yaml:"enabled"`
		} `json:"grub" yaml:"grub"`

		Initramfs struct {
			Enabled    bool `json:"enabled" yaml:"enabled"`
			Regenerate bool `json:"regenerate" yaml:"regenerate"`
		} `json:"initramfs" yaml:"initramfs"`

		Network struct {
			Enabled  bool   `json:"enabled" yaml:"enabled"`
			FixLevel string `json:"fix_level" yaml:"fix_level"`
		} `json:"network" yaml:"network"`
	} `json:"fix" yaml:"fix"`

	Convert struct {
		OutputFormat string `json:"output_format" yaml:"output_format"`
		Compress     bool   `json:"compress" yaml:"compress"`
		OutputPath   string `json:"output_path,omitempty" yaml:"output_path,omitempty"`
	} `json:"convert" yaml:"convert"`

	Validate struct {
		Enabled  bool `json:"enabled" yaml:"enabled"`
		BootTest bool `json:"boot_test" yaml:"boot_test"`
	} `json:"validate" yaml:"validate"`
}

// Manifest represents a conversion manifest
type Manifest struct {
	Version  string           `json:"version" yaml:"version"`
	Batch    bool             `json:"batch,omitempty" yaml:"batch,omitempty"`
	Pipeline ManifestPipeline `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	VMs      []struct {
		Name     string           `json:"name" yaml:"name"`
		Pipeline ManifestPipeline `json:"pipeline" yaml:"pipeline"`
	} `json:"vms,omitempty" yaml:"vms,omitempty"`
}

// handleManifestCmd handles manifest-related commands
func handleManifestCmd(daemonURL string, action string, args []string) {
	switch action {
	case "create":
		handleManifestCreate(args)
	case "validate":
		handleManifestValidate(args)
	case "submit":
		handleManifestSubmit(daemonURL, args)
	case "generate":
		handleManifestGenerate(args)
	default:
		pterm.Error.Printfln("Unknown manifest action: %s", action)
		pterm.Info.Println("Available actions: create, validate, submit, generate")
		os.Exit(1)
	}
}

// handleManifestCreate creates a new manifest interactively
func handleManifestCreate(args []string) {
	pterm.DefaultHeader.Println("📝 Manifest Builder")
	pterm.Println()

	manifest := Manifest{
		Version: "1.0",
	}

	currentStep := 1
	totalSteps := 4

	for {
		switch currentStep {
		case 1:
			// Step 1: Source configuration
			pterm.DefaultSection.Printfln("Step %d/%d: Source Configuration", currentStep, totalSteps)
			pterm.Println()

			sourceType := promptSelectWithBack("Select source type:", []string{
				"vmdk", "ova", "ovf", "vhd", "vhdx", "raw", "ami",
			}, currentStep > 1)

			if sourceType == "<back>" {
				currentStep--
				continue
			}
			manifest.Pipeline.Load.SourceType = sourceType

			sourcePath := promptInputWithBack("Enter source path:", "/path/to/disk."+sourceType, currentStep > 1)
			if sourcePath == "<back>" {
				continue
			}
			manifest.Pipeline.Load.SourcePath = sourcePath

			pterm.Println()
			currentStep++

		case 2:
			// Step 2: Pipeline configuration
			pterm.DefaultSection.Printfln("Step %d/%d: Pipeline Configuration", currentStep, totalSteps)
			pterm.Println()

			inspect := promptConfirmWithBack("Enable INSPECT stage (detect OS)?", true, true)
			if inspect == "<back>" {
				currentStep--
				continue
			}
			manifest.Pipeline.Inspect.Enabled = (inspect == "yes")
			if manifest.Pipeline.Inspect.Enabled {
				manifest.Pipeline.Inspect.DetectOS = true
			}

			fix := promptConfirmWithBack("Enable FIX stage (fstab, grub, initramfs)?", true, true)
			if fix == "<back>" {
				continue
			}
			manifest.Pipeline.Fix.Fstab.Enabled = (fix == "yes")
			if manifest.Pipeline.Fix.Fstab.Enabled {
				manifest.Pipeline.Fix.Fstab.Mode = "stabilize-all"
				manifest.Pipeline.Fix.Grub.Enabled = true
				manifest.Pipeline.Fix.Initramfs.Enabled = true
				manifest.Pipeline.Fix.Initramfs.Regenerate = true
				manifest.Pipeline.Fix.Network.Enabled = true
				manifest.Pipeline.Fix.Network.FixLevel = "full"
			}

			pterm.Println()
			currentStep++

		case 3:
			// Step 3: Output configuration
			pterm.DefaultSection.Printfln("Step %d/%d: Output Configuration", currentStep, totalSteps)
			pterm.Println()

			outputFormat := promptSelectWithBack("Select output format:", []string{
				"qcow2", "raw", "vmdk", "vdi",
			}, true)

			if outputFormat == "<back>" {
				currentStep--
				continue
			}
			manifest.Pipeline.Convert.OutputFormat = outputFormat

			compress := promptConfirmWithBack("Enable compression?", true, true)
			if compress == "<back>" {
				continue
			}
			manifest.Pipeline.Convert.Compress = (compress == "yes")

			validate := promptConfirmWithBack("Enable VALIDATE stage?", true, true)
			if validate == "<back>" {
				continue
			}
			manifest.Pipeline.Validate.Enabled = (validate == "yes")
			manifest.Pipeline.Validate.BootTest = false

			pterm.Println()
			currentStep++

		case 4:
			// Step 4: Review and save
			pterm.DefaultSection.Printfln("Step %d/%d: Review Manifest", currentStep, totalSteps)
			pterm.Println()

			// Display manifest
			data, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				pterm.Error.Printfln("Failed to generate manifest: %v", err)
				os.Exit(1)
			}

			// Pretty print with syntax highlighting
			manifestStr := string(data)
			pterm.DefaultBox.WithTitle("Generated Manifest").Println(manifestStr)
			pterm.Println()

			// Options: Save, Go Back, Cancel
			action := promptSelectWithBack("What would you like to do?", []string{
				"Save manifest to file",
				"Cancel (exit without saving)",
			}, true)

			if action == "<back>" {
				currentStep--
				continue
			}

			if action == "Cancel (exit without saving)" {
				pterm.Info.Println("Manifest creation cancelled")
				return
			}

			// Save to file
			filename := promptInputWithBack("Enter filename:", "my-vm-manifest.json", true)
			if filename == "<back>" {
				continue
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				pterm.Error.Printfln("Failed to save manifest: %v", err)
				os.Exit(1)
			}

			pterm.Success.Printfln("✅ Manifest saved to: %s", filename)
			pterm.Println()

			if promptConfirm("Submit manifest to workflow daemon?", false) {
				handleManifestSubmit("", []string{filename})
			}
			return

		default:
			return
		}
	}
}

// handleManifestValidate validates a manifest file
func handleManifestValidate(args []string) {
	if len(args) == 0 {
		pterm.Error.Println("Manifest file required")
		pterm.Info.Println("Usage: hyperctl manifest validate -file manifest.json")
		os.Exit(1)
	}

	manifestFile := args[0]
	if strings.HasPrefix(manifestFile, "-file=") {
		manifestFile = strings.TrimPrefix(manifestFile, "-file=")
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Validating manifest: %s", manifestFile))

	// Read manifest
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to read manifest: %v", err))
		os.Exit(1)
	}

	// Parse manifest
	var manifest Manifest
	ext := filepath.Ext(manifestFile)
	if ext == ".json" {
		err = json.Unmarshal(data, &manifest)
	} else if ext == ".yaml" || ext == ".yml" {
		err = yaml.Unmarshal(data, &manifest)
	} else {
		spinner.Fail("Unsupported file format (use .json or .yaml)")
		os.Exit(1)
	}

	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse manifest: %v", err))
		os.Exit(1)
	}

	// Validate manifest
	errors := validateManifest(&manifest)

	if len(errors) > 0 {
		spinner.Fail("Manifest validation failed")
		pterm.Println()
		pterm.Error.Println("Validation Errors:")
		for _, err := range errors {
			pterm.Println(fmt.Sprintf("  ❌ %s", err))
		}
		os.Exit(1)
	}

	spinner.Success("Manifest is valid!")
	pterm.Println()

	// Display summary
	displayManifestSummary(&manifest)
}

// validateManifest validates a manifest
func validateManifest(manifest *Manifest) []string {
	var errors []string

	// Check version
	if manifest.Version == "" {
		errors = append(errors, "Version is required")
	}

	// Check pipeline or batch
	if !manifest.Batch {
		// Single VM manifest
		if manifest.Pipeline.Load.SourceType == "" {
			errors = append(errors, "pipeline.load.source_type is required")
		}
		if manifest.Pipeline.Load.SourcePath == "" {
			errors = append(errors, "pipeline.load.source_path is required")
		}
		if manifest.Pipeline.Convert.OutputFormat == "" {
			errors = append(errors, "pipeline.convert.output_format is required")
		}
	} else {
		// Batch manifest
		if len(manifest.VMs) == 0 {
			errors = append(errors, "batch manifest requires at least one VM")
		}
		for i, vm := range manifest.VMs {
			if vm.Name == "" {
				errors = append(errors, fmt.Sprintf("vms[%d].name is required", i))
			}
			if vm.Pipeline.Load.SourcePath == "" {
				errors = append(errors, fmt.Sprintf("vms[%d].pipeline.load.source_path is required", i))
			}
		}
	}

	return errors
}

// displayManifestSummary displays a summary of the manifest
func displayManifestSummary(manifest *Manifest) {
	pterm.DefaultSection.Println("📋 Manifest Summary")
	pterm.Println()

	if manifest.Batch {
		pterm.Info.Printfln("Type: Batch (%d VMs)", len(manifest.VMs))
		pterm.Println()

		for i, vm := range manifest.VMs {
			pterm.Println(fmt.Sprintf("  %d. %s", i+1, vm.Name))
			pterm.Println(fmt.Sprintf("     Source: %s (%s)",
				vm.Pipeline.Load.SourcePath,
				vm.Pipeline.Load.SourceType))
			pterm.Println(fmt.Sprintf("     Output: %s",
				vm.Pipeline.Convert.OutputFormat))
		}
	} else {
		pterm.Info.Println("Type: Single VM")
		pterm.Println()

		data := [][]string{
			{"Property", "Value"},
			{"Source Type", manifest.Pipeline.Load.SourceType},
			{"Source Path", manifest.Pipeline.Load.SourcePath},
			{"Output Format", manifest.Pipeline.Convert.OutputFormat},
			{"Compression", fmt.Sprintf("%v", manifest.Pipeline.Convert.Compress)},
			{"INSPECT", fmt.Sprintf("%v", manifest.Pipeline.Inspect.Enabled)},
			{"FIX", fmt.Sprintf("%v", manifest.Pipeline.Fix.Fstab.Enabled)},
			{"VALIDATE", fmt.Sprintf("%v", manifest.Pipeline.Validate.Enabled)},
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed().
			WithData(data).
			Render()
	}
}

// handleManifestSubmit submits a manifest to the workflow daemon
func handleManifestSubmit(daemonURL string, args []string) {
	if len(args) == 0 {
		pterm.Error.Println("Manifest file required")
		pterm.Info.Println("Usage: hyperctl manifest submit -file manifest.json")
		os.Exit(1)
	}

	manifestFile := args[0]
	if strings.HasPrefix(manifestFile, "-file=") {
		manifestFile = strings.TrimPrefix(manifestFile, "-file=")
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Submitting manifest: %s", manifestFile))

	// Read manifest
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to read manifest: %v", err))
		os.Exit(1)
	}

	// Validate first
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse manifest: %v", err))
		os.Exit(1)
	}

	errors := validateManifest(&manifest)
	if len(errors) > 0 {
		spinner.Fail("Manifest validation failed")
		pterm.Println()
		for _, err := range errors {
			pterm.Error.Println("  " + err)
		}
		os.Exit(1)
	}

	// Copy to workflow directory
	workflowDir := "/var/lib/hyper2kvm/manifest-workflow/to_be_processed"

	// Create destination filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	baseName := strings.TrimSuffix(filepath.Base(manifestFile), filepath.Ext(manifestFile))
	destFile := filepath.Join(workflowDir, fmt.Sprintf("%s-%s.json", baseName, timestamp))

	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to create workflow directory: %v", err))
		os.Exit(1)
	}

	if err := os.WriteFile(destFile, data, 0644); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to submit manifest: %v", err))
		os.Exit(1)
	}

	spinner.Success("Manifest submitted successfully!")
	pterm.Println()

	pterm.Success.Printfln("✅ Job ID: %s", filepath.Base(destFile))
	pterm.Info.Printfln("📂 Location: %s", destFile)
	pterm.Println()

	pterm.Info.Println("💡 Monitor progress:")
	pterm.Println("  hyperctl workflow -op status")
	pterm.Println("  hyperctl workflow -op watch")
}

// handleManifestGenerate generates manifest from VM path
func handleManifestGenerate(args []string) {
	if len(args) < 2 {
		pterm.Error.Println("VM path and output directory required")
		pterm.Info.Println("Usage: hyperctl manifest generate <vm-path> <output-dir>")
		os.Exit(1)
	}

	vmPath := args[0]
	outputDir := args[1]

	spinner, _ := pterm.DefaultSpinner.Start("Generating manifest...")

	// Create manifest
	manifest := Manifest{
		Version: "1.0",
	}

	// Detect source type from VM path
	manifest.Pipeline.Load.SourcePath = vmPath
	if strings.HasSuffix(vmPath, ".vmdk") {
		manifest.Pipeline.Load.SourceType = "vmdk"
	} else if strings.HasSuffix(vmPath, ".ova") {
		manifest.Pipeline.Load.SourceType = "ova"
	} else if strings.HasSuffix(vmPath, ".vhd") || strings.HasSuffix(vmPath, ".vhdx") {
		manifest.Pipeline.Load.SourceType = "vhd"
	} else {
		manifest.Pipeline.Load.SourceType = "vmdk"
	}

	// Default pipeline configuration
	manifest.Pipeline.Inspect.Enabled = true
	manifest.Pipeline.Inspect.DetectOS = true

	manifest.Pipeline.Fix.Fstab.Enabled = true
	manifest.Pipeline.Fix.Fstab.Mode = "stabilize-all"
	manifest.Pipeline.Fix.Grub.Enabled = true
	manifest.Pipeline.Fix.Initramfs.Enabled = true
	manifest.Pipeline.Fix.Initramfs.Regenerate = true
	manifest.Pipeline.Fix.Network.Enabled = true
	manifest.Pipeline.Fix.Network.FixLevel = "full"

	manifest.Pipeline.Convert.OutputFormat = "qcow2"
	manifest.Pipeline.Convert.Compress = true

	manifest.Pipeline.Validate.Enabled = true
	manifest.Pipeline.Validate.BootTest = false

	// Generate filename
	vmName := strings.TrimSuffix(filepath.Base(vmPath), filepath.Ext(vmPath))
	manifestFile := filepath.Join(outputDir, vmName+"-manifest.json")

	// Save manifest
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestFile, data, 0644); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to save manifest: %v", err))
		os.Exit(1)
	}

	spinner.Success("Manifest generated!")
	pterm.Println()

	pterm.Success.Printfln("✅ Manifest saved to: %s", manifestFile)
	pterm.Println()

	displayManifestSummary(&manifest)
}

// promptSelect prompts user to select from options
func promptSelect(prompt string, options []string) string {
	pterm.Info.Println(prompt)
	for i, opt := range options {
		pterm.Println(fmt.Sprintf("  %d. %s", i+1, opt))
	}

	var choice int
	fmt.Print("Enter choice (1-" + fmt.Sprintf("%d", len(options)) + "): ")
	fmt.Scanf("%d", &choice)

	if choice < 1 || choice > len(options) {
		return options[0]
	}
	return options[choice-1]
}

// promptInput prompts user for input
func promptInput(prompt, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)

	var input string
	fmt.Scanln(&input)

	if input == "" {
		return defaultValue
	}
	return input
}

// promptConfirm prompts user for confirmation
func promptConfirm(prompt string, defaultValue bool) bool {
	defaultStr := "y/N"
	if defaultValue {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	var input string
	fmt.Scanln(&input)

	if input == "" {
		return defaultValue
	}

	input = strings.ToLower(input)
	return input == "y" || input == "yes"
}

// promptSelectWithBack prompts user to select from options with optional back button
func promptSelectWithBack(prompt string, options []string, allowBack bool) string {
	pterm.Info.Println(prompt)

	displayOptions := make([]string, len(options))
	copy(displayOptions, options)

	if allowBack {
		displayOptions = append(displayOptions, "← Go Back")
	}

	for i, opt := range displayOptions {
		pterm.Println(fmt.Sprintf("  %d. %s", i+1, opt))
	}

	var choice int
	fmt.Print("Enter choice (1-" + fmt.Sprintf("%d", len(displayOptions)) + "): ")
	fmt.Scanf("%d", &choice)

	if choice < 1 || choice > len(displayOptions) {
		return options[0]
	}

	selected := displayOptions[choice-1]
	if selected == "← Go Back" {
		return "<back>"
	}

	return selected
}

// promptInputWithBack prompts user for input with optional back button
func promptInputWithBack(prompt, defaultValue string, allowBack bool) string {
	backHint := ""
	if allowBack {
		backHint = " (type 'back' to go back)"
	}

	fmt.Printf("%s [%s]%s: ", prompt, defaultValue, backHint)

	var input string
	fmt.Scanln(&input)

	if allowBack && strings.ToLower(input) == "back" {
		return "<back>"
	}

	if input == "" {
		return defaultValue
	}
	return input
}

// promptConfirmWithBack prompts user for confirmation with optional back button
func promptConfirmWithBack(prompt string, defaultValue bool, allowBack bool) string {
	defaultStr := "y/N"
	if defaultValue {
		defaultStr = "Y/n"
	}

	backHint := ""
	if allowBack {
		backHint = " (type 'back' to go back)"
	}

	fmt.Printf("%s [%s]%s: ", prompt, defaultStr, backHint)

	var input string
	fmt.Scanln(&input)

	if allowBack && strings.ToLower(input) == "back" {
		return "<back>"
	}

	if input == "" {
		if defaultValue {
			return "yes"
		}
		return "no"
	}

	input = strings.ToLower(input)
	if input == "y" || input == "yes" {
		return "yes"
	}
	return "no"
}
