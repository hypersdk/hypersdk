// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"

	"hypersdk/config"
	"hypersdk/logger"
	"hypersdk/providers/vsphere"
)

const (
	// maxFilenameLength is the maximum allowed filename length on most filesystems
	maxFilenameLength = 255

	// jobPollInterval is the interval for polling job status
	jobPollInterval = 2 * time.Second

	// maxJobPollTime is the maximum time to poll for job completion
	maxJobPollTime = 30 * time.Minute
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffff")).
			Background(lipgloss.Color("#0066cc")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ff00"))

	unselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d97706")). // Deep burnt orange/amber
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ff0000"))

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ff00"))
)

type vmItem struct {
	vm       vsphere.VMInfo
	selected bool
}

type model struct {
	vms            []vmItem
	filteredVMs    []vmItem // Filtered/searched VMs
	cursor         int
	height         int
	width          int
	daemonURL      string
	outputDir      string
	autoConvert    bool
	autoImport     bool
	phase          string // "select", "confirm", "run-mode", "export", "convert", "done", "detail", "filter", "sort", "regex", "template", "quick-filter"
	currentExport  int
	message        string
	err            error
	confirmConvert bool
	confirmImport  bool
	runMode        string // "terminal" or "service"
	searchQuery    string // Current search query
	sortMode       string // "name", "cpu", "memory", "storage", "power"
	filterPower    string // "", "on", "off" - filter by power state
	filterOS       string // Filter by OS type
	showDetail     bool   // Show detailed view of current VM
	dryRun         bool   // Preview mode without executing
	showHelp       bool   // Show help panel

	// Export progress tracking
	exportProgress     exportProgress
	currentVMName      string
	currentFileName    string

	// Regex/pattern selection
	regexPattern       string

	// Export template
	selectedTemplate   *exportTemplate

	// Quick filter
	quickFilter        string
}

type exportProgress struct {
	currentBytes    int64
	totalBytes      int64
	currentFileIdx  int
	totalFiles      int
	speed           float64 // MB/s
	startTime       time.Time
	lastUpdateTime  time.Time
	lastBytes       int64
}

type exportTemplate struct {
	name        string
	description string
	format      string // "ovf" or "ova"
	compress    bool
	verify      bool
}

type vmsLoadedMsg struct {
	vms []vsphere.VMInfo
	err error
}

type exportDoneMsg struct {
	index int
	err   error
}

func initialModel(daemonURL, outputDir string, autoConvert, autoImport bool) model {
	return model{
		daemonURL:   daemonURL,
		outputDir:   outputDir,
		autoConvert: autoConvert,
		autoImport:  autoImport,
		phase:       "select",
		sortMode:    "name", // Default sort by name
		filterPower: "",     // No power filter by default
		filterOS:    "",     // No OS filter by default
		searchQuery: "",     // No search by default
		dryRun:      false,  // Execute by default
	}
}

func (m model) Init() tea.Cmd {
	return loadVMs(m.daemonURL)
}

func loadVMs(daemonURL string) tea.Cmd {
	return func() tea.Msg {
		// Check if environment variables are set for direct connection
		if os.Getenv("GOVC_URL") != "" {
			return loadVMsFromEnvironment()
		}

		// Fall back to daemon API
		return loadVMsFromDaemon(daemonURL)
	}
}

func loadVMsFromEnvironment() tea.Msg {
	// Load config from environment
	cfg := config.FromEnvironment()

	// Validate required fields
	if cfg.VCenterURL == "" || cfg.Username == "" || cfg.Password == "" {
		return vmsLoadedMsg{err: fmt.Errorf("missing required environment variables: GOVC_URL, GOVC_USERNAME, GOVC_PASSWORD")}
	}

	// Create logger (quiet for TUI)
	log := logger.New("error")

	// Create vSphere client
	ctx := context.Background()
	client, err := vsphere.NewVSphereClient(ctx, cfg, log)
	if err != nil {
		return vmsLoadedMsg{err: fmt.Errorf("connect to vCenter: %w", err)}
	}
	defer client.Close()

	// List VMs
	vms, err := client.ListVMs(ctx)
	if err != nil {
		return vmsLoadedMsg{err: fmt.Errorf("list VMs: %w", err)}
	}

	return vmsLoadedMsg{vms: vms}
}

func loadVMsFromDaemon(daemonURL string) tea.Msg {
	resp, err := apiRequestWithTimeout(daemonURL+"/vms/list", "GET", "", nil, 180*time.Second)
	if err != nil {
		return vmsLoadedMsg{err: err}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		VMs []vsphere.VMInfo `json:"vms"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return vmsLoadedMsg{err: err}
	}

	return vmsLoadedMsg{vms: result.VMs}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m, nil

	case vmsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = "error"
			return m, nil
		}

		if len(msg.vms) == 0 {
			m.err = fmt.Errorf("no VMs found - check daemon connection and vCenter credentials")
			m.phase = "error"
			return m, nil
		}

		m.vms = make([]vmItem, len(msg.vms))
		for i, vm := range msg.vms {
			m.vms[i] = vmItem{vm: vm, selected: false}
		}
		return m, nil

	case tea.KeyMsg:
		switch m.phase {
		case "error":
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		case "select":
			return m.handleSelectionKeys(msg)
		case "search":
			return m.handleSearchKeys(msg)
		case "detail":
			return m.handleDetailKeys(msg)
		case "confirm":
			return m.handleConfirmKeys(msg)
		case "run-mode":
			return m.handleRunModeKeys(msg)
		case "regex":
			return m.handleRegexKeys(msg)
		case "template":
			return m.handleTemplateKeys(msg)
		case "quick-filter":
			return m.handleQuickFilterKeys(msg)
		}

	case exportDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.currentExport++

		// Check if all exports done
		selectedCount := m.countSelected()

		if m.currentExport >= selectedCount {
			if m.autoConvert {
				m.phase = "convert"
				return m, m.convertAll()
			}
			m.phase = "done"
			return m, tea.Quit
		}

		return m, m.exportNext()
	}

	return m, nil
}

func (m model) handleSelectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vms := m.getVisibleVMs()

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(vms)-1 {
			m.cursor++
		}

	case " ":
		// Toggle selection for current visible VM
		if m.cursor < len(vms) {
			selectedVM := vms[m.cursor]
			// Find in original array and toggle using map for O(1) lookup
			targetPath := selectedVM.vm.Path
			for i := range m.vms {
				if m.vms[i].vm.Path == targetPath {
					m.vms[i].selected = !m.vms[i].selected
					break
				}
			}
		}

	case "a":
		// Select all (in current filtered view) - optimized with map
		visibleVMs := m.getVisibleVMs()
		visiblePaths := make(map[string]bool, len(visibleVMs))
		for _, vm := range visibleVMs {
			visiblePaths[vm.vm.Path] = true
		}
		for i := range m.vms {
			if visiblePaths[m.vms[i].vm.Path] {
				m.vms[i].selected = true
			}
		}

	case "n":
		// Deselect all (in current filtered view) - optimized with map
		visibleVMs := m.getVisibleVMs()
		visiblePaths := make(map[string]bool, len(visibleVMs))
		for _, vm := range visibleVMs {
			visiblePaths[vm.vm.Path] = true
		}
		for i := range m.vms {
			if visiblePaths[m.vms[i].vm.Path] {
				m.vms[i].selected = false
			}
		}

	case "/":
		// Enter search mode
		m.phase = "search"
		m.searchQuery = ""
		m.message = "Enter search query (name/path)..."
		return m, nil

	case "s":
		// Cycle sort mode
		m.cycleSortMode()
		m.applyFiltersAndSort()
		m.message = fmt.Sprintf("Sorted by: %s", m.sortMode)
		return m, nil

	case "f":
		// Toggle power filter
		m.togglePowerFilter()
		m.applyFiltersAndSort()
		return m, nil

	case "d", "i":
		// Show detail view of current VM
		if m.cursor < len(m.getVisibleVMs()) {
			m.showDetail = true
			m.phase = "detail"
		}
		return m, nil

	case "r":
		// Toggle dry-run mode
		m.dryRun = !m.dryRun
		if m.dryRun {
			m.message = "🔍 DRY-RUN mode enabled (preview only)"
		} else {
			m.message = "✅ DRY-RUN mode disabled (will execute)"
		}
		return m, nil

	case "c":
		// Clear all filters
		m.searchQuery = ""
		m.filterPower = ""
		m.filterOS = ""
		m.applyFiltersAndSort()
		m.message = "Filters cleared"
		return m, nil

	case "h", "?":
		// Toggle help panel
		m.showHelp = !m.showHelp
		if m.showHelp {
			m.message = "Help panel shown"
		} else {
			m.message = "Help panel hidden"
		}
		return m, nil

	case "A":
		// Bulk selection by regex pattern
		m.phase = "regex"
		m.regexPattern = ""
		m.message = "Enter regex pattern for bulk selection"
		return m, nil

	case "t", "T":
		// Select export template
		m.phase = "template"
		m.cursor = 0
		m.message = "Select export template"
		return m, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "0":
		// Quick filters
		return m.applyQuickFilter(msg.String()), nil

	case "enter":
		// Go to confirmation
		selectedCount := m.countSelected()

		if selectedCount == 0 {
			m.message = "No VMs selected!"
			return m, nil
		}

		m.phase = "confirm"
		m.message = ""
		return m, nil
	}

	return m, nil
}

func (m model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "b", "n", "N":
		// Go back to selection
		m.phase = "select"
		m.message = ""
		return m, nil

	case "y", "Y":
		// Go to run mode selection
		m.phase = "run-mode"
		return m, nil
	}

	return m, nil
}

func (m model) handleRunModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "b":
		// Go back to confirmation
		m.phase = "confirm"
		return m, nil

	case "1", "t", "T":
		// Run in terminal (interactive)
		m.runMode = "terminal"
		if m.dryRun {
			// Dry-run mode: just show preview and exit
			m.phase = "done"
			m.message = "🔍 DRY-RUN complete: No actual migration performed\nSelected VMs were previewed but not exported"
			return m, tea.Quit
		}
		m.phase = "export"
		m.currentExport = 0
		return m, m.exportNext()

	case "2", "s", "S":
		// Run as systemd service (background)
		if m.dryRun {
			m.message = "⚠️  Cannot create systemd service in dry-run mode"
			return m, nil
		}
		m.runMode = "service"
		return m, m.createSystemdService()
	}

	return m, nil
}

func (m model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "ctrl+[":
		// Exit search mode
		m.phase = "select"
		m.searchQuery = ""
		m.applyFiltersAndSort()
		m.message = "Search cancelled"
		return m, nil

	case "enter":
		// Apply search and return to selection
		m.phase = "select"
		m.applyFiltersAndSort()
		if len(m.filteredVMs) == 0 {
			m.message = fmt.Sprintf("No VMs match query: %s", m.searchQuery)
		} else {
			m.message = fmt.Sprintf("Found %d VMs matching: %s", len(m.filteredVMs), m.searchQuery)
		}
		return m, nil

	case "backspace", "ctrl+h":
		// Remove last character
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		return m, nil

	default:
		// Add character to search query (only printable characters)
		if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] <= 126 {
			m.searchQuery += msg.String()
		}
		return m, nil
	}
}

func (m model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "b", "backspace":
		// Exit detail view
		m.phase = "select"
		m.showDetail = false
		m.message = ""
		return m, nil

	case "enter", "space":
		// Select/deselect current VM and return to list
		vms := m.getVisibleVMs()
		if m.cursor < len(vms) {
			vms[m.cursor].selected = !vms[m.cursor].selected
		}
		m.phase = "select"
		m.showDetail = false
		return m, nil
	}

	return m, nil
}

func (m model) handleRegexKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "ctrl+[":
		// Exit regex mode
		m.phase = "select"
		m.regexPattern = ""
		m.message = "Regex selection cancelled"
		return m, nil

	case "enter":
		// Apply regex selection
		if m.regexPattern == "" {
			m.message = "Pattern cannot be empty"
			return m, nil
		}

		matches := 0
		for i := range m.vms {
			if matchVMPattern(m.vms[i].vm.Name, m.regexPattern) {
				m.vms[i].selected = true
				matches++
			}
		}

		m.phase = "select"
		m.message = fmt.Sprintf("✓ Selected %d VMs matching pattern: %s", matches, m.regexPattern)
		m.regexPattern = ""
		return m, nil

	case "backspace", "delete":
		// Delete last character
		if len(m.regexPattern) > 0 {
			m.regexPattern = m.regexPattern[:len(m.regexPattern)-1]
		}

	default:
		// Add character to pattern
		if len(msg.String()) == 1 {
			m.regexPattern += msg.String()
		}
	}

	return m, nil
}

func (m model) handleTemplateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "b":
		// Exit template selection
		m.phase = "select"
		m.message = "Template selection cancelled"
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(exportTemplates)-1 {
			m.cursor++
		}

	case "enter", "1", "2", "3", "4":
		// Select template
		templateIdx := m.cursor
		if msg.String() >= "1" && msg.String() <= "4" {
			templateIdx = int(msg.String()[0] - '1')
		}

		if templateIdx >= 0 && templateIdx < len(exportTemplates) {
			m.selectedTemplate = &exportTemplates[templateIdx]
			m.phase = "select"
			m.message = fmt.Sprintf("✓ Template selected: %s", exportTemplates[templateIdx].name)
		}
		return m, nil
	}

	return m, nil
}

func (m model) handleQuickFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "b":
		// Exit quick filter mode
		m.phase = "select"
		m.message = "Quick filter cancelled"
		return m, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "0":
		m = m.applyQuickFilter(msg.String())
		m.phase = "select"
		return m, nil
	}

	return m, nil
}

func (m model) applyQuickFilter(key string) model {
	// Clear current filters first
	m.searchQuery = ""
	m.filterPower = ""
	m.filterOS = ""
	m.quickFilter = key

	switch key {
	case "1":
		// Powered ON VMs
		m.filterPower = "on"
		m.message = "Filter: Powered ON VMs"

	case "2":
		// Powered OFF VMs
		m.filterPower = "off"
		m.message = "Filter: Powered OFF VMs"

	case "3":
		// Linux VMs
		m.filterOS = "linux"
		m.message = "Filter: Linux VMs"

	case "4":
		// Windows VMs
		m.filterOS = "windows"
		m.message = "Filter: Windows VMs"

	case "5":
		// High CPU (8+ cores)
		m.quickFilter = "highcpu"
		m.message = "Filter: High CPU VMs (8+ cores)"

	case "6":
		// High Memory (16GB+)
		m.quickFilter = "highmem"
		m.message = "Filter: High Memory VMs (16GB+)"

	case "7":
		// Large Storage (500GB+)
		m.quickFilter = "largestorage"
		m.message = "Filter: Large Storage VMs (500GB+)"

	case "8":
		// Reserved for future use
		m.message = "Filter: All VMs"

	case "0":
		// Clear all filters
		m.quickFilter = ""
		m.message = "All filters cleared"
	}

	m.applyFiltersAndSort()
	return m
}

func (m model) exportNext() tea.Cmd {
	// Find next selected VM
	exportIndex := 0
	for i, item := range m.vms {
		if item.selected {
			if exportIndex == m.currentExport {
				return m.exportVM(i)
			}
			exportIndex++
		}
	}
	return nil
}

func (m model) exportVM(index int) tea.Cmd {
	return func() tea.Msg {
		vm := m.vms[index].vm

		// Create output directory
		vmOutputDir := filepath.Join(m.outputDir, sanitizeFilename(vm.Name))
		os.MkdirAll(vmOutputDir, 0755)

		// Submit export job
		jobRequest := struct {
			Name       string                 `json:"name"`
			VMPath     string                 `json:"vm_path"`
			OutputPath string                 `json:"output_path"`
			Options    map[string]interface{} `json:"options"`
		}{
			Name:       "export-" + vm.Name,
			VMPath:     vm.Path,
			OutputPath: vmOutputDir,
			Options: map[string]interface{}{
				"parallel_downloads":       8,
				"remove_cdrom":             true,
				"show_individual_progress": false,
			},
		}

		reqBody, _ := json.Marshal(jobRequest)
		resp, err := apiRequest(m.daemonURL+"/jobs/submit", "POST", "application/json", reqBody)
		if err != nil {
			return exportDoneMsg{index: index, err: err}
		}
		defer resp.Body.Close()

		// Parse response to get job ID
		var submitResp struct {
			JobIDs []string `json:"job_ids"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
			return exportDoneMsg{index: index, err: fmt.Errorf("failed to parse submit response: %w", err)}
		}
		if len(submitResp.JobIDs) == 0 {
			return exportDoneMsg{index: index, err: fmt.Errorf("no job ID returned")}
		}
		jobID := submitResp.JobIDs[0]

		// Poll job status until completion
		startTime := time.Now()
		ticker := time.NewTicker(jobPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Check if we've exceeded max poll time
				if time.Since(startTime) > maxJobPollTime {
					return exportDoneMsg{index: index, err: fmt.Errorf("job polling timeout after %v", maxJobPollTime)}
				}

				// Query job status
				statusResp, err := apiRequest(m.daemonURL+"/jobs/status/"+jobID, "GET", "", nil)
				if err != nil {
					return exportDoneMsg{index: index, err: fmt.Errorf("failed to check job status: %w", err)}
				}

				var job struct {
					Status string `json:"status"`
					Error  string `json:"error,omitempty"`
				}
				if err := json.NewDecoder(statusResp.Body).Decode(&job); err != nil {
					statusResp.Body.Close()
					return exportDoneMsg{index: index, err: fmt.Errorf("failed to parse job status: %w", err)}
				}
				statusResp.Body.Close()

				// Check job status
				switch job.Status {
				case "completed":
					return exportDoneMsg{index: index, err: nil}
				case "failed":
					errMsg := job.Error
					if errMsg == "" {
						errMsg = "unknown error"
					}
					return exportDoneMsg{index: index, err: fmt.Errorf("job failed: %s", errMsg)}
				case "cancelled":
					return exportDoneMsg{index: index, err: fmt.Errorf("job was cancelled")}
				case "pending", "running":
					// Continue polling
					continue
				default:
					return exportDoneMsg{index: index, err: fmt.Errorf("unknown job status: %s", job.Status)}
				}
			}
		}
	}
}

