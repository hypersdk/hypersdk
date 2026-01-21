// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"os"
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
// logEntry represents a single log message in the TUI
type logEntry struct {
	timestamp time.Time
	level     string // "INFO", "WARN", "ERROR", "DEBUG"
	message   string
	vmName    string // associated VM if any
}

// folderNode represents a folder in the VM hierarchy tree
type folderNode struct {
	name     string
	path     string
	parent   *folderNode
	children []*folderNode
	vms      []tuiVMItem
	expanded bool
	level    int
}

// exportPreview holds estimated export information
type exportPreview struct {
	vmName            string
	totalSize         int64
	diskCount         int
	fileBreakdown     map[string]int64 // file type -> size
	estimatedDuration time.Duration
	diskSpaceNeeded   int64
	diskSpaceAvail    int64
	files             []previewFile
}

// previewFile represents a file in the export preview
type previewFile struct {
	name     string
	fileType string // "vmdk", "ovf", "mf", "cert"
	size     int64
}

// quickAction represents an action that can be performed on a VM
type quickAction struct {
	name        string
	description string
	icon        string
	category    string // "power", "snapshot", "export", "info"
	handler     func(*tuiModel, *vsphere.VMInfo) (tea.Model, tea.Cmd)
	enabled     func(*vsphere.VMInfo) bool
}

type tuiModel struct {
	vms              []tuiVMItem
	filteredVMs      []tuiVMItem
	cursor           int
	phase            string // "select", "confirm", "template", "regex", "cloud", "features", "export", "cloudupload", "done", "search", "details", "validation", "config", "stats", "queue", "history", "logs", "tree", "preview", "actions"
	detailsVM        *vsphere.VMInfo // VM to show details for
	validationReport *ValidationReport // Pre-export validation results
	configPanel      *configPanelState // Interactive config panel state
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

	// Terminal size
	termWidth  int
	termHeight int

	// Animation frame counter
	animFrame int

	// Split-screen mode
	splitScreenMode bool
	focusedPane     string // "list" or "details"

	// Export queue management
	exportQueue     []queuedExport
	queueCursor     int
	showQueueEditor bool

	// Export history view
	historyEntries       []ExportHistoryEntry
	historyCursor        int
	historyFilter        string // "all", "success", "failed"
	historySearchQuery   string
	historyDateFilter    string // "all", "today", "week", "month"
	historyProviderFilter string // "all", "vsphere", etc.

	// Live logs viewer
	logEntries      []logEntry
	logCursor       int
	logLevelFilter  string // "all", "info", "warn", "error", "debug"
	logSearchQuery  string
	autoScrollLogs  bool
	showLogsPanel   bool
	maxLogEntries   int

	// Folder tree view
	folderTree      *folderNode
	viewMode        string // "list" or "tree"
	treeItems       []interface{} // flattened tree for rendering (mix of *folderNode and tuiVMItem)
	treeCursor      int

	// Export preview
	exportPreviews  []exportPreview
	previewCursor   int
	showPreview     bool

	// Quick actions menu
	showActionsMenu bool
	actionsCursor   int
	actionsForVM    *vsphere.VMInfo

	// Configuration
	client    *vsphere.VSphereClient
	outputDir string
	log       logger.Logger
	ctx       context.Context
}

// queuedExport represents a VM in the export queue
type queuedExport struct {
	vm       vsphere.VMInfo
	priority int    // 1=high, 2=normal, 3=low
	status   string // "pending", "running", "completed", "failed"
	eta      time.Duration
	startedAt time.Time
	error    error
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

// configPanelState manages interactive configuration editing
type configPanelState struct {
	focusedField int
	fields       []configField
	isDirty      bool
}

// configField represents a single configurable field
type configField struct {
	label       string
	key         string
	value       string
	inputModel  textinput.Model
	fieldType   string // "text", "number", "bool", "select"
	options     []string // for select fields
	description string
	validator   func(string) error
}

// Modern key bindings
type tuiKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Select      key.Binding
	Confirm     key.Binding
	Back        key.Binding
	Quit        key.Binding
	Help        key.Binding
	Filter      key.Binding
	Sort        key.Binding
	Features    key.Binding
	Cloud       key.Binding
	SplitScreen key.Binding
	SwitchPane  key.Binding
	Queue       key.Binding
	MoveUp      key.Binding
	MoveDown    key.Binding
	Priority    key.Binding
	History     key.Binding
	FilterHistory key.Binding
	Logs        key.Binding
	FilterLogs  key.Binding
	ToggleAutoScroll key.Binding
	Tree        key.Binding
	ExpandFolder key.Binding
	Preview     key.Binding
	Actions     key.Binding
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

	// Enhanced progress bar with gradient colors
	progressFilledStyleTUI = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	progressEmptyStyleTUI = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Animated spinner styles for different contexts
	spinnerStyleTUI = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	spinnerLoadingStyleTUI = lipgloss.NewStyle().
				Foreground(tealInfo)

	spinnerProcessingStyleTUI = lipgloss.NewStyle().
					Foreground(amberYellow)
)

// getSpinnerFrames returns animated spinner frames
func getSpinnerFrames() []string {
	return []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
	}
}

// getDotSpinnerFrames returns dot-based spinner frames
func getDotSpinnerFrames() []string {
	return []string{
		"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷",
	}
}

// getProgressSpinnerFrames returns progress-style spinner frames
func getProgressSpinnerFrames() []string {
	return []string{
		"◐", "◓", "◑", "◒",
	}
}

// renderAnimatedProgressBar renders a smooth gradient progress bar
func renderAnimatedProgressBar(percent float64, width int) string {
	if width < 4 {
		return ""
	}

	filled := int(float64(width) * (percent / 100.0))
	if filled > width {
		filled = width
	}

	// Build progress bar with gradient effect
	var bar strings.Builder

	// Filled portion with gradient
	for i := 0; i < filled; i++ {
		if i < filled-1 {
			bar.WriteString(progressFilledStyleTUI.Render("█"))
		} else {
			// Last character with gradient
			bar.WriteString(progressFilledStyleTUI.Render("▓"))
		}
	}

	// Empty portion
	for i := filled; i < width; i++ {
		bar.WriteString(progressEmptyStyleTUI.Render("░"))
	}

	return bar.String()
}

// renderPulsingDot returns a pulsing dot for loading states
func renderPulsingDot(frame int) string {
	dots := []string{"⚬", "⚬", "⚬", "⦿", "⦿", "⦿"}
	idx := frame % len(dots)
	if idx < 3 {
		return spinnerLoadingStyleTUI.Render(dots[idx])
	}
	return spinnerProcessingStyleTUI.Render(dots[idx])
}

type vmsLoadedMsg struct {
	vms []vsphere.VMInfo
	err error
}

type exportDoneMsg struct {
	vmName string
	err    error
}

type tickMsg time.Time

type exportProgressMsg struct {
	vmName         string
	currentBytes   int64
	totalBytes     int64
	currentFileIdx int
	totalFiles     int
	fileName       string
}

type validationCompleteMsg struct {
	report *ValidationReport
	err    error
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.loadVMs, m.spinner.Tick, tickCmd())
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

	case tea.WindowSizeMsg:
		// Update terminal size and adjust UI components
		m.termWidth = msg.Width
		m.termHeight = msg.Height

		// Update progress bar width to fit terminal
		if msg.Width > 20 {
			m.progressBar.Width = min(msg.Width-10, 70)
		}

		// Update help model width
		m.helpModel.Width = msg.Width

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
		case "validation":
			return m.handleValidationKeys(msg)
		case "config":
			return m.handleConfigPanelKeys(msg)
		case "stats":
			return m.handleStatsKeys(msg)
		case "queue":
			return m.handleQueueKeys(msg)
		case "history":
			return m.handleHistoryKeys(msg)
		case "logs":
			return m.handleLogsKeys(msg)
		case "tree":
			return m.handleTreeKeys(msg)
		case "preview":
			return m.handlePreviewKeys(msg)
		case "actions":
			return m.handleActionsKeys(msg)
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
		// Increment animation frame for spinners and animations
		m.animFrame++

		var cmd tea.Cmd
		if m.phase == "export" || m.phase == "validation" || m.phase == "cloudupload" {
			// Continue ticking during animated phases
			cmd = tickCmd()
		}
		// Update spinner
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case validationCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = "error"
			return m, nil
		}
		m.validationReport = msg.report
		// Validation display is already shown, just update the model
		return m, nil
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

	case "D":
		// Show statistics dashboard
		m.phase = "stats"
		return m, nil

	case "Q":
		// Show export queue manager
		if m.countSelected() == 0 {
			m.message = "No VMs selected!"
			return m, nil
		}
		// Build queue from selected VMs
		m.exportQueue = m.buildExportQueue()
		m.queueCursor = 0
		m.showQueueEditor = true
		m.phase = "queue"
		return m, nil

	case "H":
		// Show export history
		m.phase = "history"
		m.historyCursor = 0
		m.historyFilter = "all"
		m.historyDateFilter = "all"
		m.historySearchQuery = ""
		m.historyProviderFilter = "all"
		// Load history entries
		if err := m.loadHistoryEntries(); err != nil {
			m.message = fmt.Sprintf("Failed to load history: %v", err)
		}
		return m, nil

	case "L":
		// Show live logs viewer
		m.phase = "logs"
		m.logCursor = 0
		m.logLevelFilter = "all"
		m.logSearchQuery = ""
		// Add some sample logs if empty (for demo)
		if len(m.logEntries) == 0 {
			m.addLogEntry("INFO", "Logs viewer opened", "")
			m.addLogEntry("INFO", "Ready to display export logs", "")
		}
		return m, nil

	case "]":
		// Show folder tree view
		m.viewMode = "tree"
		m.phase = "tree"
		m.treeCursor = 0
		m.buildFolderTree()
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

	case "v", "V":
		// Toggle split screen mode
		m.splitScreenMode = !m.splitScreenMode
		m.focusedPane = "list"
		return m, nil

	case "tab":
		// Switch pane in split screen mode
		if m.splitScreenMode {
			if m.focusedPane == "list" {
				m.focusedPane = "details"
			} else {
				m.focusedPane = "list"
			}
		}
		return m, nil

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

	case "x", "X":
		// Show quick actions menu
		vms := m.getVisibleVMs()
		if m.cursor < len(vms) {
			m.actionsForVM = &vms[m.cursor].vm
			m.actionsCursor = 0
			m.showActionsMenu = true
			m.phase = "actions"
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
		// Run validation first
		m.phase = "validation"
		return m, m.runValidation()

	case "v", "V":
		// Show validation without starting export
		m.phase = "validation"
		return m, m.runValidation()
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
	case "c", "C":
		// Open interactive configuration panel
		m.phase = "config"
		m.configPanel = m.newConfigPanel()
		return m, nil
	case "p", "P":
		// Show export preview
		m.phase = "preview"
		m.previewCursor = 0
		m.showPreview = true
		// Generate preview data for selected VMs
		m.exportPreviews = m.generateExportPreviews()
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

func (m tuiModel) handleValidationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "escape", "n":
		// Cancel and go back
		m.phase = "confirm"
		m.validationReport = nil
		return m, nil
	case "y", "Y", "enter":
		// Proceed with export if validation passed
		if m.validationReport != nil && m.validationReport.AllPassed {
			m.phase = "export"
			m.currentExport = 0
			m.exportProgress.startTime = time.Now()
			return m, tea.Batch(m.exportNext(), tickCmd())
		}
		// If validation failed, require explicit override
		m.message = "Validation failed! Press 'o' to override and export anyway, or Esc to cancel"
		return m, nil
	case "o", "O":
		// Override validation failures
		m.phase = "export"
		m.currentExport = 0
		m.exportProgress.startTime = time.Now()
		return m, tea.Batch(m.exportNext(), tickCmd())
	}
	return m, nil
}

