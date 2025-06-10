package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileWatcher watches configuration files for changes and triggers reloads
type FileWatcher struct {
	manager     *Manager
	watchedPath string
	lastModTime time.Time
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	interval    time.Duration
	logger      Logger
}

// Logger interface for file watcher logging
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

// DefaultLogger implements a simple logger using Go's standard log package
type DefaultLogger struct{}

func (l DefaultLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (l DefaultLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

func (l DefaultLogger) Debugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(manager *Manager, logger Logger) *FileWatcher {
	if logger == nil {
		logger = DefaultLogger{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &FileWatcher{
		manager:  manager,
		ctx:      ctx,
		cancel:   cancel,
		interval: 2 * time.Second, // Default poll interval
		logger:   logger,
	}
}

// Watch starts watching the configuration file for changes
func (fw *FileWatcher) Watch(configPath string) error {
	// Expand the path
	expandedPath, err := expandPath(configPath)
	if err != nil {
		return fmt.Errorf("expanding config path: %w", err)
	}

	// Check if file exists
	stat, err := os.Stat(expandedPath)
	if err != nil {
		return fmt.Errorf("checking config file: %w", err)
	}

	fw.watchedPath = expandedPath
	fw.lastModTime = stat.ModTime()

	fw.logger.Infof("Starting to watch config file: %s", expandedPath)

	// Start the watcher goroutine
	fw.wg.Add(1)
	go fw.watchLoop()

	return nil
}

// Stop stops watching the configuration file
func (fw *FileWatcher) Stop() {
	fw.logger.Infof("Stopping config file watcher")
	fw.cancel()
	fw.wg.Wait()
}

// SetInterval sets the polling interval for file changes
func (fw *FileWatcher) SetInterval(interval time.Duration) {
	fw.interval = interval
}

// watchLoop is the main watching loop
func (fw *FileWatcher) watchLoop() {
	defer fw.wg.Done()

	ticker := time.NewTicker(fw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.ctx.Done():
			fw.logger.Debugf("Config watcher stopped")
			return

		case <-ticker.C:
			if err := fw.checkForChanges(); err != nil {
				fw.logger.Errorf("Error checking for config changes: %v", err)
			}
		}
	}
}

// checkForChanges checks if the configuration file has been modified
func (fw *FileWatcher) checkForChanges() error {
	stat, err := os.Stat(fw.watchedPath)
	if err != nil {
		if os.IsNotExist(err) {
			fw.logger.Errorf("Config file no longer exists: %s", fw.watchedPath)
			return nil
		}
		return err
	}

	modTime := stat.ModTime()
	if modTime.After(fw.lastModTime) {
		fw.logger.Infof("Config file changed, reloading: %s", fw.watchedPath)

		// Reload the configuration
		if err := fw.reloadConfig(); err != nil {
			fw.logger.Errorf("Failed to reload config: %v", err)
			return err
		}

		fw.lastModTime = modTime
		fw.logger.Infof("Config reloaded successfully")
	}

	return nil
}

// reloadConfig reloads the configuration from the watched file
func (fw *FileWatcher) reloadConfig() error {
	// Try to load the new configuration
	if err := fw.manager.Load(fw.watchedPath); err != nil {
		fw.logger.Errorf("Failed to load new config, keeping current: %v", err)
		return err
	}

	fw.logger.Infof("Config hot-reload completed successfully")
	return nil
}

// WatchDirectory watches a directory for configuration file changes
func (fw *FileWatcher) WatchDirectory(dirPath string, pattern string) error {
	// Expand the path
	expandedPath, err := expandPath(dirPath)
	if err != nil {
		return fmt.Errorf("expanding directory path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(expandedPath); err != nil {
		return fmt.Errorf("checking directory: %w", err)
	}

	fw.watchedPath = expandedPath
	fw.logger.Infof("Starting to watch directory: %s (pattern: %s)", expandedPath, pattern)

	// Start the directory watcher goroutine
	fw.wg.Add(1)
	go fw.watchDirectoryLoop(pattern)

	return nil
}

// watchDirectoryLoop watches a directory for file changes
func (fw *FileWatcher) watchDirectoryLoop(pattern string) {
	defer fw.wg.Done()

	// Track files and their modification times
	fileModTimes := make(map[string]time.Time)

	// Initial scan
	fw.scanDirectory(fw.watchedPath, pattern, fileModTimes)

	ticker := time.NewTicker(fw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.ctx.Done():
			fw.logger.Debugf("Directory watcher stopped")
			return

		case <-ticker.C:
			if err := fw.checkDirectoryChanges(pattern, fileModTimes); err != nil {
				fw.logger.Errorf("Error checking directory changes: %v", err)
			}
		}
	}
}

// scanDirectory scans a directory for files matching the pattern
func (fw *FileWatcher) scanDirectory(dirPath, pattern string, fileModTimes map[string]time.Time) {
	matches, err := filepath.Glob(filepath.Join(dirPath, pattern))
	if err != nil {
		fw.logger.Errorf("Error globbing directory: %v", err)
		return
	}

	for _, match := range matches {
		if stat, err := os.Stat(match); err == nil && !stat.IsDir() {
			fileModTimes[match] = stat.ModTime()
		}
	}
}

// checkDirectoryChanges checks for changes in the watched directory
func (fw *FileWatcher) checkDirectoryChanges(pattern string, fileModTimes map[string]time.Time) error {
	matches, err := filepath.Glob(filepath.Join(fw.watchedPath, pattern))
	if err != nil {
		return err
	}

	// Check for new or modified files
	for _, match := range matches {
		stat, err := os.Stat(match)
		if err != nil {
			continue
		}

		if stat.IsDir() {
			continue
		}

		lastModTime, exists := fileModTimes[match]
		if !exists || stat.ModTime().After(lastModTime) {
			fw.logger.Infof("Config file changed in directory: %s", match)

			// For directory watching, we might want to reload a specific config
			// or trigger a different action. For now, just log it.
			fileModTimes[match] = stat.ModTime()
		}
	}

	// Check for deleted files
	for filePath := range fileModTimes {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fw.logger.Infof("Config file deleted: %s", filePath)
			delete(fileModTimes, filePath)
		}
	}

	return nil
}

// ConfigChangeEvent represents a configuration change event
type ConfigChangeEvent struct {
	Type     ChangeType
	Path     string
	OldValue interface{}
	NewValue interface{}
	Time     time.Time
}

// ChangeType represents the type of configuration change
type ChangeType string

const (
	ChangeTypeAdd    ChangeType = "add"
	ChangeTypeModify ChangeType = "modify"
	ChangeTypeDelete ChangeType = "delete"
)

// ChangeDetector detects specific changes in configuration
type ChangeDetector struct {
	oldConfig *Config
	newConfig *Config
}

// NewChangeDetector creates a new change detector
func NewChangeDetector(oldConfig, newConfig *Config) *ChangeDetector {
	return &ChangeDetector{
		oldConfig: oldConfig,
		newConfig: newConfig,
	}
}

// DetectChanges detects what has changed between configurations
func (cd *ChangeDetector) DetectChanges() []ConfigChangeEvent {
	var events []ConfigChangeEvent
	now := time.Now()

	// Check cache configuration changes
	if cd.oldConfig.Performance.Cache.ExpressionCacheSize != cd.newConfig.Performance.Cache.ExpressionCacheSize {
		events = append(events, ConfigChangeEvent{
			Type:     ChangeTypeModify,
			Path:     "performance.cache.expression_cache_size",
			OldValue: cd.oldConfig.Performance.Cache.ExpressionCacheSize,
			NewValue: cd.newConfig.Performance.Cache.ExpressionCacheSize,
			Time:     now,
		})
	}

	// Check concurrency configuration changes
	if cd.oldConfig.Performance.Concurrency.MaxWorkers != cd.newConfig.Performance.Concurrency.MaxWorkers {
		events = append(events, ConfigChangeEvent{
			Type:     ChangeTypeModify,
			Path:     "performance.concurrency.max_workers",
			OldValue: cd.oldConfig.Performance.Concurrency.MaxWorkers,
			NewValue: cd.newConfig.Performance.Concurrency.MaxWorkers,
			Time:     now,
		})
	}

	// Check feature flag changes
	for featureName, newValue := range cd.newConfig.Features {
		if oldValue, exists := cd.oldConfig.Features[featureName]; exists {
			if oldValue != newValue {
				events = append(events, ConfigChangeEvent{
					Type:     ChangeTypeModify,
					Path:     fmt.Sprintf("features.%s", featureName),
					OldValue: oldValue,
					NewValue: newValue,
					Time:     now,
				})
			}
		} else {
			events = append(events, ConfigChangeEvent{
				Type:     ChangeTypeAdd,
				Path:     fmt.Sprintf("features.%s", featureName),
				NewValue: newValue,
				Time:     now,
			})
		}
	}

	// Check for deleted feature flags
	for featureName, oldValue := range cd.oldConfig.Features {
		if _, exists := cd.newConfig.Features[featureName]; !exists {
			events = append(events, ConfigChangeEvent{
				Type:     ChangeTypeDelete,
				Path:     fmt.Sprintf("features.%s", featureName),
				OldValue: oldValue,
				Time:     now,
			})
		}
	}

	return events
}
