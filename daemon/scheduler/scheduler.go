// SPDX-License-Identifier: LGPL-3.0-or-later

package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"hypersdk/daemon/models"
	"hypersdk/logger"
)

// ScheduledJob represents a job scheduled for recurring execution
type ScheduledJob struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Schedule    string                `json:"schedule"` // Cron format
	JobTemplate models.JobDefinition  `json:"job_template"`
	Enabled     bool                  `json:"enabled"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	NextRun     time.Time             `json:"next_run"`
	LastRun     *time.Time            `json:"last_run,omitempty"`
	RunCount    int                   `json:"run_count"`
	Tags        []string              `json:"tags,omitempty"`
	cronEntryID cron.EntryID          `json:"-"` // Internal cron ID
}

// Scheduler manages scheduled jobs
type Scheduler struct {
	cron      *cron.Cron
	jobs      map[string]*ScheduledJob
	mu        sync.RWMutex
	log       logger.Logger
	executor  JobExecutor
}

// JobExecutor interface for executing jobs
type JobExecutor interface {
	SubmitJob(definition models.JobDefinition) error
}

// NewScheduler creates a new job scheduler
func NewScheduler(executor JobExecutor, log logger.Logger) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		jobs:     make(map[string]*ScheduledJob),
		log:      log,
		executor: executor,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.log.Info("Starting job scheduler")
	s.cron.Start()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.log.Info("Stopping job scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// AddScheduledJob adds a new scheduled job
func (s *Scheduler) AddScheduledJob(sj *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate cron schedule
	if _, err := s.cron.AddFunc(sj.Schedule, func() {}); err != nil {
		return fmt.Errorf("invalid cron schedule: %w", err)
	}

	// Set timestamps
	sj.CreatedAt = time.Now()
	sj.UpdatedAt = time.Now()

	// Add to cron if enabled
	if sj.Enabled {
		entryID, err := s.cron.AddFunc(sj.Schedule, s.createJobFunc(sj))
		if err != nil {
			return fmt.Errorf("failed to schedule job: %w", err)
		}
		sj.cronEntryID = entryID

		// Calculate next run
		entry := s.cron.Entry(entryID)
		sj.NextRun = entry.Next
	}

	s.jobs[sj.ID] = sj
	s.log.Info("Scheduled job added",
		"id", sj.ID,
		"name", sj.Name,
		"schedule", sj.Schedule,
		"enabled", sj.Enabled)

	return nil
}

// RemoveScheduledJob removes a scheduled job
func (s *Scheduler) RemoveScheduledJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sj, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("scheduled job not found: %s", id)
	}

	// Remove from cron
	if sj.Enabled {
		s.cron.Remove(sj.cronEntryID)
	}

	delete(s.jobs, id)
	s.log.Info("Scheduled job removed", "id", id, "name", sj.Name)

	return nil
}

// UpdateScheduledJob updates an existing scheduled job
func (s *Scheduler) UpdateScheduledJob(id string, updates *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sj, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("scheduled job not found: %s", id)
	}

	// Remove old cron entry
	if sj.Enabled {
		s.cron.Remove(sj.cronEntryID)
	}

	// Update fields
	if updates.Name != "" {
		sj.Name = updates.Name
	}
	if updates.Description != "" {
		sj.Description = updates.Description
	}
	if updates.Schedule != "" {
		sj.Schedule = updates.Schedule
	}
	if updates.JobTemplate.VMPath != "" {
		sj.JobTemplate = updates.JobTemplate
	}
	if updates.Tags != nil {
		sj.Tags = updates.Tags
	}

	sj.Enabled = updates.Enabled
	sj.UpdatedAt = time.Now()

	// Re-add to cron if enabled
	if sj.Enabled {
		entryID, err := s.cron.AddFunc(sj.Schedule, s.createJobFunc(sj))
		if err != nil {
			return fmt.Errorf("failed to reschedule job: %w", err)
		}
		sj.cronEntryID = entryID

		// Update next run
		entry := s.cron.Entry(entryID)
		sj.NextRun = entry.Next
	}

	s.log.Info("Scheduled job updated", "id", id, "name", sj.Name)

	return nil
}

// GetScheduledJob retrieves a scheduled job by ID
func (s *Scheduler) GetScheduledJob(id string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sj, exists := s.jobs[id]
	if !exists {
		return nil, fmt.Errorf("scheduled job not found: %s", id)
	}

	return sj, nil
}

// ListScheduledJobs returns all scheduled jobs
func (s *Scheduler) ListScheduledJobs() []*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, sj := range s.jobs {
		// Update next run time
		if sj.Enabled {
			entry := s.cron.Entry(sj.cronEntryID)
			sj.NextRun = entry.Next
		}
		jobs = append(jobs, sj)
	}

	return jobs
}

// EnableScheduledJob enables a scheduled job
func (s *Scheduler) EnableScheduledJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sj, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("scheduled job not found: %s", id)
	}

	if sj.Enabled {
		return nil // Already enabled
	}

	// Add to cron
	entryID, err := s.cron.AddFunc(sj.Schedule, s.createJobFunc(sj))
	if err != nil {
		return fmt.Errorf("failed to enable job: %w", err)
	}

	sj.cronEntryID = entryID
	sj.Enabled = true
	sj.UpdatedAt = time.Now()

	// Update next run
	entry := s.cron.Entry(entryID)
	sj.NextRun = entry.Next

	s.log.Info("Scheduled job enabled", "id", id, "name", sj.Name)

	return nil
}

// DisableScheduledJob disables a scheduled job
func (s *Scheduler) DisableScheduledJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sj, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("scheduled job not found: %s", id)
	}

	if !sj.Enabled {
		return nil // Already disabled
	}

	// Remove from cron
	s.cron.Remove(sj.cronEntryID)
	sj.Enabled = false
	sj.UpdatedAt = time.Now()

	s.log.Info("Scheduled job disabled", "id", id, "name", sj.Name)

	return nil
}

// TriggerNow manually triggers a scheduled job
func (s *Scheduler) TriggerNow(id string) error {
	s.mu.RLock()
	sj, exists := s.jobs[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("scheduled job not found: %s", id)
	}

	s.log.Info("Manually triggering scheduled job", "id", id, "name", sj.Name)

	// Execute job
	go s.executeScheduledJob(sj)

	return nil
}

// createJobFunc creates a function that executes a scheduled job
func (s *Scheduler) createJobFunc(sj *ScheduledJob) func() {
	return func() {
		s.executeScheduledJob(sj)
	}
}

// executeScheduledJob executes a scheduled job
func (s *Scheduler) executeScheduledJob(sj *ScheduledJob) {
	s.log.Info("Executing scheduled job",
		"id", sj.ID,
		"name", sj.Name,
		"schedule", sj.Schedule)

	// Create job definition with timestamp in name
	jobDef := sj.JobTemplate
	jobDef.ID = fmt.Sprintf("%s-%d", sj.ID, time.Now().Unix())
	jobDef.Name = fmt.Sprintf("%s (scheduled)", sj.JobTemplate.Name)

	// Execute job
	err := s.executor.SubmitJob(jobDef)
	if err != nil {
		s.log.Error("Failed to execute scheduled job",
			"id", sj.ID,
			"name", sj.Name,
			"error", err)
		return
	}

	// Update statistics
	s.mu.Lock()
	now := time.Now()
	sj.LastRun = &now
	sj.RunCount++
	s.mu.Unlock()

	s.log.Info("Scheduled job executed successfully",
		"id", sj.ID,
		"name", sj.Name,
		"runCount", sj.RunCount)
}

// GetScheduleStats returns statistics about scheduled jobs
func (s *Scheduler) GetScheduleStats() *ScheduleStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &ScheduleStats{
		TotalSchedules:   len(s.jobs),
		EnabledSchedules: 0,
		NextRunning:      nil,
	}

	for _, sj := range s.jobs {
		if sj.Enabled {
			stats.EnabledSchedules++
			stats.TotalRuns += sj.RunCount

			if stats.NextRunning == nil || sj.NextRun.Before(*stats.NextRunning) {
				stats.NextRunning = &sj.NextRun
			}
		}
	}

	return stats
}

// ScheduleStats holds statistics about scheduled jobs
type ScheduleStats struct {
	TotalSchedules   int        `json:"total_schedules"`
	EnabledSchedules int        `json:"enabled_schedules"`
	TotalRuns        int        `json:"total_runs"`
	NextRunning      *time.Time `json:"next_running,omitempty"`
}
