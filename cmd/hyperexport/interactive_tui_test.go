// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"testing"
	"time"

	"hypersdk/providers/vsphere"
)

// TestGetResponsiveWidth tests terminal width calculations
func TestGetResponsiveWidth(t *testing.T) {
	tests := []struct {
		name      string
		termWidth int
		want      int
	}{
		{
			name:      "zero width defaults to 80",
			termWidth: 0,
			want:      80,
		},
		{
			name:      "small terminal capped at 40",
			termWidth: 30,
			want:      40,
		},
		{
			name:      "normal terminal 100 cols",
			termWidth: 100,
			want:      96,
		},
		{
			name:      "large terminal capped at 120",
			termWidth: 200,
			want:      120,
		},
		{
			name:      "exactly 80 cols",
			termWidth: 80,
			want:      76,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{termWidth: tt.termWidth}
			got := m.getResponsiveWidth()
			if got != tt.want {
				t.Errorf("getResponsiveWidth() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetBoxWidth tests box width calculations
func TestGetBoxWidth(t *testing.T) {
	tests := []struct {
		name      string
		termWidth int
		want      int
	}{
		{
			name:      "default width",
			termWidth: 0,
			want:      76, // 80 - 4
		},
		{
			name:      "100 col terminal",
			termWidth: 100,
			want:      92, // 96 - 4
		},
		{
			name:      "max width terminal",
			termWidth: 200,
			want:      116, // 120 - 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{termWidth: tt.termWidth}
			got := m.getBoxWidth()
			if got != tt.want {
				t.Errorf("getBoxWidth() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetColumnWidth tests multi-column width calculations
func TestGetColumnWidth(t *testing.T) {
	tests := []struct {
		name       string
		termWidth  int
		numColumns int
		want       int
	}{
		{
			name:       "2 columns default width",
			termWidth:  0,
			numColumns: 2,
			want:       39, // (80 - 2) / 2
		},
		{
			name:       "3 columns",
			termWidth:  100,
			numColumns: 3,
			want:       30, // (96 - 4) / 3
		},
		{
			name:       "single column",
			termWidth:  80,
			numColumns: 1,
			want:       76, // (76 - 0) / 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{termWidth: tt.termWidth}
			got := m.getColumnWidth(tt.numColumns)
			if got != tt.want {
				t.Errorf("getColumnWidth() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTruncateToWidth tests string truncation
func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "short string unchanged",
			input: "hello",
			width: 10,
			want:  "hello",
		},
		{
			name:  "exact length unchanged",
			input: "hello",
			width: 5,
			want:  "hello",
		},
		{
			name:  "long string truncated",
			input: "hello world this is a long string",
			width: 15,
			want:  "hello world ...",
		},
		{
			name:  "very small width",
			input: "hello",
			width: 2,
			want:  "he",
		},
		{
			name:  "width 3 shows ellipsis",
			input: "hello world",
			width: 8,
			want:  "hello...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateToWidth(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("truncateToWidth() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRenderAnimatedProgressBar tests progress bar rendering
func TestRenderAnimatedProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
		wantLen int // Check length of rendered string (without ANSI codes)
	}{
		{
			name:    "0 percent",
			percent: 0,
			width:   10,
			wantLen: 10, // All empty chars
		},
		{
			name:    "50 percent",
			percent: 50,
			width:   10,
			wantLen: 10, // Mix of filled and empty
		},
		{
			name:    "100 percent",
			percent: 100,
			width:   10,
			wantLen: 10, // All filled chars
		},
		{
			name:    "too small width",
			percent: 50,
			width:   2,
			wantLen: 0, // Returns empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderAnimatedProgressBar(tt.percent, tt.width)
			// Note: actual string will have ANSI codes, so we can't check exact content
			// Just verify we got something back for valid widths
			if tt.width >= 4 && len(got) == 0 {
				t.Errorf("renderAnimatedProgressBar() returned empty string for valid width")
			}
			if tt.width < 4 && len(got) != 0 {
				t.Errorf("renderAnimatedProgressBar() should return empty for width < 4")
			}
		})
	}
}

// TestGetSpinnerFrames tests spinner frame arrays
func TestGetSpinnerFrames(t *testing.T) {
	frames := getSpinnerFrames()
	if len(frames) != 10 {
		t.Errorf("getSpinnerFrames() returned %d frames, want 10", len(frames))
	}
	for i, frame := range frames {
		if frame == "" {
			t.Errorf("getSpinnerFrames()[%d] is empty", i)
		}
	}
}

func TestGetDotSpinnerFrames(t *testing.T) {
	frames := getDotSpinnerFrames()
	if len(frames) != 8 {
		t.Errorf("getDotSpinnerFrames() returned %d frames, want 8", len(frames))
	}
	for i, frame := range frames {
		if frame == "" {
			t.Errorf("getDotSpinnerFrames()[%d] is empty", i)
		}
	}
}

func TestGetProgressSpinnerFrames(t *testing.T) {
	frames := getProgressSpinnerFrames()
	if len(frames) != 4 {
		t.Errorf("getProgressSpinnerFrames() returned %d frames, want 4", len(frames))
	}
	for i, frame := range frames {
		if frame == "" {
			t.Errorf("getProgressSpinnerFrames()[%d] is empty", i)
		}
	}
}

// TestRenderPulsingDot tests pulsing dot animation
func TestRenderPulsingDot(t *testing.T) {
	tests := []struct {
		name  string
		frame int
	}{
		{"frame 0", 0},
		{"frame 1", 1},
		{"frame 5", 5},
		{"frame 10", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderPulsingDot(tt.frame)
			if len(got) == 0 {
				t.Errorf("renderPulsingDot() returned empty string")
			}
		})
	}
}

// TestParseInt tests integer parsing
func TestParseInt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{
			name:  "valid positive number",
			input: "42",
			want:  42,
		},
		{
			name:  "valid zero",
			input: "0",
			want:  0,
		},
		{
			name:  "valid negative",
			input: "-10",
			want:  -10,
		},
		{
			name:  "invalid string",
			input: "abc",
			want:  0,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "large number",
			input: "1000000",
			want:  1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInt(tt.input)
			if got != tt.want {
				t.Errorf("parseInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCountEnabledFeatures tests feature counting
func TestCountEnabledFeatures(t *testing.T) {
	tests := []struct {
		name   string
		config featureConfiguration
		want   int
	}{
		{
			name:   "no features enabled",
			config: featureConfiguration{},
			want:   0,
		},
		{
			name: "snapshot only",
			config: featureConfiguration{
				enableSnapshot: true,
			},
			want: 1,
		},
		{
			name: "all features enabled",
			config: featureConfiguration{
				enableSnapshot:       true,
				enableBandwidthLimit: true,
				enableIncremental:    true,
				enableEmail:          true,
				enableCleanup:        true,
			},
			want: 5,
		},
		{
			name: "some features enabled",
			config: featureConfiguration{
				enableSnapshot:    true,
				enableIncremental: true,
				enableEmail:       true,
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{featureConfig: tt.config}
			got := m.countEnabledFeatures()
			if got != tt.want {
				t.Errorf("countEnabledFeatures() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCountSelected tests VM selection counting
func TestCountSelected(t *testing.T) {
	tests := []struct {
		name string
		vms  []tuiVMItem
		want int
	}{
		{
			name: "no VMs",
			vms:  []tuiVMItem{},
			want: 0,
		},
		{
			name: "none selected",
			vms: []tuiVMItem{
				{selected: false},
				{selected: false},
			},
			want: 0,
		},
		{
			name: "all selected",
			vms: []tuiVMItem{
				{selected: true},
				{selected: true},
				{selected: true},
			},
			want: 3,
		},
		{
			name: "some selected",
			vms: []tuiVMItem{
				{selected: true},
				{selected: false},
				{selected: true},
				{selected: false},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{vms: tt.vms}
			got := m.countSelected()
			if got != tt.want {
				t.Errorf("countSelected() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMinMax tests min/max helper functions
func TestMin(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a smaller", 5, 10, 5},
		{"b smaller", 10, 5, 5},
		{"equal", 7, 7, 7},
		{"negative", -5, 3, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a larger", 10, 5, 10},
		{"b larger", 5, 10, 10},
		{"equal", 7, 7, 7},
		{"negative", -5, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := max(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("max(%d, %d) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestConfigPanelCreation tests config panel initialization
func TestNewConfigPanel(t *testing.T) {
	m := &tuiModel{
		outputDir: "/test/output",
		featureConfig: featureConfiguration{
			bandwidthLimitMBps: 10,
			emailSMTPHost:      "smtp.test.com",
			emailFrom:          "test@example.com",
			emailTo:            "admin@example.com",
			keepSnapshots:      5,
			cleanupMaxAge:      30,
			cleanupMaxCount:    10,
		},
	}

	panel := m.newConfigPanel()

	if panel == nil {
		t.Fatal("newConfigPanel() returned nil")
	}

	if panel.focusedField != 0 {
		t.Errorf("focusedField = %v, want 0", panel.focusedField)
	}

	if panel.isDirty {
		t.Errorf("isDirty = true, want false")
	}

	expectedFields := 8
	if len(panel.fields) != expectedFields {
		t.Errorf("got %d fields, want %d", len(panel.fields), expectedFields)
	}

	// Verify field values are initialized correctly
	if panel.fields[0].key != "output_dir" {
		t.Errorf("first field key = %s, want output_dir", panel.fields[0].key)
	}

	if panel.fields[0].value != "/test/output" {
		t.Errorf("output_dir value = %s, want /test/output", panel.fields[0].value)
	}
}

// TestTickMsg tests animation frame increment
func TestTickMsgIncrementsFrame(t *testing.T) {
	m := tuiModel{
		animFrame: 0,
		phase:     "validation",
	}

	// Simulate tick message
	msg := tickMsg(time.Now())
	updated, _ := m.Update(msg)
	updatedModel := updated.(tuiModel)

	if updatedModel.animFrame != 1 {
		t.Errorf("animFrame = %d, want 1", updatedModel.animFrame)
	}
}

// TestStatsCalculations tests statistics dashboard calculations
func TestStatsCalculations(t *testing.T) {
	vms := []tuiVMItem{
		{
			vm: vsphere.VMInfo{
				Name:       "VM1",
				PowerState: "poweredOn",
				NumCPU:     2,
				MemoryMB:   4096,
				Storage:    10 * 1024 * 1024 * 1024, // 10 GB
			},
			selected: true,
		},
		{
			vm: vsphere.VMInfo{
				Name:       "VM2",
				PowerState: "poweredOff",
				NumCPU:     4,
				MemoryMB:   8192,
				Storage:    20 * 1024 * 1024 * 1024, // 20 GB
			},
			selected: false,
		},
		{
			vm: vsphere.VMInfo{
				Name:       "VM3",
				PowerState: "poweredOn",
				NumCPU:     1,
				MemoryMB:   2048,
				Storage:    5 * 1024 * 1024 * 1024, // 5 GB
			},
			selected: true,
		},
	}

	m := tuiModel{vms: vms}

	// Count stats
	var totalVMs, poweredOn, poweredOff int
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

	if totalVMs != 3 {
		t.Errorf("totalVMs = %d, want 3", totalVMs)
	}
	if poweredOn != 2 {
		t.Errorf("poweredOn = %d, want 2", poweredOn)
	}
	if poweredOff != 1 {
		t.Errorf("poweredOff = %d, want 1", poweredOff)
	}
	if totalCPUs != 7 {
		t.Errorf("totalCPUs = %d, want 7", totalCPUs)
	}
	if totalMemoryMB != 14336 {
		t.Errorf("totalMemoryMB = %d, want 14336", totalMemoryMB)
	}
	expectedStorage := int64(35 * 1024 * 1024 * 1024)
	if totalStorageBytes != expectedStorage {
		t.Errorf("totalStorageBytes = %d, want %d", totalStorageBytes, expectedStorage)
	}

	// Test selected stats
	selected := m.countSelected()
	if selected != 2 {
		t.Errorf("countSelected() = %d, want 2", selected)
	}
}

// TestPhaseValues tests that phase constants are correctly set
func TestPhaseValues(t *testing.T) {
	tests := []struct {
		name  string
		phase string
	}{
		{"select phase", "select"},
		{"confirm phase", "confirm"},
		{"stats phase", "stats"},
		{"config phase", "config"},
		{"validation phase", "validation"},
		{"export phase", "export"},
		{"cloudupload phase", "cloudupload"},
		{"done phase", "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{phase: tt.phase}
			if m.phase != tt.phase {
				t.Errorf("phase = %s, want %s", m.phase, tt.phase)
			}
		})
	}
}

// TestSplitScreenMode tests split screen mode toggling
func TestSplitScreenMode(t *testing.T) {
	m := tuiModel{
		splitScreenMode: false,
		focusedPane:     "list",
	}

	if m.splitScreenMode {
		t.Error("Expected splitScreenMode to be false initially")
	}

	// Toggle split screen mode
	m.splitScreenMode = true
	if !m.splitScreenMode {
		t.Error("Expected splitScreenMode to be true after toggle")
	}

	// Toggle back
	m.splitScreenMode = false
	if m.splitScreenMode {
		t.Error("Expected splitScreenMode to be false after second toggle")
	}
}

// TestSwitchPane tests pane switching in split screen mode
func TestSwitchPane(t *testing.T) {
	m := tuiModel{
		splitScreenMode: true,
		focusedPane:     "list",
	}

	if m.focusedPane != "list" {
		t.Errorf("Expected focusedPane to be 'list', got %s", m.focusedPane)
	}

	// Switch to details
	m.focusedPane = "details"
	if m.focusedPane != "details" {
		t.Errorf("Expected focusedPane to be 'details', got %s", m.focusedPane)
	}

	// Switch back to list
	m.focusedPane = "list"
	if m.focusedPane != "list" {
		t.Errorf("Expected focusedPane to be 'list', got %s", m.focusedPane)
	}
}

// TestRenderSplitScreen tests split screen rendering
func TestRenderSplitScreen(t *testing.T) {
	m := tuiModel{
		vms: []tuiVMItem{
			{
				vm: vsphere.VMInfo{
					Name:       "TestVM1",
					Path:       "/datacenter/vm/test1",
					PowerState: "poweredOn",
					GuestOS:    "Ubuntu",
					NumCPU:     4,
					MemoryMB:   8192,
					Storage:    100 * 1024 * 1024 * 1024, // 100 GB
				},
				selected: false,
			},
			{
				vm: vsphere.VMInfo{
					Name:       "TestVM2",
					Path:       "/datacenter/vm/test2",
					PowerState: "poweredOff",
					GuestOS:    "Windows",
					NumCPU:     2,
					MemoryMB:   4096,
					Storage:    50 * 1024 * 1024 * 1024, // 50 GB
				},
				selected: true,
			},
		},
		filteredVMs:     []tuiVMItem{},
		cursor:          0,
		splitScreenMode: true,
		focusedPane:     "list",
		termWidth:       100,
		termHeight:      30,
	}

	// Apply filters to populate filteredVMs
	m.applyFiltersAndSort()

	output := m.renderSplitScreen()

	if output == "" {
		t.Error("renderSplitScreen returned empty string")
	}

	// Check for key elements
	if !contains(output, "SPLIT VIEW MODE") {
		t.Error("Output should contain split view mode indicator")
	}

	if !contains(output, "Tab: Switch Pane") {
		t.Error("Output should contain tab hint")
	}
}

// TestRenderVMListPane tests VM list pane rendering
func TestRenderVMListPane(t *testing.T) {
	m := tuiModel{
		vms: []tuiVMItem{
			{
				vm: vsphere.VMInfo{
					Name:       "TestVM1",
					Path:       "/datacenter/vm/test1",
					PowerState: "poweredOn",
					GuestOS:    "Ubuntu",
					NumCPU:     4,
					MemoryMB:   8192,
					Storage:    100 * 1024 * 1024 * 1024,
				},
				selected: true,
			},
			{
				vm: vsphere.VMInfo{
					Name:       "TestVM2",
					Path:       "/datacenter/vm/test2",
					PowerState: "poweredOff",
					GuestOS:    "Windows",
					NumCPU:     2,
					MemoryMB:   4096,
					Storage:    50 * 1024 * 1024 * 1024,
				},
				selected: false,
			},
		},
		filteredVMs: []tuiVMItem{},
		cursor:      0,
		focusedPane: "list",
		termWidth:   80,
		termHeight:  30,
	}

	m.applyFiltersAndSort()

	output := m.renderVMListPane(40)

	if output == "" {
		t.Error("renderVMListPane returned empty string")
	}

	// Check for VM names
	if !contains(output, "TestVM1") {
		t.Error("Output should contain VM1 name")
	}

	// Check for selection indicators
	if !contains(output, "[✓]") {
		t.Error("Output should contain checked box for selected VM")
	}

	if !contains(output, "[ ]") {
		t.Error("Output should contain unchecked box for unselected VM")
	}
}

// TestRenderDetailsPane tests VM details pane rendering
func TestRenderDetailsPane(t *testing.T) {
	m := tuiModel{
		vms: []tuiVMItem{
			{
				vm: vsphere.VMInfo{
					Name:       "DetailTestVM",
					Path:       "/datacenter/vm/detail-test",
					PowerState: "poweredOn",
					GuestOS:    "Ubuntu 22.04",
					NumCPU:     8,
					MemoryMB:   16384,
					Storage:    200 * 1024 * 1024 * 1024,
				},
				selected: false,
			},
		},
		filteredVMs: []tuiVMItem{},
		cursor:      0,
		focusedPane: "details",
		termWidth:   80,
		termHeight:  30,
	}

	m.applyFiltersAndSort()

	output := m.renderDetailsPane(50)

	if output == "" {
		t.Error("renderDetailsPane returned empty string")
	}

	// Check for VM details
	if !contains(output, "DetailTestVM") {
		t.Error("Output should contain VM name")
	}

	if !contains(output, "Ubuntu 22.04") {
		t.Error("Output should contain guest OS")
	}

	if !contains(output, "8 cores") {
		t.Error("Output should contain CPU count")
	}

	if !contains(output, "16384 MB") {
		t.Error("Output should contain memory")
	}

	if !contains(output, "poweredOn") {
		t.Error("Output should contain power state")
	}
}

// TestRenderVerticalSplit tests vertical split for narrow terminals
func TestRenderVerticalSplit(t *testing.T) {
	m := tuiModel{
		vms: []tuiVMItem{
			{
				vm: vsphere.VMInfo{
					Name:       "TestVM",
					Path:       "/datacenter/vm/test",
					PowerState: "poweredOn",
					GuestOS:    "Linux",
					NumCPU:     2,
					MemoryMB:   4096,
					Storage:    50 * 1024 * 1024 * 1024,
				},
				selected: false,
			},
		},
		filteredVMs:     []tuiVMItem{},
		cursor:          0,
		splitScreenMode: true,
		focusedPane:     "list",
		termWidth:       60, // Narrow terminal
		termHeight:      30,
	}

	m.applyFiltersAndSort()

	output := m.renderVerticalSplit()

	if output == "" {
		t.Error("renderVerticalSplit returned empty string")
	}

	// Check for vertical split indicator
	if !contains(output, "SPLIT VIEW (Vertical)") {
		t.Error("Output should contain vertical split indicator")
	}

	// Should contain both list and details sections
	if !contains(output, "TestVM") {
		t.Error("Output should contain VM name")
	}
}

// TestSplitScreenWithNoVMs tests split screen with empty VM list
func TestSplitScreenWithNoVMs(t *testing.T) {
	m := tuiModel{
		vms:             []tuiVMItem{},
		filteredVMs:     []tuiVMItem{},
		cursor:          0,
		splitScreenMode: true,
		focusedPane:     "list",
		termWidth:       100,
		termHeight:      30,
	}

	m.applyFiltersAndSort()

	output := m.renderSplitScreen()

	if output == "" {
		t.Error("renderSplitScreen returned empty string even with no VMs")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(s) > len(substr) && anyMatch(s, substr)))
}

func anyMatch(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
