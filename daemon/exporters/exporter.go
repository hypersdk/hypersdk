// SPDX-License-Identifier: LGPL-3.0-or-later

package exporters

import (
	"context"
	"fmt"

	"hypersdk/daemon/capabilities"
	"hypersdk/daemon/models"
	"hypersdk/logger"
)

// Exporter defines the interface for VM export implementations
type Exporter interface {
	// Export performs VM export using the specific method
	Export(ctx context.Context, job *models.JobDefinition, progressCallback func(*models.JobProgress)) (*models.JobResult, error)

	// Method returns the export method name
	Method() capabilities.ExportMethod

	// Validate checks if this exporter can handle the job
	Validate(job *models.JobDefinition) error
}

// ExporterFactory creates exporters based on capabilities
type ExporterFactory struct {
	detector *capabilities.Detector
	logger   logger.Logger
}

// NewExporterFactory creates a new exporter factory
func NewExporterFactory(detector *capabilities.Detector, log logger.Logger) *ExporterFactory {
	return &ExporterFactory{
		detector: detector,
		logger:   log,
	}
}

// CreateExporter creates an exporter for the specified method
func (f *ExporterFactory) CreateExporter(method capabilities.ExportMethod) (Exporter, error) {
	// Check if method is available
	if !f.detector.IsAvailable(method) {
		return nil, fmt.Errorf("export method %s is not available", method)
	}

	caps := f.detector.GetCapabilities()
	cap := caps[method]

	switch method {
	case capabilities.ExportMethodCTL:
		return NewCTLExporter(cap.Path, f.logger), nil

	case capabilities.ExportMethodGovc:
		return NewGovcExporter(cap.Path, f.logger), nil

	case capabilities.ExportMethodOvftool:
		return NewOvftoolExporter(cap.Path, f.logger), nil

	case capabilities.ExportMethodWeb:
		return NewWebExporter(f.logger), nil

	default:
		return nil, fmt.Errorf("unknown export method: %s", method)
	}
}

// GetOrCreateDefault creates an exporter using the default (best available) method
func (f *ExporterFactory) GetOrCreateDefault() (Exporter, error) {
	defaultMethod := f.detector.GetDefaultMethod()
	return f.CreateExporter(defaultMethod)
}

// GetAvailableMethods returns a list of available export methods in priority order
func (f *ExporterFactory) GetAvailableMethods() []capabilities.ExportMethod {
	methods := []capabilities.ExportMethod{
		capabilities.ExportMethodCTL,
		capabilities.ExportMethodGovc,
		capabilities.ExportMethodOvftool,
		capabilities.ExportMethodWeb,
	}

	var available []capabilities.ExportMethod
	for _, method := range methods {
		if f.detector.IsAvailable(method) {
			available = append(available, method)
		}
	}

	return available
}

// IsAvailable checks if a specific export method is available
func (f *ExporterFactory) IsAvailable(method capabilities.ExportMethod) bool {
	return f.detector.IsAvailable(method)
}

// GetDefaultMethod returns the default (highest priority) export method
func (f *ExporterFactory) GetDefaultMethod() capabilities.ExportMethod {
	return f.detector.GetDefaultMethod()
}
