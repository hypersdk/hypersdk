// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"hypersdk/config"
	"hypersdk/providers/vsphere"
)

// VMListResponse represents the response for VM listing
type VMListResponse struct {
	VMs       []vsphere.VMInfo `json:"vms"`
	Total     int              `json:"total"`
	Timestamp time.Time        `json:"timestamp"`
}

// Handle VM listing
func (s *Server) handleListVMs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if credentials are provided in query parameters (for web UI)
	server := r.URL.Query().Get("server")
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")
	insecure := r.URL.Query().Get("insecure") == "true"

	var vms []vsphere.VMInfo
	var err error

	if server != "" && username != "" && password != "" {
		// Use provided credentials (web UI mode)
		vms, err = s.listVMsWithCredentials(server, username, password, insecure)
	} else {
		// Use configured credentials (TUI/CLI mode)
		vms, err = s.listVMsFromVSphere()
	}

	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to list VMs: %v", err)
		return
	}

	resp := VMListResponse{
		VMs:       vms,
		Total:     len(vms),
		Timestamp: time.Now(),
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// List VMs with provided credentials (for web UI)
func (s *Server) listVMsWithCredentials(server, username, password string, insecure bool) ([]vsphere.VMInfo, error) {
	ctx := context.Background()

	// Create config from provided credentials
	cfg := &config.Config{
		VCenterURL: fmt.Sprintf("https://%s/sdk", server),
		Username:   username,
		Password:   password,
		Insecure:   insecure,
		Timeout:    30 * time.Second,
	}

	// Create vSphere client
	client, err := vsphere.NewVSphereClient(ctx, cfg, s.logger)
	if err != nil {
		s.logger.Error("failed to create vSphere client", "error", err, "server", server)
		return nil, err
	}
	defer client.Close()

	// List VMs using client
	vms, err := client.ListVMs(ctx)
	if err != nil {
		s.logger.Error("failed to list VMs from vSphere", "error", err)
		return nil, err
	}

	s.logger.Info("discovered VMs from vSphere", "count", len(vms), "server", server)
	return vms, nil
}

// List all VMs from vSphere using configured credentials
func (s *Server) listVMsFromVSphere() ([]vsphere.VMInfo, error) {
	ctx := context.Background()

	// Get vSphere client from manager
	client, err := s.manager.GetVSphereClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// List VMs using client
	vms, err := client.ListVMs(ctx)
	if err != nil {
		return nil, err
	}

	return vms, nil
}
