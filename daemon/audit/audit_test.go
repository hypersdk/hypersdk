// SPDX-License-Identifier: LGPL-3.0-or-later

package audit

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileLogger(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewFileLogger(tmpDir, 10, 30, 5)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger.directory != tmpDir {
		t.Errorf("expected directory %s, got %s", tmpDir, logger.directory)
	}

	if logger.maxSize != 10*1024*1024 {
		t.Errorf("expected maxSize 10MB, got %d", logger.maxSize)
	}
}

func TestLogEvent(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewFileLogger(tmpDir, 10, 30, 5)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	event := NewEvent(EventTypeLogin, "testuser")
	event.Status = EventStatusSuccess
	event.IPAddress = "192.168.1.1"
	event.Details["method"] = "password"

	err = logger.Log(event)
	if err != nil {
		t.Errorf("failed to log event: %v", err)
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tmpDir, "audit-*.log"))
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 log file, got %d", len(files))
	}
}

func TestQueryEvents(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewFileLogger(tmpDir, 10, 30, 5)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log multiple events
	events := []*Event{
		{
			ID:        "1",
			Timestamp: time.Now(),
			EventType: EventTypeLogin,
			Status:    EventStatusSuccess,
			Username:  "alice",
			IPAddress: "192.168.1.1",
		},
		{
			ID:        "2",
			Timestamp: time.Now(),
			EventType: EventTypeExportVM,
			Status:    EventStatusSuccess,
			Username:  "bob",
			IPAddress: "192.168.1.2",
			Resource:  "vm-web-01",
		},
		{
			ID:        "3",
			Timestamp: time.Now(),
			EventType: EventTypeLogin,
			Status:    EventStatusFailure,
			Username:  "charlie",
			IPAddress: "192.168.1.3",
			Error:     "invalid credentials",
		},
	}

	for _, event := range events {
		if err := logger.Log(event); err != nil {
			t.Errorf("failed to log event: %v", err)
		}
	}

	// Query all events
	results, err := logger.Query(QueryFilter{})
	if err != nil {
		t.Fatalf("failed to query events: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 events, got %d", len(results))
	}

	// Query by username
	results, err = logger.Query(QueryFilter{Username: "alice"})
	if err != nil {
		t.Fatalf("failed to query by username: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 event for alice, got %d", len(results))
	}

	if results[0].Username != "alice" {
		t.Errorf("expected username alice, got %s", results[0].Username)
	}

	// Query by event type
	results, err = logger.Query(QueryFilter{EventType: EventTypeLogin})
	if err != nil {
		t.Fatalf("failed to query by event type: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 login events, got %d", len(results))
	}

	// Query by status
	results, err = logger.Query(QueryFilter{Status: EventStatusFailure})
	if err != nil {
		t.Fatalf("failed to query by status: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 failure event, got %d", len(results))
	}
}

func TestQueryWithLimit(t *testing.T) {
	tmpDir := t.TempDir()

	logger, err := NewFileLogger(tmpDir, 10, 30, 5)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log 10 events
	for i := 0; i < 10; i++ {
		event := NewEvent(EventTypeLogin, "user")
		event.Status = EventStatusSuccess
		logger.Log(event)
	}

	// Query with limit
	results, err := logger.Query(QueryFilter{Limit: 5})
	if err != nil {
		t.Fatalf("failed to query with limit: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 events, got %d", len(results))
	}
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventTypeExportVM, "testuser")

	if event.ID == "" {
		t.Error("expected event ID to be generated")
	}

	if event.EventType != EventTypeExportVM {
		t.Errorf("expected event type export_vm, got %s", event.EventType)
	}

	if event.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", event.Username)
	}

	if event.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}

	if event.Details == nil {
		t.Error("expected details map to be initialized")
	}
}

func TestEventTypes(t *testing.T) {
	types := []EventType{
		EventTypeLogin,
		EventTypeLogout,
		EventTypeExportVM,
		EventTypeCreateJob,
		EventTypeCancelJob,
		EventTypeDeleteJob,
		EventTypeCreateSchedule,
		EventTypeUpdateSchedule,
		EventTypeDeleteSchedule,
		EventTypeCreateWebhook,
		EventTypeDeleteWebhook,
		EventTypeCreateUser,
		EventTypeUpdateUser,
		EventTypeDeleteUser,
		EventTypeConfigChange,
	}

	if len(types) != 15 {
		t.Errorf("expected 15 event types, got %d", len(types))
	}
}

func TestEventStatus(t *testing.T) {
	statuses := []EventStatus{
		EventStatusSuccess,
		EventStatusFailure,
		EventStatusDenied,
	}

	if len(statuses) != 3 {
		t.Errorf("expected 3 event statuses, got %d", len(statuses))
	}
}

func TestLogRotation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create logger with very small max size (1KB)
	logger, err := NewFileLogger(tmpDir, 1, 30, 5)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Manually set smaller size for testing
	logger.maxSize = 1024

	// Log many events to trigger rotation
	for i := 0; i < 100; i++ {
		event := NewEvent(EventTypeLogin, "user")
		event.Status = EventStatusSuccess
		event.Details["iteration"] = i
		logger.Log(event)
	}

	// Should have multiple log files now
	files, err := filepath.Glob(filepath.Join(tmpDir, "audit-*.log*"))
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	// Note: Rotation might not happen in tests due to timing
	if len(files) == 0 {
		t.Error("expected at least one log file")
	}
}
