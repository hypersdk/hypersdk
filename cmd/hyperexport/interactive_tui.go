// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	vms              []tuiVMItem
	filteredVMs      []tuiVMItem
	cursor           int
	phase            string // "select", "confirm", "template", "regex", "cloud", "features", "export", "cloudupload", "done", "search", "details"
	detailsVM        *vsphere.VMInfo // VM to show details for
	searchQuery      string
	sortMode         string // "name", "cpu", "memory", "storage", "power"
	filterPower      string // "", "on", "off"
	filterOS         string
	quickFilter      string
	showHelp         bool
	regexPattern     string
	selectedTemplate *exportTemplate
	message          string
	err              error

	// Export state
	currentExport   int
	exportProgress  exportProgressState
	currentVMName   string
	currentFileName string

	// Cloud configuration
	cloudConfig       *cloudConfig
	enableCloudUpload bool

	// Advanced Features Configuration
	featureConfig featureConfiguration

	// Modern UI components
	progressBar progress.Model
	helpModel   help.Model
	spinner     spinner.Model
	searchInput textinput.Model
	keys        tuiKeyMap

	// Configuration
	client    *vsphere.VSphereClient
	outputDir string
	log       logger.Logger
	ctx       context.Context
}

// featureConfiguration holds advanced export features
type featureConfiguration struct {
	// Snapshot settings
	enableSnapshot     bool
	snapshotMemory     bool
	snapshotQuiesce    bool
	deleteSnapshot     bool
	keepSnapshots      int
	consolidateSnaps   bool

	// Bandwidth settings
	enableBandwidthLimit bool
	bandwidthLimitMBps   int64
	adaptiveBandwidth    bool

	// Incremental export settings
	enableIncremental bool
	showIncrementalInfo bool

	// Email notifications
	enableEmail       bool
	emailSMTPHost     string
	emailSMTPPort     int
	emailFrom         string
	emailTo           string
	emailOnStart      bool
	emailOnComplete   bool
	emailOnFailure    bool

	// Cleanup settings
	enableCleanup   bool
	cleanupMaxAge   int // days
	cleanupMaxCount int
	cleanupDryRun   bool
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

// Modern key bindings
type tuiKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Confirm  key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Filter   key.Binding
	Sort     key.Binding
	Features key.Binding
	Cloud    key.Binding
}

func (k tuiKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k tuiKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Confirm},
		{k.Filter, k.Sort, k.Features, k.Cloud},
		{k.Back, k.Help, k.Quit},
	}
}

// Vision-optimized color palette (based on human eye sensitivity: green > yellow > red).
var (
	// Core palette - dark orange in the sweet spot (#D35400 - #C75B12).
	// Avoids eye strain (too bright #FFA500) and muddy unreadable (too brown #8B4513).
	deepOrange    = lipgloss.Color("#D35400") // Primary accent (optimal contrast, no glare)
	tealInfo      = lipgloss.Color("#5DADE2") // Directories/info
	successGreen  = lipgloss.Color("#A3BE8C") // Success messages (eye-sensitive green)
	warmRed       = lipgloss.Color("#E74C3C") // Errors (bright for visibility)
	amberYellow   = lipgloss.Color("#F39C12") // Highlights/warnings (readable yellow)
	offWhite      = lipgloss.Color("#F5F5DC") // Normal text (cream)
	darkCharcoal  = lipgloss.Color("#0B0C10") // Background (very dark)
	lightCharcoal = lipgloss.Color("#1C1E22") // Subtle backgrounds
	mutedGray     = lipgloss.Color("#6B7280") // Muted elements

	// Semantic color mappings.
	primaryColor   = deepOrange    // Orange as signal
	secondaryColor = tealInfo      // Teal for information
	successColor   = successGreen  // Green for success
	warningColor   = amberYellow   // Amber for warnings
	errorColor     = warmRed       // Red for errors
	mutedColor     = mutedGray     // Gray for muted text
	highlightColor = deepOrange    // Orange highlights
	textColor      = offWhite      // Default text
	darkBg         = darkCharcoal  // Primary background
	lightBg        = lightCharcoal // Secondary background

	// Enhanced title with warm orange accent.
	titleStyleTUI = lipgloss.NewStyle().
			Bold(true).
			Foreground(deepOrange).
			Background(darkBg).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tealInfo)

	// Warm selection style.
	selectedStyleTUI = lipgloss.NewStyle().
				Bold(true).
				Foreground(deepOrange).
				Background(lightBg)

	unselectedStyleTUI = lipgloss.NewStyle().
				Foreground(mutedColor)

	infoStyleTUI = lipgloss.NewStyle().
			Foreground(tealInfo).
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

