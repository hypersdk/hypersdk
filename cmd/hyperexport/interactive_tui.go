// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hypersdk/logger"
	"hypersdk/providers/vsphere"
)

// exportTemplate defines an export configuration preset
type exportTemplate struct {
	name        string
	description string
	format      string // "ovf" or "ova"
	compress    bool
	verify      bool
}

// Export templates (same as hyperctl)
var exportTemplates = []exportTemplate{
	{
		name:        "Quick Export",
		description: "Fast export without compression (OVF format)",
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
		description: "OVF format for fast development cycles",
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

// TUI model for interactive hyperexport
type tuiModel struct {
	vms            []tuiVMItem
	filteredVMs    []tuiVMItem
	cursor         int
	phase          string // "select", "confirm", "template", "regex", "cloud", "export", "cloudupload", "done"
	searchQuery    string
	sortMode       string // "name", "cpu", "memory", "storage", "power"
	filterPower    string // "", "on", "off"
	filterOS       string
	quickFilter    string
	showHelp       bool
	regexPattern   string
	selectedTemplate *exportTemplate
	message        string
	err            error

	// Export state
	currentExport  int
	exportProgress exportProgressState
	currentVMName  string
	currentFileName string

	// Cloud configuration
	cloudConfig    *cloudConfig
	enableCloudUpload bool

	// Configuration
	client    *vsphere.VSphereClient
	outputDir string
	log       logger.Logger
	ctx       context.Context
}

type tuiVMItem struct {
	vm       vsphere.VMInfo
	selected bool
}

type exportProgressState struct {
	currentBytes   int64
	totalBytes     int64
	currentFileIdx int
	totalFiles     int
	speed          float64 // MB/s
	startTime      time.Time
	lastUpdateTime time.Time
	lastBytes      int64
}

// Color palette and styles (same as hyperctl)
var (
	primaryColor   = lipgloss.Color("#00ffff")
	secondaryColor = lipgloss.Color("#ff00ff")
	successColor   = lipgloss.Color("#00ff00")
	warningColor   = lipgloss.Color("#ffaa00")
	errorColor     = lipgloss.Color("#ff0000")
	mutedColor     = lipgloss.Color("#666666")
	highlightColor = lipgloss.Color("#ffff00")

	titleStyleTUI = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(lipgloss.Color("#0066cc")).
			Padding(0, 1)

	selectedStyleTUI = lipgloss.NewStyle().
				Bold(true).
				Foreground(successColor)

	unselectedStyleTUI = lipgloss.NewStyle().
				Foreground(mutedColor)

	infoStyleTUI = lipgloss.NewStyle().
			Foreground(warningColor).
			Italic(true)

	helpStyleTUI = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	errorStyleTUI = lipgloss.NewStyle().
			Bold(true).
			Foreground(errorColor)

	successStyleTUI = lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)

	statsStyleTUI = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	panelStyleTUI = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Margin(1, 0)

	badgeSuccessStyleTUI = lipgloss.NewStyle().
				Background(successColor).
				Foreground(lipgloss.Color("#000000")).
				Padding(0, 1).
				Bold(true)

	badgeWarningStyleTUI = lipgloss.NewStyle().
				Background(warningColor).
				Foreground(lipgloss.Color("#000000")).
				Padding(0, 1).
				Bold(true)

	progressBarStyleTUI = lipgloss.NewStyle().
				Foreground(successColor).
				Background(mutedColor)

	progressLabelStyleTUI = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	keyStyleTUI = lipgloss.NewStyle().
			Foreground(highlightColor).
			Bold(true)

	keyDescStyleTUI = lipgloss.NewStyle().
			Foreground(mutedColor)
)

type vmsLoadedMsg struct {
	vms []vsphere.VMInfo
	err error
}

type exportDoneMsg struct {
	vmName string
	err    error
}

func (m tuiModel) Init() tea.Cmd {
	return m.loadVMs
}

func (m tuiModel) loadVMs() tea.Msg {
	vms, err := m.client.ListVMs(m.ctx)
	if err != nil {
		return vmsLoadedMsg{err: err}
	}
	return vmsLoadedMsg{vms: vms}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case vmsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = "error"
			return m, tea.Quit
		}

		m.vms = make([]tuiVMItem, len(msg.vms))
		for i, vm := range msg.vms {
			m.vms[i] = tuiVMItem{vm: vm, selected: false}
		}
		m.phase = "select"
		m.applyFiltersAndSort()
		return m, nil

	case tea.KeyMsg:
		switch m.phase {
		case "select":
			return m.handleSelectionKeys(msg)
		case "confirm":
			return m.handleConfirmKeys(msg)
		case "regex":
			return m.handleRegexKeys(msg)
		case "template":
			return m.handleTemplateKeys(msg)
		case "cloud":
			// Cloud phase is handled by cloudSelectionModel
			return m, nil
		}

	case cloudConfigCompleteMsg:
		m.cloudConfig = msg.config
		m.enableCloudUpload = true
		m.phase = "confirm"
		return m, nil

	case exportDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.currentExport++
		selectedCount := m.countSelected()
		if m.currentExport >= selectedCount {
			m.phase = "done"
			return m, tea.Quit
		}
		return m, m.exportNext()
	}

	return m, nil
}

