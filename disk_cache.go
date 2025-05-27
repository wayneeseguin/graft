package spruce

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DiskCache provides persistent caching to disk
type DiskCache struct {
	config    DiskCacheConfig
	entries   map[string]*CacheEntry
	mu        sync.RWMutex
	metrics   DiskCacheMetrics
}

// DiskCacheConfig configures the disk cache
type DiskCacheConfig struct {
	StoragePath string        // Directory for cache files
	MaxSize     int           // Maximum number of entries
	TTL         time.Duration // Time to live for entries
	Persistence bool          // Enable persistent storage
	FilePrefix  string        // Prefix for cache files
}

// DiskCacheMetrics tracks disk cache performance
type DiskCacheMetrics struct {
	Hits        int64
	Misses      int64
	Writes      int64
	Reads       int64
	Deletes     int64
	Size        int64
	DiskUsage   int64
	LoadTime    time.Duration
	SaveTime    time.Duration
	mu          sync.RWMutex
}

// NewDiskCache creates a new disk cache
func NewDiskCache(config DiskCacheConfig) (*DiskCache, error) {
	// Set defaults
	if config.FilePrefix == "" {
		config.FilePrefix = "spruce_cache"
	}
	if config.TTL == 0 {
		config.TTL = time.Hour
	}
	if config.MaxSize == 0 {
		config.MaxSize = 10000
	}
	
	// Create storage directory if it doesn't exist
	if config.Persistence {
		if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %v", err)
		}
	}
	
	dc := &DiskCache{
		config:  config,
		entries: make(map[string]*CacheEntry),
	}
	
	// Load existing cache if persistence is enabled
	if config.Persistence {
		if err := dc.LoadFromDisk(); err != nil {
			return nil, fmt.Errorf("failed to load cache from disk: %v", err)
		}
	}
	
	return dc, nil
}

// Get retrieves an entry from the disk cache
func (dc *DiskCache) Get(key string) (*CacheEntry, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	entry, found := dc.entries[key]
	if !found {
		dc.metrics.incrementMisses()
		return nil, false
	}
	
	// Check TTL
	if time.Since(entry.Timestamp) > entry.TTL {
		// Entry expired, remove it
		dc.mu.RUnlock()
		dc.mu.Lock()
		delete(dc.entries, key)
		dc.mu.Unlock()
		dc.mu.RLock()
		
		dc.metrics.incrementMisses()
		return nil, false
	}
	
	// Update access time and hit count
	entry.LastAccessed = time.Now()
	entry.HitCount++
	
	dc.metrics.incrementHits()
	return entry, true
}

// Set stores an entry in the disk cache
func (dc *DiskCache) Set(key string, entry *CacheEntry) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// Check if we need to evict entries
	if len(dc.entries) >= dc.config.MaxSize {
		dc.evictLRU()
	}
	
	dc.entries[key] = entry
	dc.metrics.incrementWrites()
	dc.metrics.setSize(int64(len(dc.entries)))
	
	// Persist to disk if enabled
	if dc.config.Persistence {
		go dc.persistEntry(key, entry)
	}
}

// Delete removes an entry from the disk cache
func (dc *DiskCache) Delete(key string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	if _, found := dc.entries[key]; found {
		delete(dc.entries, key)
		dc.metrics.incrementDeletes()
		dc.metrics.setSize(int64(len(dc.entries)))
		
		// Remove from disk if persistence is enabled
		if dc.config.Persistence {
			go dc.removeFromDisk(key)
		}
	}
}

// Clear removes all entries from the cache
func (dc *DiskCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	dc.entries = make(map[string]*CacheEntry)
	dc.metrics.setSize(0)
	
	// Clear disk storage if persistence is enabled
	if dc.config.Persistence {
		go dc.clearDisk()
	}
}

// evictLRU removes the least recently used entry
func (dc *DiskCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range dc.entries {
		if oldestKey == "" || entry.LastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccessed
		}
	}
	
	if oldestKey != "" {
		delete(dc.entries, oldestKey)
		
		if dc.config.Persistence {
			go dc.removeFromDisk(oldestKey)
		}
	}
}