// newConfigPanel creates an interactive configuration panel
func (m *tuiModel) newConfigPanel() *configPanelState {
	panel := &configPanelState{
		focusedField: 0,
		fields:       make([]configField, 0),
		isDirty:      false,
	}

	// Helper to create text input
	makeInput := func(placeholder, value string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.SetValue(value)
		ti.CharLimit = 200
		ti.Width = 60
		return ti
	}

	// Add configuration fields
	fields := []configField{
		{
			label:       "Output Directory",
			key:         "output_dir",
			value:       m.outputDir,
			inputModel:  makeInput("/path/to/exports", m.outputDir),
			fieldType:   "text",
			description: "Directory where VM exports will be saved",
		},
		{
			label:       "Bandwidth Limit (MB/s)",
			key:         "bandwidth_limit",
			value:       fmt.Sprintf("%d", m.featureConfig.bandwidthLimitMBps),
			inputModel:  makeInput("0 = unlimited", fmt.Sprintf("%d", m.featureConfig.bandwidthLimitMBps)),
			fieldType:   "number",
			description: "Maximum upload/download speed (0 for unlimited)",
		},
		{
			label:       "Email SMTP Host",
			key:         "email_smtp_host",
			value:       m.featureConfig.emailSMTPHost,
			inputModel:  makeInput("smtp.example.com", m.featureConfig.emailSMTPHost),
			fieldType:   "text",
			description: "SMTP server hostname for email notifications",
		},
		{
			label:       "Email From Address",
			key:         "email_from",
			value:       m.featureConfig.emailFrom,
			inputModel:  makeInput("exports@example.com", m.featureConfig.emailFrom),
			fieldType:   "text",
			description: "From email address for notifications",
		},
		{
			label:       "Email To Address",
			key:         "email_to",
			value:       m.featureConfig.emailTo,
			inputModel:  makeInput("admin@example.com", m.featureConfig.emailTo),
			fieldType:   "text",
			description: "Recipient email address (comma-separated for multiple)",
		},
		{
			label:       "Keep Snapshots Count",
			key:         "keep_snapshots",
			value:       fmt.Sprintf("%d", m.featureConfig.keepSnapshots),
			inputModel:  makeInput("0 = keep all", fmt.Sprintf("%d", m.featureConfig.keepSnapshots)),
			fieldType:   "number",
			description: "Number of snapshots to retain (0 = keep all)",
		},
		{
			label:       "Cleanup Max Age (days)",
			key:         "cleanup_max_age",
			value:       fmt.Sprintf("%d", m.featureConfig.cleanupMaxAge),
			inputModel:  makeInput("30", fmt.Sprintf("%d", m.featureConfig.cleanupMaxAge)),
			fieldType:   "number",
			description: "Delete exports older than this many days",
		},
		{
			label:       "Cleanup Max Count",
			key:         "cleanup_max_count",
			value:       fmt.Sprintf("%d", m.featureConfig.cleanupMaxCount),
			inputModel:  makeInput("10", fmt.Sprintf("%d", m.featureConfig.cleanupMaxCount)),
			fieldType:   "number",
			description: "Keep only N most recent exports",
		},
	}

	panel.fields = fields
	// Focus first field
	if len(panel.fields) > 0 {
		panel.fields[0].inputModel.Focus()
	}

	return panel
}

// handleConfigPanelKeys handles keyboard input for config panel
func (m tuiModel) handleConfigPanelKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.configPanel == nil {
		// Initialize config panel if not present
		m.configPanel = m.newConfigPanel()
	}

	panel := m.configPanel

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape":
		// Cancel and go back
		m.phase = "confirm"
		m.configPanel = nil
		return m, nil

	case "up", "shift+tab":
		// Move to previous field
		if panel.focusedField > 0 {
			panel.fields[panel.focusedField].inputModel.Blur()
			panel.focusedField--
			panel.fields[panel.focusedField].inputModel.Focus()
		}
		return m, nil

	case "down", "tab":
		// Move to next field
		if panel.focusedField < len(panel.fields)-1 {
			panel.fields[panel.focusedField].inputModel.Blur()
			panel.focusedField++
			panel.fields[panel.focusedField].inputModel.Focus()
		}
		return m, nil

	case "enter":
		// Move to next field or save if at last field
		if panel.focusedField < len(panel.fields)-1 {
			panel.fields[panel.focusedField].inputModel.Blur()
			panel.focusedField++
			panel.fields[panel.focusedField].inputModel.Focus()
		} else {
			// Save configuration
			return m.saveConfigPanel()
		}
		return m, nil

	case "ctrl+s":
		// Save configuration
		return m.saveConfigPanel()

	default:
		// Update the focused field's input model
		var cmd tea.Cmd
		panel.fields[panel.focusedField].inputModel, cmd = panel.fields[panel.focusedField].inputModel.Update(msg)
		panel.isDirty = true
		return m, cmd
	}
}

// saveConfigPanel saves the configuration from the panel
func (m tuiModel) saveConfigPanel() (tea.Model, tea.Cmd) {
	if m.configPanel == nil {
		return m, nil
	}

	// Update model with values from config panel
	for _, field := range m.configPanel.fields {
		value := field.inputModel.Value()
		switch field.key {
		case "output_dir":
			m.outputDir = value
		case "bandwidth_limit":
			if val := parseInt(value); val >= 0 {
				m.featureConfig.bandwidthLimitMBps = val
			}
		case "email_smtp_host":
			m.featureConfig.emailSMTPHost = value
		case "email_from":
			m.featureConfig.emailFrom = value
		case "email_to":
			m.featureConfig.emailTo = value
		case "keep_snapshots":
			if val := parseInt(value); val >= 0 {
				m.featureConfig.keepSnapshots = int(val)
			}
		case "cleanup_max_age":
			if val := parseInt(value); val >= 0 {
				m.featureConfig.cleanupMaxAge = int(val)
			}
		case "cleanup_max_count":
			if val := parseInt(value); val >= 0 {
				m.featureConfig.cleanupMaxCount = int(val)
			}
		}
	}

	m.message = "Configuration saved successfully!"
	m.phase = "confirm"
	m.configPanel = nil
	return m, nil
}

// parseInt parses a string to int64, returns 0 on error
func parseInt(s string) int64 {
	var val int64
	fmt.Sscanf(s, "%d", &val)
	return val
}

// handleStatsKeys handles keyboard input for stats dashboard
func (m tuiModel) handleStatsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "escape":
		// Go back to select screen
		m.phase = "select"
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
	case "validation":
		return m.renderValidation()
	case "config":
		return m.renderConfigPanel()
	case "stats":
		return m.renderStats()
	case "queue":
		return m.renderQueue()
	case "history":
		return m.renderHistory()
	case "logs":
		return m.renderLogs()
	case "tree":
		return m.renderTree()
	case "preview":
		return m.renderPreview()
	case "actions":
		return m.renderActions()
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

// renderSplitScreen renders a split-screen layout with VM list and details side-by-side
func (m tuiModel) renderSplitScreen() string {
	var b strings.Builder

	// Calculate pane widths based on terminal width
	termWidth := m.getResponsiveWidth()
	leftPaneWidth := termWidth * 45 / 100  // 45% for list
	rightPaneWidth := termWidth * 50 / 100 // 50% for details

	if termWidth < 80 {
		// On narrow terminals, use vertical split instead
		return m.renderVerticalSplit()
	}

	// Header with split screen indicator
	header := lipgloss.NewStyle().
		Foreground(tealInfo).
		Background(darkBg).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(deepOrange).
		Width(termWidth - 4).
		Render("╔═══ SPLIT VIEW MODE ═══╗  VMs │ Details  (Tab: Switch Pane | V: Exit Split View)")

	b.WriteString(header)
	b.WriteString("\n\n")

	// Render left pane (VM list)
	leftPane := m.renderVMListPane(leftPaneWidth)

	// Render right pane (VM details)
	rightPane := m.renderDetailsPane(rightPaneWidth)

	// Create bordered styles for each pane
	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(func() lipgloss.Color {
			if m.focusedPane == "list" {
				return tealInfo
			}
			return lipgloss.Color("#4B5563")
		}()).
		Width(leftPaneWidth).
		Height(m.termHeight - 10)

	rightStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(func() lipgloss.Color {
			if m.focusedPane == "details" {
				return tealInfo
			}
			return lipgloss.Color("#4B5563")
		}()).
		Width(rightPaneWidth).
		Height(m.termHeight - 10)

	// Join panes horizontally
	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(leftPane),
		"  ",
		rightStyle.Render(rightPane),
	)

	b.WriteString(panes)
	b.WriteString("\n")

	// Footer with keyboard shortcuts
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("Tab: Switch Pane | ↑/↓: Navigate | Space: Select | Enter: Export | V: Exit Split View | Q: Quit")

	b.WriteString(footer)

	return b.String()
}

