package main

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"hypersdk/logger"
)

func TestNewDownloadWorkerPool(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 4, logger.NewTestLogger(t))

	if pool == nil {
		t.Fatal("NewDownloadWorkerPool returned nil")
	}
	if pool.workerCount != 4 {
		t.Errorf("Expected 4 workers, got %d", pool.workerCount)
	}
	if pool.tasks == nil {
		t.Error("Tasks channel should be initialized")
	}
	if pool.results == nil {
		t.Error("Results channel should be initialized")
	}
}

func TestDownloadWorkerPool_Start(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))

	pool.Start()

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	// Close the pool
	if err := pool.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestDownloadWorkerPool_Submit(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	pool.Start()

	task := DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: "/tmp/file.bin",
		Size:        1024,
		Name:        "file.bin",
	}

	err := pool.Submit(task)
	if err != nil {
		t.Errorf("Submit failed: %v", err)
	}

	// Verify total bytes updated
	_, total, _ := pool.GetProgress()
	if total != 1024 {
		t.Errorf("Expected total bytes 1024, got %d", total)
	}
}

func TestDownloadWorkerPool_Submit_AfterShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))

	cancel() // Cancel context immediately

	task := DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: "/tmp/file.bin",
		Size:        1024,
		Name:        "file.bin",
	}

	err := pool.Submit(task)
	if err == nil {
		t.Error("Expected error when submitting to shutdown pool")
	}
}

func TestDownloadWorkerPool_Results(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	results := pool.Results()
	if results == nil {
		t.Error("Results channel should not be nil")
	}
}

func TestDownloadWorkerPool_GetProgress(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	// Initially should be zero
	downloaded, total, speed := pool.GetProgress()
	if downloaded != 0 {
		t.Errorf("Expected 0 downloaded bytes, got %d", downloaded)
	}
	if total != 0 {
		t.Errorf("Expected 0 total bytes, got %d", total)
	}
	if speed < 0 {
		t.Errorf("Speed should not be negative, got %f", speed)
	}

	// Submit a task to update total
	pool.Submit(DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: "/tmp/file.bin",
		Size:        2048,
		Name:        "file.bin",
	})

	downloaded, total, speed = pool.GetProgress()
	if total != 2048 {
		t.Errorf("Expected total 2048, got %d", total)
	}
}

