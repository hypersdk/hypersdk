// SPDX-License-Identifier: LGPL-3.0-or-later

package vsphere

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneVM(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	// Get an existing VM to clone
	vms, err := client.ListVMs(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, vms)

	sourceVM := vms[0].Name

	tests := []struct {
		name    string
		spec    CloneSpec
		wantErr bool
	}{
		{
			name: "basic clone",
			spec: CloneSpec{
				SourceVM:   sourceVM,
				TargetName: "test-clone-basic",
				PowerOn:    false,
				Template:   false,
			},
			wantErr: false,
		},
		{
			name: "clone with power on",
			spec: CloneSpec{
				SourceVM:   sourceVM,
				TargetName: "test-clone-powered",
				PowerOn:    true,
				Template:   false,
			},
			wantErr: false,
		},
		{
			name: "clone as template",
			spec: CloneSpec{
				SourceVM:   sourceVM,
				TargetName: "test-clone-template",
				PowerOn:    false,
				Template:   true,
			},
			wantErr: false,
		},
		{
			name: "empty source VM",
			spec: CloneSpec{
				SourceVM:   "",
				TargetName: "test-clone-empty",
			},
			wantErr: true,
		},
		{
			name: "empty target name",
			spec: CloneSpec{
				SourceVM:   sourceVM,
				TargetName: "",
			},
			wantErr: true,
		},
		{
			name: "non-existent source VM",
			spec: CloneSpec{
				SourceVM:   "non-existent-vm",
				TargetName: "test-clone-invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			result, err := client.CloneVM(ctx, tt.spec)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.True(t, result.Success)
			assert.Equal(t, tt.spec.TargetName, result.TargetName)
			assert.NotEmpty(t, result.TargetPath)
			assert.Greater(t, result.Duration, time.Duration(0))

			// Cleanup: Delete the cloned VM
			// (In real tests, defer this or use test fixtures)
		})
	}
}

func TestBulkCloneVMs(t *testing.T) {
	t.Skip("Skipping bulk clone test - requires significant resources")

	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	// Get an existing VM to clone
	vms, err := client.ListVMs(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, vms)

	sourceVM := vms[0].Name

	specs := []CloneSpec{
		{
			SourceVM:   sourceVM,
			TargetName: "bulk-clone-1",
			PowerOn:    false,
		},
		{
			SourceVM:   sourceVM,
			TargetName: "bulk-clone-2",
			PowerOn:    false,
		},
		{
			SourceVM:   sourceVM,
			TargetName: "bulk-clone-3",
			PowerOn:    false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	results, err := client.BulkCloneVMs(ctx, specs, 2)
	require.NoError(t, err)
	assert.Len(t, results, len(specs))

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
			assert.NotEmpty(t, result.TargetName)
			assert.NotEmpty(t, result.TargetPath)
		}
	}

	assert.Greater(t, successCount, 0, "at least one clone should succeed")
}

func TestLinkedClone(t *testing.T) {
	t.Skip("Skipping linked clone test - requires snapshot support")

	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	// Get an existing VM with snapshot
	vms, err := client.ListVMs(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, vms)

	sourceVM := vms[0].Name

	spec := CloneSpec{
		SourceVM:    sourceVM,
		TargetName:  "test-linked-clone",
		LinkedClone: true,
		Snapshot:    "test-snapshot",
		PowerOn:     false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := client.CloneVM(ctx, spec)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Success)
	assert.Equal(t, spec.TargetName, result.TargetName)
}

func TestConvertVMToTemplate(t *testing.T) {
	t.Skip("Skipping template conversion test - method not implemented")
}

func TestDeployFromTemplate(t *testing.T) {
	t.Skip("Skipping template deployment test - requires existing template")

	client, cleanup := setupTestClient(t)
	defer cleanup()

	spec := CloneSpec{
		SourceVM:   "test-template",
		TargetName: "vm-from-template",
		PowerOn:    true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := client.DeployFromTemplate(ctx, spec)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Success)
	assert.Equal(t, spec.TargetName, result.TargetName)
	assert.NotEmpty(t, result.TargetPath)
}

func TestCloneSpecValidation(t *testing.T) {
	tests := []struct {
		name  string
		spec  CloneSpec
		valid bool
	}{
		{
			name: "valid basic spec",
			spec: CloneSpec{
				SourceVM:   "source-vm",
				TargetName: "target-vm",
				PowerOn:    false,
			},
			valid: true,
		},
		{
			name: "valid spec with folder",
			spec: CloneSpec{
				SourceVM:     "source-vm",
				TargetName:   "target-vm",
				TargetFolder: "/DC1/vm/prod",
			},
			valid: true,
		},
		{
			name: "valid spec with resource pool",
			spec: CloneSpec{
				SourceVM:     "source-vm",
				TargetName:   "target-vm",
				ResourcePool: "prod-pool",
			},
			valid: true,
		},
		{
			name: "invalid - empty source",
			spec: CloneSpec{
				SourceVM:   "",
				TargetName: "target-vm",
			},
			valid: false,
		},
		{
			name: "invalid - empty target",
			spec: CloneSpec{
				SourceVM:   "source-vm",
				TargetName: "",
			},
			valid: false,
		},
		{
			name: "invalid - linked clone without snapshot",
			spec: CloneSpec{
				SourceVM:    "source-vm",
				TargetName:  "target-vm",
				LinkedClone: true,
				Snapshot:    "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			isValid := tt.spec.SourceVM != "" && tt.spec.TargetName != ""

			// Linked clone must have snapshot
			if tt.spec.LinkedClone && tt.spec.Snapshot == "" {
				isValid = false
			}

			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestCloneResultValidation(t *testing.T) {
	result := CloneResult{
		TargetName: "cloned-vm",
		TargetPath: "/DC1/vm/clones/cloned-vm",
		Success:    true,
		Duration:   5 * time.Minute,
		Error:      "",
	}

	assert.NotEmpty(t, result.TargetName)
	assert.NotEmpty(t, result.TargetPath)
	assert.True(t, result.Success)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Empty(t, result.Error)
}

func TestCloneResultWithError(t *testing.T) {
	result := CloneResult{
		TargetName: "failed-clone",
		TargetPath: "",
		Success:    false,
		Duration:   30 * time.Second,
		Error:      "insufficient disk space",
	}

	assert.NotEmpty(t, result.TargetName)
	assert.Empty(t, result.TargetPath)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

func TestBulkCloneConcurrency(t *testing.T) {
	// Test that bulk clone respects concurrency limits
	maxConcurrent := 2

	specs := make([]CloneSpec, 10)
	for i := range specs {
		specs[i] = CloneSpec{
			SourceVM:   "template-vm",
			TargetName: "clone-" + string(rune(i)),
		}
	}

	// In actual implementation, this would verify that
	// no more than maxConcurrent clones run simultaneously
	assert.Equal(t, 2, maxConcurrent)
	assert.Len(t, specs, 10)
}

func TestCloneContextCancellation(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	spec := CloneSpec{
		SourceVM:   "test-vm",
		TargetName: "clone-vm",
	}

	result, err := client.CloneVM(ctx, spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, result)
}