// renderVMListPane renders the left pane with VM list
func (m tuiModel) renderVMListPane(width int) string {
	var b strings.Builder

	// Pane title
	title := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(fmt.Sprintf("📋 VM List (%s)", func() string {
			if m.focusedPane == "list" {
				return "ACTIVE"
			}
			return "inactive"
		}()))

	b.WriteString(title)
	b.WriteString("\n\n")

	// Stats
	selectedCount := m.countSelected()
	totalCount := len(m.vms)
	visibleCount := len(m.getVisibleVMs())

	stats := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Render(fmt.Sprintf("Total: %d | Visible: %d | Selected: %d", totalCount, visibleCount, selectedCount))

	b.WriteString(stats)
	b.WriteString("\n\n")

	// VM list
	vms := m.getVisibleVMs()
	maxVisible := m.termHeight - 15
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := m.cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(vms) {
		end = len(vms)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end && i < len(vms); i++ {
		vm := vms[i]
		cursor := " "
		if i == m.cursor {
			cursor = "❯"
		}

		checkbox := "[ ]"
		if vm.selected {
			checkbox = "[✓]"
		}

		// Truncate name if too long
		name := vm.vm.Name
		maxNameLen := width - 15
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		style := lipgloss.NewStyle()
		if i == m.cursor {
			style = style.Foreground(tealInfo).Bold(true)
		} else if vm.selected {
			style = style.Foreground(successGreen)
		}

		line := fmt.Sprintf("%s %s %s", cursor, checkbox, name)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// renderDetailsPane renders the right pane with VM details
func (m tuiModel) renderDetailsPane(width int) string {
	var b strings.Builder

	// Pane title
	title := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(fmt.Sprintf("📊 VM Details (%s)", func() string {
			if m.focusedPane == "details" {
				return "ACTIVE"
			}
			return "inactive"
		}()))

	b.WriteString(title)
	b.WriteString("\n\n")

	// Get currently selected VM
	vms := m.getVisibleVMs()
	if m.cursor >= len(vms) {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Render("No VM selected"))
		return b.String()
	}

	vm := vms[m.cursor].vm

	// VM details
	details := []struct {
		label string
		value string
		color lipgloss.Color
	}{
		{"Name", vm.Name, tealInfo},
		{"Path", vm.Path, lipgloss.Color("#6B7280")},
		{"Power", vm.PowerState, func() lipgloss.Color {
			if vm.PowerState == "poweredOn" {
				return successGreen
			}
			return amberYellow
		}()},
		{"OS", vm.GuestOS, lipgloss.Color("#9CA3AF")},
		{"CPU", fmt.Sprintf("%d cores", vm.NumCPU), lipgloss.Color("#60A5FA")},
		{"Memory", fmt.Sprintf("%d MB", vm.MemoryMB), lipgloss.Color("#34D399")},
		{"Storage", fmt.Sprintf("%.2f GB", float64(vm.Storage)/(1024*1024*1024)), lipgloss.Color("#F59E0B")},
	}

	for _, detail := range details {
		label := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Width(12).
			Render(detail.label + ":")

		value := lipgloss.NewStyle().
			Foreground(detail.color).
			Bold(true).
			Render(detail.value)

		b.WriteString(label + " " + value)
		b.WriteString("\n")
	}

	return b.String()
}

// renderVerticalSplit renders a vertical split for narrow terminals
func (m tuiModel) renderVerticalSplit() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render("╔═══ SPLIT VIEW (Vertical) ═══╗"))
	b.WriteString("\n\n")

	// VM List on top
	b.WriteString(m.renderVMListPane(m.termWidth - 4))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.termWidth-4))
	b.WriteString("\n")

	// Details on bottom
	b.WriteString(m.renderDetailsPane(m.termWidth - 4))

	return b.String()
}

func (m tuiModel) renderSelection() string {
	// If split screen mode is enabled, use split screen layout
	if m.splitScreenMode {
		return m.renderSplitScreen()
	}

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
	b.WriteString(helpStyleTUI.Render("y/Enter: Validate & export | p: Preview | v: Validation only | c: Config | u: Cloud | f: Features | n/Esc: Back | q: Quit"))

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

func (m tuiModel) renderValidation() string {
	var b strings.Builder

	boxWidth := m.getBoxWidth()
	headerWidth := m.getHeaderWidth()

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(lightCharcoal).
		Width(headerWidth).
		Align(lipgloss.Center).
		Render("🔍 Pre-Export Validation")
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.validationReport == nil {
		// Show animated loading spinner
		frames := getSpinnerFrames()
		spinnerChar := frames[m.animFrame%len(frames)]
		b.WriteString(spinnerLoadingStyleTUI.Render(spinnerChar))
		b.WriteString(" ")
		b.WriteString(lipgloss.NewStyle().Foreground(tealInfo).Render("Running validation checks..."))
		return b.String()
	}

	report := m.validationReport

	// Overall status
	var statusBox string
	if report.AllPassed {
		statusBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(successGreen).
			Padding(1, 2).
			Width(boxWidth).
			Render(lipgloss.NewStyle().
				Foreground(successGreen).
				Bold(true).
				Render("✓ All validation checks passed!"))
	} else {
		statusBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(errorColor).
			Padding(1, 2).
			Width(boxWidth).
			Render(lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true).
				Render("⚠ Some validation checks failed"))
	}
	b.WriteString(statusBox)
	b.WriteString("\n\n")

	// Validation checks
	checksBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedGray).
		Padding(1, 2).
		Width(boxWidth)

	var checks strings.Builder
	for _, check := range report.Checks {
		var icon string
		var style lipgloss.Style

		if !check.Passed {
			icon = "✗"
			style = lipgloss.NewStyle().Foreground(errorColor).Bold(true)
		} else if check.Warning {
			icon = "⚠"
			style = lipgloss.NewStyle().Foreground(warningColor)
		} else {
			icon = "✓"
			style = lipgloss.NewStyle().Foreground(successGreen)
		}

		checkLine := fmt.Sprintf("%s %-25s %s",
			style.Render(icon),
			check.Name,
			check.Message)
		checks.WriteString(checkLine)
		checks.WriteString("\n")
	}

	b.WriteString(checksBox.Render(checks.String()))
	b.WriteString("\n\n")

	// Warnings summary
	if report.HasWarnings {
		warningBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warningColor).
			Padding(0, 1)

		b.WriteString(warningBox.Render(
			lipgloss.NewStyle().Foreground(warningColor).Render(
				"⚠ Review warnings above before proceeding")))
		b.WriteString("\n\n")
	}

	// Help text
	if report.AllPassed {
		b.WriteString(helpStyleTUI.Render("y/Enter: Start export | Esc: Go back | q: Quit"))
	} else {
		b.WriteString(helpStyleTUI.Render("o: Override and export anyway | Esc: Go back | q: Quit"))
	}

	if m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(errorStyleTUI.Render(m.message))
	}

	return b.String()
}

// renderConfigPanel renders the interactive configuration panel
func (m tuiModel) renderConfigPanel() string {
	var b strings.Builder

	boxWidth := m.getBoxWidth()
	headerWidth := m.getHeaderWidth()

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(lightCharcoal).
		Width(headerWidth).
		Align(lipgloss.Center).
		Render("⚙️  Configuration Editor")
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.configPanel == nil {
		return errorStyleTUI.Render("Config panel not initialized")
	}

	panel := m.configPanel

	// Instructions box
	instructionsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tealInfo).
		Padding(0, 1).
		Width(boxWidth)

	instructions := lipgloss.NewStyle().Foreground(tealInfo).Render(
		"💡 Use ↑/↓ or Tab/Shift+Tab to navigate | Enter to move to next field | Ctrl+S to save")
	b.WriteString(instructionsBox.Render(instructions))
	b.WriteString("\n\n")

	// Render each config field
	for i, field := range panel.fields {
		isFocused := i == panel.focusedField

		// Field box styling
		var fieldBox lipgloss.Style
		if isFocused {
			fieldBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2).
				Width(boxWidth)
		} else {
			fieldBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(mutedGray).
				Padding(1, 2).
				Width(boxWidth)
		}

		// Field content
		var fieldContent strings.Builder

		// Label
		labelStyle := lipgloss.NewStyle().
			Bold(true)
		if isFocused {
			labelStyle = labelStyle.Foreground(primaryColor)
		} else {
			labelStyle = labelStyle.Foreground(textColor)
		}
		fieldContent.WriteString(labelStyle.Render(field.label))
		fieldContent.WriteString("\n")

		// Description
		descStyle := lipgloss.NewStyle().
			Foreground(mutedGray).
			Italic(true)
		fieldContent.WriteString(descStyle.Render(field.description))
		fieldContent.WriteString("\n\n")

		// Input field
		fieldContent.WriteString(field.inputModel.View())

		b.WriteString(fieldBox.Render(fieldContent.String()))
		b.WriteString("\n")
	}

	// Status message
	if panel.isDirty {
		b.WriteString("\n")
		statusBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warningColor).
			Padding(0, 1)
		b.WriteString(statusBox.Render(
			lipgloss.NewStyle().Foreground(warningColor).Render(
				"⚠ Unsaved changes")))
	}

	b.WriteString("\n\n")

	// Help text
	b.WriteString(helpStyleTUI.Render(
		"Ctrl+S or Enter on last field: Save | Esc: Cancel | q: Quit"))

	return b.String()
}