func (m tuiModel) handleSelectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if m.cursor < len(vms) {
			selectedVM := vms[m.cursor]
			for i := range m.vms {
				if m.vms[i].vm.Path == selectedVM.vm.Path {
					m.vms[i].selected = !m.vms[i].selected
					break
				}
			}
		}

	case "a":
		// Select all visible
		visiblePaths := make(map[string]bool)
		for _, vm := range m.getVisibleVMs() {
			visiblePaths[vm.vm.Path] = true
		}
		for i := range m.vms {
			if visiblePaths[m.vms[i].vm.Path] {
				m.vms[i].selected = true
			}
		}

	case "n":
		// Deselect all visible
		visiblePaths := make(map[string]bool)
		for _, vm := range m.getVisibleVMs() {
			visiblePaths[vm.vm.Path] = true
		}
		for i := range m.vms {
			if visiblePaths[m.vms[i].vm.Path] {
				m.vms[i].selected = false
			}
		}

	case "A":
		// Regex selection
		m.phase = "regex"
		m.regexPattern = ""
		return m, nil

	case "t", "T":
		// Template selection
		m.phase = "template"
		m.cursor = 0
		return m, nil

	case "1", "2", "3", "4", "5", "6", "7", "0":
		m.applyQuickFilter(msg.String())
		return m, nil

	case "/":
		m.searchQuery = ""
		m.message = "Type to search..."
		return m, nil

	case "s":
		m.cycleSortMode()
		m.applyFiltersAndSort()
		return m, nil

	case "c":
		m.searchQuery = ""
		m.filterPower = ""
		m.filterOS = ""
		m.quickFilter = ""
		m.applyFiltersAndSort()
		m.message = "Filters cleared"
		return m, nil

	case "h", "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "enter":
		if m.countSelected() == 0 {
			m.message = "No VMs selected!"
			return m, nil
		}
		m.phase = "confirm"
		return m, nil

	case "u", "U":
		// Cloud upload configuration
		if m.countSelected() == 0 {
			m.message = "No VMs selected!"
			return m, nil
		}
		m.phase = "cloud"
		cloudModel := newCloudSelectionModel(&m)
		return cloudModel, nil
	}

	return m, nil
}

func (m tuiModel) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "escape", "n":
		m.phase = "select"
		return m, nil
	case "y", "Y", "enter":
		m.phase = "export"
		m.currentExport = 0
		return m, m.exportNext()
	case "u", "U":
		// Configure cloud upload from confirm screen
		m.phase = "cloud"
		cloudModel := newCloudSelectionModel(&m)
		return cloudModel, nil
	}
	return m, nil
}

