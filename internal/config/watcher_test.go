package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockLogger implements Logger interface for testing
type MockLogger struct {
	mu       sync.Mutex
	messages []LogMessage
	counts   struct {
		info  int64
		error int64
		debug int64
	}
}

type LogMessage struct {
	Level   string
	Message string
	Time    time.Time
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.AddInt64(&m.counts.info, 1)
	m.messages = append(m.messages, LogMessage{
		Level:   "INFO",
		Message: fmt.Sprintf(format, args...),
		Time:    time.Now(),
	})
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.AddInt64(&m.counts.error, 1)
	m.messages = append(m.messages, LogMessage{
		Level:   "ERROR",
		Message: fmt.Sprintf(format, args...),
		Time:    time.Now(),
	})
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.AddInt64(&m.counts.debug, 1)
	m.messages = append(m.messages, LogMessage{
		Level:   "DEBUG",
		Message: fmt.Sprintf(format, args...),
		Time:    time.Now(),
	})
}

func (m *MockLogger) GetMessages() []LogMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]LogMessage(nil), m.messages...)
}

func (m *MockLogger) GetCounts() (info, error, debug int64) {
	return atomic.LoadInt64(&m.counts.info),
		atomic.LoadInt64(&m.counts.error),
		atomic.LoadInt64(&m.counts.debug)
}

// TestFileWatcher_Creation tests the creation of file watcher
func TestFileWatcher_Creation(t *testing.T) {
	manager := NewManager()

	t.Run("with default logger", func(t *testing.T) {
		fw := NewFileWatcher(manager, nil)
		if fw == nil {
			t.Fatal("expected file watcher to be created")
		}
		if fw.manager != manager {
			t.Error("expected manager to be set")
		}
		if fw.interval != 2*time.Second {
			t.Errorf("expected default interval of 2s, got %v", fw.interval)
		}
		fw.Stop()
	})

	t.Run("with custom logger", func(t *testing.T) {
		logger := &MockLogger{}
		fw := NewFileWatcher(manager, logger)
		if fw == nil {
			t.Fatal("expected file watcher to be created")
		}
		if fw.logger != logger {
			t.Error("expected custom logger to be set")
		}
		fw.Stop()
	})
}