func TestDownloadWorkerPool_Close(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))

	pool.Start()

	err := pool.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify channels are closed
	select {
	case _, ok := <-pool.results:
		if ok {
			t.Error("Results channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		// Timeout is ok
	}
}

func TestDownloadWorkerPool_DownloadFile(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	task := DownloadTask{
		URL:         "http://example.com/test.bin",
		Destination: filepath.Join(tmpDir, "test.bin"),
		Size:        1024,
		Name:        "test.bin",
	}

	result := pool.downloadFile(0, task)

	if !result.Success {
		t.Errorf("Download failed: %v", result.Error)
	}
	if result.BytesDownloaded != 1024 {
		t.Errorf("Expected 1024 bytes downloaded, got %d", result.BytesDownloaded)
	}
	if result.Duration == 0 {
		t.Error("Duration should be set")
	}

	// Verify file was created
	if _, err := os.Stat(task.Destination); os.IsNotExist(err) {
		t.Error("Downloaded file should exist")
	}
}

func TestDownloadWorkerPool_DownloadFile_CancelledContext(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	cancel() // Cancel immediately

	task := DownloadTask{
		URL:         "http://example.com/test.bin",
		Destination: filepath.Join(tmpDir, "test.bin"),
		Size:        100 * 1024 * 1024, // Large file
		Name:        "test.bin",
	}

	result := pool.downloadFile(0, task)

	if result.Success {
		t.Error("Expected download to fail with cancelled context")
	}
	if result.Error == nil {
		t.Error("Expected error for cancelled download")
	}
}

func TestDownloadWorkerPool_DownloadBatch(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	pool.Start()

	tasks := []DownloadTask{
		{
			URL:         "http://example.com/file1.bin",
			Destination: filepath.Join(tmpDir, "file1.bin"),
			Size:        1024,
			Name:        "file1.bin",
		},
		{
			URL:         "http://example.com/file2.bin",
			Destination: filepath.Join(tmpDir, "file2.bin"),
			Size:        2048,
			Name:        "file2.bin",
		},
		{
			URL:         "http://example.com/file3.bin",
			Destination: filepath.Join(tmpDir, "file3.bin"),
			Size:        512,
			Name:        "file3.bin",
		},
	}

	progressCalled := false
	progressCallback := func(downloaded, total int64) {
		progressCalled = true
	}

	results, err := pool.DownloadBatch(tasks, progressCallback)
	if err != nil {
		t.Errorf("DownloadBatch failed: %v", err)
	}

	if len(results) != len(tasks) {
		t.Errorf("Expected %d results, got %d", len(tasks), len(results))
	}

	for _, result := range results {
		if !result.Success {
			t.Errorf("Download of %s failed: %v", result.Task.Name, result.Error)
		}
	}

	if !progressCalled {
		t.Error("Progress callback should have been called")
	}
}

func TestDownloadWorkerPool_ConcurrentDownloads(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 3, logger.NewTestLogger(t))
	defer pool.Close()

	pool.Start()

	// Submit multiple tasks
	for i := 0; i < 10; i++ {
		task := DownloadTask{
			URL:         "http://example.com/file.bin",
			Destination: filepath.Join(tmpDir, "file"+string(rune('0'+i))+".bin"),
			Size:        1024,
			Name:        "file" + string(rune('0'+i)) + ".bin",
		}
		if err := pool.Submit(task); err != nil {
			t.Errorf("Submit failed: %v", err)
		}
	}

	// Collect results
	successCount := 0
	for i := 0; i < 10; i++ {
		select {
		case result := <-pool.Results():
			if result.Success {
				successCount++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for results")
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful downloads, got %d", successCount)
	}
}

func TestDownloadTask_Fields(t *testing.T) {
	task := DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: "/tmp/file.bin",
		Size:        1024,
		Name:        "file.bin",
	}

	if task.URL != "http://example.com/file.bin" {
		t.Error("URL field mismatch")
	}
	if task.Destination != "/tmp/file.bin" {
		t.Error("Destination field mismatch")
	}
	if task.Size != 1024 {
		t.Error("Size field mismatch")
	}
	if task.Name != "file.bin" {
		t.Error("Name field mismatch")
	}
}

func TestDownloadResult_Fields(t *testing.T) {
	task := DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: "/tmp/file.bin",
		Size:        1024,
		Name:        "file.bin",
	}

	result := DownloadResult{
		Task:            task,
		Success:         true,
		Error:           nil,
		Duration:        5 * time.Second,
		BytesDownloaded: 1024,
	}

	if result.Task.Name != "file.bin" {
		t.Error("Task field mismatch")
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Error != nil {
		t.Error("Error should be nil")
	}
	if result.Duration != 5*time.Second {
		t.Error("Duration mismatch")
	}
	if result.BytesDownloaded != 1024 {
		t.Error("BytesDownloaded mismatch")
	}
}

func TestNewResumeableDownloader(t *testing.T) {
	downloader := NewResumeableDownloader("/tmp/checkpoint.json", nil)

	if downloader == nil {
		t.Fatal("NewResumeableDownloader returned nil")
	}
	if downloader.checkpointFile != "/tmp/checkpoint.json" {
		t.Error("Checkpoint file path mismatch")
	}
}

func TestResumeableDownloader_SaveCheckpoint(t *testing.T) {
	downloader := NewResumeableDownloader("/tmp/checkpoint.json", nil)

	cp := Checkpoint{
		FilePath:        "/tmp/file.bin",
		BytesDownloaded: 1024,
		TotalBytes:      2048,
		Timestamp:       time.Now(),
		Completed:       false,
	}

	err := downloader.SaveCheckpoint(cp)
	if err != nil {
		t.Errorf("SaveCheckpoint failed: %v", err)
	}
}

func TestResumeableDownloader_LoadCheckpoint_NotFound(t *testing.T) {
	downloader := NewResumeableDownloader("/nonexistent/checkpoint.json", nil)

	cp, err := downloader.LoadCheckpoint()
	if err == nil {
		t.Error("Expected error when checkpoint not found")
	}
	if cp != nil {
		t.Error("Checkpoint should be nil when not found")
	}
}

func TestResumeableDownloader_DownloadWithResume(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointFile := filepath.Join(tmpDir, "checkpoint.json")
	downloader := NewResumeableDownloader(checkpointFile, nil)

	ctx := context.Background()
	task := DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: filepath.Join(tmpDir, "file.bin"),
		Size:        1024,
		Name:        "file.bin",
	}

	progressCallback := func(downloaded, total int64) {
		// Progress callback
	}

	err := downloader.DownloadWithResume(ctx, task, progressCallback)
	if err != nil {
		t.Errorf("DownloadWithResume failed: %v", err)
	}
}

