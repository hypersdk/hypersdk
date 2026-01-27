# Backend Integration Guide for Workflow Components

## Required API Endpoints

### 1. Provider Management Endpoints

#### GET /api/providers/list
**Purpose:** List all configured providers

**Request:** None

**Response:**
```json
[
  {
    "provider": "vsphere",
    "name": "Production vCenter",
    "enabled": true,
    "connected": true,
    "lastChecked": "2026-01-29T10:00:00Z",
    "config": {
      "host": "vcenter.example.com",
      "datacenter": "DC1"
      // Don't include passwords
    }
  },
  {
    "provider": "aws",
    "name": "AWS Production",
    "enabled": true,
    "connected": false,
    "error": "Invalid credentials",
    "config": {
      "region": "us-east-1"
    }
  }
]
```

**Implementation Example:**
```go
// daemon/api/provider_handlers.go
package api

import (
    "encoding/json"
    "net/http"
)

type ProviderInfo struct {
    Provider    string            `json:"provider"`
    Name        string            `json:"name"`
    Enabled     bool              `json:"enabled"`
    Connected   bool              `json:"connected"`
    LastChecked string            `json:"lastChecked,omitempty"`
    Error       string            `json:"error,omitempty"`
    Config      map[string]string `json:"config"`
}

func (s *Server) handleProvidersList(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Get providers from config or database
    providers := s.store.GetProviders()

    // Filter out sensitive data
    for i := range providers {
        delete(providers[i].Config, "password")
        delete(providers[i].Config, "secret_key")
        delete(providers[i].Config, "client_secret")
        delete(providers[i].Config, "private_key")
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(providers)
}

// Register in server.go
mux.HandleFunc("/api/providers/list", s.handleProvidersList)
```

#### POST /api/providers/add
**Purpose:** Add or update provider configuration

**Request:**
```json
{
  "provider": "vsphere",
  "config": {
    "host": "vcenter.example.com",
    "username": "administrator@vsphere.local",
    "password": "secret123",
    "datacenter": "DC1",
    "insecure": "true"
  }
}
```

**Response:**
```json
{
  "success": true,
  "provider": "vsphere",
  "message": "Provider added successfully"
}
```

**Implementation:**
```go
func (s *Server) handleProviderAdd(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Provider string            `json:"provider"`
        Config   map[string]string `json:"config"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate provider type
    if !isValidProvider(req.Provider) {
        http.Error(w, "Invalid provider", http.StatusBadRequest)
        return
    }

    // Encrypt sensitive fields
    encryptedConfig := encryptSensitiveFields(req.Config)

    // Store in database
    if err := s.store.SaveProvider(req.Provider, encryptedConfig); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success":  true,
        "provider": req.Provider,
        "message":  "Provider added successfully",
    })
}