type exportProgressMsg struct {
	vmName         string
	currentBytes   int64
	totalBytes     int64
	currentFileIdx int
	totalFiles     int
	fileName       string
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.loadVMs, m.spinner.Tick)
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
		case "features":
			return m.handleFeaturesKeys(msg)
		case "search":
			return m.handleSearchKeys(msg)
		case "details":
			return m.handleDetailsKeys(msg)
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

	case exportProgressMsg:
		// Update progress state
		m.currentVMName = msg.vmName
		m.currentFileName = msg.fileName
		m.exportProgress.currentBytes = msg.currentBytes
		m.exportProgress.totalBytes = msg.totalBytes
		m.exportProgress.currentFileIdx = msg.currentFileIdx
		m.exportProgress.totalFiles = msg.totalFiles

		// Calculate speed
		now := time.Now()
		if !m.exportProgress.lastUpdateTime.IsZero() {
			elapsed := now.Sub(m.exportProgress.lastUpdateTime).Seconds()
			if elapsed > 0 {
				bytesDiff := msg.currentBytes - m.exportProgress.lastBytes
				m.exportProgress.speed = float64(bytesDiff) / elapsed / (1024 * 1024) // MB/s
			}
		}
		m.exportProgress.lastUpdateTime = now
		m.exportProgress.lastBytes = msg.currentBytes

		return m, nil

	case tickMsg:
		var cmd tea.Cmd
		if m.phase == "export" {
			// Continue ticking during export
			cmd = tickCmd()
		}
		// Update spinner
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
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
		m.phase = "search"
		m.searchInput.Reset()
		m.searchInput.Focus()
		m.searchInput.Placeholder = "Type to search VMs..."
		return m, textinput.Blink

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

	case "f", "F":
		// Advanced features configuration
		m.phase = "features"
		m.cursor = 0
		return m, nil

	case "i", "I":
		// Show VM details
		vms := m.getVisibleVMs()
		if m.cursor < len(vms) {
			m.detailsVM = &vms[m.cursor].vm
			m.phase = "details"
		}
		return m, nil
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
		m.exportProgress.startTime = time.Now()
		return m, tea.Batch(m.exportNext(), tickCmd())
	case "u", "U":
		// Configure cloud upload from confirm screen
		m.phase = "cloud"
		cloudModel := newCloudSelectionModel(&m)
		return cloudModel, nil
	case "f", "F":
		// Configure advanced features from confirm screen
		m.phase = "features"
		m.cursor = 0
		return m, nil
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

func (m tuiModel) handleFeaturesKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxItems := 14 // Total number of feature options

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
		if m.cursor < maxItems-1 {
			m.cursor++
		}
	case " ", "enter":
		// Toggle the feature at cursor position
		switch m.cursor {
		case 0:
			m.featureConfig.enableSnapshot = !m.featureConfig.enableSnapshot
		case 1:
			m.featureConfig.snapshotMemory = !m.featureConfig.snapshotMemory
		case 2:
			m.featureConfig.snapshotQuiesce = !m.featureConfig.snapshotQuiesce
		case 3:
			m.featureConfig.deleteSnapshot = !m.featureConfig.deleteSnapshot
		case 4:
			m.featureConfig.consolidateSnaps = !m.featureConfig.consolidateSnaps
		case 5:
			m.featureConfig.enableBandwidthLimit = !m.featureConfig.enableBandwidthLimit
		case 6:
			m.featureConfig.adaptiveBandwidth = !m.featureConfig.adaptiveBandwidth
		case 7:
			m.featureConfig.enableIncremental = !m.featureConfig.enableIncremental
		case 8:
			m.featureConfig.showIncrementalInfo = !m.featureConfig.showIncrementalInfo
		case 9:
			m.featureConfig.enableEmail = !m.featureConfig.enableEmail
		case 10:
			m.featureConfig.emailOnStart = !m.featureConfig.emailOnStart
		case 11:
			m.featureConfig.emailOnComplete = !m.featureConfig.emailOnComplete
		case 12:
			m.featureConfig.enableCleanup = !m.featureConfig.enableCleanup
		case 13:
			m.featureConfig.cleanupDryRun = !m.featureConfig.cleanupDryRun
		}
	case "s", "S":
		// Save and return to previous phase
		m.phase = "select"
		count := m.countEnabledFeatures()
		m.message = fmt.Sprintf("✓ %d advanced features configured", count)
		return m, nil
	}
	return m, nil
}

