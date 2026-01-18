// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"testing"

	"hypersdk/providers/vsphere"
)

// TestSanitizeFilename tests the filename sanitization function for security
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid_simple_filename",
			input:    "my-file.txt",
			expected: "my-file.txt",
		},
		{
			name:     "path_traversal_dot_dot_slash",
			input:    "../../../etc/passwd",
			expected: "etc-passwd", // Leading dashes trimmed
		},
		{
			name:     "path_traversal_backslash",
			input:    "..\\..\\windows\\system32",
			expected: "windows-system32", // Leading dashes trimmed
		},
		{
			name:     "forward_slash_separator",
			input:    "folder/file/name.txt",
			expected: "folder-file-name.txt",
		},
		{
			name:     "backslash_separator",
			input:    "folder\\file\\name.txt",
			expected: "folder-file-name.txt",
		},
		{
			name:     "null_byte_injection",
			input:    "file\x00malicious.txt",
			expected: "filemalicious.txt",
		},
		{
			name:     "special_characters",
			input:    "file:name*with?special<chars>|test.txt",
			expected: "file-name-with-special-chars--test.txt",
		},
		{
			name:     "spaces_in_filename",
			input:    "my file name.txt",
			expected: "my-file-name.txt",
		},
		{
			name:     "leading_dots_and_dashes",
			input:    "...---file.txt---...",
			expected: "file.txt",
		},
		{
			name:     "empty_after_sanitization",
			input:    "...---...",
			expected: "unnamed-vm",
		},
		{
			name:     "quotes_removed",
			input:    "my\"file\"name.txt",
			expected: "my-file-name.txt",
		},
		{
			name:     "mixed_dangerous_chars",
			input:    "../folder/file:name*test.ovf",
			expected: "folder-file-name-test.ovf", // Leading dashes trimmed
		},
		{
			name:     "windows_drive_letter",
			input:    "C:\\Windows\\System32\\file.txt",
			expected: "C--Windows-System32-file.txt",
		},
		{
			name:     "relative_path_current_dir",
			input:    "./file.txt",
			expected: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Verify result doesn't contain dangerous characters
			if result != "" && result != "unnamed-vm" {
				validateSanitizedFilename(t, result)
			}
		})
	}
}

// TestSanitizeFilenameLength tests that very long filenames are truncated
func TestSanitizeFilenameLength(t *testing.T) {
	// Create a 300-character filename
	longName := ""
	for i := 0; i < 300; i++ {
		longName += "a"
	}

	result := sanitizeFilename(longName)
	if len(result) > 255 {
		t.Errorf("sanitizeFilename() produced name longer than 255 chars: %d", len(result))
	}
	if len(result) != 255 {
		t.Errorf("sanitizeFilename() should truncate to exactly 255 chars, got %d", len(result))
	}
}