// renderStats renders the statistics dashboard
func (m tuiModel) renderStats() string {
	var b strings.Builder

	boxWidth := m.getBoxWidth()
	headerWidth := m.getHeaderWidth()

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(lightCharcoal).
		Width(headerWidth).
		Align(lipgloss.Center).
		Render("📊 Statistics Dashboard")
	b.WriteString(header)
	b.WriteString("\n\n")

	// VM Inventory Statistics
	var totalVMs int
	var poweredOn, poweredOff int
	var totalCPUs int32
	var totalMemoryMB int32
	var totalStorageBytes int64

	for _, vm := range m.vms {
		totalVMs++
		totalCPUs += vm.vm.NumCPU
		totalMemoryMB += vm.vm.MemoryMB
		totalStorageBytes += vm.vm.Storage
		if vm.vm.PowerState == "poweredOn" {
			poweredOn++
		} else {
			poweredOff++
		}
	}

	// VM Inventory Box
	inventoryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tealInfo).
		Padding(1, 2).
		Width(boxWidth)

	var inventory strings.Builder
	inventory.WriteString(lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render("🖥️  VM Inventory"))
	inventory.WriteString("\n\n")

	inventory.WriteString(fmt.Sprintf("Total VMs: %s\n",
		lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(fmt.Sprintf("%d", totalVMs))))
	inventory.WriteString(fmt.Sprintf("  ● Powered On: %s\n",
		lipgloss.NewStyle().Foreground(successGreen).Render(fmt.Sprintf("%d", poweredOn))))
	inventory.WriteString(fmt.Sprintf("  ○ Powered Off: %s\n",
		lipgloss.NewStyle().Foreground(mutedGray).Render(fmt.Sprintf("%d", poweredOff))))
	inventory.WriteString("\n")
	inventory.WriteString(fmt.Sprintf("Total vCPUs: %s\n",
		lipgloss.NewStyle().Foreground(textColor).Render(fmt.Sprintf("%d", totalCPUs))))
	inventory.WriteString(fmt.Sprintf("Total Memory: %s\n",
		lipgloss.NewStyle().Foreground(textColor).Render(fmt.Sprintf("%.1f GB", float64(totalMemoryMB)/1024))))
	inventory.WriteString(fmt.Sprintf("Total Storage: %s\n",
		lipgloss.NewStyle().Foreground(textColor).Render(formatBytes(totalStorageBytes))))

	b.WriteString(inventoryBox.Render(inventory.String()))
	b.WriteString("\n\n")

	// Selected VMs Box
	selectedCount := m.countSelected()
	if selectedCount > 0 {
		var selectedCPUs int32
		var selectedMemoryMB int32
		var selectedStorageBytes int64

		for _, vm := range m.vms {
			if vm.selected {
				selectedCPUs += vm.vm.NumCPU
				selectedMemoryMB += vm.vm.MemoryMB
				selectedStorageBytes += vm.vm.Storage
			}
		}

		selectedBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Width(boxWidth)

		var selected strings.Builder
		selected.WriteString(lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render("✓ Selected for Export"))
		selected.WriteString("\n\n")

		selected.WriteString(fmt.Sprintf("Selected VMs: %s\n",
			lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(fmt.Sprintf("%d", selectedCount))))
		selected.WriteString(fmt.Sprintf("Total vCPUs: %s\n",
			lipgloss.NewStyle().Foreground(textColor).Render(fmt.Sprintf("%d", selectedCPUs))))
		selected.WriteString(fmt.Sprintf("Total Memory: %s\n",
			lipgloss.NewStyle().Foreground(textColor).Render(fmt.Sprintf("%.1f GB", float64(selectedMemoryMB)/1024))))
		selected.WriteString(fmt.Sprintf("Total Storage: %s\n",
			lipgloss.NewStyle().Foreground(textColor).Render(formatBytes(selectedStorageBytes))))

		b.WriteString(selectedBox.Render(selected.String()))
		b.WriteString("\n\n")
	}

	// Configuration Summary Box
	configBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(amberYellow).
		Padding(1, 2).
		Width(boxWidth)

	var configSummary strings.Builder
	configSummary.WriteString(lipgloss.NewStyle().
		Foreground(amberYellow).
		Bold(true).
		Render("⚙️  Configuration"))
	configSummary.WriteString("\n\n")

	configSummary.WriteString(fmt.Sprintf("Output Directory: %s\n",
		lipgloss.NewStyle().Foreground(textColor).Render(m.outputDir)))

	if m.enableCloudUpload && m.cloudConfig != nil {
		configSummary.WriteString(fmt.Sprintf("Cloud Upload: %s (%s)\n",
			lipgloss.NewStyle().Foreground(successGreen).Render("Enabled"),
			m.cloudConfig.provider))
	} else {
		configSummary.WriteString(fmt.Sprintf("Cloud Upload: %s\n",
			lipgloss.NewStyle().Foreground(mutedGray).Render("Disabled")))
	}

	enabledFeatures := m.countEnabledFeatures()
	if enabledFeatures > 0 {
		configSummary.WriteString(fmt.Sprintf("Advanced Features: %s\n",
			lipgloss.NewStyle().Foreground(successGreen).Render(fmt.Sprintf("%d enabled", enabledFeatures))))
	} else {
		configSummary.WriteString(fmt.Sprintf("Advanced Features: %s\n",
			lipgloss.NewStyle().Foreground(mutedGray).Render("None")))
	}

	b.WriteString(configBox.Render(configSummary.String()))
	b.WriteString("\n\n")

	// Help text
	b.WriteString(helpStyleTUI.Render("Esc: Back to VM list | q: Quit"))

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
// min returns the minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getResponsiveWidth calculates responsive width based on terminal size
func (m tuiModel) getResponsiveWidth() int {
	if m.termWidth == 0 {
		return 80 // Default width
	}

	// Use 90% of terminal width, with min/max bounds
	width := min(max(m.termWidth-4, 40), 120)
	return width
}

// getBoxWidth returns width for lipgloss boxes based on terminal size
func (m tuiModel) getBoxWidth() int {
	width := m.getResponsiveWidth()
	return width - 4 // Account for padding and borders
}

// getHeaderWidth returns width for headers
func (m tuiModel) getHeaderWidth() int {
	return m.getResponsiveWidth()
}

// getColumnWidth calculates column width for multi-column layouts
func (m tuiModel) getColumnWidth(numColumns int) int {
	totalWidth := m.getResponsiveWidth()
	// Account for spacing between columns
	spacing := (numColumns - 1) * 2
	return (totalWidth - spacing) / numColumns
}

// truncateToWidth truncates a string to fit within a width
func truncateToWidth(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width < 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

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

func (m tuiModel) renderCloudUpload() string {
	var b strings.Builder

	boxWidth := m.getBoxWidth()
	headerWidth := m.getHeaderWidth()

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(lightCharcoal).
		Width(headerWidth).
		Align(lipgloss.Center).
		Render("☁️  Uploading to Cloud")
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.cloudConfig == nil {
		return errorStyleTUI.Render("No cloud configuration found")
	}

	// Cloud provider info
	providerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tealInfo).
		Padding(1, 2).
		Width(boxWidth)

	var providerInfo strings.Builder
	providerInfo.WriteString(lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(fmt.Sprintf("📦 Provider: %s", m.cloudConfig.provider)))
	providerInfo.WriteString("\n")
	providerInfo.WriteString(lipgloss.NewStyle().Foreground(textColor).Render(
		fmt.Sprintf("   Bucket: %s", m.cloudConfig.bucket)))
	if m.cloudConfig.prefix != "" {
		providerInfo.WriteString("\n")
		providerInfo.WriteString(lipgloss.NewStyle().Foreground(textColor).Render(
			fmt.Sprintf("   Prefix: %s", m.cloudConfig.prefix)))
	}

	b.WriteString(providerBox.Render(providerInfo.String()))
	b.WriteString("\n\n")

	// Upload progress
	progressBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		Width(boxWidth)

	var progress strings.Builder

	if m.exportProgress.totalBytes > 0 {
		uploadPercent := float64(m.exportProgress.currentBytes) / float64(m.exportProgress.totalBytes) * 100
		progress.WriteString(lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render(fmt.Sprintf("⬆️  Upload Progress: %.1f%%", uploadPercent)))
		progress.WriteString("\n\n")

		// Progress bar (animated with gradient)
		barWidth := min(boxWidth-10, 60)
		animatedBar := renderAnimatedProgressBar(uploadPercent, barWidth)
		progress.WriteString(animatedBar)
		progress.WriteString(" ")
		progress.WriteString(progressLabelStyleTUI.Render(fmt.Sprintf("%.1f%%", uploadPercent)))
		progress.WriteString("\n\n")

		// Transfer stats
		progress.WriteString(lipgloss.NewStyle().Foreground(textColor).Render(
			fmt.Sprintf("📊 %s / %s",
				formatBytes(m.exportProgress.currentBytes),
				formatBytes(m.exportProgress.totalBytes))))
		progress.WriteString("\n")

		// Upload speed
		if m.exportProgress.speed > 0 {
			progress.WriteString(lipgloss.NewStyle().Foreground(tealInfo).Render(
				fmt.Sprintf("⚡ %.2f MB/s", m.exportProgress.speed)))
			progress.WriteString("\n")

			// ETA
			if m.exportProgress.totalBytes > m.exportProgress.currentBytes {
				remainingBytes := m.exportProgress.totalBytes - m.exportProgress.currentBytes
				remainingMB := float64(remainingBytes) / (1024 * 1024)
				etaSeconds := remainingMB / m.exportProgress.speed
				etaDuration := time.Duration(etaSeconds * float64(time.Second))

				progress.WriteString(lipgloss.NewStyle().Foreground(amberYellow).Render(
					fmt.Sprintf("⏱  ETA: %s", formatDuration(etaDuration))))
			}
		}
	} else {
		// Animated spinner for preparing files
		frames := getDotSpinnerFrames()
		spinnerChar := frames[m.animFrame%len(frames)]
		progress.WriteString(spinnerProcessingStyleTUI.Render(spinnerChar))
		progress.WriteString(" ")
		progress.WriteString(lipgloss.NewStyle().Foreground(amberYellow).Render("Preparing files for upload..."))
	}

	b.WriteString(progressBox.Render(progress.String()))
	b.WriteString("\n\n")

	// Current file
	if m.currentFileName != "" {
		fileBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedGray).
			Padding(0, 1)

		b.WriteString(fileBox.Render(
			lipgloss.NewStyle().Foreground(mutedGray).Render(
				fmt.Sprintf("📄 %s (%d/%d)",
					truncateString(m.currentFileName, 60),
					m.exportProgress.currentFileIdx+1,
					m.exportProgress.totalFiles))))
		b.WriteString("\n\n")
	}

	// Help
	helpView := m.helpModel.ShortHelpView([]key.Binding{m.keys.Quit})
	b.WriteString(lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Render("💡 " + helpView + " | Upload in progress..."))

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
  i         Show VM details
  x         Quick actions menu (power, snapshot, export)
  u         Cloud upload (S3/Azure/GCS/SFTP)
  t         Export templates
  f         Advanced features
  s         Cycle sort
  c         Clear filters

Views:
  v         Toggle split-screen
  ]         Folder tree view
  H         Export history
  L         Live logs viewer
  Q         Queue manager

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

// getSelectedVMs returns a list of selected VMs
func (m tuiModel) getSelectedVMs() []tuiVMItem {
	selectedVMs := make([]tuiVMItem, 0)
	for _, item := range m.vms {
		if item.selected {
			selectedVMs = append(selectedVMs, item)
		}
	}
	return selectedVMs
}

// buildExportQueue creates export queue from selected VMs
func (m tuiModel) buildExportQueue() []queuedExport {
	queue := make([]queuedExport, 0)
	for _, item := range m.vms {
		if item.selected {
			queue = append(queue, queuedExport{
				vm:       item.vm,
				priority: 2, // Normal priority by default
				status:   "pending",
				eta:      0,
			})
		}
	}
	return queue
}

// handleQueueKeys handles keyboard input in queue management phase
func (m tuiModel) handleQueueKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		// Return to selection
		m.phase = "select"
		m.showQueueEditor = false
		return m, nil

	case "up", "k":
		if m.queueCursor > 0 {
			m.queueCursor--
		}

	case "down", "j":
		if m.queueCursor < len(m.exportQueue)-1 {
			m.queueCursor++
		}

	case "K", "shift+up":
		// Move current item up in queue
		if m.queueCursor > 0 {
			m.exportQueue[m.queueCursor], m.exportQueue[m.queueCursor-1] =
				m.exportQueue[m.queueCursor-1], m.exportQueue[m.queueCursor]
			m.queueCursor--
		}

	case "J", "shift+down":
		// Move current item down in queue
		if m.queueCursor < len(m.exportQueue)-1 {
			m.exportQueue[m.queueCursor], m.exportQueue[m.queueCursor+1] =
				m.exportQueue[m.queueCursor+1], m.exportQueue[m.queueCursor]
			m.queueCursor++
		}

	case "p", "P":
		// Cycle priority: normal -> high -> low -> normal
		current := &m.exportQueue[m.queueCursor]
		switch current.priority {
		case 1: // high -> low
			current.priority = 3
		case 2: // normal -> high
			current.priority = 1
		case 3: // low -> normal
			current.priority = 2
		}

	case "enter":
		// Confirm and proceed to export
		m.phase = "confirm"
		m.showQueueEditor = false
		return m, nil
	}

	return m, nil
}

