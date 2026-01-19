// SPDX-License-Identifier: LGPL-3.0-or-later

package vsphere

import (
	"time"

	"hypersdk/providers/common"
)

const (
	defaultDirPerm   = 0755
	leaseWaitTimeout = 5 * time.Minute
	downloadTimeout  = 2 * time.Hour
)

type VMInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	PowerState string `json:"power_state"`
	GuestOS    string `json:"guest_os"`
	MemoryMB   int32  `json:"memory_mb"`
	NumCPU     int32  `json:"num_cpu"`
	Storage    int64  `json:"storage_bytes"`
}

type ExportResult struct {
	OutputDir    string
	OVFPath      string
	OVAPath      string // Path to OVA file (if Format is "ova")
	Format       string // "ovf" or "ova"
	Files        []string
	TotalSize    int64
	Duration     time.Duration
	ManifestPath string // Path to Artifact Manifest v1.0 JSON file

	// Conversion result (Phase 2)
	ConversionResult *common.ConversionResult
}
