// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug_level", "debug"},
		{"info_level", "info"},
		{"warn_level", "warn"},
		{"error_level", "error"},
		{"empty_level", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.level)
			if log == nil {
				t.Fatal("New() returned nil logger")
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	// Test that all log methods can be called without panic
	log := New("debug")

	// These should not panic
	log.Debug("Debug message")
	log.Info("Info message")
	log.Warn("Warn message")
	log.Error("Error message")
}

func TestLoggerWithKeyValues(t *testing.T) {
	log := New("debug")

	// Test logging with key-value pairs
	log.Info("Test message", "key1", "value1", "key2", "value2")
	log.Debug("Debug with context", "vm_path", "/datacenter/vm/test", "status", "running")
	log.Warn("Warning with context", "error_count", 5)
	log.Error("Error with context", "failed_vm", "test-vm", "reason", "timeout")
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warning"}, // alternative spelling
		{"error", "error"},
		{"invalid", "invalid"}, // should default to INFO
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.level)
			if log == nil {
				t.Fatalf("New(%s) returned nil", tt.level)
			}

			// Should not panic
			log.Debug("test")
			log.Info("test")
			log.Warn("test")
			log.Error("test")
		})
	}
}

func TestStandardLogger(t *testing.T) {
	// Create a debug level logger
	log := New("debug")

	// Cast to concrete type for testing
	stdLog, ok := log.(*StandardLogger)
	if !ok {
		t.Fatal("Expected *StandardLogger type")
	}

	if stdLog.logger == nil {
		t.Error("StandardLogger.logger should not be nil")
	}

	// Verify level was set correctly
	if stdLog.level != DEBUG {
		t.Errorf("Expected DEBUG level, got %v", stdLog.level)
	}
}

func TestLoggerConcurrency(t *testing.T) {
	log := New("info")
	done := make(chan bool, 100)

	// Log from multiple goroutines
	for i := 0; i < 100; i++ {
		go func(index int) {
			log.Info("Concurrent log", "index", index)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// If we get here without panic, concurrency is safe
}

func TestNewTestLogger(t *testing.T) {
	testLog := NewTestLogger(t)
	if testLog == nil {
		t.Fatal("NewTestLogger() returned nil")
	}

	// Verify it implements the Logger interface
	var _ Logger = testLog
}

func TestTestLogger_AllLevels(t *testing.T) {
	testLog := NewTestLogger(t)

	// Test all log levels without panic
	testLog.Debug("Debug message")
	testLog.Info("Info message")
	testLog.Warn("Warn message")
	testLog.Error("Error message")
}

func TestTestLogger_WithKeyValues(t *testing.T) {
	testLog := NewTestLogger(t)

	// Test logging with key-value pairs
	testLog.Debug("Debug with context", "key1", "value1")
	testLog.Info("Info with multiple pairs", "vm_name", "test-vm", "status", "running", "progress", 50)
	testLog.Warn("Warning with one pair", "error_count", 3)
	testLog.Error("Error with context", "vm_path", "/datacenter/vm/test", "error", "timeout")
}

func TestTestLogger_WithOddKeyValues(t *testing.T) {
	testLog := NewTestLogger(t)

	// Test with odd number of key-value pairs (should handle gracefully)
	testLog.Info("Message with odd pairs", "key1", "value1", "key2")
	testLog.Debug("Debug with single value", "lonely_key")
}

func TestTestLogger_EmptyKeyValues(t *testing.T) {
	testLog := NewTestLogger(t)

	// Test with no key-value pairs
	testLog.Debug("Just a message")
	testLog.Info("Another message")
	testLog.Warn("Warning message")
	testLog.Error("Error message")
}

func TestTestLogger_Format(t *testing.T) {
	testLog := NewTestLogger(t).(*TestLogger)

	tests := []struct {
		name          string
		level         string
		msg           string
		keysAndValues []interface{}
	}{
		{
			name:          "no pairs",
			level:         "INFO",
			msg:           "test message",
			keysAndValues: nil,
		},
		{
			name:          "one pair",
			level:         "DEBUG",
			msg:           "debug message",
			keysAndValues: []interface{}{"key1", "value1"},
		},
		{
			name:          "multiple pairs",
			level:         "WARN",
			msg:           "warning",
			keysAndValues: []interface{}{"key1", "value1", "key2", "value2"},
		},
		{
			name:          "odd number",
			level:         "ERROR",
			msg:           "error",
			keysAndValues: []interface{}{"key1", "value1", "key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testLog.format(tt.level, tt.msg, tt.keysAndValues...)
			if result == "" {
				t.Error("format() returned empty string")
			}
			// Should contain level and message
			if len(result) < len(tt.level)+len(tt.msg) {
				t.Errorf("format() result too short: %s", result)
			}
		})
	}
}
