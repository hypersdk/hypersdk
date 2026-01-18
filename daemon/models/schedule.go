// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"time"
)

// ScheduledJob represents a job scheduled for recurring execution
type ScheduledJob struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schedule    string         `json:"schedule"` // Cron format
	JobTemplate JobDefinition  `json:"job_template"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	NextRun     time.Time      `json:"next_run"`
	LastRun     *time.Time     `json:"last_run,omitempty"`
	RunCount    int            `json:"run_count"`
	Tags        []string       `json:"tags,omitempty"`
}
