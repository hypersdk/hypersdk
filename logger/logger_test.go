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
		{"warning", "warning"},  // alternative spelling
		{"error", "error"},
		{"invalid", "invalid"},  // should default to INFO
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