func (m tuiModel) handleRegexKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "escape":
		m.phase = "select"
		m.regexPattern = ""
		return m, nil
	case "enter":
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
		m.message = fmt.Sprintf("✓ Selected %d VMs matching: %s", matches, m.regexPattern)
		m.regexPattern = ""
		return m, nil
	case "backspace", "delete":
		if len(m.regexPattern) > 0 {
			m.regexPattern = m.regexPattern[:len(m.regexPattern)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.regexPattern += msg.String()
		}
	}
	return m, nil
}

func (m tuiModel) handleTemplateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "escape":
		m.phase = "select"
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
		idx := m.cursor
		if msg.String() >= "1" && msg.String() <= "4" {
			idx = int(msg.String()[0] - '1')
		}
		if idx >= 0 && idx < len(exportTemplates) {
			m.selectedTemplate = &exportTemplates[idx]
			m.phase = "select"
			m.message = fmt.Sprintf("✓ Template: %s", exportTemplates[idx].name)
		}
		return m, nil
	}
	return m, nil
}

func (m tuiModel) View() string {
	switch m.phase {
	case "select":
		return m.renderSelection()
	case "confirm":
		return m.renderConfirm()
	case "regex":
		return m.renderRegex()
	case "template":
		return m.renderTemplate()
	case "export":
		return m.renderExport()
	case "cloudupload":
		return m.renderCloudUpload()
	case "done":
		return m.renderDone()
	case "error":
		return errorStyleTUI.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
	}
	return "Loading VMs..."
}

func (m tuiModel) renderSelection() string {
	var b strings.Builder

	b.WriteString(titleStyleTUI.Render("HyperExport - Interactive VM Export"))
	b.WriteString("\n\n")

	if len(m.vms) == 0 {
		b.WriteString(infoStyleTUI.Render("Loading VMs..."))
		return b.String()
	}

	// Status bar
	selectedCount := m.countSelected()
	totalCount := len(m.vms)
	visibleCount := len(m.getVisibleVMs())

	statusText := fmt.Sprintf("📊 Total: %d | Visible: %d | ✅ Selected: %d", totalCount, visibleCount, selectedCount)
	if m.searchQuery != "" {
		statusText += fmt.Sprintf(" | 🔍 %s", m.searchQuery)
	}
	if m.filterPower != "" {
		statusText += fmt.Sprintf(" | ⚡ %s", m.filterPower)
	}
	if m.quickFilter != "" {
		statusText += fmt.Sprintf(" | 🚀 %s", m.quickFilter)
	}
	b.WriteString(infoStyleTUI.Render(statusText))
	b.WriteString("\n\n")

	// VM list
	vms := m.getVisibleVMs()
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
		cursor := "  "
		if m.cursor == i {
			cursor = "▶ "
		}

		checkbox := "[ ]"
		if item.selected {
			checkbox = "[✓]"
		}

		powerIcon := "🔴"
		if item.vm.PowerState == "poweredOn" {
			powerIcon = "🟢"
		}

		vmInfo := fmt.Sprintf("%-35s %s %2dC %4.0fG %8s",
			truncateString(item.vm.Name, 35),
			powerIcon,
			item.vm.NumCPU,
			float64(item.vm.MemoryMB)/1024,
			formatBytesCompact(item.vm.Storage))

		style := unselectedStyleTUI
		if item.selected {
			style = selectedStyleTUI
		}

		line := cursor + checkbox + " " + vmInfo
		if m.cursor == i {
			line = style.Bold(true).Underline(true).Render(line)
		} else {
			line = style.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	if m.showHelp {
		b.WriteString(m.renderHelp())
	} else {
		b.WriteString(titleStyleTUI.Render("🎯 Controls:"))
		b.WriteString("\n")
		b.WriteString(helpStyleTUI.Render("Navigation: ↑/k: Up | ↓/j: Down | Space: Select | Enter: Continue"))
		b.WriteString("\n")
		b.WriteString(helpStyleTUI.Render("Selection:  a: All | n: None | A: Regex | 1-7: Quick filters"))
		b.WriteString("\n")
		b.WriteString(helpStyleTUI.Render("Actions:    u: Cloud Upload | t: Templates | s: Sort | c: Clear"))
		b.WriteString("\n")
		b.WriteString(helpStyleTUI.Render("Other:      h/?: Help | q: Quit"))
	}

	if m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(infoStyleTUI.Render(m.message))
	}

	return b.String()
}

