// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
)

// Enhanced styles with theme support
var (
	// Color palette
	primaryColor   = lipgloss.Color("#00ffff")   // Cyan
	secondaryColor = lipgloss.Color("#ff00ff")   // Magenta
	successColor   = lipgloss.Color("#00ff00")   // Green
	warningColor   = lipgloss.Color("#ffaa00")   // Orange
	errorColor     = lipgloss.Color("#ff0000")   // Red
	mutedColor     = lipgloss.Color("#666666")   // Gray
	highlightColor = lipgloss.Color("#ffff00")   // Yellow
	bgColor        = lipgloss.Color("#1a1a1a")   // Dark background
	fgColor        = lipgloss.Color("#e0e0e0")   // Light foreground

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(bgColor).
			Bold(true).
			Padding(0, 1)

	// Progress bar styles
	progressBarStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Background(mutedColor)

	progressLabelStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Margin(1, 0)

	statsStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// Badge styles
	badgeStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(bgColor).
			Padding(0, 1).
			Bold(true)

	badgeSuccessStyle = lipgloss.NewStyle().
				Background(successColor).
				Foreground(bgColor).
				Padding(0, 1).
				Bold(true)

	badgeWarningStyle = lipgloss.NewStyle().
				Background(warningColor).
				Foreground(bgColor).
				Padding(0, 1).
				Bold(true)

	// Text styles
	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	// Keyboard shortcut styles
	keyStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Bold(true)

	keyDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)

// renderStatusBar creates an enhanced status bar
func renderStatusBar(m model) string {
	parts := []string{}

	// Connection status
	connIcon := "🔗"
	connText := "Connected"
	if os.Getenv("GOVC_URL") != "" {
		connText = "Direct"
	} else {
		connText = "Daemon"
	}
	parts = append(parts, fmt.Sprintf("%s %s", connIcon, connText))

	// Selected count
	selectedCount := m.countSelected()
	totalCount := len(m.vms)
	visibleCount := len(m.getVisibleVMs())

	selectionText := fmt.Sprintf("📊 %d/%d VMs", visibleCount, totalCount)
	if selectedCount > 0 {
		selectionText += fmt.Sprintf(" | ✅ %d selected", selectedCount)
	}
	parts = append(parts, selectionText)

	// Active filters
	filters := []string{}
	if m.searchQuery != "" {
		filters = append(filters, fmt.Sprintf("🔍 %s", m.searchQuery))
	}
	if m.filterPower != "" {
		filters = append(filters, fmt.Sprintf("⚡ %s", m.filterPower))
	}
	if m.sortMode != "name" {
		filters = append(filters, fmt.Sprintf("📑 %s", m.sortMode))
	}
	if len(filters) > 0 {
		parts = append(parts, strings.Join(filters, " | "))
	}

	// Dry-run indicator
	if m.dryRun {
		parts = append(parts, badgeWarningStyle.Render("DRY-RUN"))
	}

	// Join all parts
	return statusBarStyle.Render(strings.Join(parts, "  •  "))
}

// renderProgressBar creates a visual progress bar
func renderProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(float64(width) * percentage)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	label := fmt.Sprintf("%d/%d (%.0f%%)", current, total, percentage*100)

	return progressLabelStyle.Render(label) + " " + progressBarStyle.Render(bar)
}

// renderStatsPanel creates a statistics panel
func renderStatsPanel(m model) string {
	var totalCPUs int32
	var totalMemoryMB int32
	var totalStorage int64
	var poweredOn, poweredOff int

	for _, item := range m.getVisibleVMs() {
		totalCPUs += item.vm.NumCPU
		totalMemoryMB += item.vm.MemoryMB
		totalStorage += item.vm.Storage

		if item.vm.PowerState == "poweredOn" {
			poweredOn++
		} else {
			poweredOff++
		}
	}

	stats := []string{
		fmt.Sprintf("💻 VMs: %d", len(m.getVisibleVMs())),
		fmt.Sprintf("🟢 ON: %d", poweredOn),
		fmt.Sprintf("🔴 OFF: %d", poweredOff),
		fmt.Sprintf("⚡ CPUs: %d", totalCPUs),
		fmt.Sprintf("💾 RAM: %.1f GB", float64(totalMemoryMB)/1024),
		fmt.Sprintf("💿 Storage: %s", formatBytes(totalStorage)),
	}

	return statsStyle.Render(strings.Join(stats, "  |  "))
}

