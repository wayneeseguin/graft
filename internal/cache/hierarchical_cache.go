package cache

import (
	"fmt"
	"sync"
	"time"
	
	"github.com/wayneeseguin/graft/pkg/graft"
)

// HierarchicalCache implements L1 (memory) and L2 (disk) cache layers
type HierarchicalCache struct {
	l1Cache *ConcurrentCache // Fast in-memory cache
	l2Cache *DiskCache       // Persistent disk cache
	config  HierarchicalCacheConfig
	metrics HierarchicalCacheMetrics
	mu      sync.RWMutex
}

// HierarchicalCacheConfig configures the hierarchical cache
type HierarchicalCacheConfig struct {
	L1Size       int           // L1 cache size
	L2Size       int           // L2 cache size
	L2Enabled    bool          // Enable L2 disk cache
	StoragePath  string        // Disk storage path
	TTL          time.Duration // Default TTL
	Persistence  bool          // Persist across restarts
	SyncInterval time.Duration // L1->L2 sync interval
}

// HierarchicalCacheMetrics tracks cache performance
type HierarchicalCacheMetrics struct {
	L1Hits         int64
	L1Misses       int64
	L2Hits         int64
	L2Misses       int64
	Promotions     int64 // L2 -> L1
	Demotions      int64 // L1 -> L2
	SyncOperations int64
	Errors         int64
	mu             sync.RWMutex
}

// CacheEntry represents a cache entry with metadata
type CacheEntry struct {
	Key          string        `json:"key"`
	Value        interface{}   `json:"value"`
	Timestamp    time.Time     `json:"timestamp"`
	LastAccessed time.Time     `json:"last_accessed"`
	HitCount     int64         `json:"hit_count"`
	Size         int64         `json:"size"`
	TTL          time.Duration `json:"ttl"`
}

// DefaultHierarchicalCacheConfig provides sensible defaults
func DefaultHierarchicalCacheConfig() HierarchicalCacheConfig {
	return HierarchicalCacheConfig{
		L1Size:       1000,
		L2Size:       10000,
		L2Enabled:    true,
		StoragePath:  "/tmp/graft_cache",
		TTL:          30 * time.Minute,
		Persistence:  true,
		SyncInterval: 5 * time.Minute,
	}
}

// NewHierarchicalCache creates a new hierarchical cache
func NewHierarchicalCache(config HierarchicalCacheConfig) (*HierarchicalCache, error) {
	// Create L1 cache
	l1Cache := NewConcurrentCache(CacheConfig{
		Capacity: config.L1Size,
		TTL:      config.TTL,
	})

	var l2Cache *DiskCache
	var err error

	// Create L2 cache if enabled
	if config.L2Enabled {
		l2Cache, err = NewDiskCache(DiskCacheConfig{
			StoragePath: config.StoragePath,
			MaxSize:     config.L2Size,
			TTL:         config.TTL,
			Persistence: config.Persistence,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create L2 cache: %v", err)
		}
	}

	hc := &HierarchicalCache{
		l1Cache: l1Cache,
		l2Cache: l2Cache,
		config:  config,
	}

	// Start background sync if L2 is enabled
	if config.L2Enabled && config.SyncInterval > 0 {
		go hc.backgroundSync()
	}

	// Load from persistent storage if enabled
	if config.L2Enabled && config.Persistence {
		hc.loadFromDisk()
	}

	return hc, nil
}

// Get retrieves a value from the cache hierarchy
func (hc *HierarchicalCache) Get(key string) (interface{}, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Try L1 cache first
	if value, found := hc.l1Cache.Get(key); found {
		hc.metrics.incrementL1Hits()
		return value, true
	}
	hc.metrics.incrementL1Misses()

	// Try L2 cache if enabled
	if hc.config.L2Enabled && hc.l2Cache != nil {
		if entry, found := hc.l2Cache.Get(key); found {
			hc.metrics.incrementL2Hits()

			// Promote to L1 cache
			hc.promoteToL1(key, entry.Value)
			return entry.Value, true
		}
		hc.metrics.incrementL2Misses()
	}

	return nil, false
}

// Set stores a value in the cache hierarchy
func (hc *HierarchicalCache) Set(key string, value interface{}) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Always store in L1 first
	hc.l1Cache.Set(key, value)

	// Optionally store in L2
	if hc.config.L2Enabled && hc.l2Cache != nil {
		entry := &CacheEntry{
			Key:          key,
			Value:        value,
			Timestamp:    time.Now(),
			LastAccessed: time.Now(),
			HitCount:     0,
			Size:         hc.estimateSize(value),
			TTL:          hc.config.TTL,
		}
		hc.l2Cache.Set(key, entry)
	}
}

// Delete removes a value from both cache levels
func (hc *HierarchicalCache) Delete(key string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.l1Cache.Delete(key)

	if hc.config.L2Enabled && hc.l2Cache != nil {
		hc.l2Cache.Delete(key)
	}
}

// Clear removes all cached entries
func (hc *HierarchicalCache) Clear() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.l1Cache.Clear()

	if hc.config.L2Enabled && hc.l2Cache != nil {
		hc.l2Cache.Clear()
	}

	hc.metrics = HierarchicalCacheMetrics{}
}

// promoteToL1 moves an entry from L2 to L1 cache
func (hc *HierarchicalCache) promoteToL1(key string, value interface{}) {
	hc.l1Cache.Set(key, value)
	hc.metrics.incrementPromotions()
}