func (m tuiModel) renderConfirm() string {
	var b strings.Builder

	b.WriteString(titleStyleTUI.Render("📋 Confirm Export"))
	b.WriteString("\n\n")

	selectedVMs := []tuiVMItem{}
	for _, item := range m.vms {
		if item.selected {
			selectedVMs = append(selectedVMs, item)
		}
	}

	var totalStorage int64
	var totalCPUs int32
	var totalMemory int32

	for _, item := range selectedVMs {
		totalStorage += item.vm.Storage
		totalCPUs += item.vm.NumCPU
		totalMemory += item.vm.MemoryMB

		vmDetails := fmt.Sprintf("📦 %s | %d CPU | %.1f GB | %s",
			item.vm.Name,
			item.vm.NumCPU,
			float64(item.vm.MemoryMB)/1024,
			formatBytesCompact(item.vm.Storage))
		b.WriteString(selectedStyleTUI.Render(vmDetails))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(titleStyleTUI.Render("📊 Summary"))
	b.WriteString("\n")
	summary := fmt.Sprintf("VMs: %d | CPUs: %d | Memory: %.1f GB | Storage: %s",
		len(selectedVMs),
		totalCPUs,
		float64(totalMemory)/1024,
		formatBytesCompact(totalStorage))
	b.WriteString(infoStyleTUI.Render(summary))
	b.WriteString("\n\n")

	// Cloud upload info
	if m.enableCloudUpload && m.cloudConfig != nil {
		b.WriteString(titleStyleTUI.Render("☁️  Cloud Upload"))
		b.WriteString("\n")
		cloudInfo := fmt.Sprintf("Provider: %s | Bucket: %s",
			m.cloudConfig.provider,
			m.cloudConfig.bucket)
		if m.cloudConfig.prefix != "" {
			cloudInfo += fmt.Sprintf(" | Prefix: %s", m.cloudConfig.prefix)
		}
		b.WriteString(successStyleTUI.Render("✓ " + cloudInfo))
		b.WriteString("\n\n")
	} else {
		b.WriteString(infoStyleTUI.Render("☁️  Cloud upload: Not configured (press 'u' to configure)"))
		b.WriteString("\n\n")
	}

	// Disk space check
	diskSpace := getDiskSpace(m.outputDir)
	if diskSpace > totalStorage {
		b.WriteString(successStyleTUI.Render(fmt.Sprintf("✓ Disk space OK: %s available", formatBytesCompact(diskSpace))))
	} else {
		b.WriteString(errorStyleTUI.Render(fmt.Sprintf("⚠ WARNING: Need %s, only %s available!", formatBytesCompact(totalStorage), formatBytesCompact(diskSpace))))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyleTUI.Render("y/Y/Enter: Start export | u: Cloud upload | n/Esc: Go back | q: Quit"))

	return b.String()
}

func (m tuiModel) renderRegex() string {
	var b strings.Builder

	b.WriteString(titleStyleTUI.Render("🎯 Bulk Selection by Pattern"))
	b.WriteString("\n\n")
	b.WriteString(infoStyleTUI.Render("Enter regex pattern to match VM names:"))
	b.WriteString("\n\n")

	b.WriteString(selectedStyleTUI.Render(fmt.Sprintf("Pattern: %s█", m.regexPattern)))
	b.WriteString("\n\n")

	if m.regexPattern != "" {
		matches := 0
		for _, item := range m.vms {
			if matchVMPattern(item.vm.Name, m.regexPattern) {
				matches++
			}
		}
		if matches > 0 {
			b.WriteString(successStyleTUI.Render(fmt.Sprintf("✓ Matches %d VMs", matches)))
		} else {
			b.WriteString(errorStyleTUI.Render("⚠ No matches"))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyleTUI.Render("Enter: Select | Esc: Cancel | Backspace: Delete"))

	return b.String()
}

func (m tuiModel) renderTemplate() string {
	var b strings.Builder

	b.WriteString(titleStyleTUI.Render("📋 Export Templates"))
	b.WriteString("\n\n")

	for i, tmpl := range exportTemplates {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		style := unselectedStyleTUI
		if i == m.cursor {
			style = selectedStyleTUI
		}

		header := fmt.Sprintf("%s[%d] %s", cursor, i+1, tmpl.name)
		b.WriteString(style.Bold(true).Render(header))
		b.WriteString("\n")
		b.WriteString(infoStyleTUI.Render(fmt.Sprintf("    %s", tmpl.description)))
		b.WriteString("\n")
		settings := fmt.Sprintf("    Format: %s | Compress: %v | Verify: %v",
			tmpl.format, tmpl.compress, tmpl.verify)
		b.WriteString(helpStyleTUI.Render(settings))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyleTUI.Render("↑/↓: Navigate | Enter/1-4: Select | Esc: Back"))

	return b.String()
}

func (m tuiModel) renderExport() string {
	var b strings.Builder

	selectedVMs := []tuiVMItem{}
	for _, item := range m.vms {
		if item.selected {
			selectedVMs = append(selectedVMs, item)
		}
	}

	b.WriteString(titleStyleTUI.Render("📦 Exporting VMs"))
	b.WriteString("\n\n")

	// Overall progress
	overallProgress := fmt.Sprintf("%d / %d VMs completed", m.currentExport, len(selectedVMs))
	b.WriteString(progressLabelStyleTUI.Render(overallProgress))
	b.WriteString("\n\n")

	// VM list with status
	for i, item := range selectedVMs {
		status := ""
		icon := ""

		if i < m.currentExport {
			status = "Completed"
			icon = "✅"
		} else if i == m.currentExport {
			status = "Exporting..."
			icon = "⏳"
		} else {
			status = "Pending"
			icon = "⏸ "
		}

		vmLine := fmt.Sprintf("%s %s - %s", icon, truncateString(item.vm.Name, 40), status)
		b.WriteString(vmLine)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyleTUI.Render("Export in progress... Press q to cancel"))

	return b.String()
}

func (m tuiModel) renderDone() string {
	return successStyleTUI.Render("✅ Export complete!\n\nPress q to quit")
}

func (m tuiModel) renderHelp() string {
	help := `⌨️  Keyboard Shortcuts

Navigation:
  ↑/k       Move up
  ↓/j       Move down
  Space     Select/deselect
  Enter     Continue

Selection:
  a         Select all (visible)
  n         Deselect all
  A         Regex pattern
  1-7       Quick filters

Filters:
  1         Powered ON
  2         Powered OFF
  3         Linux VMs
  4         Windows VMs
  5         High CPU (8+)
  6         High Memory (16GB+)
  7         Large Storage (500GB+)

Actions:
  u         Cloud upload (S3/Azure/GCS/SFTP)
  t         Export templates
  s         Cycle sort
  c         Clear filters

Other:
  h/?       Toggle help
  q         Quit
`
	return panelStyleTUI.Render(help)
}

// Helper methods
func (m *tuiModel) getVisibleVMs() []tuiVMItem {
	if m.filteredVMs != nil {
		return m.filteredVMs
	}
	return m.vms
}

func (m tuiModel) countSelected() int {
	count := 0
	for _, item := range m.vms {
		if item.selected {
			count++
		}
	}
	return count
}

func (m *tuiModel) applyFiltersAndSort() {
	filtered := make([]tuiVMItem, 0, len(m.vms))

	for _, item := range m.vms {
		// Apply search
		if m.searchQuery != "" {
			query := strings.ToLower(m.searchQuery)
			if !strings.Contains(strings.ToLower(item.vm.Name), query) &&
				!strings.Contains(strings.ToLower(item.vm.Path), query) {
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

	// Sort
	switch m.sortMode {
	case "cpu":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].vm.NumCPU > filtered[j].vm.NumCPU
		})
	case "memory":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].vm.MemoryMB > filtered[j].vm.MemoryMB
		})
	case "storage":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].vm.Storage > filtered[j].vm.Storage
		})
	case "power":
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].vm.PowerState == "poweredOn" && filtered[j].vm.PowerState != "poweredOn" {
				return true
			}
			return false
		})
	default: // name
		sort.Slice(filtered, func(i, j int) bool {
			return strings.ToLower(filtered[i].vm.Name) < strings.ToLower(filtered[j].vm.Name)
		})
	}

	m.filteredVMs = filtered

	if len(filtered) > 0 && m.cursor >= len(filtered) {
		m.cursor = len(filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *tuiModel) cycleSortMode() {
	modes := []string{"name", "cpu", "memory", "storage", "power"}
	for i, mode := range modes {
		if m.sortMode == mode {
			m.sortMode = modes[(i+1)%len(modes)]
			return
		}
	}
	m.sortMode = "name"
}

func (m *tuiModel) applyQuickFilter(key string) {
	m.searchQuery = ""
	m.filterPower = ""
	m.filterOS = ""
	m.quickFilter = ""

	switch key {
	case "1":
		m.filterPower = "on"
		m.message = "Filter: Powered ON"
	case "2":
		m.filterPower = "off"
		m.message = "Filter: Powered OFF"
	case "3":
		m.filterOS = "linux"
		m.message = "Filter: Linux"
	case "4":
		m.filterOS = "windows"
		m.message = "Filter: Windows"
	case "5":
		m.quickFilter = "highcpu"
		m.message = "Filter: High CPU (8+)"
	case "6":
		m.quickFilter = "highmem"
		m.message = "Filter: High Memory (16GB+)"
	case "7":
		m.quickFilter = "largestorage"
		m.message = "Filter: Large Storage (500GB+)"
	case "0":
		m.message = "Filters cleared"
	}

	m.applyFiltersAndSort()
}

func (m tuiModel) exportNext() tea.Cmd {
	return func() tea.Msg {
		// Find next selected VM
		exportIndex := 0
		for _, item := range m.vms {
			if item.selected {
				if exportIndex == m.currentExport {
					// Export this VM
					// TODO: Implement actual export with parallel downloads
					time.Sleep(2 * time.Second) // Simulate export
					return exportDoneMsg{vmName: item.vm.Name, err: nil}
				}
				exportIndex++
			}
		}
		return exportDoneMsg{vmName: "", err: fmt.Errorf("no more VMs to export")}
	}
}

// Utility functions
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatBytesCompact(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < unit*unit {
		return fmt.Sprintf("%.0fK", float64(bytes)/unit)
	}
	if bytes < unit*unit*unit {
		return fmt.Sprintf("%.0fM", float64(bytes)/(unit*unit))
	}
	if bytes < unit*unit*unit*unit {
		return fmt.Sprintf("%.1fG", float64(bytes)/(unit*unit*unit))
	}
	return fmt.Sprintf("%.1fT", float64(bytes)/(unit*unit*unit*unit))
}

func getDiskSpace(path string) int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	return int64(stat.Bavail) * int64(stat.Bsize)
}

func matchVMPattern(vmName, pattern string) bool {
	vmLower := strings.ToLower(vmName)
	patternLower := strings.ToLower(pattern)

	if strings.HasPrefix(patternLower, "^") {
		patternLower = strings.TrimPrefix(patternLower, "^")
		return strings.HasPrefix(vmLower, patternLower)
	}

	if strings.HasSuffix(patternLower, "$") {
		patternLower = strings.TrimSuffix(patternLower, "$")
		return strings.HasSuffix(vmLower, patternLower)
	}

	return strings.Contains(vmLower, patternLower)
}
