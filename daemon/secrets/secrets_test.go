// SPDX-License-Identifier: LGPL-3.0-or-later

package secrets

import (
	"context"
	"testing"
	"time"
)

func TestNewSecretManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "memory backend",
			config: &Config{
				Backend: "memory",
			},
			wantErr: false,
		},
		{
			name: "vault backend without config",
			config: &Config{
				Backend: "vault",
			},
			wantErr: true,
		},
		{
			name: "aws backend without config",
			config: &Config{
				Backend: "aws",
			},
			wantErr: true,
		},
		{
			name: "azure backend without config",
			config: &Config{
				Backend: "azure",
			},
			wantErr: true,
		},
		{
			name: "unsupported backend",
			config: &Config{
				Backend: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSecretManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSecretManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryManager_GetSet(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	secret := &Secret{
		Name: "test-secret",
		Type: SecretTypeVCenter,
		Value: map[string]string{
			"username": "admin",
			"password": "secret",
		},
		Metadata: map[string]string{
			"environment": "production",
		},
	}

	// Test Set
	err := mgr.Set(ctx, secret)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	retrieved, err := mgr.Get(ctx, "test-secret")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != secret.Name {
		t.Errorf("expected name %s, got %s", secret.Name, retrieved.Name)
	}

	if retrieved.Type != secret.Type {
		t.Errorf("expected type %s, got %s", secret.Type, retrieved.Type)
	}

	if retrieved.Value["username"] != "admin" {
		t.Errorf("expected username admin, got %s", retrieved.Value["username"])
	}

	if retrieved.Version != "1" {
		t.Errorf("expected version 1, got %s", retrieved.Version)
	}
}

func TestMemoryManager_GetNonExistent(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	_, err := mgr.Get(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent secret")
	}
}

func TestMemoryManager_Delete(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	secret := &Secret{
		Name: "to-delete",
		Type: SecretTypeAWS,
		Value: map[string]string{
			"key": "value",
		},
	}

	// Create secret
	err := mgr.Set(ctx, secret)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete secret
	err = mgr.Delete(ctx, "to-delete")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = mgr.Get(ctx, "to-delete")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestMemoryManager_List(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	// Create multiple secrets
	secrets := []*Secret{
		{
			Name:  "vcenter-1",
			Type:  SecretTypeVCenter,
			Value: map[string]string{"host": "vcenter1.example.com"},
		},
		{
			Name:  "vcenter-2",
			Type:  SecretTypeVCenter,
			Value: map[string]string{"host": "vcenter2.example.com"},
		},
		{
			Name:  "aws-1",
			Type:  SecretTypeAWS,
			Value: map[string]string{"region": "us-east-1"},
		},
	}

	for _, secret := range secrets {
		err := mgr.Set(ctx, secret)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// List all secrets
	all, err := mgr.List(ctx, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("expected 3 secrets, got %d", len(all))
	}

	// List by type
	vcenterSecrets, err := mgr.List(ctx, SecretTypeVCenter)
	if err != nil {
		t.Fatalf("List by type failed: %v", err)
	}

	if len(vcenterSecrets) != 2 {
		t.Errorf("expected 2 vcenter secrets, got %d", len(vcenterSecrets))
	}
}

func TestMemoryManager_Rotate(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	// Create initial secret
	secret := &Secret{
		Name: "rotatable",
		Type: SecretTypeAPIKey,
		Value: map[string]string{
			"key": "old-key",
		},
	}

	err := mgr.Set(ctx, secret)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get initial version
	initial, err := mgr.Get(ctx, "rotatable")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	initialVersion := initial.Version

	// Rotate secret
	newValue := map[string]string{
		"key": "new-key",
	}

	err = mgr.Rotate(ctx, "rotatable", newValue)
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}

	// Verify rotation
	rotated, err := mgr.Get(ctx, "rotatable")
	if err != nil {
		t.Fatalf("Get after rotate failed: %v", err)
	}

	if rotated.Value["key"] != "new-key" {
		t.Errorf("expected new-key, got %s", rotated.Value["key"])
	}

	if rotated.Version == initialVersion {
		t.Error("expected version to change after rotation")
	}
}

func TestMemoryManager_Update(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	// Create secret
	secret := &Secret{
		Name:  "updateable",
		Type:  SecretTypeDatabase,
		Value: map[string]string{"password": "old"},
	}

	err := mgr.Set(ctx, secret)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	firstCreated, _ := mgr.Get(ctx, "updateable")

	// Update secret
	time.Sleep(10 * time.Millisecond)
	secret.Value["password"] = "new"
	err = mgr.Set(ctx, secret)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	updated, err := mgr.Get(ctx, "updateable")
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}

	if updated.Value["password"] != "new" {
		t.Errorf("expected new password")
	}

	// CreatedAt should remain same
	if !updated.CreatedAt.Equal(firstCreated.CreatedAt) {
		t.Error("CreatedAt should not change on update")
	}

	// UpdatedAt should be newer
	if !updated.UpdatedAt.After(firstCreated.UpdatedAt) {
		t.Error("UpdatedAt should be newer after update")
	}

	// Version should increment
	if updated.Version != "2" {
		t.Errorf("expected version 2, got %s", updated.Version)
	}
}

func TestMemoryManager_Health(t *testing.T) {
	ctx := context.Background()
	mgr := NewMemoryManager()

	err := mgr.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestMemoryManager_Close(t *testing.T) {
	mgr := NewMemoryManager()

	err := mgr.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestCachedSecretManager(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryManager()
	cached := NewCachedSecretManager(backend, 100*time.Millisecond)

	secret := &Secret{
		Name:  "cached",
		Type:  SecretTypeVCenter,
		Value: map[string]string{"key": "value"},
	}

	// Set secret
	err := cached.Set(ctx, secret)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// First get - from backend
	retrieved1, err := cached.Get(ctx, "cached")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved1.Value["key"] != "value" {
		t.Error("unexpected value")
	}

	// Second get - from cache
	retrieved2, err := cached.Get(ctx, "cached")
	if err != nil {
		t.Fatalf("Cached get failed: %v", err)
	}

	if retrieved2.Value["key"] != "value" {
		t.Error("unexpected cached value")
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Get after expiry - from backend again
	retrieved3, err := cached.Get(ctx, "cached")
	if err != nil {
		t.Fatalf("Get after expiry failed: %v", err)
	}

	if retrieved3.Value["key"] != "value" {
		t.Error("unexpected value after expiry")
	}
}

func TestCachedSecretManager_InvalidateCache(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryManager()
	cached := NewCachedSecretManager(backend, 1*time.Hour)

	secret := &Secret{
		Name:  "invalidatable",
		Type:  SecretTypeAWS,
		Value: map[string]string{"key": "old"},
	}

	// Set and cache
	cached.Set(ctx, secret)
	cached.Get(ctx, "invalidatable")

	// Update backend directly
	secret.Value["key"] = "new"
	backend.Set(ctx, secret)

	// Invalidate cache
	cached.InvalidateSecret("invalidatable")

	// Get should fetch new value
	retrieved, err := cached.Get(ctx, "invalidatable")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Value["key"] != "new" {
		t.Error("expected new value after cache invalidation")
	}
}

func TestSecretTypes(t *testing.T) {
	types := []SecretType{
		SecretTypeVCenter,
		SecretTypeAWS,
		SecretTypeAzure,
		SecretTypeGCP,
		SecretTypeDatabase,
		SecretTypeAPIKey,
		SecretTypeSSHKey,
		SecretTypeTLS,
	}

	if len(types) != 8 {
		t.Errorf("expected 8 secret types, got %d", len(types))
	}
}