// renderQueue renders the export queue management interface
func (m tuiModel) renderQueue() string {
	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Foreground(tealInfo).
		Background(darkBg).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(deepOrange).
		Width(m.getResponsiveWidth() - 4).
		Render("╔═══ EXPORT QUEUE MANAGER ═══╗  Reorder & Prioritize Exports")

	b.WriteString(header)
	b.WriteString("\n\n")

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Italic(true).
		Render("K/Shift+↑: Move Up  |  J/Shift+↓: Move Down  |  P: Change Priority  |  Enter: Confirm  |  Esc: Cancel")

	b.WriteString(instructions)
	b.WriteString("\n\n")

	// Queue header
	queueHeader := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(fmt.Sprintf("📋 Export Queue (%d VMs)", len(m.exportQueue)))

	b.WriteString(queueHeader)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.getResponsiveWidth()-4))
	b.WriteString("\n\n")

	// Queue items
	for i, item := range m.exportQueue {
		cursor := "  "
		if i == m.queueCursor {
			cursor = "❯ "
		}

		// Priority indicator
		priorityIndicator := ""
		priorityColor := lipgloss.Color("#9CA3AF")
		switch item.priority {
		case 1:
			priorityIndicator = "[HIGH]"
			priorityColor = warmRed
		case 2:
			priorityIndicator = "[NORM]"
			priorityColor = lipgloss.Color("#60A5FA")
		case 3:
			priorityIndicator = "[LOW]"
			priorityColor = lipgloss.Color("#6B7280")
		}

		priority := lipgloss.NewStyle().
			Foreground(priorityColor).
			Bold(true).
			Width(8).
			Render(priorityIndicator)

		// Position indicator
		position := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Width(5).
			Render(fmt.Sprintf("#%d", i+1))

		// VM name
		name := item.vm.Name
		maxNameLen := m.getResponsiveWidth() - 30
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		nameStyle := lipgloss.NewStyle()
		if i == m.queueCursor {
			nameStyle = nameStyle.Foreground(tealInfo).Bold(true)
		}

		vmName := nameStyle.Render(name)

		// VM specs (size indicator)
		specs := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("%.1f GB", float64(item.vm.Storage)/(1024*1024*1024)))

		line := fmt.Sprintf("%s%s %s  %s  %s", cursor, position, priority, vmName, specs)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Summary
	highCount := 0
	normalCount := 0
	lowCount := 0
	for _, item := range m.exportQueue {
		switch item.priority {
		case 1:
			highCount++
		case 2:
			normalCount++
		case 3:
			lowCount++
		}
	}

	summary := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tealInfo).
		Padding(0, 2).
		Render(fmt.Sprintf("Summary: %d High  |  %d Normal  |  %d Low  |  Total: %d VMs",
			highCount, normalCount, lowCount, len(m.exportQueue)))

	b.WriteString(summary)
	b.WriteString("\n\n")

	// Footer help
	footer := lipgloss.NewStyle().
		Foreground(successGreen).
		Bold(true).
		Render("Press Enter to proceed with export in this order")

	b.WriteString(footer)

	return b.String()
}

// loadHistoryEntries loads export history from disk
func (m *tuiModel) loadHistoryEntries() error {
	historyFile, err := GetDefaultHistoryFile()
	if err != nil {
		return fmt.Errorf("get history file: %w", err)
	}

	history := NewExportHistory(historyFile, m.log)
	entries, err := history.GetHistory()
	if err != nil {
		// If file doesn't exist, that's OK - just no history yet
		if os.IsNotExist(err) {
			m.historyEntries = []ExportHistoryEntry{}
			return nil
		}
		return fmt.Errorf("load history: %w", err)
	}

	m.historyEntries = entries
	return nil
}