func (m model) convertAll() tea.Cmd {
	return func() tea.Msg {
		for i, item := range m.vms {
			if item.selected {
				vmOutputDir := filepath.Join(m.outputDir, sanitizeFilename(item.vm.Name))

				// Find OVF file
				ovfFiles, err := filepath.Glob(filepath.Join(vmOutputDir, "*.ovf"))
				if err != nil || len(ovfFiles) == 0 {
					// No OVF file, skip
					continue
				}

				ovfFile := ovfFiles[0]

				// Call hyper2kvm to do the migration
				// hyper2kvm -input <ovf-file> -output <output-dir>
				cmd := exec.Command("hyper2kvm",
					"-input", ovfFile,
					"-output", vmOutputDir,
					"-format", "qcow2",
				)

				// Set environment for hyper2kvm if needed
				cmd.Env = os.Environ()

				// Run hyper2kvm
				output, err := cmd.CombinedOutput()
				if err != nil {
					return exportDoneMsg{
						index: i,
						err:   fmt.Errorf("hyper2kvm failed for %s: %v\nOutput: %s", item.vm.Name, err, string(output)),
					}
				}

				// If auto-import is enabled, also import to libvirt
				if m.autoImport {
					// Find the qcow2 file created by hyper2kvm
					qcow2Files, err := filepath.Glob(filepath.Join(vmOutputDir, "*.qcow2"))
					if err == nil && len(qcow2Files) > 0 {
						qcow2File := qcow2Files[0]

						// Import to libvirt using virt-install
						importCmd := exec.Command("virt-install",
							"--name", item.vm.Name,
							"--import",
							"--disk", qcow2File+",bus=virtio",
							"--memory", fmt.Sprintf("%d", item.vm.MemoryMB),
							"--vcpus", fmt.Sprintf("%d", item.vm.NumCPU),
							"--os-variant", "generic",
							"--network", "bridge=virbr0",
							"--graphics", "vnc",
							"--noautoconsole",
						)

						if err := importCmd.Run(); err != nil {
							// Don't fail the whole process if import fails
							// Just log it
							fmt.Fprintf(os.Stderr, "Warning: Failed to import %s to libvirt: %v\n", item.vm.Name, err)
						}
					}
				}
			}
		}
		return exportDoneMsg{index: -1, err: nil}
	}
}

