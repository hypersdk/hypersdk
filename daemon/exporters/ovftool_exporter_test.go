// SPDX-License-Identifier: LGPL-3.0-or-later

package exporters

import (
	"context"
	"testing"

	"hypersdk/daemon/capabilities"
	"hypersdk/daemon/models"
	"hypersdk/logger"
)

func TestNewOvftoolExporter(t *testing.T) {
	log := logger.New("info")
	exporter := NewOvftoolExporter("/usr/bin/ovftool", log)

	if exporter == nil {
		t.Fatal("NewOvftoolExporter returned nil")
	}

	if exporter.ovftoolPath != "/usr/bin/ovftool" {
		t.Errorf("Expected path /usr/bin/ovftool, got %s", exporter.ovftoolPath)
	}
}

func TestOvftoolExporter_Method(t *testing.T) {
	log := logger.New("info")
	exporter := NewOvftoolExporter("/usr/bin/ovftool", log)

	if exporter.Method() != capabilities.ExportMethodOvftool {
		t.Errorf("Expected method ovftool, got %s", exporter.Method())
	}
}

func TestOvftoolExporter_Validate(t *testing.T) {
	log := logger.New("info")
	exporter := NewOvftoolExporter("/usr/bin/ovftool", log)

	tests := []struct {
		name    string
		job     *models.JobDefinition
		wantErr bool
	}{
		{
			name: "valid job",
			job: &models.JobDefinition{
				VMPath:    "/datacenter/vm/test-vm",
				OutputDir: "/tmp/output",
				VCenter: &models.VCenterConfig{
					Server:   "vcenter.example.com",
					Username: "admin",
					Password: "password",
				},
			},
			wantErr: false,
		},
		{
			name: "missing vm_path",
			job: &models.JobDefinition{
				OutputDir: "/tmp/output",
				VCenter: &models.VCenterConfig{
					Server: "vcenter.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing vcenter credentials",
			job: &models.JobDefinition{
				VMPath:    "/datacenter/vm/test-vm",
				OutputDir: "/tmp/output",
				VCenter: &models.VCenterConfig{
					Server: "vcenter.example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exporter.Validate(tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOvftoolExporter_Export_InvalidJob(t *testing.T) {
	log := logger.New("info")
	exporter := NewOvftoolExporter("/usr/bin/ovftool", log)

	invalidJob := &models.JobDefinition{}

	ctx := context.Background()
	_, err := exporter.Export(ctx, invalidJob, nil)

	if err == nil {
		t.Error("Expected error for invalid job, got nil")
	}
}
