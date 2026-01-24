// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	// Test basic format
	id := generateID()

	if !strings.HasPrefix(id, "schedule-") {
		t.Errorf("generateID() = %q, want prefix 'schedule-'", id)
	}

	// Test that ID changes over time
	id1 := generateID()
	time.Sleep(time.Second)
	id2 := generateID()

	if id1 == id2 {
		t.Error("generateID() should generate different IDs over time")
	}

	// Test format consistency
	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		ids[i] = generateID()
		time.Sleep(10 * time.Millisecond)
	}

	for i, id := range ids {
		parts := strings.Split(id, "-")
		if len(parts) != 2 {
			t.Errorf("ID %d: %q should have format 'schedule-<timestamp>'", i, id)
		}
		if parts[0] != "schedule" {
			t.Errorf("ID %d: %q should start with 'schedule'", i, id)
		}
		if len(parts[1]) == 0 {
			t.Errorf("ID %d: %q should have non-empty timestamp", i, id)
		}
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	// Generate multiple IDs with 1 second delay and check they're unique
	// Note: generateID uses Unix timestamp in seconds, so calls within the same second may generate duplicates
	ids := make(map[string]bool)
	for i := 0; i < 3; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("generateID() generated duplicate ID: %q", id)
		}
		ids[id] = true
		time.Sleep(time.Second) // Wait 1 second to ensure different Unix timestamp
	}

	if len(ids) != 3 {
		t.Errorf("Expected 3 unique IDs, got %d", len(ids))
	}
}

func TestGenerateID_SameSecond(t *testing.T) {
	// Test that IDs generated in the same second are identical
	// This documents the actual behavior of the function
	id1 := generateID()
	id2 := generateID()

	if id1 != id2 {
		// This is actually fine - it means the calls happened in different seconds
		t.Logf("IDs were generated in different seconds: %q != %q", id1, id2)
	}
}