// getFilteredHistory returns history entries matching current filters
func (m tuiModel) getFilteredHistory() []ExportHistoryEntry {
	filtered := make([]ExportHistoryEntry, 0)
	now := time.Now()

	for _, entry := range m.historyEntries {
		// Apply success/failure filter
		if m.historyFilter == "success" && !entry.Success {
			continue
		}
		if m.historyFilter == "failed" && entry.Success {
			continue
		}

		// Apply date filter
		switch m.historyDateFilter {
		case "today":
			if entry.Timestamp.Before(now.Add(-24 * time.Hour)) {
				continue
			}
		case "week":
			if entry.Timestamp.Before(now.Add(-7 * 24 * time.Hour)) {
				continue
			}
		case "month":
			if entry.Timestamp.Before(now.Add(-30 * 24 * time.Hour)) {
				continue
			}
		}

		// Apply search query
		if m.historySearchQuery != "" {
			query := strings.ToLower(m.historySearchQuery)
			if !strings.Contains(strings.ToLower(entry.VMName), query) &&
				!strings.Contains(strings.ToLower(entry.VMPath), query) {
				continue
			}
		}

		// Apply provider filter
		if m.historyProviderFilter != "all" && entry.Provider != m.historyProviderFilter {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// handleHistoryKeys handles keyboard input in history view
func (m tuiModel) handleHistoryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.getFilteredHistory()

	switch msg.String() {
	case "ctrl+c", "q", "esc":
		// Return to selection
		m.phase = "select"
		return m, nil

	case "up", "k":
		if m.historyCursor > 0 {
			m.historyCursor--
		}

	case "down", "j":
		if m.historyCursor < len(entries)-1 {
			m.historyCursor++
		}

	case "f", "F":
		// Cycle through filters
		switch m.historyFilter {
		case "all":
			m.historyFilter = "success"
		case "success":
			m.historyFilter = "failed"
		case "failed":
			m.historyFilter = "all"
		}
		m.historyCursor = 0

	case "d", "D":
		// Cycle through date filters
		switch m.historyDateFilter {
		case "all":
			m.historyDateFilter = "today"
		case "today":
			m.historyDateFilter = "week"
		case "week":
			m.historyDateFilter = "month"
		case "month":
			m.historyDateFilter = "all"
		}
		m.historyCursor = 0

	case "r", "R":
		// Refresh history
		if err := m.loadHistoryEntries(); err != nil {
			m.message = fmt.Sprintf("Failed to reload history: %v", err)
		}
	}

	return m, nil
}

// renderHistory renders the export history view
func (m tuiModel) renderHistory() string {
	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Foreground(tealInfo).
		Background(darkBg).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(deepOrange).
		Width(m.getResponsiveWidth() - 4).
		Render("╔═══ EXPORT HISTORY ═══╗  View Past Exports")

	b.WriteString(header)
	b.WriteString("\n\n")

	// Active filters indicator
	filterParts := []string{}
	if m.historyFilter != "all" {
		filterParts = append(filterParts, fmt.Sprintf("Status: %s", m.historyFilter))
	}
	if m.historyDateFilter != "all" {
		filterParts = append(filterParts, fmt.Sprintf("Time: %s", m.historyDateFilter))
	}
	if m.historySearchQuery != "" {
		filterParts = append(filterParts, fmt.Sprintf("Search: %s", m.historySearchQuery))
	}

	if len(filterParts) > 0 {
		filters := lipgloss.NewStyle().
			Foreground(amberYellow).
			Italic(true).
			Render("Active Filters: " + strings.Join(filterParts, " | "))
		b.WriteString(filters)
		b.WriteString("\n\n")
	}

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Italic(true).
		Render("F: Filter Status  |  D: Date Range  |  R: Refresh  |  Esc: Back")

	b.WriteString(instructions)
	b.WriteString("\n\n")

	// Get filtered entries
	entries := m.getFilteredHistory()

	if len(entries) == 0 {
		noHistory := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Italic(true).
			Render("No export history found")
		b.WriteString(noHistory)
		return b.String()
	}

	// Summary stats
	totalExports := len(entries)
	successCount := 0
	failedCount := 0
	var totalSize int64
	var totalDuration time.Duration

	for _, entry := range entries {
		if entry.Success {
			successCount++
		} else {
			failedCount++
		}
		totalSize += entry.TotalSize
		totalDuration += entry.Duration
	}

	summary := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(fmt.Sprintf("📊 %d Total | ✓ %d Success | ✗ %d Failed | 💾 %.2f GB | ⏱ %s avg",
			totalExports, successCount, failedCount,
			float64(totalSize)/(1024*1024*1024),
			(totalDuration / time.Duration(totalExports)).Round(time.Second)))

	b.WriteString(summary)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.getResponsiveWidth()-4))
	b.WriteString("\n\n")

	// History entries (show last 15)
	maxVisible := 15
	start := 0
	if len(entries) > maxVisible {
		start = len(entries) - maxVisible
		if m.historyCursor < start {
			start = m.historyCursor
		}
		if m.historyCursor >= start+maxVisible {
			start = m.historyCursor - maxVisible + 1
		}
	}

	end := start + maxVisible
	if end > len(entries) {
		end = len(entries)
	}

	// Reverse order (most recent first)
	for i := end - 1; i >= start; i-- {
		entry := entries[i]

		cursor := "  "
		if i == m.historyCursor {
			cursor = "❯ "
		}

		// Status indicator
		statusIcon := "✓"
		statusColor := successGreen
		if !entry.Success {
			statusIcon = "✗"
			statusColor = warmRed
		}

		status := lipgloss.NewStyle().
			Foreground(statusColor).
			Bold(true).
			Width(3).
			Render(statusIcon)

		// Timestamp
		timestamp := entry.Timestamp.Format("01/02 15:04")
		timeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Width(12)
		timeStr := timeStyle.Render(timestamp)

		// VM name
		name := entry.VMName
		maxNameLen := m.getResponsiveWidth() - 50
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		nameStyle := lipgloss.NewStyle()
		if i == m.historyCursor {
			nameStyle = nameStyle.Foreground(tealInfo).Bold(true)
		}
		vmName := nameStyle.Render(name)

		// Format & size
		details := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render(fmt.Sprintf("%s | %.1f GB | %s",
				entry.Format,
				float64(entry.TotalSize)/(1024*1024*1024),
				entry.Duration.Round(time.Second)))

		line := fmt.Sprintf("%s%s %s  %s  %s", cursor, status, timeStr, vmName, details)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Show details of selected entry if cursor is valid
	if m.historyCursor < len(entries) && m.historyCursor >= 0 {
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", m.getResponsiveWidth()-4))
		b.WriteString("\n")

		selectedEntry := entries[m.historyCursor]
		b.WriteString(m.renderHistoryDetails(selectedEntry))
	}

	return b.String()
}

// renderHistoryDetails renders detailed information about a history entry
func (m tuiModel) renderHistoryDetails(entry ExportHistoryEntry) string {
	var b strings.Builder

	detailsTitle := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render("📝 Export Details")

	b.WriteString(detailsTitle)
	b.WriteString("\n\n")

	details := []struct {
		label string
		value string
		color lipgloss.Color
	}{
		{"VM Name", entry.VMName, tealInfo},
		{"VM Path", entry.VMPath, lipgloss.Color("#6B7280")},
		{"Provider", entry.Provider, lipgloss.Color("#9CA3AF")},
		{"Format", entry.Format, lipgloss.Color("#60A5FA")},
		{"Size", fmt.Sprintf("%.2f GB", float64(entry.TotalSize)/(1024*1024*1024)), lipgloss.Color("#34D399")},
		{"Duration", entry.Duration.Round(time.Second).String(), lipgloss.Color("#F59E0B")},
		{"Files", fmt.Sprintf("%d", entry.FilesCount), lipgloss.Color("#9CA3AF")},
		{"Timestamp", entry.Timestamp.Format("2006-01-02 15:04:05"), lipgloss.Color("#6B7280")},
		{"Output Dir", entry.OutputDir, lipgloss.Color("#6B7280")},
	}

	for _, detail := range details {
		label := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Width(14).
			Render(detail.label + ":")

		value := lipgloss.NewStyle().
			Foreground(detail.color).
			Render(detail.value)

		b.WriteString(label + " " + value)
		b.WriteString("\n")
	}

	// Show error if failed
	if !entry.Success && entry.ErrorMessage != "" {
		b.WriteString("\n")
		errorLabel := lipgloss.NewStyle().
			Foreground(warmRed).
			Bold(true).
			Render("Error:")

		errorMsg := lipgloss.NewStyle().
			Foreground(warmRed).
			Render(" " + entry.ErrorMessage)

		b.WriteString(errorLabel + errorMsg)
	}

	return b.String()
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

func (m tuiModel) runValidation() tea.Cmd {
	return func() tea.Msg {
		// Get selected VMs
		selectedVMs := []vsphere.VMInfo{}
		for _, item := range m.vms {
			if item.selected {
				selectedVMs = append(selectedVMs, item.vm)
			}
		}

		if len(selectedVMs) == 0 {
			return validationCompleteMsg{
				err: fmt.Errorf("no VMs selected"),
			}
		}

		// Calculate total storage requirement
		var totalStorage int64
		for _, vm := range selectedVMs {
			totalStorage += vm.Storage
		}

		// Run pre-export validation
		validator := NewPreExportValidator(m.log)

		// For now, validate against the first VM (we'll aggregate later)
		report := validator.ValidateExport(m.ctx, selectedVMs[0], m.outputDir, totalStorage)

		return validationCompleteMsg{
			report: report,
			err:    nil,
		}
	}
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

// addLogEntry adds a new log entry to the logs viewer
func (m *tuiModel) addLogEntry(level, message, vmName string) {
	entry := logEntry{
		timestamp: time.Now(),
		level:     level,
		message:   message,
		vmName:    vmName,
	}

	m.logEntries = append(m.logEntries, entry)

	// Limit log entries to maxLogEntries
	if len(m.logEntries) > m.maxLogEntries {
		m.logEntries = m.logEntries[len(m.logEntries)-m.maxLogEntries:]
	}

	// Auto-scroll to bottom if enabled
	if m.autoScrollLogs {
		filtered := m.getFilteredLogs()
		if len(filtered) > 0 {
			m.logCursor = len(filtered) - 1
		}
	}
}

// getFilteredLogs returns logs filtered by level and search query
func (m tuiModel) getFilteredLogs() []logEntry {
	filtered := make([]logEntry, 0)

	for _, entry := range m.logEntries {
		// Apply level filter
		if m.logLevelFilter != "all" {
			if strings.ToLower(entry.level) != strings.ToLower(m.logLevelFilter) {
				continue
			}
		}

		// Apply search query
		if m.logSearchQuery != "" {
			query := strings.ToLower(m.logSearchQuery)
			if !strings.Contains(strings.ToLower(entry.message), query) &&
				!strings.Contains(strings.ToLower(entry.vmName), query) {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// handleLogsKeys handles keyboard input in logs view
func (m tuiModel) handleLogsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.getFilteredLogs()

	switch msg.String() {
	case "ctrl+c", "q", "esc":
		// Return to selection or hide logs panel
		if m.phase == "export" {
			m.showLogsPanel = false
		} else {
			m.phase = "select"
		}
		return m, nil

	case "up", "k":
		if m.logCursor > 0 {
			m.logCursor--
			m.autoScrollLogs = false
		}

	case "down", "j":
		if m.logCursor < len(entries)-1 {
			m.logCursor++
		}

	case "g":
		m.logCursor = 0
		m.autoScrollLogs = false

	case "G":
		if len(entries) > 0 {
			m.logCursor = len(entries) - 1
		}
		m.autoScrollLogs = true

	case "l", "L":
		switch m.logLevelFilter {
		case "all":
			m.logLevelFilter = "info"
		case "info":
			m.logLevelFilter = "warn"
		case "warn":
			m.logLevelFilter = "error"
		case "error":
			m.logLevelFilter = "debug"
		case "debug":
			m.logLevelFilter = "all"
		}
		m.logCursor = 0

	case "a", "A":
		m.autoScrollLogs = !m.autoScrollLogs
		if m.autoScrollLogs && len(entries) > 0 {
			m.logCursor = len(entries) - 1
		}

	case "c", "C":
		m.logEntries = []logEntry{}
		m.logCursor = 0
	}

	return m, nil
}

// renderLogs renders the live logs viewer
func (m tuiModel) renderLogs() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(tealInfo).
		Background(darkBg).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(deepOrange).
		Width(m.getResponsiveWidth() - 4).
		Render("╔═══ LIVE LOGS VIEWER ═══╗  Real-time Export Logs")

	b.WriteString(header)
	b.WriteString("\n\n")

	filterParts := []string{}
	if m.logLevelFilter != "all" {
		filterParts = append(filterParts, fmt.Sprintf("Level: %s", strings.ToUpper(m.logLevelFilter)))
	}
	if m.autoScrollLogs {
		filterParts = append(filterParts, "[AUTO-SCROLL ON]")
	} else {
		filterParts = append(filterParts, "[AUTO-SCROLL OFF]")
	}

	if len(filterParts) > 0 {
		filters := lipgloss.NewStyle().
			Foreground(amberYellow).
			Italic(true).
			Render("Active: " + strings.Join(filterParts, " | "))
		b.WriteString(filters)
		b.WriteString("\n\n")
	}

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Italic(true).
		Render("L: Filter Level  |  A: Toggle Auto-Scroll  |  G/g: Jump  |  C: Clear  |  Esc: Back")

	b.WriteString(instructions)
	b.WriteString("\n\n")

	entries := m.getFilteredLogs()

	if len(entries) == 0 {
		noLogs := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Italic(true).
			Render("No log entries found")
		b.WriteString(noLogs)
		return b.String()
	}

	totalLogs := len(entries)
	infoCount := 0
	warnCount := 0
	errorCount := 0
	debugCount := 0

	for _, entry := range entries {
		switch strings.ToUpper(entry.level) {
		case "INFO":
			infoCount++
		case "WARN", "WARNING":
			warnCount++
		case "ERROR":
			errorCount++
		case "DEBUG":
			debugCount++
		}
	}

	summary := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(fmt.Sprintf("📊 %d Total | ℹ %d Info | ⚠ %d Warn | ✗ %d Error | 🐛 %d Debug",
			totalLogs, infoCount, warnCount, errorCount, debugCount))

	b.WriteString(summary)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.getResponsiveWidth()-4))
	b.WriteString("\n\n")

	maxVisible := 20
	start := 0
	if len(entries) > maxVisible {
		start = len(entries) - maxVisible
		if m.logCursor < start {
			start = m.logCursor
		}
		if m.logCursor >= start+maxVisible {
			start = m.logCursor - maxVisible + 1
		}
	}

	end := start + maxVisible
	if end > len(entries) {
		end = len(entries)
	}

	for i := start; i < end; i++ {
		entry := entries[i]

		cursor := "  "
		if i == m.logCursor {
			cursor = "❯ "
		}

		var levelIcon string
		var levelColor lipgloss.Color
		switch strings.ToUpper(entry.level) {
		case "INFO":
			levelIcon = "ℹ"
			levelColor = tealInfo
		case "WARN", "WARNING":
			levelIcon = "⚠"
			levelColor = amberYellow
		case "ERROR":
			levelIcon = "✗"
			levelColor = warmRed
		case "DEBUG":
			levelIcon = "🐛"
			levelColor = mutedGray
		default:
			levelIcon = "·"
			levelColor = offWhite
		}

		levelStr := strings.ToUpper(entry.level)
		if len(levelStr) > 3 {
			levelStr = levelStr[:3]
		}
		level := lipgloss.NewStyle().
			Foreground(levelColor).
			Bold(true).
			Width(5).
			Render(levelIcon + " " + levelStr)

		timestamp := entry.timestamp.Format("15:04:05")
		timeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Width(10)
		timeStr := timeStyle.Render(timestamp)

		vmPart := ""
		if entry.vmName != "" {
			vmStyle := lipgloss.NewStyle().
				Foreground(deepOrange).
				Bold(true)
			vmPart = vmStyle.Render("[" + entry.vmName + "] ")
		}

		message := entry.message
		maxMsgLen := m.getResponsiveWidth() - 60
		if maxMsgLen > 3 && len(message) > maxMsgLen {
			message = message[:maxMsgLen-3] + "..."
		}

		msgStyle := lipgloss.NewStyle()
		if i == m.logCursor {
			msgStyle = msgStyle.Foreground(offWhite).Bold(true)
		}
		msgStr := msgStyle.Render(message)

		line := fmt.Sprintf("%s%s %s %s%s", cursor, level, timeStr, vmPart, msgStr)
		b.WriteString(line)
		b.WriteString("\n")
	}

	if start > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(mutedGray).
			Italic(true).
			Render(fmt.Sprintf("... %d more above (g: top) ...", start)))
	}
	if end < len(entries) {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(mutedGray).
			Italic(true).
			Render(fmt.Sprintf("... %d more below (G: bottom) ...", len(entries)-end)))
	}

	return b.String()
}