// demoteToL2 moves entries from L1 to L2 cache
func (hc *HierarchicalCache) demoteToL2(evictedEntries map[string]interface{}) {
	if !hc.config.L2Enabled || hc.l2Cache == nil {
		return
	}

	for key, value := range evictedEntries {
		entry := &CacheEntry{
			Key:          key,
			Value:        value,
			Timestamp:    time.Now(),
			LastAccessed: time.Now(),
			HitCount:     0,
			Size:         hc.estimateSize(value),
			TTL:          hc.config.TTL,
		}
		hc.l2Cache.Set(key, entry)
		hc.metrics.incrementDemotions()
	}
}

// backgroundSync periodically syncs L1 to L2
func (hc *HierarchicalCache) backgroundSync() {
	ticker := time.NewTicker(hc.config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		hc.syncL1ToL2()
	}
}

// syncL1ToL2 syncs L1 cache entries to L2
func (hc *HierarchicalCache) syncL1ToL2() {
	if !hc.config.L2Enabled || hc.l2Cache == nil {
		return
	}

	hc.mu.RLock()
	l1Entries := hc.getAllL1Entries()
	hc.mu.RUnlock()

	for key, value := range l1Entries {
		// Check if entry exists in L2, if not add it
		if _, found := hc.l2Cache.Get(key); !found {
			entry := &CacheEntry{
				Key:          key,
				Value:        value,
				Timestamp:    time.Now(),
				LastAccessed: time.Now(),
				HitCount:     0,
				Size:         hc.estimateSize(value),
				TTL:          hc.config.TTL,
			}
			hc.l2Cache.Set(key, entry)
		}
	}

	hc.metrics.incrementSyncOperations()
}

// loadFromDisk loads persistent cache data
func (hc *HierarchicalCache) loadFromDisk() {
	if !hc.config.L2Enabled || hc.l2Cache == nil {
		return
	}

	// L2 cache handles loading from disk
	if err := hc.l2Cache.LoadFromDisk(); err != nil {
		hc.metrics.incrementErrors()
	}
}

// estimateSize estimates the size of a cached value
func (hc *HierarchicalCache) estimateSize(value interface{}) int64 {
	// Simple size estimation
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case *graft.Expr:
		// Rough estimate for expression size
		return 100
	default:
		// Default estimate
		return 50
	}
}

// GetMetrics returns cache performance metrics
func (hc *HierarchicalCache) GetMetrics() HierarchicalCacheMetrics {
	hc.metrics.mu.RLock()
	defer hc.metrics.mu.RUnlock()

	// Return a copy without the mutex
	return HierarchicalCacheMetrics{
		L1Hits:         hc.metrics.L1Hits,
		L1Misses:       hc.metrics.L1Misses,
		L2Hits:         hc.metrics.L2Hits,
		L2Misses:       hc.metrics.L2Misses,
		Promotions:     hc.metrics.Promotions,
		Demotions:      hc.metrics.Demotions,
		SyncOperations: hc.metrics.SyncOperations,
		Errors:         hc.metrics.Errors,
	}
}

// getAllL1Entries gets all entries from L1 cache
func (hc *HierarchicalCache) getAllL1Entries() map[string]interface{} {
	// This is a simplified implementation
	// In practice, we'd need to iterate through all shards
	entries := make(map[string]interface{})

	// For now, we'll return an empty map since the actual implementation
	// would require accessing internal shard data
	return entries
}

// GetDetailedMetrics returns detailed metrics including L1 and L2 stats
func (hc *HierarchicalCache) GetDetailedMetrics() map[string]interface{} {
	metrics := hc.GetMetrics()
	l1Stats := hc.l1Cache.Metrics()

	result := map[string]interface{}{
		"l1": map[string]interface{}{
			"hits":     metrics.L1Hits,
			"misses":   metrics.L1Misses,
			"hit_rate": float64(metrics.L1Hits) / float64(metrics.L1Hits+metrics.L1Misses),
			"size":     l1Stats.Size,
			"capacity": hc.config.L1Size,
		},
		"promotions": metrics.Promotions,
		"demotions":  metrics.Demotions,
		"sync_ops":   metrics.SyncOperations,
		"errors":     metrics.Errors,
		"l2_enabled": hc.config.L2Enabled,
	}

	if hc.config.L2Enabled && hc.l2Cache != nil {
		l2Stats := hc.l2Cache.GetMetrics()
		result["l2"] = map[string]interface{}{
			"hits":       metrics.L2Hits,
			"misses":     metrics.L2Misses,
			"hit_rate":   float64(metrics.L2Hits) / float64(metrics.L2Hits+metrics.L2Misses),
			"size":       l2Stats.Size,
			"capacity":   hc.config.L2Size,
			"disk_usage": l2Stats.DiskUsage,
		}
	}

	return result
}

// Close shuts down the hierarchical cache
func (hc *HierarchicalCache) Close() error {
	// Sync to disk before closing (without holding lock)
	if hc.config.L2Enabled && hc.l2Cache != nil {
		hc.syncL1ToL2()
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.config.L2Enabled && hc.l2Cache != nil {
		return hc.l2Cache.Close()
	}

	return nil
}

// Metrics helper methods
func (m *HierarchicalCacheMetrics) incrementL1Hits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.L1Hits++
}

func (m *HierarchicalCacheMetrics) incrementL1Misses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.L1Misses++
}

func (m *HierarchicalCacheMetrics) incrementL2Hits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.L2Hits++
}

func (m *HierarchicalCacheMetrics) incrementL2Misses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.L2Misses++
}

func (m *HierarchicalCacheMetrics) incrementPromotions() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Promotions++
}

func (m *HierarchicalCacheMetrics) incrementDemotions() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Demotions++
}

func (m *HierarchicalCacheMetrics) incrementSyncOperations() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SyncOperations++
}

func (m *HierarchicalCacheMetrics) incrementErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Errors++
}
