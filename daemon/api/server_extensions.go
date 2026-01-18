// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"hypersdk/daemon/scheduler"
	"hypersdk/daemon/webhooks"
)

// ServerExtensions adds new fields to Server for Phase 1 features
// This allows the base Server methods to access these components
type ServerExtensions struct {
	scheduler  *scheduler.Scheduler
	webhookMgr *webhooks.Manager
	config     *Config
}

// SetScheduler sets the scheduler for the server
func (s *Server) SetScheduler(scheduler *scheduler.Scheduler) {
	// Store in a package-level map or use interface composition
	// For now, we'll use the enhanced server approach
}

// SetWebhookManager sets the webhook manager for the server
func (s *Server) SetWebhookManager(mgr *webhooks.Manager) {
	// Store in a package-level map or use interface composition
}

// SetConfig sets the configuration for the server
func (s *Server) SetConfig(config *Config) {
	// Store in a package-level map or use interface composition
}