// Register
mux.HandleFunc("/api/providers/add", s.handleProviderAdd)
```

#### POST /api/providers/test
**Purpose:** Test provider connection

**Request:**
```json
{
  "provider": "vsphere"
}
```

**Response:**
```json
{
  "success": true,
  "provider": "vsphere",
  "message": "Connection successful",
  "details": {
    "version": "7.0.3",
    "datacenters": 2,
    "vms": 150
  }
}
```

**Or on failure:**
```json
{
  "success": false,
  "provider": "vsphere",
  "error": "Authentication failed: Invalid credentials"
}
```

**Implementation:**
```go
func (s *Server) handleProviderTest(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Provider string `json:"provider"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get provider from registry
    provider := registry.GetProvider(req.Provider)
    if provider == nil {
        http.Error(w, "Provider not found", http.StatusNotFound)
        return
    }

    // Get stored config
    config := s.store.GetProviderConfig(req.Provider)

    // Test connection
    ctx := context.Background()
    if err := provider.Connect(ctx, config); err != nil {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success":  false,
            "provider": req.Provider,
            "error":    err.Error(),
        })
        return
    }
    defer provider.Disconnect()

    // Validate credentials
    if err := provider.ValidateCredentials(ctx); err != nil {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success":  false,
            "provider": req.Provider,
            "error":    err.Error(),
        })
        return
    }

    // Get additional info
    details := make(map[string]interface{})
    // Provider-specific details here

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success":  true,
        "provider": req.Provider,
        "message":  "Connection successful",
        "details":  details,
    })
}

// Register
mux.HandleFunc("/api/providers/test", s.handleProviderTest)
```

### 2. VM Discovery Endpoint

#### POST /api/vms/list
**Purpose:** List VMs from specified provider

**Request:**
```json
{
  "provider": "vsphere",
  "filter": {
    "datacenter": "DC1",
    "status": "poweredOn"
  }
}
```

**Response:**
```json
[
  {
    "id": "vm-123",
    "name": "web-server-01",
    "provider": "vsphere",
    "status": "running",
    "power_state": "poweredOn",
    "cpu_count": 4,
    "memory_mb": 8192,
    "os": "Ubuntu 20.04",
    "disk_gb": 100,
    "ip_address": "10.0.1.50",
    "datacenter": "DC1",
    "cluster": "Cluster1",
    "tags": ["production", "web"]
  },
  {
    "id": "vm-456",
    "name": "db-server-01",
    "provider": "vsphere",
    "status": "running",
    "power_state": "poweredOn",
    "cpu_count": 8,
    "memory_mb": 16384,
    "os": "Red Hat Enterprise Linux 8",
    "disk_gb": 500,
    "ip_address": "10.0.1.51",
    "datacenter": "DC1",
    "cluster": "Cluster1",
    "tags": ["production", "database"]
  }
]
```

**Implementation:**
```go
func (s *Server) handleVMsList(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Provider string            `json:"provider"`
        Filter   map[string]string `json:"filter"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get provider
    provider := registry.GetProvider(req.Provider)
    if provider == nil {
        http.Error(w, "Provider not found", http.StatusNotFound)
        return
    }

    // Get stored config and connect
    config := s.store.GetProviderConfig(req.Provider)
    ctx := context.Background()

    if err := provider.Connect(ctx, config); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer provider.Disconnect()

    // Convert filter to VMFilter
    vmFilter := providers.VMFilter{
        // Map req.Filter fields to VMFilter
    }

    // List VMs
    vms, err := provider.ListVMs(ctx, vmFilter)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Convert to response format
    response := make([]map[string]interface{}, len(vms))
    for i, vm := range vms {
        response[i] = map[string]interface{}{
            "id":          vm.ID,
            "name":        vm.Name,
            "provider":    req.Provider,
            "status":      vm.Status,
            "power_state": vm.PowerState,
            "cpu_count":   vm.CPUCount,
            "memory_mb":   vm.MemoryMB,
            "os":          vm.OS,
            "disk_gb":     vm.DiskGB,
            "ip_address":  vm.IPAddress,
            "datacenter":  vm.Datacenter,
            "cluster":     vm.Cluster,
            "tags":        vm.Tags,
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Register
mux.HandleFunc("/api/vms/list", s.handleVMsList)
```

## Database Schema

### Provider Storage Table

```sql
CREATE TABLE IF NOT EXISTS providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    name TEXT,
    enabled BOOLEAN DEFAULT 1,
    connected BOOLEAN DEFAULT 0,
    config_encrypted TEXT,
    last_checked TIMESTAMP,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_provider ON providers(provider);
```

### Provider Store Interface

```go
// daemon/store/provider_store.go
package store

type ProviderStore interface {
    GetProviders() ([]ProviderInfo, error)
    GetProviderConfig(provider string) (map[string]string, error)
    SaveProvider(provider string, config map[string]string) error
    DeleteProvider(provider string) error
    UpdateProviderStatus(provider string, connected bool, err string) error
}

type SQLiteProviderStore struct {
    db *sql.DB
}

func (s *SQLiteProviderStore) GetProviders() ([]ProviderInfo, error) {
    rows, err := s.db.Query(`
        SELECT provider, name, enabled, connected,
               config_encrypted, last_checked, error
        FROM providers
        WHERE enabled = 1
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var providers []ProviderInfo
    for rows.Next() {
        var p ProviderInfo
        var configEncrypted string
        var lastChecked sql.NullString
        var errorMsg sql.NullString

        err := rows.Scan(
            &p.Provider, &p.Name, &p.Enabled, &p.Connected,
            &configEncrypted, &lastChecked, &errorMsg,
        )
        if err != nil {
            return nil, err
        }

        // Decrypt config
        p.Config = decrypt(configEncrypted)

        if lastChecked.Valid {
            p.LastChecked = lastChecked.String
        }
        if errorMsg.Valid {
            p.Error = errorMsg.String
        }

        providers = append(providers, p)
    }

    return providers, nil
}

func (s *SQLiteProviderStore) SaveProvider(provider string, config map[string]string) error {
    configEncrypted := encrypt(config)

    _, err := s.db.Exec(`
        INSERT INTO providers (provider, config_encrypted, updated_at)
        VALUES (?, ?, CURRENT_TIMESTAMP)
        ON CONFLICT(provider) DO UPDATE SET
            config_encrypted = excluded.config_encrypted,
            updated_at = CURRENT_TIMESTAMP
    `, provider, configEncrypted)

    return err
}
```

## Security Implementation

### Encryption Utilities

```go
// daemon/store/encryption.go
package store

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "io"
)

var encryptionKey = []byte("your-32-byte-encryption-key-here") // Load from env

func encrypt(config map[string]string) string {
    // Serialize config to JSON
    data, _ := json.Marshal(config)

    // Create cipher
    block, _ := aes.NewCipher(encryptionKey)
    gcm, _ := cipher.NewGCM(block)

    // Create nonce
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)

    // Encrypt
    ciphertext := gcm.Seal(nonce, nonce, data, nil)

    // Encode to base64
    return base64.StdEncoding.EncodeToString(ciphertext)
}

func decrypt(encrypted string) map[string]string {
    // Decode from base64
    data, _ := base64.StdEncoding.DecodeString(encrypted)

    // Create cipher
    block, _ := aes.NewCipher(encryptionKey)
    gcm, _ := cipher.NewGCM(block)

    // Extract nonce
    nonceSize := gcm.NonceSize()
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]

    // Decrypt
    plaintext, _ := gcm.Open(nil, nonce, ciphertext, nil)

    // Deserialize
    var config map[string]string
    json.Unmarshal(plaintext, &config)

    return config
}

func encryptSensitiveFields(config map[string]string) map[string]string {
    sensitive := []string{
        "password", "secret_key", "client_secret",
        "private_key", "access_key_secret",
    }

    result := make(map[string]string)
    for k, v := range config {
        if contains(sensitive, k) {
            result[k] = encryptValue(v)
        } else {
            result[k] = v
        }
    }
    return result
}
```

## Configuration Updates

### Update config.yaml

```yaml
# Add provider storage section
providers:
  storage:
    type: sqlite  # or "database", "vault"
    path: /var/lib/hypervisord/providers.db
    encryption_key_env: HYPERSDK_ENCRYPTION_KEY

  # Optionally pre-configure providers
  defaults:
    vsphere:
      enabled: true
      host: ${VSPHERE_HOST}
      username: ${VSPHERE_USER}
      password: ${VSPHERE_PASS}
```

## Complete Example: Minimal Backend

Here's a minimal working example to get started:

```go
// daemon/api/provider_handlers.go
package api

import (
    "context"
    "encoding/json"
    "net/http"
)

// Simple in-memory storage (replace with database)
var providers = make(map[string]map[string]string)

func (s *Server) handleProvidersList(w http.ResponseWriter, r *http.Request) {
    list := []map[string]interface{}{}
    for name, config := range providers {
        list = append(list, map[string]interface{}{
            "provider":  name,
            "enabled":   true,
            "connected": true,
            "config":    config,
        })
    }
    json.NewEncoder(w).Encode(list)
}

func (s *Server) handleProviderAdd(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Provider string            `json:"provider"`
        Config   map[string]string `json:"config"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    providers[req.Provider] = req.Config

    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "provider": req.Provider,
    })
}

func (s *Server) handleProviderTest(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Provider string `json:"provider"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Simple check
    _, exists := providers[req.Provider]

    json.NewEncoder(w).Encode(map[string]interface{}{
        "success":  exists,
        "provider": req.Provider,
    })
}

func (s *Server) handleVMsList(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Provider string `json:"provider"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Use existing provider code
    provider := s.registry.GetProvider(req.Provider)
    if provider == nil {
        http.Error(w, "Provider not found", http.StatusNotFound)
        return
    }

    ctx := context.Background()
    vms, _ := provider.ListVMs(ctx, VMFilter{})

    json.NewEncoder(w).Encode(vms)
}

// Register in server.go
func (s *Server) setupRoutes() {
    s.mux.HandleFunc("/api/providers/list", s.handleProvidersList)
    s.mux.HandleFunc("/api/providers/add", s.handleProviderAdd)
    s.mux.HandleFunc("/api/providers/test", s.handleProviderTest)
    s.mux.HandleFunc("/api/vms/list", s.handleVMsList)
}
```

## Testing the Integration

### Test Script

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"

# 1. Add provider
echo "Adding vSphere provider..."
curl -X POST $BASE_URL/api/providers/add \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "vsphere",
    "config": {
      "host": "vcenter.example.com",
      "username": "admin",
      "password": "pass123"
    }
  }'

# 2. List providers
echo -e "\n\nListing providers..."
curl $BASE_URL/api/providers/list

# 3. Test connection
echo -e "\n\nTesting connection..."
curl -X POST $BASE_URL/api/providers/test \
  -H "Content-Type: application/json" \
  -d '{"provider": "vsphere"}'

# 4. List VMs
echo -e "\n\nListing VMs..."
curl -X POST $BASE_URL/api/vms/list \
  -H "Content-Type: application/json" \
  -d '{"provider": "vsphere"}'
```

---

**Implementation Priority:**
1. ✅ Start with minimal in-memory version
2. ⏭️ Add database persistence
3. ⏭️ Add encryption
4. ⏭️ Add authentication/authorization
5. ⏭️ Add audit logging
