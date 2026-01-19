// SPDX-License-Identifier: LGPL-3.0-or-later

package cache

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Backend != "memory" {
		t.Errorf("expected backend memory, got %s", config.Backend)
	}

	if config.MaxMemorySize != 100*1024*1024 {
		t.Errorf("expected max memory 100MB, got %d", config.MaxMemorySize)
	}

	if config.DefaultTTL != 1*time.Hour {
		t.Errorf("expected default TTL 1h, got %v", config.DefaultTTL)
	}

	if !config.EnableFallback {
		t.Error("expected fallback to be enabled")
	}
}

func TestNewCacheMemory(t *testing.T) {
	config := DefaultConfig()
	cache, err := NewCache(config)

	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	if cache == nil {
		t.Fatal("expected cache to be created")
	}

	defer cache.Close()
}

func TestNewCacheInvalidBackend(t *testing.T) {
	config := DefaultConfig()
	config.Backend = "invalid"

	_, err := NewCache(config)
	if err == nil {
		t.Error("expected error with invalid backend")
	}
}

func TestMemoryCacheSetGet(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Set value
	err := cache.Set(ctx, "key1", "value1", 1*time.Hour)
	if err != nil {
		t.Errorf("failed to set value: %v", err)
	}

	// Get value
	value, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("failed to get value: %v", err)
	}

	if value != "value1" {
		t.Errorf("expected value1, got %v", value)
	}
}

func TestMemoryCacheGetMiss(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	_, err := cache.Get(ctx, "nonexistent")
	if err != ErrCacheMiss {
		t.Errorf("expected cache miss error, got %v", err)
	}
}

func TestMemoryCacheExpiration(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Set value with short TTL
	cache.Set(ctx, "key1", "value1", 100*time.Millisecond)

	// Get immediately
	value, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("failed to get value: %v", err)
	}
	if value != "value1" {
		t.Errorf("expected value1, got %v", value)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Get after expiration
	_, err = cache.Get(ctx, "key1")
	if err != ErrCacheMiss {
		t.Errorf("expected cache miss after expiration, got %v", err)
	}
}

func TestMemoryCacheDelete(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Set and delete
	cache.Set(ctx, "key1", "value1", 1*time.Hour)
	cache.Delete(ctx, "key1")

	// Verify deleted
	_, err := cache.Get(ctx, "key1")
	if err != ErrCacheMiss {
		t.Errorf("expected cache miss after delete, got %v", err)
	}
}