// renderHelpPanel creates a help panel with keyboard shortcuts
func renderHelpPanel() string {
	shortcuts := [][]string{
		{"Navigation", ""},
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"Space", "Select/deselect"},
		{"Enter", "Continue"},
		{"", ""},
		{"Selection", ""},
		{"a", "Select all (visible)"},
		{"n", "Deselect all"},
		{"A", "Select by pattern"},
		{"", ""},
		{"Filters", ""},
		{"/", "Search"},
		{"s", "Cycle sort"},
		{"f", "Toggle power filter"},
		{"c", "Clear all filters"},
		{"", ""},
		{"View", ""},
		{"d/i", "VM details"},
		{"h/?", "Toggle help"},
		{"r", "Toggle dry-run"},
		{"", ""},
		{"Actions", ""},
		{"q", "Quit"},
		{"Esc", "Go back"},
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("⌨️  Keyboard Shortcuts"))
	b.WriteString("\n\n")

	for _, shortcut := range shortcuts {
		key := shortcut[0]
		desc := shortcut[1]

		if key == "" {
			b.WriteString("\n")
			continue
		}

		if desc == "" {
			// Section header
			b.WriteString(selectedStyle.Render(key))
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				keyStyle.Render(fmt.Sprintf("%-8s", key)),
				keyDescStyle.Render(desc)))
		}
	}

	return panelStyle.Render(b.String())
}

// renderExportPreview shows estimated export details
func renderExportPreview(m model) string {
	var b strings.Builder

	selectedVMs := []vmItem{}
	for _, item := range m.vms {
		if item.selected {
			selectedVMs = append(selectedVMs, item)
		}
	}

	if len(selectedVMs) == 0 {
		return ""
	}

	b.WriteString(titleStyle.Render("📋 Export Preview"))
	b.WriteString("\n\n")

	var totalStorage int64
	var totalCPUs int32
	var totalMemory int32

	for _, item := range selectedVMs {
		totalStorage += item.vm.Storage
		totalCPUs += item.vm.NumCPU
		totalMemory += item.vm.MemoryMB
	}

	// Estimate export time (rough calculation: 100MB/s throughput)
	estimatedSeconds := totalStorage / (100 * 1024 * 1024)

	// Check disk space
	diskSpace := getDiskSpace(m.outputDir)
	hasSpace := diskSpace > totalStorage

	stats := []string{
		fmt.Sprintf("VMs to export: %d", len(selectedVMs)),
		fmt.Sprintf("Total size: %s", formatBytes(totalStorage)),
		fmt.Sprintf("Estimated time: %s", formatDuration(int(estimatedSeconds))),
		fmt.Sprintf("Total CPUs: %d", totalCPUs),
		fmt.Sprintf("Total RAM: %.1f GB", float64(totalMemory)/1024),
	}

	for _, stat := range stats {
		b.WriteString(infoStyle.Render("  • " + stat))
		b.WriteString("\n")
	}

	// Disk space warning
	b.WriteString("\n")
	if hasSpace {
		b.WriteString(successStyle.Render(fmt.Sprintf("✓ Sufficient disk space: %s available", formatBytes(diskSpace))))
	} else {
		b.WriteString(errorStyle.Render(fmt.Sprintf("⚠ WARNING: Insufficient disk space!")))
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Required: %s | Available: %s", formatBytes(totalStorage), formatBytes(diskSpace))))
	}

	return panelStyle.Render(b.String())
}

// getDiskSpace returns available disk space for a path
func getDiskSpace(path string) int64 {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0
	}

	// Available space = available blocks * block size
	return int64(stat.Bavail) * int64(stat.Bsize)
}

// formatDuration formats seconds into human-readable duration
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// renderQuickFilters shows quick filter options
func renderQuickFilters() string {
	filters := []struct {
		key  string
		desc string
	}{
		{"1", "Powered ON VMs"},
		{"2", "Powered OFF VMs"},
		{"3", "Linux VMs"},
		{"4", "Windows VMs"},
		{"5", "High CPU (8+)"},
		{"6", "High Memory (16GB+)"},
		{"7", "Large Storage (500GB+)"},
		{"0", "Clear filters"},
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("🚀 Quick Filters"))
	b.WriteString("\n\n")

	for _, f := range filters {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			keyStyle.Render(f.key),
			keyDescStyle.Render(f.desc)))
	}

	return panelStyle.Render(b.String())
}

