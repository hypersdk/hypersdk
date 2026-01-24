// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		value    string
		expected bool
	}{
		{
			name:     "value exists in slice",
			slice:    []string{"apple", "banana", "cherry"},
			value:    "banana",
			expected: true,
		},
		{
			name:     "value does not exist in slice",
			slice:    []string{"apple", "banana", "cherry"},
			value:    "orange",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			value:    "test",
			expected: false,
		},
		{
			name:     "value is first element",
			slice:    []string{"first", "second", "third"},
			value:    "first",
			expected: true,
		},
		{
			name:     "value is last element",
			slice:    []string{"first", "second", "third"},
			value:    "third",
			expected: true,
		},
		{
			name:     "slice with one element - match",
			slice:    []string{"only"},
			value:    "only",
			expected: true,
		},
		{
			name:     "slice with one element - no match",
			slice:    []string{"only"},
			value:    "other",
			expected: false,
		},
		{
			name:     "case sensitive - different case",
			slice:    []string{"Test", "TEST", "test"},
			value:    "Test",
			expected: true,
		},
		{
			name:     "case sensitive - no match",
			slice:    []string{"Test", "TEST"},
			value:    "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.value)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.value, result, tt.expected)
			}
		})
	}
}