func (m model) View() string {
	switch m.phase {
	case "error":
		return m.renderError()
	case "select":
		return m.renderSelection()
	case "search":
		return m.renderSearch()
	case "detail":
		return m.renderDetail()
	case "confirm":
		return m.renderConfirm()
	case "run-mode":
		return m.renderRunMode()
	case "export":
		return m.renderExport()
	case "convert":
		return m.renderConvert()
	case "done":
		return m.renderDone()
	case "regex":
		return renderRegexSelection(m)
	case "template":
		return renderExportTemplates(m.cursor)
	case "quick-filter":
		return renderQuickFilterMenu(m)
	}

	// Show loading message based on connection mode
	if os.Getenv("GOVC_URL") != "" {
		return "Loading VMs from vCenter (direct connection)...\n\n" +
			infoStyle.Render(fmt.Sprintf("vCenter: %s", os.Getenv("GOVC_URL")))
	}
	return "Loading VMs from daemon...\n\n" +
		infoStyle.Render(fmt.Sprintf("Daemon: %s", m.daemonURL))
}

func (m model) renderSelection() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Interactive VM Migration Tool"))
	b.WriteString("\n\n")

	if len(m.vms) == 0 {
		b.WriteString(infoStyle.Render("Loading VMs..."))
		return b.String()
	}

	// Get visible VMs (filtered/sorted)
	vms := m.getVisibleVMs()

	// Enhanced status bar
	b.WriteString(renderStatusBar(m))
	b.WriteString("\n\n")

	// Statistics panel
	b.WriteString(renderStatsPanel(m))
	b.WriteString("\n\n")

	// VM list (show window of items from visible VMs)
	start := m.cursor - 10
	if start < 0 {
		start = 0
	}
	end := start + 20
	if end > len(vms) {
		end = len(vms)
	}

	for i := start; i < end; i++ {
		item := vms[i]

		// Cursor indicator
		cursor := "  "
		if m.cursor == i {
			cursor = "▶ "
		}

		// Selection checkbox
		checkbox := "[ ]"
		if item.selected {
			checkbox = "[✓]"
		}

		// VM info
		vmInfo := fmt.Sprintf("%-40s | %s | %d CPU | %.1f GB | %s",
			truncate(item.vm.Name, 40),
			colorPowerState(item.vm.PowerState),
			item.vm.NumCPU,
			float64(item.vm.MemoryMB)/1024,
			formatBytes(item.vm.Storage))

		// Style
		style := unselectedStyle
		if item.selected {
			style = selectedStyle
		}

		line := cursor + checkbox + " " + vmInfo
		if m.cursor == i {
			line = style.Bold(true).Render(line)
		} else {
			line = style.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Help - show full panel or brief hint
	b.WriteString("\n")
	if m.showHelp {
		b.WriteString(renderHelpPanel())
		b.WriteString("\n")
	} else {
		b.WriteString(titleStyle.Render("🎯 Controls:"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Navigation: ↑/k: Up | ↓/j: Down | Space: Select/deselect | Enter: Continue →"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Selection:  a: Select all | n: Deselect all | A: Regex pattern | 1-8: Quick filters"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Search:     /: Search VMs | s: Cycle sort (" + m.sortMode + ") | f: Filter power | c: Clear filters"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("View:       d/i: Detail view | h/?: Toggle help | t: Templates | r: Toggle dry-run"))
		if m.dryRun {
			b.WriteString(" ")
			b.WriteString(infoStyle.Render("[DRY-RUN]"))
		}
		b.WriteString(" | ")
		b.WriteString(errorStyle.Render("q: Quit"))
	}

	if m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(infoStyle.Render(m.message))
	}

	return b.String()
}

func (m model) renderSearch() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🔍 Search VMs"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(infoStyle.Render("Type to search by name, path, or OS:"))
	b.WriteString("\n\n")

	// Search box
	searchBox := fmt.Sprintf("Search: %s█", m.searchQuery)
	b.WriteString(selectedStyle.Render(searchBox))
	b.WriteString("\n\n")

	// Preview results
	if m.searchQuery != "" {
		// Apply filter temporarily to show preview
		tempModel := m
		tempModel.applyFiltersAndSort()

		b.WriteString(titleStyle.Render(fmt.Sprintf("Preview: %d matches", len(tempModel.filteredVMs))))
		b.WriteString("\n\n")

		// Show first few matches
		maxPreview := 10
		if len(tempModel.filteredVMs) < maxPreview {
			maxPreview = len(tempModel.filteredVMs)
		}

		for i := 0; i < maxPreview; i++ {
			item := tempModel.filteredVMs[i]
			vmInfo := fmt.Sprintf("  • %-40s | %s | %d CPU | %.1f GB",
				truncate(item.vm.Name, 40),
				colorPowerState(item.vm.PowerState),
				item.vm.NumCPU,
				float64(item.vm.MemoryMB)/1024)

			b.WriteString(unselectedStyle.Render(vmInfo))
			b.WriteString("\n")
		}

		if len(tempModel.filteredVMs) > maxPreview {
			b.WriteString("\n")
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ... and %d more", len(tempModel.filteredVMs)-maxPreview)))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(helpStyle.Render("Start typing to search..."))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter: Apply search | Esc: Cancel | Backspace: Delete character | q: Quit"))

	return b.String()
}

func (m model) renderDetail() string {
	var b strings.Builder

	vms := m.getVisibleVMs()
	if m.cursor >= len(vms) {
		m.phase = "select"
		return m.renderSelection()
	}

	item := vms[m.cursor]

	// Title
	b.WriteString(titleStyle.Render("📊 VM Details"))
	b.WriteString("\n\n")

	// Enhanced VM card
	b.WriteString(renderVMCard(item))
	b.WriteString("\n\n")

	// Help
	b.WriteString(helpStyle.Render("Space/Enter: Toggle selection | Esc/b: Back to list | q: Quit"))

	return b.String()
}

func (m model) renderConfirm() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("📋 Confirm Migration"))
	b.WriteString("\n\n")

	// Show selected VMs with detailed info
	b.WriteString(infoStyle.Render("Selected VMs for migration:"))
	b.WriteString("\n\n")

	var totalCPUs int32
	var totalMemoryMB int32
	var totalStorage int64

	for _, item := range m.vms {
		if item.selected {
			totalCPUs += item.vm.NumCPU
			totalMemoryMB += item.vm.MemoryMB
			totalStorage += item.vm.Storage

			// VM details box
			vmDetails := fmt.Sprintf(
				"📦 %s\n"+
					"   Path: %s\n"+
					"   Power: %s | CPU: %d | Memory: %.1f GB | Storage: %s\n"+
					"   OS: %s",
				item.vm.Name,
				item.vm.Path,
				item.vm.PowerState,
				item.vm.NumCPU,
				float64(item.vm.MemoryMB)/1024,
				formatBytes(item.vm.Storage),
				item.vm.GuestOS,
			)

			b.WriteString(selectedStyle.Render(vmDetails))
			b.WriteString("\n\n")
		}
	}

	// Summary
	b.WriteString(titleStyle.Render("📊 Migration Summary"))
	b.WriteString("\n\n")

	summary := fmt.Sprintf(
		"Total Resources:\n"+
			"  🖥️  Total CPUs: %d\n"+
			"  💾 Total Memory: %.1f GB\n"+
			"  💿 Total Storage: %s\n",
		totalCPUs,
		float64(totalMemoryMB)/1024,
		formatBytes(totalStorage),
	)
	b.WriteString(infoStyle.Render(summary))
	b.WriteString("\n\n")

	// Export settings
	b.WriteString(titleStyle.Render("⚙️  Export Settings"))
	b.WriteString("\n\n")

	settings := fmt.Sprintf(
		"Output Directory: %s\n"+
			"Auto-convert to qcow2: %s (using hyper2kvm)\n"+
			"Auto-import to libvirt: %s\n",
		m.outputDir,
		boolToYesNo(m.autoConvert),
		boolToYesNo(m.autoImport),
	)
	b.WriteString(infoStyle.Render(settings))
	b.WriteString("\n\n")

	if m.autoConvert {
		b.WriteString(helpStyle.Render("Note: hyper2kvm will convert OVF+VMDK to qcow2 format"))
		b.WriteString("\n\n")
	}

	// Export preview with disk space validation
	preview := renderExportPreview(m)
	if preview != "" {
		b.WriteString(preview)
		b.WriteString("\n\n")
	}

	// Confirmation prompt
	if m.dryRun {
		b.WriteString(infoStyle.Bold(true).Render("🔍 DRY-RUN MODE: Preview Only (No Actual Migration)"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("In dry-run mode, you'll see what would be migrated without performing actual migration."))
	} else {
		b.WriteString(successStyle.Bold(true).Render("⚡ Start Migration for Selected VMs?"))
		b.WriteString("\n\n")
		b.WriteString(infoStyle.Render("This will export, convert, and optionally import the selected VMs."))
	}
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("y/Y: Yes, proceed | n/N: No, go back | Esc/b: Back to VM selection | q: Quit"))

	return b.String()
}