// renderBulkSelectionPanel shows bulk selection options
func renderBulkSelectionPanel(m model) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🎯 Bulk Selection"))
	b.WriteString("\n\n")

	b.WriteString(infoStyle.Render("Enter pattern to match VM names:"))
	b.WriteString("\n\n")

	// Show pattern input
	pattern := m.searchQuery
	b.WriteString(selectedStyle.Render(fmt.Sprintf("Pattern: %s█", pattern)))
	b.WriteString("\n\n")

	// Preview matches
	if pattern != "" {
		matches := 0
		for _, item := range m.vms {
			if strings.Contains(strings.ToLower(item.vm.Name), strings.ToLower(pattern)) {
				matches++
			}
		}

		if matches > 0 {
			b.WriteString(successStyle.Render(fmt.Sprintf("✓ Will select %d VMs matching pattern", matches)))
		} else {
			b.WriteString(warningStyle.Render("No VMs match this pattern"))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter: Select all matching | Esc: Cancel | Backspace: Delete char"))

	return panelStyle.Render(b.String())
}

// renderExportProgress shows real-time export progress
func renderExportProgress(m model) string {
	var b strings.Builder

	selectedCount := m.countSelected()
	current := m.currentExport + 1

	// Title
	b.WriteString(titleStyle.Render("📦 Exporting VMs"))
	b.WriteString("\n\n")

	// Overall progress bar
	b.WriteString(renderProgressBar(m.currentExport, selectedCount, 50))
	b.WriteString("\n\n")

	// VM list with status
	exportIndex := 0
	for _, item := range m.vms {
		if item.selected {
			status := ""
			icon := ""

			if exportIndex < m.currentExport {
				status = "✅ Completed"
				icon = "✓"
			} else if exportIndex == m.currentExport {
				status = "⏳ Exporting..."
				icon = "▶"
			} else {
				status = "⏸  Pending"
				icon = "·"
			}

			vmLine := fmt.Sprintf("%s %s - %s",
				icon,
				truncate(item.vm.Name, 40),
				status)

			if exportIndex == m.currentExport {
				b.WriteString(selectedStyle.Bold(true).Render(vmLine))
			} else if exportIndex < m.currentExport {
				b.WriteString(successStyle.Render(vmLine))
			} else {
				b.WriteString(unselectedStyle.Render(vmLine))
			}
			b.WriteString("\n")

			exportIndex++
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("Exporting VM %d of %d...", current, selectedCount)))

	return panelStyle.Render(b.String())
}

// renderCompactVMList renders VMs in a more compact, multi-column format
func renderCompactVMList(vms []vmItem, cursor int) string {
	var b strings.Builder

	for i, item := range vms {
		isCursor := i == cursor

		// Compact format: checkbox + name + key stats
		checkbox := "[ ]"
		if item.selected {
			checkbox = "[✓]"
		}

		cursor := "  "
		if isCursor {
			cursor = "▶ "
		}

		// Colored power indicator
		powerIcon := "🔴"
		if item.vm.PowerState == "poweredOn" {
			powerIcon = "🟢"
		}

		// Compact VM info
		vmInfo := fmt.Sprintf("%-35s %s %2dC %4.0fG %8s",
			truncate(item.vm.Name, 35),
			powerIcon,
			item.vm.NumCPU,
			float64(item.vm.MemoryMB)/1024,
			formatBytesCompact(item.vm.Storage))

		// Style
		style := unselectedStyle
		if item.selected {
			style = selectedStyle
		}

		line := cursor + checkbox + " " + vmInfo
		if isCursor {
			line = style.Bold(true).Underline(true).Render(line)
		} else {
			line = style.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// formatBytesCompact formats bytes in compact format
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

// renderVMCard renders a detailed VM card
func renderVMCard(item vmItem) string {
	var b strings.Builder

	// VM name header
	b.WriteString(titleStyle.Render(fmt.Sprintf("📦 %s", item.vm.Name)))
	b.WriteString("\n\n")

	// Power state with icon
	powerStatus := "🔴 Powered OFF"
	if item.vm.PowerState == "poweredOn" {
		powerStatus = "🟢 Powered ON"
	}

	// Info grid
	info := [][]string{
		{"Power:", powerStatus},
		{"Path:", item.vm.Path},
		{"OS:", item.vm.GuestOS},
		{"", ""},
		{"CPUs:", fmt.Sprintf("%d cores", item.vm.NumCPU)},
		{"Memory:", fmt.Sprintf("%.1f GB (%d MB)", float64(item.vm.MemoryMB)/1024, item.vm.MemoryMB)},
		{"Storage:", fmt.Sprintf("%s (%d bytes)", formatBytes(item.vm.Storage), item.vm.Storage)},
	}

	for _, row := range info {
		if row[0] == "" {
			b.WriteString("\n")
			continue
		}

		b.WriteString(fmt.Sprintf("  %-10s %s\n",
			statsStyle.Render(row[0]),
			infoStyle.Render(row[1])))
	}

	// Selection status
	b.WriteString("\n")
	if item.selected {
		b.WriteString(badgeSuccessStyle.Render(" ✓ SELECTED FOR MIGRATION "))
	} else {
		b.WriteString(unselectedStyle.Render("  Not selected"))
	}

	return panelStyle.Render(b.String())
}