// TestFileWatcher_Watch tests basic file watching functionality
func TestFileWatcher_Watch(t *testing.T) {
	// Create a temporary config file
	tmpDir, err := ioutil.TempDir("", "graft-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "test-config.yaml")
	configContent := `
performance:
  cache:
    expression_cache_size: 1000
`
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	logger := &MockLogger{}
	fw := NewFileWatcher(manager, logger)
	fw.SetInterval(100 * time.Millisecond) // Fast interval for testing

	t.Run("watch existing file", func(t *testing.T) {
		err := fw.Watch(configPath)
		if err != nil {
			t.Fatalf("failed to watch file: %v", err)
		}

		// Wait for initial watch to start
		time.Sleep(50 * time.Millisecond)

		messages := logger.GetMessages()
		found := false
		for _, msg := range messages {
			if msg.Level == "INFO" && strings.Contains(msg.Message, "Starting to watch config file") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected log message about starting to watch")
		}

		fw.Stop()
	})

	t.Run("watch non-existent file", func(t *testing.T) {
		fw2 := NewFileWatcher(manager, logger)
		err := fw2.Watch(filepath.Join(tmpDir, "non-existent.yaml"))
		if err == nil {
			t.Error("expected error watching non-existent file")
			fw2.Stop()
		}
	})
}

// TestFileWatcher_FileModification tests file modification detection
func TestFileWatcher_FileModification(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "graft-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "test-config.yaml")
	initialContent := `
performance:
  cache:
    expression_cache_size: 1000
`
	if err := ioutil.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	if err := manager.Load(configPath); err != nil {
		t.Fatal(err)
	}

	logger := &MockLogger{}
	fw := NewFileWatcher(manager, logger)
	fw.SetInterval(100 * time.Millisecond)

	if err := fw.Watch(configPath); err != nil {
		t.Fatal(err)
	}

	// Wait for watcher to start
	time.Sleep(150 * time.Millisecond)

	// Modify the file
	modifiedContent := `
performance:
  cache:
    expression_cache_size: 2000
`
	// Touch the file with a slight delay to ensure mod time changes
	time.Sleep(10 * time.Millisecond)
	if err := ioutil.WriteFile(configPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for change detection
	time.Sleep(200 * time.Millisecond)

	fw.Stop()

	// Check logs
	messages := logger.GetMessages()
	foundChange := false
	foundReload := false

	for _, msg := range messages {
		if msg.Level == "INFO" && strings.Contains(msg.Message, "Config file changed") {
			foundChange = true
		}
		if msg.Level == "INFO" && strings.Contains(msg.Message, "Config reloaded successfully") {
			foundReload = true
		}
	}

	if !foundChange {
		t.Error("expected log message about config file change")
	}
	if !foundReload {
		t.Error("expected log message about successful reload")
	}

	// Verify the config was actually reloaded
	currentConfig := manager.Get()
	if currentConfig.Performance.Cache.ExpressionCacheSize != 2000 {
		t.Errorf("expected cache size to be 2000, got %d",
			currentConfig.Performance.Cache.ExpressionCacheSize)
	}
}

// TestFileWatcher_FileDeleted tests behavior when watched file is deleted
func TestFileWatcher_FileDeleted(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "graft-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := ioutil.WriteFile(configPath, []byte("test: value"), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	logger := &MockLogger{}
	fw := NewFileWatcher(manager, logger)
	fw.SetInterval(100 * time.Millisecond)

	if err := fw.Watch(configPath); err != nil {
		t.Fatal(err)
	}

	// Wait for watcher to start
	time.Sleep(150 * time.Millisecond)

	// Delete the file
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}

	// Wait for deletion detection
	time.Sleep(200 * time.Millisecond)

	fw.Stop()

	// Check logs
	messages := logger.GetMessages()
	foundDeletion := false

	for _, msg := range messages {
		if msg.Level == "ERROR" && strings.Contains(msg.Message, "Config file no longer exists") {
			foundDeletion = true
			break
		}
	}

	if !foundDeletion {
		t.Error("expected error log about deleted file")
	}
}

// TestFileWatcher_DirectoryWatch tests directory watching functionality
func TestFileWatcher_DirectoryWatch(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "graft-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial config files
	config1 := filepath.Join(tmpDir, "config1.yaml")
	config2 := filepath.Join(tmpDir, "config2.yaml")

	if err := ioutil.WriteFile(config1, []byte("test: 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(config2, []byte("test: 2"), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	logger := &MockLogger{}
	fw := NewFileWatcher(manager, logger)
	fw.SetInterval(100 * time.Millisecond)

	t.Run("watch directory with pattern", func(t *testing.T) {
		err := fw.WatchDirectory(tmpDir, "*.yaml")
		if err != nil {
			t.Fatalf("failed to watch directory: %v", err)
		}

		// Wait for initial scan
		time.Sleep(150 * time.Millisecond)

		// Create a new file
		config3 := filepath.Join(tmpDir, "config3.yaml")
		if err := ioutil.WriteFile(config3, []byte("test: 3"), 0644); err != nil {
			t.Fatal(err)
		}

		// Modify existing file
		time.Sleep(10 * time.Millisecond)
		if err := ioutil.WriteFile(config1, []byte("test: modified"), 0644); err != nil {
			t.Fatal(err)
		}

		// Wait for changes to be detected
		time.Sleep(200 * time.Millisecond)

		// Delete a file
		if err := os.Remove(config2); err != nil {
			t.Fatal(err)
		}

		// Wait for deletion to be detected
		time.Sleep(200 * time.Millisecond)

		fw.Stop()

		// Check logs
		messages := logger.GetMessages()
		foundModified := false
		foundDeleted := false

		for _, msg := range messages {
			if msg.Level == "INFO" && strings.Contains(msg.Message, "Config file changed in directory") {
				foundModified = true
			}
			if msg.Level == "INFO" && strings.Contains(msg.Message, "Config file deleted") {
				foundDeleted = true
			}
		}

		if !foundModified {
			t.Error("expected log about modified file")
		}
		if !foundDeleted {
			t.Error("expected log about deleted file")
		}
	})
}

// TestFileWatcher_ConcurrentAccess tests thread safety
func TestFileWatcher_ConcurrentAccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "graft-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := ioutil.WriteFile(configPath, []byte("test: value"), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	logger := &MockLogger{}
	fw := NewFileWatcher(manager, logger)
	fw.SetInterval(50 * time.Millisecond)

	if err := fw.Watch(configPath); err != nil {
		t.Fatal(err)
	}

	// Run concurrent modifications
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Modify file
			content := fmt.Sprintf("test: %d", id)
			if err := ioutil.WriteFile(configPath, []byte(content), 0644); err != nil {
				errors <- err
				return
			}

			time.Sleep(60 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent modification error: %v", err)
	}

	// Let watcher process changes
	time.Sleep(200 * time.Millisecond)

	fw.Stop()

	// Verify no panics and logger received messages
	info, errCount, _ := logger.GetCounts()
	if info == 0 {
		t.Error("expected info messages")
	}
	if errCount > 0 {
		t.Logf("Encountered %d errors during concurrent test", errCount)
	}
}

// TestFileWatcher_LargeConfigPerformance tests performance with large configs
func TestFileWatcher_LargeConfigPerformance(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "graft-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a large config file
	configPath := filepath.Join(tmpDir, "large-config.yaml")
	largeConfig := generateLargeConfig(1000) // 1000 entries

	if err := ioutil.WriteFile(configPath, []byte(largeConfig), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	logger := &MockLogger{}
	fw := NewFileWatcher(manager, logger)
	fw.SetInterval(100 * time.Millisecond)

	// Measure initial load time
	start := time.Now()
	if err := manager.Load(configPath); err != nil {
		t.Fatal(err)
	}
	loadDuration := time.Since(start)
	t.Logf("Initial load of large config took: %v", loadDuration)

	// Start watching
	if err := fw.Watch(configPath); err != nil {
		t.Fatal(err)
	}

	// Wait for watcher to start
	time.Sleep(150 * time.Millisecond)

	// Modify the large file
	modifiedConfig := generateLargeConfig(1100) // Add 100 more entries
	reloadStart := time.Now()

	if err := ioutil.WriteFile(configPath, []byte(modifiedConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for reload
	time.Sleep(300 * time.Millisecond)
	reloadDuration := time.Since(reloadStart)

	fw.Stop()

	t.Logf("Reload of modified large config took: %v", reloadDuration)

	// Check that reload happened
	messages := logger.GetMessages()
	foundReload := false
	for _, msg := range messages {
		if msg.Level == "INFO" && strings.Contains(msg.Message, "Config reloaded successfully") {
			foundReload = true
			break
		}
	}

	if !foundReload {
		t.Error("expected successful reload of large config")
	}

	// Performance assertions
	if loadDuration > 5*time.Second {
		t.Errorf("initial load too slow: %v", loadDuration)
	}
	if reloadDuration > 5*time.Second {
		t.Errorf("reload too slow: %v", reloadDuration)
	}
}

// TestChangeDetector tests configuration change detection
func TestChangeDetector(t *testing.T) {
	oldConfig := &Config{
		Performance: PerformanceConfig{
			Cache: CacheConfig{
				ExpressionCacheSize: 1000,
			},
			Concurrency: ConcurrencyConfig{
				MaxWorkers: 4,
			},
		},
		Features: map[string]bool{
			"feature1": true,
			"feature2": false,
			"feature3": true,
		},
	}

	t.Run("detect modified values", func(t *testing.T) {
		newConfig := &Config{
			Performance: PerformanceConfig{
				Cache: CacheConfig{
					ExpressionCacheSize: 2000,
				},
				Concurrency: ConcurrencyConfig{
					MaxWorkers: 8,
				},
			},
			Features: map[string]bool{
				"feature1": true,
				"feature2": true, // Changed
				"feature3": true,
			},
		}

		detector := NewChangeDetector(oldConfig, newConfig)
		changes := detector.DetectChanges()

		expectedChanges := 3 // cache size, max workers, feature2
		if len(changes) != expectedChanges {
			t.Errorf("expected %d changes, got %d", expectedChanges, len(changes))
		}

		// Verify specific changes
		foundCacheChange := false
		foundWorkerChange := false
		foundFeatureChange := false

		for _, change := range changes {
			switch change.Path {
			case "performance.cache.expression_cache_size":
				foundCacheChange = true
				if change.Type != ChangeTypeModify {
					t.Errorf("expected modify type for cache size")
				}
				if change.OldValue != 1000 || change.NewValue != 2000 {
					t.Errorf("incorrect values for cache size change")
				}
			case "performance.concurrency.max_workers":
				foundWorkerChange = true
			case "features.feature2":
				foundFeatureChange = true
				if change.OldValue != false || change.NewValue != true {
					t.Errorf("incorrect values for feature2 change")
				}
			}
		}

		if !foundCacheChange || !foundWorkerChange || !foundFeatureChange {
			t.Error("expected changes not found")
		}
	})

	t.Run("detect added features", func(t *testing.T) {
		newConfig := &Config{
			Performance: oldConfig.Performance,
			Features: map[string]bool{
				"feature1": true,
				"feature2": false,
				"feature3": true,
				"feature4": true, // New
			},
		}

		detector := NewChangeDetector(oldConfig, newConfig)
		changes := detector.DetectChanges()

		foundNewFeature := false
		for _, change := range changes {
			if change.Path == "features.feature4" && change.Type == ChangeTypeAdd {
				foundNewFeature = true
				if change.NewValue != true {
					t.Error("incorrect value for new feature")
				}
			}
		}

		if !foundNewFeature {
			t.Error("expected new feature to be detected")
		}
	})

	t.Run("detect deleted features", func(t *testing.T) {
		newConfig := &Config{
			Performance: oldConfig.Performance,
			Features: map[string]bool{
				"feature1": true,
				// feature2 deleted
				"feature3": true,
			},
		}

		detector := NewChangeDetector(oldConfig, newConfig)
		changes := detector.DetectChanges()

		foundDeletedFeature := false
		for _, change := range changes {
			if change.Path == "features.feature2" && change.Type == ChangeTypeDelete {
				foundDeletedFeature = true
				if change.OldValue != false {
					t.Error("incorrect old value for deleted feature")
				}
			}
		}

		if !foundDeletedFeature {
			t.Error("expected deleted feature to be detected")
		}
	})
}

// Helper functions

func generateLargeConfig(numEntries int) string {
	config := "features:\n"
	for i := 0; i < numEntries; i++ {
		config += fmt.Sprintf("  feature_%d: %v\n", i, i%2 == 0)
	}
	config += "\nperformance:\n  cache:\n    expression_cache_size: 10000\n"
	return config
}

// Benchmarks

func BenchmarkFileWatcher_CheckChanges(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "graft-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "bench-config.yaml")
	if err := ioutil.WriteFile(configPath, []byte("test: value"), 0644); err != nil {
		b.Fatal(err)
	}

	manager := NewManager()
	fw := NewFileWatcher(manager, nil)
	fw.watchedPath = configPath
	fw.lastModTime = time.Now().Add(-time.Hour)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = fw.checkForChanges()
	}
}

func BenchmarkChangeDetector_LargeConfig(b *testing.B) {
	// Create configs with many features
	oldFeatures := make(map[string]bool)
	newFeatures := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("feature_%d", i)
		oldFeatures[key] = i%2 == 0
		newFeatures[key] = i%3 == 0 // Different pattern for changes
	}

	oldConfig := &Config{Features: oldFeatures}
	newConfig := &Config{Features: newFeatures}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		detector := NewChangeDetector(oldConfig, newConfig)
		_ = detector.DetectChanges()
	}
}