func (m model) renderRunMode() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🔧 Choose Execution Mode"))
	b.WriteString("\n\n")

	// Dry-run indicator
	if m.dryRun {
		b.WriteString(infoStyle.Bold(true).Render("🔍 DRY-RUN MODE ACTIVE"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("(Preview only - no actual migration will be performed)"))
		b.WriteString("\n\n")
	}

	// Explanation
	b.WriteString(infoStyle.Render("How would you like to run the migration?"))
	b.WriteString("\n\n")

	// Option 1: Terminal
	b.WriteString(titleStyle.Render("1. Run in Terminal (Interactive)"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render(
		"  ✓ Watch progress in real-time\n" +
			"  ✓ See immediate feedback\n" +
			"  ✓ Requires keeping terminal open\n" +
			"  ⚠  Terminal must stay active during migration",
	))
	b.WriteString("\n\n")

	// Option 2: Service
	b.WriteString(titleStyle.Render("2. Run as Systemd Service (Background)"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render(
		"  ✓ Runs in background\n" +
			"  ✓ Can close terminal and come back later\n" +
			"  ✓ Survives SSH disconnections\n" +
			"  ✓ Check status with: journalctl -u vm-migration@<job-id>\n" +
			"  ℹ  Perfect for long migrations or remote work",
	))
	b.WriteString("\n\n")

	// Selected count reminder
	selectedCount := m.countSelected()
	b.WriteString(helpStyle.Render(fmt.Sprintf("Migrating %d VMs", selectedCount)))
	b.WriteString("\n\n")

	// Prompt
	b.WriteString(successStyle.Bold(true).Render("Choose Execution Mode:"))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("1/t: Run in Terminal | 2/s: Run as Systemd Service"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Esc/b: Back to confirmation | q: Quit"))

	return b.String()
}

func (m model) renderExport() string {
	// Use enhanced real-time export progress display
	return renderRealTimeExportProgress(m)
}

func (m model) renderConvert() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔄 Converting VMs with hyper2kvm"))
	b.WriteString("\n\n")

	b.WriteString(infoStyle.Render("Running hyper2kvm for each exported VM..."))
	b.WriteString("\n\n")

	for _, item := range m.vms {
		if item.selected {
			b.WriteString(fmt.Sprintf("⟳ %s - Converting OVF to qcow2...\n", item.vm.Name))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("This may take several minutes depending on VM size"))

	return b.String()
}

func (m model) renderError() string {
	var b strings.Builder

	b.WriteString(errorStyle.Render("❌ Error"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(m.err.Error())
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("Troubleshooting:"))
	b.WriteString("\n\n")

	// Check connection mode and provide relevant troubleshooting
	if os.Getenv("GOVC_URL") != "" {
		b.WriteString(infoStyle.Render("  Connection Mode: Direct (using environment variables)"))
		b.WriteString("\n\n")
		b.WriteString("  • Verify environment variables are set:\n")
		b.WriteString(fmt.Sprintf("    GOVC_URL=%s\n", os.Getenv("GOVC_URL")))
		b.WriteString(fmt.Sprintf("    GOVC_USERNAME=%s\n", os.Getenv("GOVC_USERNAME")))
		b.WriteString("    GOVC_PASSWORD=*** (set: ")
		if os.Getenv("GOVC_PASSWORD") != "" {
			b.WriteString("yes)\n")
		} else {
			b.WriteString("no)\n")
		}
		b.WriteString(fmt.Sprintf("    GOVC_INSECURE=%s\n\n", os.Getenv("GOVC_INSECURE")))
		b.WriteString("  • Test vCenter connection manually:\n")
		b.WriteString("    govc about\n\n")
		b.WriteString("  • Check vCenter is accessible:\n")
		b.WriteString(fmt.Sprintf("    ping %s\n\n", os.Getenv("GOVC_URL")))
	} else {
		b.WriteString(infoStyle.Render("  Connection Mode: Daemon API"))
		b.WriteString("\n\n")
		b.WriteString("  • Check that hyper2kvmd daemon is running:\n")
		b.WriteString("    sudo systemctl status hyper2kvmd\n\n")
		b.WriteString("  • Check daemon logs:\n")
		b.WriteString("    sudo journalctl -u hyper2kvmd -f\n\n")
		b.WriteString("  • Verify daemon is accessible:\n")
		b.WriteString(fmt.Sprintf("    curl %s/vms/list\n\n", m.daemonURL))
		b.WriteString("  • Check vCenter credentials in /etc/hyper2kvm/config.yaml\n\n")
		b.WriteString("  • OR use direct connection by setting environment:\n")
		b.WriteString("    export GOVC_URL='https://vcenter/sdk'\n")
		b.WriteString("    export GOVC_USERNAME='user@vsphere.local'\n")
		b.WriteString("    export GOVC_PASSWORD='password'\n")
		b.WriteString("    export GOVC_INSECURE=1\n\n")
	}

	b.WriteString(helpStyle.Render("Press 'q' to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) renderDone() string {
	var b strings.Builder

	if m.runMode == "service" {
		// Show service information
		b.WriteString(successStyle.Render("✅ Migration Service Started!"))
		b.WriteString("\n\n")

		if m.message != "" {
			b.WriteString(infoStyle.Render(m.message))
			b.WriteString("\n\n")
		}

		b.WriteString(helpStyle.Render("The migration is running in the background.\nYou can safely close this terminal and check back later."))
		b.WriteString("\n")

		return b.String()
	}

	// Terminal mode - show completion
	b.WriteString(successStyle.Render("✅ Migration Complete!"))
	b.WriteString("\n\n")

	for _, item := range m.vms {
		if item.selected {
			vmOutputDir := filepath.Join(m.outputDir, sanitizeFilename(item.vm.Name))
			b.WriteString(fmt.Sprintf("📁 %s → %s\n", item.vm.Name, vmOutputDir))
		}
	}

	if m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(infoStyle.Render(m.message))
	}

	return b.String()
}

func colorPowerState(state string) string {
	if state == "poweredOn" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render("ON ")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("OFF")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func sanitizeFilename(name string) string {
	// Prevent path traversal and invalid filename characters
	// Replace directory separators
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")

	// Replace path traversal attempts
	name = strings.ReplaceAll(name, "..", "-")

	// Replace other invalid characters
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "\x00", "") // Remove null bytes
	name = strings.ReplaceAll(name, ":", "-")   // Colon (problematic on Windows)
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "\"", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")

	// Trim leading/trailing dots and dashes (reserved names like "." and "..")
	name = strings.Trim(name, ".-")

	// Ensure name is not empty after sanitization
	if name == "" {
		name = "unnamed-vm"
	}

	// Limit length to prevent filesystem issues
	if len(name) > maxFilenameLength {
		name = name[:maxFilenameLength]
	}

	return name
}

func boolToYesNo(b bool) string {
	if b {
		return successStyle.Render("Yes ✓")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("No")
}

// getVisibleVMs returns the current filtered/sorted view of VMs
func (m model) getVisibleVMs() []vmItem {
	if len(m.filteredVMs) > 0 {
		return m.filteredVMs
	}
	return m.vms
}

// countSelected returns the number of selected VMs across all VMs (not just visible)
func (m model) countSelected() int {
	count := 0
	for _, item := range m.vms {
		if item.selected {
			count++
		}
	}
	return count
}

// cycleSortMode cycles through sort modes
func (m *model) cycleSortMode() {
	switch m.sortMode {
	case "name":
		m.sortMode = "cpu"
	case "cpu":
		m.sortMode = "memory"
	case "memory":
		m.sortMode = "storage"
	case "storage":
		m.sortMode = "power"
	case "power":
		m.sortMode = "name"
	default:
		m.sortMode = "name"
	}
}

// togglePowerFilter cycles through power filter options
func (m *model) togglePowerFilter() {
	switch m.filterPower {
	case "":
		m.filterPower = "on"
		m.message = "Filter: Show powered ON VMs only"
	case "on":
		m.filterPower = "off"
		m.message = "Filter: Show powered OFF VMs only"
	case "off":
		m.filterPower = ""
		m.message = "Filter: Show all VMs"
	default:
		m.filterPower = ""
	}
}

// applyFiltersAndSort applies current filters and sorting to the VM list
func (m *model) applyFiltersAndSort() {
	// Start with all VMs
	filtered := make([]vmItem, 0, len(m.vms))

	for _, item := range m.vms {
		// Apply search filter
		if m.searchQuery != "" {
			query := strings.ToLower(m.searchQuery)
			nameMatch := strings.Contains(strings.ToLower(item.vm.Name), query)
			pathMatch := strings.Contains(strings.ToLower(item.vm.Path), query)
			osMatch := strings.Contains(strings.ToLower(item.vm.GuestOS), query)

			if !nameMatch && !pathMatch && !osMatch {
				continue
			}
		}

		// Apply power filter
		if m.filterPower == "on" && item.vm.PowerState != "poweredOn" {
			continue
		}
		if m.filterPower == "off" && item.vm.PowerState == "poweredOn" {
			continue
		}

		// Apply OS filter
		if m.filterOS != "" {
			if !strings.Contains(strings.ToLower(item.vm.GuestOS), strings.ToLower(m.filterOS)) {
				continue
			}
		}

		// Apply quick filters
		if m.quickFilter != "" {
			switch m.quickFilter {
			case "highcpu":
				if item.vm.NumCPU < 8 {
					continue
				}
			case "highmem":
				if item.vm.MemoryMB < 16*1024 {
					continue
				}
			case "largestorage":
				if item.vm.Storage < 500*1024*1024*1024 {
					continue
				}
			}
		}

		filtered = append(filtered, item)
	}

	// Apply sorting - using sort.Slice for O(n log n) performance
	switch m.sortMode {
	case "cpu":
		// Sort by CPU count (descending)
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].vm.NumCPU > filtered[j].vm.NumCPU
		})
	case "memory":
		// Sort by memory (descending)
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].vm.MemoryMB > filtered[j].vm.MemoryMB
		})
	case "storage":
		// Sort by storage (descending)
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].vm.Storage > filtered[j].vm.Storage
		})
	case "power":
		// Sort by power state (ON first)
		sort.Slice(filtered, func(i, j int) bool {
			// poweredOn < other states (so ON comes first)
			if filtered[i].vm.PowerState == "poweredOn" && filtered[j].vm.PowerState != "poweredOn" {
				return true
			}
			if filtered[i].vm.PowerState != "poweredOn" && filtered[j].vm.PowerState == "poweredOn" {
				return false
			}
			// If both same power state, sort by name
			return strings.ToLower(filtered[i].vm.Name) < strings.ToLower(filtered[j].vm.Name)
		})
	case "name":
		// Sort by name (ascending)
		sort.Slice(filtered, func(i, j int) bool {
			return strings.ToLower(filtered[i].vm.Name) < strings.ToLower(filtered[j].vm.Name)
		})
	}

	m.filteredVMs = filtered

	// Adjust cursor bounds to stay within filtered list
	if len(filtered) > 0 {
		if m.cursor >= len(filtered) {
			m.cursor = len(filtered) - 1
		} else if m.cursor < 0 {
			m.cursor = 0
		}
	} else {
		m.cursor = 0
	}
}