func (m tuiModel) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEscape:
		// Cancel search and return to select
		m.phase = "select"
		m.searchQuery = ""
		m.searchInput.Blur()
		m.applyFiltersAndSort()
		m.message = "Search cancelled"
		return m, nil

	case tea.KeyEnter:
		// Apply search and return to select
		m.phase = "select"
		m.searchQuery = m.searchInput.Value()
		m.searchInput.Blur()
		m.applyFiltersAndSort()
		if m.searchQuery != "" {
			visibleCount := len(m.getVisibleVMs())
			m.message = fmt.Sprintf("🔍 Found %d VMs matching '%s'", visibleCount, m.searchQuery)
		}
		return m, nil

	default:
		// Update the search input and filter in real-time
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.searchQuery = m.searchInput.Value()
		m.applyFiltersAndSort()
	}

	return m, cmd
}

func (m tuiModel) handleDetailsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "escape", "i", "I", "enter":
		// Return to selection
		m.phase = "select"
		m.detailsVM = nil
		return m, nil
	}
	return m, nil
}

func (m tuiModel) countEnabledFeatures() int {
	count := 0
	if m.featureConfig.enableSnapshot {
		count++
	}
	if m.featureConfig.enableBandwidthLimit {
		count++
	}
	if m.featureConfig.enableIncremental {
		count++
	}
	if m.featureConfig.enableEmail {
		count++
	}
	if m.featureConfig.enableCleanup {
		count++
	}
	return count
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
	case "features":
		return m.renderFeatures()
	case "search":
		return m.renderSearch()
	case "details":
		return m.renderDetails()
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

// Cool ASCII art banner
func renderCoolBanner() string {
	banner := `
╦ ╦╦ ╦╔═╗╔═╗╦═╗  ╔═╗═╗ ╦╔═╗╔═╗╦═╗╔╦╗
╠═╣╚╦╝╠═╝║╣ ╠╦╝  ║╣ ╔╩╦╝╠═╝║ ║╠╦╝ ║
╩ ╩ ╩ ╩  ╚═╝╩╚═  ╚═╝╩ ╚═╩  ╚═╝╩╚═ ╩ `

	// Gradient styling for the banner
	gradientBanner := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(banner)

	subtitle := lipgloss.NewStyle().
		Foreground(amberYellow).
		Italic(true).
		Render("        ⚡ Multi-Cloud VM Export Platform ⚡")

	// Create a bordered box around the banner
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(tealInfo).
		Padding(1, 2).
		Align(lipgloss.Center)

	return box.Render(gradientBanner + "\n" + subtitle)
}

func (m tuiModel) renderSelection() string {
	var b strings.Builder

	// Cool banner instead of simple title
	b.WriteString(renderCoolBanner())
	b.WriteString("\n\n")

	if len(m.vms) == 0 {
		b.WriteString(infoStyleTUI.Render("Loading VMs..."))
		return b.String()
	}

	// Cool status bar with gradient background
	selectedCount := m.countSelected()
	totalCount := len(m.vms)
	visibleCount := len(m.getVisibleVMs())

	// Create modern status bar
	statusBar := lipgloss.NewStyle().
		Foreground(tealInfo).
		Background(darkBg).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(deepOrange).
		Width(80)

	statusParts := []string{
		lipgloss.NewStyle().Foreground(tealInfo).Render(fmt.Sprintf("📊 %d Total", totalCount)),
		lipgloss.NewStyle().Foreground(tealInfo).Render(fmt.Sprintf("👁  %d Visible", visibleCount)),
		lipgloss.NewStyle().Foreground(successGreen).Render(fmt.Sprintf("✓ %d Selected", selectedCount)),
	}

	if m.searchQuery != "" {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(deepOrange).Render(fmt.Sprintf("🔍 %s", m.searchQuery)))
	}
	if m.filterPower != "" {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(amberYellow).Render(fmt.Sprintf("⚡ %s", m.filterPower)))
	}

	b.WriteString(statusBar.Render(strings.Join(statusParts, " │ ")))
	b.WriteString("\n\n")

	// Animated hint message with glowing effect
	var hintMsg string
	var hintStyle lipgloss.Style

	if selectedCount == 0 {
		hintMsg = "💡 Press SPACE to select VMs │ A to select all │ ENTER to continue"
		hintStyle = lipgloss.NewStyle().
			Foreground(deepOrange).
			Background(lightCharcoal).
			Italic(true).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(deepOrange)
	} else if selectedCount == 1 {
		hintMsg = fmt.Sprintf("✓ 1 VM selected │ Select more with SPACE │ %d VMs available", visibleCount-1)
		hintStyle = lipgloss.NewStyle().
			Foreground(successGreen).
			Background(darkBg).
			Bold(true).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(successGreen)
	} else {
		hintMsg = fmt.Sprintf("🚀 %d VMs ready for export │ Press ENTER to continue", selectedCount)
		hintStyle = lipgloss.NewStyle().
			Foreground(successGreen).
			Background(darkBg).
			Bold(true).
			Padding(0, 2).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(tealInfo)
	}

	b.WriteString(hintStyle.Render(hintMsg))
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

	// Create a bordered container for VM list
	vmListBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(deepOrange).
		Padding(1, 2)

	var vmListContent strings.Builder

	for i := start; i < end; i++ {
		item := vms[i]

		// Cool cursor with neon effect
		cursor := "   "
		cursorStyle := lipgloss.NewStyle()
		if m.cursor == i {
			cursor = " ▶ "
			cursorStyle = lipgloss.NewStyle().Foreground(tealInfo).Bold(true)
		}

		// Animated checkbox with glow effect
		checkbox := "☐ "
		checkboxStyle := lipgloss.NewStyle().Foreground(mutedGray)
		if item.selected {
			checkbox = "☑ "
			checkboxStyle = lipgloss.NewStyle().
				Foreground(successGreen).
				Bold(true)
		}

		// Power state with cool icons
		var powerIcon string
		var powerColor lipgloss.Color
		if item.vm.PowerState == "poweredOn" {
			powerIcon = "⚡"
			powerColor = successGreen
		} else {
			powerIcon = "○"
			powerColor = mutedGray
		}
		powerStyle := lipgloss.NewStyle().Foreground(powerColor)

		// VM name with dynamic styling
		vmNameStyle := lipgloss.NewStyle().Foreground(tealInfo)
		if item.selected {
			vmNameStyle = lipgloss.NewStyle().
				Foreground(successGreen).
				Bold(true)
		}
		if m.cursor == i {
			vmNameStyle = vmNameStyle.Background(lightBg)
		}

		// Resource info with color coding
		cpuInfo := lipgloss.NewStyle().Foreground(tealInfo).Render(fmt.Sprintf("%2dC", item.vm.NumCPU))
		memInfo := lipgloss.NewStyle().Foreground(amberYellow).Render(fmt.Sprintf("%4.0fG", float64(item.vm.MemoryMB)/1024))
		storageInfo := lipgloss.NewStyle().Foreground(deepOrange).Render(formatBytesCompact(item.vm.Storage))

		// Construct the line with all components
		line := fmt.Sprintf("%s%s%s %-35s %s %s %s %s",
			cursorStyle.Render(cursor),
			checkboxStyle.Render(checkbox),
			powerStyle.Render(powerIcon),
			vmNameStyle.Render(truncateString(item.vm.Name, 35)),
			cpuInfo,
			memInfo,
			storageInfo,
			"",
		)

		// Add selection highlight
		if m.cursor == i {
			lineStyle := lipgloss.NewStyle().
				Background(lightBg).
				Foreground(tealInfo).
				Width(78)
			vmListContent.WriteString(lineStyle.Render(line))
		} else {
			vmListContent.WriteString(line)
		}
		vmListContent.WriteString("\n")
	}

	b.WriteString(vmListBox.Render(vmListContent.String()))
	b.WriteString("\n")

	// Modern help section
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, false, false, false).
		BorderForeground(primaryColor).
		Foreground(mutedColor).
		Padding(1, 0).
		Render(""))
	b.WriteString("\n")

	if m.showHelp {
		// Full help with all keybindings
		helpView := m.helpModel.FullHelpView(m.keys.FullHelp())
		b.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render(helpView))
	} else {
		// Short help with essential keys
		shortHelp := m.helpModel.ShortHelpView(m.keys.ShortHelp())
		b.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("💡 " + shortHelp))
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

	// Advanced features info
	enabledCount := m.countEnabledFeatures()
	if enabledCount > 0 {
		b.WriteString(titleStyleTUI.Render("⚡ Advanced Features"))
		b.WriteString("\n")
		features := []string{}
		if m.featureConfig.enableSnapshot {
			features = append(features, "Snapshots")
		}
		if m.featureConfig.enableBandwidthLimit {
			features = append(features, "Bandwidth Limiting")
		}
		if m.featureConfig.enableIncremental {
			features = append(features, "Incremental Export")
		}
		if m.featureConfig.enableEmail {
			features = append(features, "Email Notifications")
		}
		if m.featureConfig.enableCleanup {
			features = append(features, "Cleanup")
		}
		b.WriteString(successStyleTUI.Render(fmt.Sprintf("✓ Enabled: %s", strings.Join(features, ", "))))
		b.WriteString("\n\n")
	} else {
		b.WriteString(infoStyleTUI.Render("⚡ Advanced features: Not configured (press 'f' to configure)"))
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
	b.WriteString(helpStyleTUI.Render("y/Y/Enter: Start export | u: Cloud upload | f: Features | n/Esc: Go back | q: Quit"))

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

func (m tuiModel) renderSearch() string {
	var b strings.Builder

	b.WriteString(titleStyleTUI.Render("🔍 Live Search"))
	b.WriteString("\n\n")
	b.WriteString(infoStyleTUI.Render("Search VMs by name or path (type to filter in real-time):"))
	b.WriteString("\n\n")

	// Search input box
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(deepOrange).
		Padding(0, 1).
		Width(60)

	b.WriteString(searchBox.Render(m.searchInput.View()))
	b.WriteString("\n\n")

	// Show matching results in real-time
	visibleVMs := m.getVisibleVMs()
	if m.searchQuery != "" {
		if len(visibleVMs) > 0 {
			resultStyle := lipgloss.NewStyle().Foreground(successGreen).Bold(true)
			b.WriteString(resultStyle.Render(fmt.Sprintf("✓ %d VMs match your search", len(visibleVMs))))
			b.WriteString("\n\n")

			// Show preview of matches
			previewCount := 10
			if len(visibleVMs) < previewCount {
				previewCount = len(visibleVMs)
			}

			previewBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(mutedGray).
				Padding(1, 2).
				Width(70)

			var previewList strings.Builder
			for i := 0; i < previewCount; i++ {
				vm := visibleVMs[i]
				// Highlight matching text
				name := highlightMatch(vm.vm.Name, m.searchQuery)
				previewList.WriteString(fmt.Sprintf("  • %s\n", name))
			}

			if len(visibleVMs) > previewCount {
				previewList.WriteString(lipgloss.NewStyle().Foreground(mutedGray).Render(
					fmt.Sprintf("  ... and %d more", len(visibleVMs)-previewCount)))
			}

			b.WriteString(previewBox.Render(previewList.String()))
		} else {
			b.WriteString(errorStyleTUI.Render("⚠ No VMs match your search"))
		}
	} else {
		hintStyle := lipgloss.NewStyle().Foreground(mutedGray).Italic(true)
		b.WriteString(hintStyle.Render(fmt.Sprintf("💡 %d VMs available. Start typing to filter...", len(m.vms))))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyleTUI.Render("Enter: Apply search | Esc: Cancel | Type: Filter in real-time"))

	return b.String()
}

// highlightMatch highlights matching substring in text
func highlightMatch(text, query string) string {
	if query == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	index := strings.Index(lowerText, lowerQuery)

	if index == -1 {
		return text
	}

	before := text[:index]
	match := text[index : index+len(query)]
	after := text[index+len(query):]

	highlightStyle := lipgloss.NewStyle().Foreground(deepOrange).Background(lightCharcoal).Bold(true)
	return before + highlightStyle.Render(match) + after
}

func (m tuiModel) renderDetails() string {
	var b strings.Builder

	if m.detailsVM == nil {
		return errorStyleTUI.Render("No VM selected for details")
	}

	vm := *m.detailsVM

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(lightCharcoal).
		Width(80).
		Align(lipgloss.Center).
		Render("🖥️  VM Information")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Main info box
	mainInfoBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(deepOrange).
		Padding(1, 2).
		Width(76)

	var mainInfo strings.Builder

	// VM Name
	mainInfo.WriteString(lipgloss.NewStyle().
		Foreground(successGreen).
		Bold(true).
		Render(fmt.Sprintf("🏷️  Name: %s", vm.Name)))
	mainInfo.WriteString("\n\n")

	// Power State
	var powerIcon string
	var powerColor lipgloss.Color
	if vm.PowerState == "poweredOn" {
		powerIcon = "⚡"
		powerColor = successGreen
	} else {
		powerIcon = "○"
		powerColor = mutedGray
	}
	mainInfo.WriteString(lipgloss.NewStyle().Foreground(powerColor).Render(
		fmt.Sprintf("%s Power State: %s", powerIcon, vm.PowerState)))
	mainInfo.WriteString("\n\n")

	// Guest OS
	mainInfo.WriteString(lipgloss.NewStyle().Foreground(tealInfo).Render(
		fmt.Sprintf("💿 Guest OS: %s", vm.GuestOS)))
	mainInfo.WriteString("\n\n")

	// Path
	mainInfo.WriteString(lipgloss.NewStyle().Foreground(mutedGray).Render(
		fmt.Sprintf("📁 Path: %s", vm.Path)))

	b.WriteString(mainInfoBox.Render(mainInfo.String()))
	b.WriteString("\n\n")

	// Resources section
	resourcesBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tealInfo).
		Padding(1, 2).
		Width(76)

	var resources strings.Builder
	resources.WriteString(lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render("⚙️  Resources"))
	resources.WriteString("\n\n")

	// CPU
	resources.WriteString(lipgloss.NewStyle().Foreground(amberYellow).Render(
		fmt.Sprintf("  🔢 vCPUs:         %d", vm.NumCPU)))
	resources.WriteString("\n")

	// Memory
	memoryGB := float64(vm.MemoryMB) / 1024.0
	resources.WriteString(lipgloss.NewStyle().Foreground(amberYellow).Render(
		fmt.Sprintf("  🧠 Memory:        %d MB (%.1f GB)", vm.MemoryMB, memoryGB)))
	resources.WriteString("\n")

	// Storage
	storageGB := float64(vm.Storage) / (1024 * 1024 * 1024)
	resources.WriteString(lipgloss.NewStyle().Foreground(amberYellow).Render(
		fmt.Sprintf("  💾 Storage:       %s (%.1f GB)", formatBytes(vm.Storage), storageGB)))

	b.WriteString(resourcesBox.Render(resources.String()))
	b.WriteString("\n\n")

	// Estimated export size
	estimateBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warningColor).
		Padding(1, 2).
		Width(76)

	var estimate strings.Builder
	estimate.WriteString(lipgloss.NewStyle().
		Foreground(warningColor).
		Bold(true).
		Render("📦 Export Information"))
	estimate.WriteString("\n\n")

	estimate.WriteString(lipgloss.NewStyle().Foreground(textColor).Render(
		fmt.Sprintf("  Estimated Size:   ~%s", formatBytes(vm.Storage))))
	estimate.WriteString("\n")

	// Check disk space
	diskSpace := getDiskSpace(m.outputDir)
	if diskSpace > vm.Storage {
		estimate.WriteString(lipgloss.NewStyle().Foreground(successGreen).Render(
			fmt.Sprintf("  Available Space:  %s ✓", formatBytes(diskSpace))))
	} else {
		estimate.WriteString(lipgloss.NewStyle().Foreground(errorColor).Render(
			fmt.Sprintf("  Available Space:  %s ⚠ INSUFFICIENT!", formatBytes(diskSpace))))
	}

	b.WriteString(estimateBox.Render(estimate.String()))
	b.WriteString("\n\n")

	// Help
	b.WriteString(helpStyleTUI.Render("Press i/Esc/Enter to return to VM list"))

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

