// SPDX-License-Identifier: LGPL-3.0-or-later

package exporters

import (
	"context"
	"testing"

	"hypersdk/daemon/capabilities"
	"hypersdk/daemon/models"
	"hypersdk/logger"
)

func TestNewCTLExporter(t *testing.T) {
	log := logger.New("info")
	exporter := NewCTLExporter("/usr/bin/hyperctl", log)

	if exporter == nil {
		t.Fatal("NewCTLExporter returned nil")
	}

	if exporter.ctlPath != "/usr/bin/hyperctl" {
		t.Errorf("Expected path /usr/bin/hyperctl, got %s", exporter.ctlPath)
	}

	if exporter.logger == nil {
		t.Error("logger not set")
	}
}

func TestCTLExporter_Method(t *testing.T) {
	log := logger.New("info")
	exporter := NewCTLExporter("/usr/bin/hyperctl", log)

	if exporter.Method() != capabilities.ExportMethodCTL {
		t.Errorf("Expected method ctl, got %s", exporter.Method())
	}
}

func TestCTLExporter_Validate(t *testing.T) {
	log := logger.New("info")
	exporter := NewCTLExporter("/usr/bin/hyperctl", log)

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
			name: "missing output_dir",
			job: &models.JobDefinition{
				VMPath: "/datacenter/vm/test-vm",
				VCenter: &models.VCenterConfig{
					Server: "vcenter.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing vcenter server",
			job: &models.JobDefinition{
				VMPath:    "/datacenter/vm/test-vm",
				OutputDir: "/tmp/output",
				VCenter: &models.VCenterConfig{
					Username: "admin",
				},
			},
			wantErr: true,
		},
		{
			name: "nil vcenter config",
			job: &models.JobDefinition{
				VMPath:    "/datacenter/vm/test-vm",
				OutputDir: "/tmp/output",
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

func TestCTLExporter_BuildArgs(t *testing.T) {
	log := logger.New("info")
	exporter := NewCTLExporter("/usr/bin/hyperctl", log)

	job := &models.JobDefinition{
		VMPath:    "/datacenter/vm/test-vm",
		OutputDir: "/tmp/output",
		Format:    "ova",
		VCenter: &models.VCenterConfig{
			Server:   "vcenter.example.com",
			Username: "admin",
			Password: "secret",
			Insecure: true,
		},
		Compress: true,
		Thin:     true,
	}

	// We can't easily test the actual command execution, but we can validate the job
	err := exporter.Validate(job)
	if err != nil {
		t.Errorf("Valid job failed validation: %v", err)
	}
}

func TestCTLExporter_Export_InvalidJob(t *testing.T) {
	log := logger.New("info")
	exporter := NewCTLExporter("/usr/bin/hyperctl", log)

	invalidJob := &models.JobDefinition{
		// Missing required fields
	}

	ctx := context.Background()
	_, err := exporter.Export(ctx, invalidJob, nil)

	// Should fail validation
	if err == nil {
		t.Error("Expected error for invalid job, got nil")
	}
}