func (m model) createSystemdService() tea.Cmd {
	return func() tea.Msg {
		// Generate unique job ID
		jobID := fmt.Sprintf("vm-migration-%d", os.Getpid())

		// Create a script file that will be executed by systemd
		scriptPath := fmt.Sprintf("/tmp/%s.sh", jobID)
		scriptContent := "#!/bin/bash\n\n"
		scriptContent += "# VM Migration Script\n"
		scriptContent += "# Generated by hyperctl\n\n"

		// Add export and conversion commands for each selected VM
		for _, item := range m.vms {
			if item.selected {
				vmName := sanitizeFilename(item.vm.Name)
				vmOutputDir := filepath.Join(m.outputDir, vmName)

				// Create output directory
				scriptContent += fmt.Sprintf("mkdir -p '%s'\n\n", vmOutputDir)

				// Export VM
				scriptContent += fmt.Sprintf("echo 'Exporting %s...'\n", item.vm.Name)
				scriptContent += "# Submit export job via hyperctl\n"
				scriptContent += fmt.Sprintf("hyperctl submit -vm '%s' -output '%s'\n\n", item.vm.Path, vmOutputDir)

				// Wait for export to complete (poll job status)
				scriptContent += "# Wait for export to complete\n"
				scriptContent += "sleep 10\n\n"

				if m.autoConvert {
					// Convert with hyper2kvm
					scriptContent += fmt.Sprintf("echo 'Converting %s with hyper2kvm...'\n", item.vm.Name)
					scriptContent += fmt.Sprintf("OVF_FILE=$(ls '%s'/*.ovf | head -1)\n", vmOutputDir)
					scriptContent += "if [ -f \"$OVF_FILE\" ]; then\n"
					scriptContent += fmt.Sprintf("  hyper2kvm -input \"$OVF_FILE\" -output '%s' -format qcow2\n", vmOutputDir)
					scriptContent += "fi\n\n"
				}

				if m.autoImport {
					// Import to libvirt
					scriptContent += fmt.Sprintf("echo 'Importing %s to libvirt...'\n", item.vm.Name)
					scriptContent += fmt.Sprintf("QCOW2_FILE=$(ls '%s'/*.qcow2 | head -1)\n", vmOutputDir)
					scriptContent += "if [ -f \"$QCOW2_FILE\" ]; then\n"
					scriptContent += fmt.Sprintf("  virt-install --name '%s' --import --disk \"$QCOW2_FILE\",bus=virtio \\\n", item.vm.Name)
					scriptContent += fmt.Sprintf("    --memory %d --vcpus %d --os-variant generic \\\n", item.vm.MemoryMB, item.vm.NumCPU)
					scriptContent += "    --network bridge=virbr0 --graphics vnc --noautoconsole\n"
					scriptContent += "fi\n\n"
				}
			}
		}

		scriptContent += "echo 'Migration complete!'\n"

		// Write script file
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			return exportDoneMsg{index: -1, err: fmt.Errorf("failed to create script: %v", err)}
		}

		// Create systemd transient service
		cmd := exec.Command("systemd-run",
			"--user",
			"--unit", jobID,
			"--description", fmt.Sprintf("VM Migration: %d VMs", len(m.vms)),
			"--working-directory", m.outputDir,
			scriptPath,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Try without --user flag (requires sudo)
			cmd = exec.Command("sudo", "systemd-run",
				"--unit", jobID,
				"--description", fmt.Sprintf("VM Migration: %d VMs", len(m.vms)),
				"--working-directory", m.outputDir,
				scriptPath,
			)

			output, err = cmd.CombinedOutput()
			if err != nil {
				return exportDoneMsg{index: -1, err: fmt.Errorf("failed to create systemd service: %v\nOutput: %s", err, string(output))}
			}
		}

		// Service created successfully
		m.message = fmt.Sprintf("✅ Migration started as systemd service: %s\n\nCheck status with:\n  journalctl -u %s -f\n\nScript: %s", jobID, jobID, scriptPath)
		m.phase = "done"

		return exportDoneMsg{index: -1, err: nil}
	}
}