// validateSanitizedFilename checks that a sanitized filename doesn't contain dangerous characters
func validateSanitizedFilename(t *testing.T, s string) {
	t.Helper()

	dangerousChars := []string{"/", "\\", "..", "\x00", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range dangerousChars {
		if containsSubstring(s, char) {
			t.Errorf("Sanitized filename contains dangerous character %q: %q", char, s)
		}
	}

	// Ensure it doesn't start or end with dots or dashes
	if len(s) > 0 && (s[0] == '.' || s[0] == '-') {
		t.Errorf("Sanitized filename should not start with '.' or '-': %q", s)
	}
	if len(s) > 0 && (s[len(s)-1] == '.' || s[len(s)-1] == '-') {
		t.Errorf("Sanitized filename should not end with '.' or '-': %q", s)
	}
}

// containsSubstring checks if string contains a substring (helper for testing)
func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestCountSelected tests the countSelected helper method
func TestCountSelected(t *testing.T) {
	tests := []struct {
		name     string
		vms      []vmItem
		expected int
	}{
		{
			name:     "no_vms",
			vms:      []vmItem{},
			expected: 0,
		},
		{
			name: "none_selected",
			vms: []vmItem{
				{vm: vsphere.VMInfo{Name: "vm1"}, selected: false},
				{vm: vsphere.VMInfo{Name: "vm2"}, selected: false},
				{vm: vsphere.VMInfo{Name: "vm3"}, selected: false},
			},
			expected: 0,
		},
		{
			name: "all_selected",
			vms: []vmItem{
				{vm: vsphere.VMInfo{Name: "vm1"}, selected: true},
				{vm: vsphere.VMInfo{Name: "vm2"}, selected: true},
				{vm: vsphere.VMInfo{Name: "vm3"}, selected: true},
			},
			expected: 3,
		},
		{
			name: "some_selected",
			vms: []vmItem{
				{vm: vsphere.VMInfo{Name: "vm1"}, selected: true},
				{vm: vsphere.VMInfo{Name: "vm2"}, selected: false},
				{vm: vsphere.VMInfo{Name: "vm3"}, selected: true},
				{vm: vsphere.VMInfo{Name: "vm4"}, selected: false},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{vms: tt.vms}
			result := m.countSelected()
			if result != tt.expected {
				t.Errorf("countSelected() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestCycleSortMode tests the sort mode cycling
func TestCycleSortMode(t *testing.T) {
	tests := []struct {
		current  string
		expected string
	}{
		{"name", "cpu"},
		{"cpu", "memory"},
		{"memory", "storage"},
		{"storage", "power"},
		{"power", "name"},
		{"unknown", "name"}, // Default to name
	}

	for _, tt := range tests {
		t.Run("from_"+tt.current, func(t *testing.T) {
			m := model{sortMode: tt.current}
			m.cycleSortMode()
			if m.sortMode != tt.expected {
				t.Errorf("cycleSortMode() from %s = %s, want %s", tt.current, m.sortMode, tt.expected)
			}
		})
	}
}

// TestTogglePowerFilter tests the power filter toggle
func TestTogglePowerFilter(t *testing.T) {
	tests := []struct {
		current  string
		expected string
	}{
		{"", "on"},
		{"on", "off"},
		{"off", ""},
		{"unknown", ""}, // Default to empty
	}

	for _, tt := range tests {
		t.Run("from_"+tt.current, func(t *testing.T) {
			m := model{filterPower: tt.current}
			m.togglePowerFilter()
			if m.filterPower != tt.expected {
				t.Errorf("togglePowerFilter() from %s = %s, want %s", tt.current, m.filterPower, tt.expected)
			}
		})
	}
}

// TestGetVisibleVMs tests the visible VMs getter
func TestGetVisibleVMs(t *testing.T) {
	allVMs := []vmItem{
		{vm: vsphere.VMInfo{Name: "vm1"}},
		{vm: vsphere.VMInfo{Name: "vm2"}},
		{vm: vsphere.VMInfo{Name: "vm3"}},
	}

	filteredVMs := []vmItem{
		{vm: vsphere.VMInfo{Name: "vm1"}},
		{vm: vsphere.VMInfo{Name: "vm3"}},
	}

	tests := []struct {
		name         string
		vms          []vmItem
		filteredVMs  []vmItem
		expectedLen  int
		expectedName string // Name of first VM
	}{
		{
			name:         "no_filter_returns_all",
			vms:          allVMs,
			filteredVMs:  nil,
			expectedLen:  3,
			expectedName: "vm1",
		},
		{
			name:         "with_filter_returns_filtered",
			vms:          allVMs,
			filteredVMs:  filteredVMs,
			expectedLen:  2,
			expectedName: "vm1",
		},
		{
			name:         "empty_vms",
			vms:          []vmItem{},
			filteredVMs:  nil,
			expectedLen:  0,
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{vms: tt.vms, filteredVMs: tt.filteredVMs}
			result := m.getVisibleVMs()

			if len(result) != tt.expectedLen {
				t.Errorf("getVisibleVMs() len = %d, want %d", len(result), tt.expectedLen)
			}

			if tt.expectedLen > 0 && result[0].vm.Name != tt.expectedName {
				t.Errorf("getVisibleVMs()[0].Name = %s, want %s", result[0].vm.Name, tt.expectedName)
			}
		})
	}
}

// TestApplyFiltersAndSort tests the filtering and sorting logic
func TestApplyFiltersAndSort(t *testing.T) {
	vms := []vmItem{
		{vm: vsphere.VMInfo{Name: "zebra", Path: "/dc/vm/zebra", PowerState: "poweredOn", GuestOS: "ubuntu", NumCPU: 4, MemoryMB: 8192, Storage: 107374182400}},
		{vm: vsphere.VMInfo{Name: "alpha", Path: "/dc/vm/alpha", PowerState: "poweredOff", GuestOS: "windows", NumCPU: 2, MemoryMB: 4096, Storage: 53687091200}},
		{vm: vsphere.VMInfo{Name: "beta", Path: "/dc/vm/beta", PowerState: "poweredOn", GuestOS: "centos", NumCPU: 8, MemoryMB: 16384, Storage: 214748364800}},
		{vm: vsphere.VMInfo{Name: "gamma", Path: "/dc/vm/gamma", PowerState: "poweredOff", GuestOS: "ubuntu", NumCPU: 2, MemoryMB: 2048, Storage: 26843545600}},
	}

	tests := []struct {
		name           string
		searchQuery    string
		filterPower    string
		sortMode       string
		expectedCount  int
		expectedFirst  string // Name of first VM after filtering/sorting
		expectedLast   string // Name of last VM
	}{
		{
			name:          "no_filters_sort_by_name",
			searchQuery:   "",
			filterPower:   "",
			sortMode:      "name",
			expectedCount: 4,
			expectedFirst: "alpha",
			expectedLast:  "zebra",
		},
		{
			name:          "search_ubuntu",
			searchQuery:   "ubuntu",
			filterPower:   "",
			sortMode:      "name",
			expectedCount: 2,
			expectedFirst: "gamma",
			expectedLast:  "zebra",
		},
		{
			name:          "filter_powered_on",
			searchQuery:   "",
			filterPower:   "on",
			sortMode:      "name",
			expectedCount: 2,
			expectedFirst: "beta",
			expectedLast:  "zebra",
		},
		{
			name:          "filter_powered_off",
			searchQuery:   "",
			filterPower:   "off",
			sortMode:      "name",
			expectedCount: 2,
			expectedFirst: "alpha",
			expectedLast:  "gamma",
		},
		{
			name:          "sort_by_cpu",
			searchQuery:   "",
			filterPower:   "",
			sortMode:      "cpu",
			expectedCount: 4,
			expectedFirst: "beta", // 8 CPUs
			expectedLast:  "",     // Don't check last - alpha and gamma both have 2 CPUs
		},
		{
			name:          "sort_by_memory",
			searchQuery:   "",
			filterPower:   "",
			sortMode:      "memory",
			expectedCount: 4,
			expectedFirst: "beta", // 16384 MB
			expectedLast:  "gamma", // 2048 MB
		},
		{
			name:          "sort_by_storage",
			searchQuery:   "",
			filterPower:   "",
			sortMode:      "storage",
			expectedCount: 4,
			expectedFirst: "beta", // Largest
			expectedLast:  "gamma", // Smallest
		},
		{
			name:          "sort_by_power",
			searchQuery:   "",
			filterPower:   "",
			sortMode:      "power",
			expectedCount: 4,
			expectedFirst: "beta", // poweredOn
			expectedLast:  "gamma", // poweredOff
		},
		{
			name:          "combined_search_and_filter",
			searchQuery:   "ubuntu",
			filterPower:   "on",
			sortMode:      "name",
			expectedCount: 1,
			expectedFirst: "zebra",
			expectedLast:  "zebra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				vms:         vms,
				searchQuery: tt.searchQuery,
				filterPower: tt.filterPower,
				sortMode:    tt.sortMode,
			}

			m.applyFiltersAndSort()

			if len(m.filteredVMs) != tt.expectedCount {
				t.Errorf("applyFiltersAndSort() count = %d, want %d", len(m.filteredVMs), tt.expectedCount)
			}

			if tt.expectedCount > 0 {
				if m.filteredVMs[0].vm.Name != tt.expectedFirst {
					t.Errorf("applyFiltersAndSort() first = %s, want %s", m.filteredVMs[0].vm.Name, tt.expectedFirst)
				}

				// Only check last VM if expectedLast is specified
				if tt.expectedLast != "" {
					lastIdx := len(m.filteredVMs) - 1
					if m.filteredVMs[lastIdx].vm.Name != tt.expectedLast {
						t.Errorf("applyFiltersAndSort() last = %s, want %s", m.filteredVMs[lastIdx].vm.Name, tt.expectedLast)
					}
				}
			}
		})
	}
}

// TestApplyFiltersAndSort_CursorAdjustment tests cursor adjustment logic
func TestApplyFiltersAndSort_CursorAdjustment(t *testing.T) {
	vms := []vmItem{
		{vm: vsphere.VMInfo{Name: "vm1"}},
		{vm: vsphere.VMInfo{Name: "vm2"}},
		{vm: vsphere.VMInfo{Name: "vm3"}},
		{vm: vsphere.VMInfo{Name: "vm4"}},
		{vm: vsphere.VMInfo{Name: "vm5"}},
	}

	tests := []struct {
		name           string
		initialCursor  int
		searchQuery    string
		expectedCursor int
		expectedVMs    int
	}{
		{
			name:           "cursor_within_bounds",
			initialCursor:  2,
			searchQuery:    "", // All 5 VMs
			expectedCursor: 2,
			expectedVMs:    5,
		},
		{
			name:           "cursor_beyond_filtered_list",
			initialCursor:  4,
			searchQuery:    "vm1", // Only 1 VM
			expectedCursor: 0,     // Adjusted to last valid position
			expectedVMs:    1,
		},
		{
			name:           "cursor_negative",
			initialCursor:  -1,
			searchQuery:    "",
			expectedCursor: 0, // Adjusted to 0
			expectedVMs:    5,
		},
		{
			name:           "empty_filtered_list",
			initialCursor:  2,
			searchQuery:    "nonexistent",
			expectedCursor: 0, // Cursor set to 0 when empty
			expectedVMs:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				vms:         vms,
				cursor:      tt.initialCursor,
				searchQuery: tt.searchQuery,
				sortMode:    "name",
			}

			m.applyFiltersAndSort()

			if m.cursor != tt.expectedCursor {
				t.Errorf("cursor = %d, want %d", m.cursor, tt.expectedCursor)
			}

			if len(m.filteredVMs) != tt.expectedVMs {
				t.Errorf("filtered VMs count = %d, want %d", len(m.filteredVMs), tt.expectedVMs)
			}
		})
	}
}

// TestSearchQueryCaseInsensitive tests case-insensitive search
func TestSearchQueryCaseInsensitive(t *testing.T) {
	vms := []vmItem{
		{vm: vsphere.VMInfo{Name: "Ubuntu-Server", Path: "/dc/vm/ubuntu", GuestOS: "Ubuntu Linux"}},
		{vm: vsphere.VMInfo{Name: "Windows-Server", Path: "/dc/vm/windows", GuestOS: "Windows Server 2022"}},
	}

	tests := []struct {
		query         string
		expectedCount int
		expectedName  string
	}{
		{"ubuntu", 1, "Ubuntu-Server"},
		{"UBUNTU", 1, "Ubuntu-Server"},
		{"Ubuntu", 1, "Ubuntu-Server"},
		{"windows", 1, "Windows-Server"},
		{"WINDOWS", 1, "Windows-Server"},
		{"server", 2, "Ubuntu-Server"}, // Both have "server" in name or OS
		{"linux", 1, "Ubuntu-Server"},
	}

	for _, tt := range tests {
		t.Run("search_"+tt.query, func(t *testing.T) {
			m := model{
				vms:         vms,
				searchQuery: tt.query,
				sortMode:    "name",
			}

			m.applyFiltersAndSort()

			if len(m.filteredVMs) != tt.expectedCount {
				t.Errorf("search %q: count = %d, want %d", tt.query, len(m.filteredVMs), tt.expectedCount)
			}

			if tt.expectedCount > 0 && m.filteredVMs[0].vm.Name != tt.expectedName {
				t.Errorf("search %q: first VM = %s, want %s", tt.query, m.filteredVMs[0].vm.Name, tt.expectedName)
			}
		})
	}
}