func (m tuiModel) renderFeatures() string {
	var b strings.Builder

	b.WriteString(titleStyleTUI.Render("⚡ Advanced Features Configuration"))
	b.WriteString("\n\n")

	// Feature options
	features := []struct {
		name        string
		description string
		enabled     bool
	}{
		// Snapshot features
		{"Snapshot Management", "Create VM snapshots before export", m.featureConfig.enableSnapshot},
		{"  Include Memory", "Include VM memory in snapshot", m.featureConfig.snapshotMemory},
		{"  Quiesce Filesystem", "Quiesce filesystem for consistency", m.featureConfig.snapshotQuiesce},
		{"  Delete After Export", "Remove snapshot after export completes", m.featureConfig.deleteSnapshot},
		{"  Consolidate Snapshots", "Merge all snapshots into base disks", m.featureConfig.consolidateSnaps},

		// Bandwidth limiting
		{"Bandwidth Limiting", "Control network bandwidth usage", m.featureConfig.enableBandwidthLimit},
		{"  Adaptive Bandwidth", "Automatically adjust based on network", m.featureConfig.adaptiveBandwidth},

		// Incremental export
		{"Incremental Export", "Export only changed disks", m.featureConfig.enableIncremental},
		{"  Show Analysis Only", "Preview savings without exporting", m.featureConfig.showIncrementalInfo},

		// Email notifications
		{"Email Notifications", "Send email alerts for export events", m.featureConfig.enableEmail},
		{"  Email on Start", "Notify when export starts", m.featureConfig.emailOnStart},
		{"  Email on Complete", "Notify when export completes", m.featureConfig.emailOnComplete},

		// Cleanup
		{"Export Cleanup", "Automatically clean up old exports", m.featureConfig.enableCleanup},
		{"  Dry Run Mode", "Preview cleanup without deleting", m.featureConfig.cleanupDryRun},
	}

	for i, feature := range features {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		checkbox := "[ ]"
		if feature.enabled {
			checkbox = "[✓]"
		}

		style := unselectedStyleTUI
		if i == m.cursor {
			style = selectedStyleTUI
		}

		line := fmt.Sprintf("%s%s %s", cursor, checkbox, feature.name)
		b.WriteString(style.Render(line))
		b.WriteString("\n")

		if i == m.cursor {
			b.WriteString(infoStyleTUI.Render(fmt.Sprintf("    %s", feature.description)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Summary
	enabledCount := m.countEnabledFeatures()
	if enabledCount > 0 {
		b.WriteString(successStyleTUI.Render(fmt.Sprintf("✓ %d features enabled", enabledCount)))
	} else {
		b.WriteString(infoStyleTUI.Render("No advanced features enabled"))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyleTUI.Render("↑/↓: Navigate | Space/Enter: Toggle | s: Save & Back | Esc: Cancel"))

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

	// Modern header with gradient
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(lightCharcoal).
		Width(80).
		Align(lipgloss.Center).
		Render("🚀 Exporting Virtual Machines")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Modern progress bar
	progressPercent := float64(m.currentExport) / float64(len(selectedVMs))
	progressBar := m.progressBar.ViewAs(progressPercent)

	progressInfo := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render(fmt.Sprintf("Overall Progress: %d / %d VMs (%.0f%%)",
			m.currentExport,
			len(selectedVMs),
			progressPercent*100))

	b.WriteString(progressInfo)
	b.WriteString("\n")
	b.WriteString(progressBar)
	b.WriteString("\n\n")

	// Current VM info with real-time progress
	if m.currentExport < len(selectedVMs) {
		currentVM := selectedVMs[m.currentExport]

		// Build progress details
		var progressDetails strings.Builder
		progressDetails.WriteString(fmt.Sprintf("⏳ Currently Exporting: %s\n",
			lipgloss.NewStyle().Bold(true).Foreground(successColor).Render(currentVM.vm.Name)))
		progressDetails.WriteString(fmt.Sprintf("   Path: %s\n", currentVM.vm.Path))

		// Real-time transfer statistics
		if m.exportProgress.totalBytes > 0 {
			filePercent := float64(m.exportProgress.currentBytes) / float64(m.exportProgress.totalBytes) * 100
			progressDetails.WriteString(fmt.Sprintf("\n   📊 Transfer: %s / %s (%.1f%%)\n",
				formatBytes(m.exportProgress.currentBytes),
				formatBytes(m.exportProgress.totalBytes),
				filePercent))

			// File progress bar
			fileBar := m.progressBar.ViewAs(filePercent / 100.0)
			progressDetails.WriteString(fmt.Sprintf("   %s\n", fileBar))
		}

		// Current file being transferred
		if m.currentFileName != "" {
			progressDetails.WriteString(fmt.Sprintf("\n   📄 File: %s (%d/%d)\n",
				truncateString(m.currentFileName, 50),
				m.exportProgress.currentFileIdx+1,
				m.exportProgress.totalFiles))
		}

		// Transfer speed
		if m.exportProgress.speed > 0 {
			speedStyle := lipgloss.NewStyle().Foreground(tealInfo).Bold(true)
			progressDetails.WriteString(fmt.Sprintf("\n   ⚡ Speed: %s\n",
				speedStyle.Render(fmt.Sprintf("%.2f MB/s", m.exportProgress.speed))))

			// Calculate ETA
			if m.exportProgress.totalBytes > m.exportProgress.currentBytes && m.exportProgress.speed > 0 {
				remainingBytes := m.exportProgress.totalBytes - m.exportProgress.currentBytes
				remainingMB := float64(remainingBytes) / (1024 * 1024)
				etaSeconds := remainingMB / m.exportProgress.speed
				etaDuration := time.Duration(etaSeconds * float64(time.Second))

				etaStyle := lipgloss.NewStyle().Foreground(amberYellow)
				progressDetails.WriteString(fmt.Sprintf("   ⏱  ETA: %s\n",
					etaStyle.Render(formatDuration(etaDuration))))
			}
		}

		// Elapsed time
		if !m.exportProgress.startTime.IsZero() {
			elapsed := time.Since(m.exportProgress.startTime)
			progressDetails.WriteString(fmt.Sprintf("\n   ⌛ Elapsed: %s",
				formatDuration(elapsed)))
		}

		currentBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Render(progressDetails.String())
		b.WriteString(currentBox)
		b.WriteString("\n\n")
	}

	// VM list with modern icons and styling
	vmListBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Padding(1, 2).
		Width(78)

	var vmList strings.Builder
	for i, item := range selectedVMs {
		var status string
		var icon string
		var style lipgloss.Style

		if i < m.currentExport {
			status = "Completed"
			icon = "✅"
			style = lipgloss.NewStyle().Foreground(successColor)
		} else if i == m.currentExport {
			// Show spinner for current export
			status = m.spinner.View() + " Exporting..."
			icon = ""
			style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
		} else {
			status = "Pending"
			icon = "⏸️ "
			style = lipgloss.NewStyle().Foreground(mutedColor)
		}

		vmLine := style.Render(fmt.Sprintf("%s %-45s %s",
			icon,
			truncateString(item.vm.Name, 45),
			status))
		vmList.WriteString(vmLine)
		vmList.WriteString("\n")
	}

	b.WriteString(vmListBox.Render(vmList.String()))
	b.WriteString("\n\n")

	// Bandwidth usage visualization
	if m.exportProgress.speed > 0 {
		bandwidthBox := m.renderBandwidthGraph()
		b.WriteString(bandwidthBox)
		b.WriteString("\n\n")
	}

	// Modern help with bubbles/help component
	helpView := m.helpModel.ShortHelpView([]key.Binding{m.keys.Quit})
	b.WriteString(lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Render("💡 " + helpView + " | Export in progress..."))

	return b.String()
}

// renderBandwidthGraph creates a simple bar graph for bandwidth usage
func (m tuiModel) renderBandwidthGraph() string {
	maxBandwidth := 100.0 // Assume 100 MB/s max for visualization
	if m.featureConfig.enableBandwidthLimit && m.featureConfig.bandwidthLimitMBps > 0 {
		maxBandwidth = float64(m.featureConfig.bandwidthLimitMBps)
	}

	barWidth := 40
	currentBars := int(m.exportProgress.speed / maxBandwidth * float64(barWidth))
	if currentBars > barWidth {
		currentBars = barWidth
	}

	bars := strings.Repeat("█", currentBars)
	empty := strings.Repeat("░", barWidth-currentBars)

	bandwidthStr := fmt.Sprintf("📈 Bandwidth: %s%s %.2f/%.0f MB/s",
		lipgloss.NewStyle().Foreground(successGreen).Render(bars),
		lipgloss.NewStyle().Foreground(mutedGray).Render(empty),
		m.exportProgress.speed,
		maxBandwidth)

	return lipgloss.NewStyle().
		Foreground(tealInfo).
		Render(bandwidthStr)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func (m tuiModel) renderDone() string {
	return successStyleTUI.Render("✅ Export complete!\n\nPress q to quit")
}

func (m tuiModel) renderHelp() string {
	help := `⌨️  Keyboard Shortcuts

Navigation:
  ↑/k       Move up
  ↓/j       Move down
  Space     Toggle checkbox [ ] / [x] - Multi-select VMs
  Enter     Continue to confirmation

Multi-Selection:
  a         Select all visible VMs [x]
  n         Clear all selections [ ]
  A         Bulk select by regex pattern
  1-7       Quick filters (select VMs matching criteria)

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
  f         Advanced features
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