// buildFolderTree builds a hierarchical folder tree from VM paths
func (m *tuiModel) buildFolderTree() {
	root := &folderNode{
		name:     "VMs",
		path:     "/",
		children: []*folderNode{},
		vms:      []tuiVMItem{},
		expanded: true,
		level:    0,
	}

	for _, vmItem := range m.vms {
		pathParts := strings.Split(strings.Trim(vmItem.vm.Path, "/"), "/")
		current := root

		// Build folder hierarchy
		for i, part := range pathParts[:len(pathParts)-1] {
			found := false
			for _, child := range current.children {
				if child.name == part {
					current = child
					found = true
					break
				}
			}
			if !found {
				newNode := &folderNode{
					name:     part,
					path:     strings.Join(pathParts[:i+1], "/"),
					parent:   current,
					children: []*folderNode{},
					vms:      []tuiVMItem{},
					expanded: false,
					level:    i + 1,
				}
				current.children = append(current.children, newNode)
				current = newNode
			}
		}
		current.vms = append(current.vms, vmItem)
	}

	m.folderTree = root
	m.flattenTree()
}

// flattenTree converts the tree structure into a flat list for rendering
func (m *tuiModel) flattenTree() {
	m.treeItems = []interface{}{}
	var flatten func(*folderNode)
	flatten = func(node *folderNode) {
		if node != m.folderTree {
			m.treeItems = append(m.treeItems, node)
		}
		if node.expanded {
			for _, child := range node.children {
				flatten(child)
			}
			for _, vm := range node.vms {
				m.treeItems = append(m.treeItems, vm)
			}
		}
	}
	flatten(m.folderTree)
}

// toggleFolderAtCursor toggles expand/collapse of folder at cursor
func (m *tuiModel) toggleFolderAtCursor() {
	if m.treeCursor >= len(m.treeItems) {
		return
	}
	if folder, ok := m.treeItems[m.treeCursor].(*folderNode); ok {
		folder.expanded = !folder.expanded
		m.flattenTree()
	}
}

// renderTree renders the folder tree view
func (m tuiModel) renderTree() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render("📁 FOLDER TREE VIEW  (T: List View | Enter: Expand/Collapse | Space: Select)")
	b.WriteString(header)
	b.WriteString("\n\n")

	if len(m.treeItems) == 0 {
		b.WriteString("No VMs in tree\n")
		return b.String()
	}

	maxVisible := 20
	start := 0
	if m.treeCursor >= maxVisible {
		start = m.treeCursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.treeItems) {
		end = len(m.treeItems)
	}

	for i := start; i < end; i++ {
		cursor := "  "
		if i == m.treeCursor {
			cursor = "❯ "
		}

		indent := ""
		var line string

		switch item := m.treeItems[i].(type) {
		case *folderNode:
			indent = strings.Repeat("  ", item.level)
			icon := "📁"
			if item.expanded {
				icon = "📂"
			}
			folderStyle := lipgloss.NewStyle().Foreground(amberYellow).Bold(true)
			line = fmt.Sprintf("%s%s%s %s (%d VMs)", cursor, indent, icon, folderStyle.Render(item.name), len(item.vms))

		case tuiVMItem:
			level := 2
			if folder, ok := m.treeItems[i-1].(*folderNode); ok {
				level = folder.level + 1
			}
			indent = strings.Repeat("  ", level)
			checkbox := "[ ]"
			if item.selected {
				checkbox = "[✓]"
			}
			vmStyle := lipgloss.NewStyle()
			if i == m.treeCursor {
				vmStyle = vmStyle.Foreground(tealInfo).Bold(true)
			}
			line = fmt.Sprintf("%s%s%s %s", cursor, indent, checkbox, vmStyle.Render(item.vm.Name))
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	statusLine := lipgloss.NewStyle().
		Foreground(mutedGray).
		Render(fmt.Sprintf("\n%d items | Selected: %d", len(m.treeItems), m.countSelected()))
	b.WriteString(statusLine)

	return b.String()
}

// renderPreview renders the export preview screen
func (m tuiModel) renderPreview() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(deepOrange).
		Bold(true).
		Render("📊 EXPORT PREVIEW  (Esc/P: Back | Enter: Continue | ↑↓: Navigate)")
	b.WriteString(header)
	b.WriteString("\n\n")

	if len(m.exportPreviews) == 0 {
		b.WriteString(errorStyleTUI.Render("No VMs selected for export"))
		b.WriteString("\n")
		return b.String()
	}

	// Summary section
	summaryTitle := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render("Export Summary")
	b.WriteString(summaryTitle)
	b.WriteString("\n")

	totalSize := int64(0)
	totalDuration := time.Duration(0)
	totalDisks := 0
	totalFiles := 0
	for _, preview := range m.exportPreviews {
		totalSize += preview.totalSize
		totalDuration += preview.estimatedDuration
		totalDisks += preview.diskCount
		totalFiles += len(preview.files)
	}

	summaryBox := panelStyleTUI.Render(
		fmt.Sprintf(
			"  VMs: %s\n"+
				"  Total Size: %s\n"+
				"  Total Disks: %s\n"+
				"  Total Files: %s\n"+
				"  Estimated Duration: %s",
			statsStyleTUI.Render(fmt.Sprintf("%d", len(m.exportPreviews))),
			statsStyleTUI.Render(formatBytes(totalSize)),
			statsStyleTUI.Render(fmt.Sprintf("%d", totalDisks)),
			statsStyleTUI.Render(fmt.Sprintf("%d", totalFiles)),
			statsStyleTUI.Render(totalDuration.Round(time.Second).String()),
		),
	)
	b.WriteString(summaryBox)
	b.WriteString("\n\n")

	// Individual VM previews
	previewTitle := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render("VM Details")
	b.WriteString(previewTitle)
	b.WriteString("\n\n")

	for i, preview := range m.exportPreviews {
		cursor := "  "
		vmStyle := lipgloss.NewStyle()
		if i == m.previewCursor {
			cursor = "❯ "
			vmStyle = vmStyle.Foreground(deepOrange).Bold(true)
		}

		vmName := vmStyle.Render(preview.vmName)
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, vmName))

		// VM details
		detailStyle := lipgloss.NewStyle().Foreground(mutedGray)
		b.WriteString(detailStyle.Render(fmt.Sprintf("    Size: %s | Disks: %d | Duration: ~%s\n",
			formatBytes(preview.totalSize),
			preview.diskCount,
			preview.estimatedDuration.Round(time.Second),
		)))

		// Disk space check
		if preview.diskSpaceAvail > 0 {
			spaceStatus := "✓"
			spaceStyle := successStyleTUI
			spaceMsg := "Sufficient space"

			if preview.diskSpaceNeeded > preview.diskSpaceAvail {
				spaceStatus = "✗"
				spaceStyle = errorStyleTUI
				spaceMsg = "INSUFFICIENT SPACE"
			} else if preview.diskSpaceAvail-preview.diskSpaceNeeded < preview.diskSpaceAvail/5 {
				spaceStatus = "⚠"
				spaceStyle = lipgloss.NewStyle().Foreground(warningColor).Bold(true)
				spaceMsg = "Low space warning"
			}

			b.WriteString(fmt.Sprintf("    %s Disk Space: %s (Available: %s, Needed: %s)\n",
				spaceStatus,
				spaceStyle.Render(spaceMsg),
				formatBytes(preview.diskSpaceAvail),
				formatBytes(preview.diskSpaceNeeded),
			))
		}

		// File breakdown (only for selected VM)
		if i == m.previewCursor {
			b.WriteString(detailStyle.Render("    Files:\n"))
			for _, file := range preview.files {
				fileTypeIcon := "📄"
				switch file.fileType {
				case "vmdk":
					fileTypeIcon = "💾"
				case "ovf":
					fileTypeIcon = "📋"
				case "mf":
					fileTypeIcon = "📝"
				}
				b.WriteString(detailStyle.Render(fmt.Sprintf("      %s %s (%s)\n",
					fileTypeIcon,
					file.name,
					formatBytes(file.size),
				)))
			}
		}

		b.WriteString("\n")
	}

	// Help footer
	helpText := helpStyleTUI.Render("Press Enter to continue with export, Esc/P to go back")
	b.WriteString("\n")
	b.WriteString(helpText)
	b.WriteString("\n")

	return b.String()
}

// renderActions renders the quick actions menu
func (m tuiModel) renderActions() string {
	var b strings.Builder

	header := lipgloss.NewStyle().
		Foreground(deepOrange).
		Bold(true).
		Render("⚡ QUICK ACTIONS  (Esc/X: Back | Enter: Execute | ↑↓: Navigate)")
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.actionsForVM == nil {
		b.WriteString(errorStyleTUI.Render("No VM selected"))
		return b.String()
	}

	// VM info header
	vmHeader := lipgloss.NewStyle().
		Foreground(tealInfo).
		Bold(true).
		Render(fmt.Sprintf("VM: %s", m.actionsForVM.Name))
	b.WriteString(vmHeader)
	b.WriteString("\n")

	vmDetails := lipgloss.NewStyle().
		Foreground(mutedGray).
		Render(fmt.Sprintf("    Power: %s | CPU: %d | Memory: %d MB | Storage: %s",
			m.actionsForVM.PowerState,
			m.actionsForVM.NumCPU,
			m.actionsForVM.MemoryMB,
			formatBytes(m.actionsForVM.Storage),
		))
	b.WriteString(vmDetails)
	b.WriteString("\n\n")

	// Get available actions for this VM
	actions := m.getAvailableActions(m.actionsForVM)

	// Group actions by category
	categories := map[string][]quickAction{
		"power":    {},
		"snapshot": {},
		"export":   {},
		"info":     {},
	}

	for _, action := range actions {
		categories[action.category] = append(categories[action.category], action)
	}

	// Render actions by category
	categoryOrder := []string{"power", "snapshot", "export", "info"}
	categoryTitles := map[string]string{
		"power":    "⚡ Power Management",
		"snapshot": "📸 Snapshot Operations",
		"export":   "📦 Export Operations",
		"info":     "ℹ️  Information",
	}

	actionIndex := 0
	for _, catKey := range categoryOrder {
		catActions := categories[catKey]
		if len(catActions) == 0 {
			continue
		}

		categoryTitle := lipgloss.NewStyle().
			Foreground(amberYellow).
			Bold(true).
			Render(categoryTitles[catKey])
		b.WriteString(categoryTitle)
		b.WriteString("\n")

		for _, action := range catActions {
			cursor := "  "
			actionStyle := lipgloss.NewStyle()

			if actionIndex == m.actionsCursor {
				cursor = "❯ "
				actionStyle = actionStyle.
					Foreground(deepOrange).
					Bold(true).
					Background(lightBg)
			} else {
				actionStyle = actionStyle.Foreground(offWhite)
			}

			enabled := action.enabled(m.actionsForVM)
			disabledIndicator := ""
			if !enabled {
				actionStyle = actionStyle.Foreground(mutedGray)
				disabledIndicator = " (disabled)"
			}

			line := fmt.Sprintf("%s%s %s%s", cursor, action.icon, action.name, disabledIndicator)
			b.WriteString(actionStyle.Render(line))
			b.WriteString("\n")

			if actionIndex == m.actionsCursor {
				desc := lipgloss.NewStyle().
					Foreground(mutedGray).
					Italic(true).
					Render(fmt.Sprintf("    %s", action.description))
				b.WriteString(desc)
				b.WriteString("\n")
			}

			actionIndex++
		}
		b.WriteString("\n")
	}

	// Help footer
	helpText := helpStyleTUI.Render("Press Enter to execute action, Esc/X to go back")
	b.WriteString("\n")
	b.WriteString(helpText)
	b.WriteString("\n")

	return b.String()
}

