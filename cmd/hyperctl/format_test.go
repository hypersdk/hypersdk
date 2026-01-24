// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes less than 1KB",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "exactly 1KB",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024 * 5,
			expected: "5.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1024 * 1024 * 1024 * 10,
			expected: "10.0 GB",
		},
		{
			name:     "terabytes",
			bytes:    1024 * 1024 * 1024 * 1024 * 2,
			expected: "2.0 TB",
		},
		{
			name:     "petabytes",
			bytes:    1024 * 1024 * 1024 * 1024 * 1024 * 3,
			expected: "3.0 PB",
		},
		{
			name:     "fractional GB",
			bytes:    1024 * 1024 * 1024 * 5 / 2,
			expected: "2.5 GB",
		},
		{
			name:     "small fractional MB",
			bytes:    1024 * 1024 * 3 / 2,
			expected: "1.5 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{
			name:     "zero seconds",
			seconds:  0,
			expected: "0s",
		},
		{
			name:     "less than a minute",
			seconds:  45,
			expected: "45s",
		},
		{
			name:     "exactly 1 minute",
			seconds:  60,
			expected: "1m 0s",
		},
		{
			name:     "minutes and seconds",
			seconds:  125,
			expected: "2m 5s",
		},
		{
			name:     "exactly 1 hour",
			seconds:  3600,
			expected: "1h 0m",
		},
		{
			name:     "hours and minutes",
			seconds:  3665,
			expected: "1h 1m",
		},
		{
			name:     "multiple hours",
			seconds:  7325,
			expected: "2h 2m",
		},
		{
			name:     "large duration",
			seconds:  86400,
			expected: "24h 0m",
		},
		{
			name:     "59 seconds",
			seconds:  59,
			expected: "59s",
		},
		{
			name:     "59 minutes 59 seconds",
			seconds:  3599,
			expected: "59m 59s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.seconds)
			if result != tt.expected {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestFormatBytesCompact(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0B",
		},
		{
			name:     "bytes less than 1KB",
			bytes:    512,
			expected: "512B",
		},
		{
			name:     "exactly 1KB",
			bytes:    1024,
			expected: "1K",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2K",
		},
		{
			name:     "fractional kilobytes",
			bytes:    1536,
			expected: "2K", // rounds to 2K
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024 * 5,
			expected: "5M",
		},
		{
			name:     "gigabytes",
			bytes:    1024 * 1024 * 1024 * 10,
			expected: "10.0G",
		},
		{
			name:     "terabytes",
			bytes:    1024 * 1024 * 1024 * 1024 * 2,
			expected: "2.0T",
		},
		{
			name:     "fractional gigabytes",
			bytes:    1024 * 1024 * 1024 * 5 / 2,
			expected: "2.5G",
		},
		{
			name:     "large terabytes",
			bytes:    1024 * 1024 * 1024 * 1024 * 15 / 10,
			expected: "1.5T",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytesCompact(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytesCompact(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}