func runInteractive(daemonURL, outputDir string, autoConvert, autoImport bool) {
	// Show banner first
	showBanner()
	pterm.Println()
	pterm.Info.Println("Entering interactive mode...")
	pterm.Println()

	// Check if we have a TTY
	if !isatty() {
		pterm.Error.Println("❌ Interactive mode requires a terminal (TTY)")
		pterm.Info.Println("")
		pterm.Info.Println("Please run this command inside tmux or a real terminal:")
		pterm.Info.Println("")
		pterm.Info.Println("  Method 1: Start tmux first")
		pterm.Info.Println("    tmux")
		pterm.Info.Println("    hyperctl migrate")
		pterm.Info.Println("")
		pterm.Info.Println("  Method 2: Use the helper script")
		pterm.Info.Println("    /tmp/start-migration-tui.sh")
		pterm.Info.Println("")
		os.Exit(1)
	}

	// Run bubbletea program
	p := tea.NewProgram(
		initialModel(daemonURL, outputDir, autoConvert, autoImport),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		pterm.Error.Printfln("Error running interactive mode: %v", err)
		os.Exit(1)
	}

	// Check if there was an error in the final model
	if m, ok := finalModel.(model); ok && m.err != nil {
		pterm.Println()
		pterm.Error.Printfln("Migration error: %v", m.err)
		os.Exit(1)
	}

	pterm.Println()
	pterm.Success.Println("✅ Interactive migration complete!")
}

func isatty() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