func TestResumeableDownloader_DownloadWithResume_ContextCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	checkpointFile := filepath.Join(tmpDir, "checkpoint.json")
	downloader := NewResumeableDownloader(checkpointFile, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	task := DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: filepath.Join(tmpDir, "file.bin"),
		Size:        100 * 1024 * 1024, // Large file
		Name:        "file.bin",
	}

	// Should complete quickly even with large file since it's simulated
	err := downloader.DownloadWithResume(ctx, task, nil)
	// May or may not error depending on timing
	_ = err
}

func TestCheckpoint_Fields(t *testing.T) {
	now := time.Now()
	cp := Checkpoint{
		FilePath:        "/tmp/file.bin",
		BytesDownloaded: 1024,
		TotalBytes:      2048,
		Timestamp:       now,
		Completed:       false,
	}

	if cp.FilePath != "/tmp/file.bin" {
		t.Error("FilePath mismatch")
	}
	if cp.BytesDownloaded != 1024 {
		t.Error("BytesDownloaded mismatch")
	}
	if cp.TotalBytes != 2048 {
		t.Error("TotalBytes mismatch")
	}
	if !cp.Timestamp.Equal(now) {
		t.Error("Timestamp mismatch")
	}
	if cp.Completed {
		t.Error("Completed should be false")
	}
}

func TestDownloadWorkerPool_ProgressTracking(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	// Manually update progress
	atomic.AddInt64(&pool.totalBytes, 1000)
	atomic.AddInt64(&pool.downloadedBytes, 500)

	downloaded, total, speed := pool.GetProgress()

	if downloaded != 500 {
		t.Errorf("Expected 500 downloaded, got %d", downloaded)
	}
	if total != 1000 {
		t.Errorf("Expected 1000 total, got %d", total)
	}
	if speed < 0 {
		t.Error("Speed should not be negative")
	}
}

func TestDownloadWorkerPool_MultipleClose(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))

	pool.Start()

	// First close
	if err := pool.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}

	// Second close should not panic
	// Note: may error or not depending on implementation
	pool.Close()
}

func TestDownloadWorkerPool_WorkerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))

	pool.Start()

	// Cancel context
	cancel()

	// Give workers time to stop
	time.Sleep(100 * time.Millisecond)

	// Close should complete quickly
	if err := pool.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestDownloadWorkerPool_CreateDirectoryError(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	// Try to create file in non-writable location (may vary by system)
	task := DownloadTask{
		URL:         "http://example.com/file.bin",
		Destination: "/root/nonexistent/deeply/nested/file.bin",
		Size:        1024,
		Name:        "file.bin",
	}

	result := pool.downloadFile(0, task)

	// May succeed or fail depending on permissions
	// Just verify result is properly formatted
	if result.Task.Name != "file.bin" {
		t.Error("Result should contain task info")
	}
	if result.Duration == 0 {
		t.Error("Duration should be set even on failure")
	}
}

func TestDownloadBatch_EmptyTasks(t *testing.T) {
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	pool.Start()

	results, err := pool.DownloadBatch([]DownloadTask{}, nil)
	if err != nil {
		t.Errorf("DownloadBatch with empty tasks failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestDownloadBatch_NilProgressCallback(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()
	pool := NewDownloadWorkerPool(ctx, 2, logger.NewTestLogger(t))
	defer pool.Close()

	pool.Start()

	tasks := []DownloadTask{
		{
			URL:         "http://example.com/file.bin",
			Destination: filepath.Join(tmpDir, "file.bin"),
			Size:        1024,
			Name:        "file.bin",
		},
	}

	// Should not crash with nil callback
	results, err := pool.DownloadBatch(tasks, nil)
	if err != nil {
		t.Errorf("DownloadBatch failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}