// handleActionsKeys handles keyboard input in actions menu
func (m tuiModel) handleActionsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := m.getAvailableActions(m.actionsForVM)
	totalActions := len(actions)

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "x", "X":
		// Return to select phase
		m.phase = "select"
		m.showActionsMenu = false
		m.actionsForVM = nil
		return m, nil

	case "up", "k":
		if m.actionsCursor > 0 {
			m.actionsCursor--
		}

	case "down", "j":
		if m.actionsCursor < totalActions-1 {
			m.actionsCursor++
		}

	case "enter":
		// Execute the selected action
		if m.actionsCursor < len(actions) {
			action := actions[m.actionsCursor]
			if action.enabled(m.actionsForVM) {
				m.phase = "select"
				m.showActionsMenu = false
				return action.handler(&m, m.actionsForVM)
			} else {
				m.message = "This action is not available for the current VM state"
			}
		}
	}

	return m, nil
}

// getAvailableActions returns list of available actions for a VM
func (m tuiModel) getAvailableActions(vm *vsphere.VMInfo) []quickAction {
	if vm == nil {
		return []quickAction{}
	}

	actions := []quickAction{
		// Power actions
		{
			name:        "Power On",
			description: "Power on the virtual machine",
			icon:        "▶️",
			category:    "power",
			enabled: func(v *vsphere.VMInfo) bool {
				return v.PowerState == "poweredOff"
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.message = fmt.Sprintf("Power on operation not yet implemented for %s", v.Name)
				return *m, nil
			},
		},
		{
			name:        "Power Off",
			description: "Power off the virtual machine (graceful shutdown)",
			icon:        "⏹️",
			category:    "power",
			enabled: func(v *vsphere.VMInfo) bool {
				return v.PowerState == "poweredOn"
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.message = fmt.Sprintf("Power off operation not yet implemented for %s", v.Name)
				return *m, nil
			},
		},
		{
			name:        "Reset (Hard Reboot)",
			description: "Force reset the virtual machine (equivalent to power cycle)",
			icon:        "🔄",
			category:    "power",
			enabled: func(v *vsphere.VMInfo) bool {
				return v.PowerState == "poweredOn"
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.message = fmt.Sprintf("Reset operation not yet implemented for %s", v.Name)
				return *m, nil
			},
		},
		{
			name:        "Suspend",
			description: "Suspend the virtual machine to save its current state",
			icon:        "⏸️",
			category:    "power",
			enabled: func(v *vsphere.VMInfo) bool {
				return v.PowerState == "poweredOn"
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.message = fmt.Sprintf("Suspend operation not yet implemented for %s", v.Name)
				return *m, nil
			},
		},

		// Snapshot actions
		{
			name:        "Create Snapshot",
			description: "Create a point-in-time snapshot of the VM",
			icon:        "📸",
			category:    "snapshot",
			enabled: func(v *vsphere.VMInfo) bool {
				return true // Always available
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.message = fmt.Sprintf("Snapshot creation not yet implemented for %s", v.Name)
				return *m, nil
			},
		},
		{
			name:        "Delete All Snapshots",
			description: "Remove all snapshots and reclaim disk space",
			icon:        "🗑️",
			category:    "snapshot",
			enabled: func(v *vsphere.VMInfo) bool {
				return true // Would need to check if snapshots exist
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.message = fmt.Sprintf("Snapshot deletion not yet implemented for %s", v.Name)
				return *m, nil
			},
		},
		{
			name:        "Consolidate Snapshots",
			description: "Consolidate snapshot disks and commit changes",
			icon:        "📊",
			category:    "snapshot",
			enabled: func(v *vsphere.VMInfo) bool {
				return true
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.message = fmt.Sprintf("Snapshot consolidation not yet implemented for %s", v.Name)
				return *m, nil
			},
		},

		// Export actions
		{
			name:        "Quick Export (OVF)",
			description: "Fast export without compression",
			icon:        "📤",
			category:    "export",
			enabled: func(v *vsphere.VMInfo) bool {
				return true
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				// Select this VM and proceed to export
				for i := range m.vms {
					m.vms[i].selected = (m.vms[i].vm.Path == v.Path)
				}
				m.phase = "confirm"
				return *m, nil
			},
		},
		{
			name:        "Production Export (OVA)",
			description: "Compressed OVA with verification",
			icon:        "📦",
			category:    "export",
			enabled: func(v *vsphere.VMInfo) bool {
				return true
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				// Select this VM and proceed to export with template
				for i := range m.vms {
					m.vms[i].selected = (m.vms[i].vm.Path == v.Path)
				}
				m.selectedTemplate = &exportTemplates[1] // Production template
				m.phase = "confirm"
				return *m, nil
			},
		},

		// Info actions
		{
			name:        "Show Detailed Info",
			description: "Display comprehensive VM information",
			icon:        "ℹ️",
			category:    "info",
			enabled: func(v *vsphere.VMInfo) bool {
				return true
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				m.detailsVM = v
				m.phase = "details"
				return *m, nil
			},
		},
		{
			name:        "Run Pre-Export Validation",
			description: "Check if VM is ready for export",
			icon:        "✅",
			category:    "info",
			enabled: func(v *vsphere.VMInfo) bool {
				return true
			},
			handler: func(m *tuiModel, v *vsphere.VMInfo) (tea.Model, tea.Cmd) {
				// Select this VM and show validation
				for i := range m.vms {
					m.vms[i].selected = (m.vms[i].vm.Path == v.Path)
				}
				m.phase = "validation"
				return *m, m.runValidation()
			},
		},
	}

	return actions
}

// handleTreeKeys handles keyboard input in tree view
func (m tuiModel) handleTreeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "]":
		m.viewMode = "list"
		m.phase = "select"
		return m, nil

	case "up", "k":
		if m.treeCursor > 0 {
			m.treeCursor--
		}

	case "down", "j":
		if m.treeCursor < len(m.treeItems)-1 {
			m.treeCursor++
		}

	case "enter":
		m.toggleFolderAtCursor()

	case " ":
		if m.treeCursor < len(m.treeItems) {
			if vmItem, ok := m.treeItems[m.treeCursor].(tuiVMItem); ok {
				for i := range m.vms {
					if m.vms[i].vm.Path == vmItem.vm.Path {
						m.vms[i].selected = !m.vms[i].selected
						m.buildFolderTree()
						break
					}
				}
			} else {
				m.toggleFolderAtCursor()
			}
		}

	case "esc", "b":
		m.viewMode = "list"
		m.phase = "select"
		return m, nil
	}

	return m, nil
}

// handlePreviewKeys handles keyboard input in export preview mode
func (m tuiModel) handlePreviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "escape", "p", "P", "b":
		// Return to confirm phase
		m.phase = "confirm"
		m.showPreview = false
		return m, nil

	case "up", "k":
		if m.previewCursor > 0 {
			m.previewCursor--
		}

	case "down", "j":
		if m.previewCursor < len(m.exportPreviews)-1 {
			m.previewCursor++
		}

	case "enter", "y":
		// Return to confirm and proceed
		m.phase = "confirm"
		m.showPreview = false
		return m, nil
	}

	return m, nil
}

// generateExportPreviews creates export preview data for all selected VMs
func (m tuiModel) generateExportPreviews() []exportPreview {
	previews := make([]exportPreview, 0)
	selectedVMs := m.getSelectedVMs()

	for _, vmItem := range selectedVMs {
		vm := vmItem.vm

		// Calculate estimated values
		// Estimate number of disks based on storage size (most VMs have 1-2 disks)
		estimatedDisks := 1
		if vm.Storage > 500*1024*1024*1024 { // > 500GB probably has multiple disks
			estimatedDisks = 2
		}

		preview := exportPreview{
			vmName:    vm.Name,
			totalSize: vm.Storage,
			diskCount: estimatedDisks,
			fileBreakdown: make(map[string]int64),
			files:     make([]previewFile, 0),
		}

		// Estimate file breakdown
		// VMDKs are typically the largest files
		vmdkSize := vm.Storage
		ovfSize := int64(1024 * 100) // ~100KB for OVF descriptor
		mfSize := int64(1024)         // ~1KB for manifest

		preview.fileBreakdown["vmdk"] = vmdkSize
		preview.fileBreakdown["ovf"] = ovfSize
		preview.fileBreakdown["mf"] = mfSize

		// Generate file list (estimate disk files based on storage)
		diskSize := vm.Storage / int64(estimatedDisks)
		for i := 0; i < estimatedDisks; i++ {
			preview.files = append(preview.files, previewFile{
				name:     fmt.Sprintf("%s-disk%d.vmdk", sanitizeForPath(vm.Name), i+1),
				fileType: "vmdk",
				size:     diskSize,
			})
		}
		preview.files = append(preview.files, previewFile{
			name:     fmt.Sprintf("%s.ovf", sanitizeForPath(vm.Name)),
			fileType: "ovf",
			size:     ovfSize,
		})
		preview.files = append(preview.files, previewFile{
			name:     fmt.Sprintf("%s.mf", sanitizeForPath(vm.Name)),
			fileType: "mf",
			size:     mfSize,
		})

		// Estimate duration (assume 50 MB/s average transfer speed)
		averageSpeedMBps := float64(50)
		totalSizeMB := float64(vm.Storage) / (1024 * 1024)
		estimatedSeconds := totalSizeMB / averageSpeedMBps
		preview.estimatedDuration = time.Duration(estimatedSeconds) * time.Second

		// Check disk space availability
		var stat syscall.Statfs_t
		outputPath := m.outputDir
		if outputPath == "" {
			outputPath = "./exports"
		}

		// Create directory if it doesn't exist
		os.MkdirAll(outputPath, 0755)

		err := syscall.Statfs(outputPath, &stat)
		if err == nil {
			preview.diskSpaceAvail = int64(stat.Bavail) * int64(stat.Bsize)
		}

		// Calculate disk space needed (with 10% overhead)
		preview.diskSpaceNeeded = vm.Storage + (vm.Storage / 10)

		previews = append(previews, preview)
	}

	return previews
}