func TestMemoryCacheExists(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Check non-existent key
	exists, err := cache.Exists(ctx, "key1")
	if err != nil {
		t.Errorf("failed to check existence: %v", err)
	}
	if exists {
		t.Error("expected key to not exist")
	}

	// Set value
	cache.Set(ctx, "key1", "value1", 1*time.Hour)

	// Check existent key
	exists, err = cache.Exists(ctx, "key1")
	if err != nil {
		t.Errorf("failed to check existence: %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}
}

func TestMemoryCacheClear(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Set multiple values
	cache.Set(ctx, "key1", "value1", 1*time.Hour)
	cache.Set(ctx, "key2", "value2", 1*time.Hour)
	cache.Set(ctx, "key3", "value3", 1*time.Hour)

	// Clear
	cache.Clear(ctx)

	// Verify all cleared
	_, err := cache.Get(ctx, "key1")
	if err != ErrCacheMiss {
		t.Error("expected cache miss after clear")
	}

	_, err = cache.Get(ctx, "key2")
	if err != ErrCacheMiss {
		t.Error("expected cache miss after clear")
	}

	_, err = cache.Get(ctx, "key3")
	if err != ErrCacheMiss {
		t.Error("expected cache miss after clear")
	}
}

func TestMemoryCacheStats(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Perform operations
	cache.Set(ctx, "key1", "value1", 1*time.Hour)
	cache.Set(ctx, "key2", "value2", 1*time.Hour)
	cache.Get(ctx, "key1") // hit
	cache.Get(ctx, "key3") // miss
	cache.Delete(ctx, "key2")

	stats := cache.Stats()

	if stats.Sets != 2 {
		t.Errorf("expected 2 sets, got %d", stats.Sets)
	}

	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}

	if stats.Deletes != 1 {
		t.Errorf("expected 1 delete, got %d", stats.Deletes)
	}
}

func TestMemoryCacheEviction(t *testing.T) {
	config := DefaultConfig()
	config.MaxMemorySize = 1024 // 1 KB limit

	cache := newMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	// Create large values to exceed limit
	largeValue := make([]byte, 512)
	for i := range largeValue {
		largeValue[i] = 'A'
	}

	// Set multiple large values
	cache.Set(ctx, "key1", string(largeValue), 1*time.Hour)
	cache.Set(ctx, "key2", string(largeValue), 1*time.Hour)
	cache.Set(ctx, "key3", string(largeValue), 1*time.Hour) // Should trigger eviction

	stats := cache.Stats()
	if stats.Evictions == 0 {
		t.Error("expected evictions to occur")
	}
}

func TestMemoryCacheExpirationLoop(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Set values with short TTL
	cache.Set(ctx, "key1", "value1", 1*time.Millisecond)
	cache.Set(ctx, "key2", "value2", 1*time.Millisecond)

	// Wait for expiration (eviction loop runs every minute, so check manually)
	time.Sleep(10 * time.Millisecond)

	// Try to get expired keys - this verifies expiration works
	_, err1 := cache.Get(ctx, "key1")
	_, err2 := cache.Get(ctx, "key2")

	if err1 != ErrCacheMiss {
		t.Errorf("expected cache miss for key1, got %v", err1)
	}

	if err2 != ErrCacheMiss {
		t.Errorf("expected cache miss for key2, got %v", err2)
	}
}

func TestMemoryCacheComplexTypes(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Test map
	testMap := map[string]interface{}{
		"name": "test",
		"age":  30,
		"tags": []string{"a", "b", "c"},
	}

	cache.Set(ctx, "map", testMap, 1*time.Hour)
	value, err := cache.Get(ctx, "map")
	if err != nil {
		t.Errorf("failed to get map: %v", err)
	}

	resultMap, ok := value.(map[string]interface{})
	if !ok {
		t.Error("expected map type")
	}

	if resultMap["name"] != "test" {
		t.Errorf("expected name test, got %v", resultMap["name"])
	}

	// Test number - JSON may preserve type or convert to float64
	testNumber := 42
	cache.Set(ctx, "number", testNumber, 1*time.Hour)
	value, err = cache.Get(ctx, "number")
	if err != nil {
		t.Errorf("failed to get number: %v", err)
	}

	// Verify we got a number back (could be int or float64)
	switch v := value.(type) {
	case int:
		if v != 42 {
			t.Errorf("expected 42, got %v", v)
		}
	case float64:
		if v != 42.0 {
			t.Errorf("expected 42.0, got %v", v)
		}
	default:
		t.Errorf("expected number type, got %T", value)
	}
}

func TestMemoryCacheConcurrency(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Concurrent writes and reads
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Set(ctx, "key", j, 1*time.Hour)
				cache.Get(ctx, "key")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := cache.Stats()
	if stats.Sets != 1000 {
		t.Errorf("expected 1000 sets, got %d", stats.Sets)
	}
}

func TestMemoryCacheNilConfig(t *testing.T) {
	cache, err := NewCache(nil)
	if err != nil {
		t.Fatalf("failed to create cache with nil config: %v", err)
	}

	if cache == nil {
		t.Fatal("expected cache to be created")
	}

	defer cache.Close()
}

func TestMemoryCacheMultipleOperations(t *testing.T) {
	cache := newMemoryCache(DefaultConfig())
	defer cache.Close()

	ctx := context.Background()

	// Test sequence of operations
	testCases := []struct {
		key   string
		value interface{}
		ttl   time.Duration
	}{
		{"str", "test string", 1 * time.Hour},
		{"int", 42, 1 * time.Hour},
		{"float", 3.14, 1 * time.Hour},
		{"bool", true, 1 * time.Hour},
	}

	// Set all
	for _, tc := range testCases {
		if err := cache.Set(ctx, tc.key, tc.value, tc.ttl); err != nil {
			t.Errorf("failed to set %s: %v", tc.key, err)
		}
	}

	// Get all
	for _, tc := range testCases {
		value, err := cache.Get(ctx, tc.key)
		if err != nil {
			t.Errorf("failed to get %s: %v", tc.key, err)
		}

		// Check type
		switch tc.value.(type) {
		case string:
			if value != tc.value {
				t.Errorf("expected %v, got %v", tc.value, value)
			}
		}
	}

	// Verify exists
	for _, tc := range testCases {
		exists, err := cache.Exists(ctx, tc.key)
		if err != nil {
			t.Errorf("failed to check existence of %s: %v", tc.key, err)
		}
		if !exists {
			t.Errorf("expected %s to exist", tc.key)
		}
	}

	// Delete all
	for _, tc := range testCases {
		if err := cache.Delete(ctx, tc.key); err != nil {
			t.Errorf("failed to delete %s: %v", tc.key, err)
		}
	}

	// Verify deleted
	for _, tc := range testCases {
		exists, err := cache.Exists(ctx, tc.key)
		if err != nil {
			t.Errorf("failed to check existence of %s: %v", tc.key, err)
		}
		if exists {
			t.Errorf("expected %s to not exist after delete", tc.key)
		}
	}
}