// LoadFromDisk loads cache entries from persistent storage
func (dc *DiskCache) LoadFromDisk() error {
	if !dc.config.Persistence {
		return nil
	}
	
	start := time.Now()
	defer func() {
		dc.metrics.setLoadTime(time.Since(start))
	}()
	
	// Load index file
	indexPath := filepath.Join(dc.config.StoragePath, dc.config.FilePrefix+"_index.json")
	indexData, err := ioutil.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing cache
		}
		return fmt.Errorf("failed to read cache index: %v", err)
	}
	
	var index map[string]string
	if err := json.Unmarshal(indexData, &index); err != nil {
		return fmt.Errorf("failed to parse cache index: %v", err)
	}
	
	// Load individual cache files
	for key, filename := range index {
		entryPath := filepath.Join(dc.config.StoragePath, filename)
		entryData, err := ioutil.ReadFile(entryPath)
		if err != nil {
			continue // Skip corrupted entries
		}
		
		var entry CacheEntry
		if err := json.Unmarshal(entryData, &entry); err != nil {
			continue // Skip corrupted entries
		}
		
		// Check if entry is still valid
		if time.Since(entry.Timestamp) <= entry.TTL {
			dc.entries[key] = &entry
		} else {
			// Remove expired entry
			os.Remove(entryPath)
		}
	}
	
	dc.metrics.setSize(int64(len(dc.entries)))
	dc.metrics.incrementReads()
	
	return nil
}

// SaveToDisk saves all cache entries to persistent storage
func (dc *DiskCache) SaveToDisk() error {
	if !dc.config.Persistence {
		return nil
	}
	
	start := time.Now()
	defer func() {
		dc.metrics.setSaveTime(time.Since(start))
	}()
	
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	index := make(map[string]string)
	
	// Save each entry to its own file
	for key, entry := range dc.entries {
		filename := dc.generateFilename(key)
		entryPath := filepath.Join(dc.config.StoragePath, filename)
		
		entryData, err := json.Marshal(entry)
		if err != nil {
			continue // Skip entries that can't be serialized
		}
		
		if err := ioutil.WriteFile(entryPath, entryData, 0644); err != nil {
			continue // Skip entries that can't be written
		}
		
		index[key] = filename
	}
	
	// Save index file
	indexData, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("failed to marshal cache index: %v", err)
	}
	
	indexPath := filepath.Join(dc.config.StoragePath, dc.config.FilePrefix+"_index.json")
	if err := ioutil.WriteFile(indexPath, indexData, 0644); err != nil {
		return fmt.Errorf("failed to write cache index: %v", err)
	}
	
	return nil
}

// persistEntry saves a single entry to disk
func (dc *DiskCache) persistEntry(key string, entry *CacheEntry) {
	filename := dc.generateFilename(key)
	entryPath := filepath.Join(dc.config.StoragePath, filename)
	
	entryData, err := json.Marshal(entry)
	if err != nil {
		return
	}
	
	ioutil.WriteFile(entryPath, entryData, 0644)
}

// removeFromDisk removes an entry file from disk
func (dc *DiskCache) removeFromDisk(key string) {
	filename := dc.generateFilename(key)
	entryPath := filepath.Join(dc.config.StoragePath, filename)
	os.Remove(entryPath)
}

// clearDisk removes all cache files from disk
func (dc *DiskCache) clearDisk() {
	pattern := filepath.Join(dc.config.StoragePath, dc.config.FilePrefix+"*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	
	for _, path := range matches {
		os.Remove(path)
	}
}

// generateFilename generates a filename for a cache entry
func (dc *DiskCache) generateFilename(key string) string {
	// Use the cache key generator to create a safe filename
	safeKey := FastTokenKey(key)
	safeKey = strings.ReplaceAll(safeKey, "/", "_")
	safeKey = strings.ReplaceAll(safeKey, ":", "_")
	return fmt.Sprintf("%s_%s.json", dc.config.FilePrefix, safeKey)
}

// GetMetrics returns disk cache metrics
func (dc *DiskCache) GetMetrics() DiskCacheMetrics {
	dc.metrics.mu.RLock()
	defer dc.metrics.mu.RUnlock()
	
	// Return a copy without the mutex
	return DiskCacheMetrics{
		Hits:      dc.metrics.Hits,
		Misses:    dc.metrics.Misses,
		Writes:    dc.metrics.Writes,
		Reads:     dc.metrics.Reads,
		Deletes:   dc.metrics.Deletes,
		Size:      dc.metrics.Size,
		DiskUsage: dc.metrics.DiskUsage,
		LoadTime:  dc.metrics.LoadTime,
		SaveTime:  dc.metrics.SaveTime,
	}
}

// Close shuts down the disk cache and saves to disk
func (dc *DiskCache) Close() error {
	if dc.config.Persistence {
		return dc.SaveToDisk()
	}
	return nil
}

// Metrics helper methods
func (m *DiskCacheMetrics) incrementHits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Hits++
}

func (m *DiskCacheMetrics) incrementMisses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Misses++
}

func (m *DiskCacheMetrics) incrementWrites() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Writes++
}

func (m *DiskCacheMetrics) incrementReads() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Reads++
}

func (m *DiskCacheMetrics) incrementDeletes() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Deletes++
}

func (m *DiskCacheMetrics) setSize(size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Size = size
}

func (m *DiskCacheMetrics) setLoadTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LoadTime = duration
}

func (m *DiskCacheMetrics) setSaveTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SaveTime = duration
}