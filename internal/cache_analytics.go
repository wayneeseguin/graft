package internal

import (
	"math"
	"sort"
	"sync"
	"time"
)

// CacheAnalytics provides detailed cache performance analytics
type CacheAnalytics struct {
	mu         sync.RWMutex
	caches     map[string]*CacheStats
	startTime  time.Time
	window     time.Duration
	hotKeys    *HotKeyTracker
}

// CacheStats tracks statistics for a single cache
type CacheStats struct {
	Name           string
	Hits           int64
	Misses         int64
	Evictions      int64
	Size           int64
	MaxSize        int64
	TotalLoadTime  time.Duration
	LastAccessTime time.Time
	
	// Detailed tracking
	hitsByKey      map[string]int64
	missesByKey    map[string]int64
	loadTimeByKey  map[string]time.Duration
	evictionReasons map[string]int64
	
	mu sync.RWMutex
}

// HotKeyTracker tracks frequently accessed keys
type HotKeyTracker struct {
	mu          sync.RWMutex
	keyAccess   map[string]*KeyAccessInfo
	window      time.Duration
	topN        int
}

// KeyAccessInfo tracks access information for a key
type KeyAccessInfo struct {
	Key         string
	AccessCount int64
	LastAccess  time.Time
	AvgLoadTime time.Duration
	HitRate     float64
}

// NewCacheAnalytics creates a new cache analytics instance
func NewCacheAnalytics(window time.Duration) *CacheAnalytics {
	return &CacheAnalytics{
		caches:    make(map[string]*CacheStats),
		startTime: time.Now(),
		window:    window,
		hotKeys:   NewHotKeyTracker(window, 100),
	}
}

// NewHotKeyTracker creates a new hot key tracker
func NewHotKeyTracker(window time.Duration, topN int) *HotKeyTracker {
	return &HotKeyTracker{
		keyAccess: make(map[string]*KeyAccessInfo),
		window:    window,
		topN:      topN,
	}
}

// RecordHit records a cache hit
func (ca *CacheAnalytics) RecordHit(cacheName, key string) {
	ca.mu.Lock()
	stats, exists := ca.caches[cacheName]
	if !exists {
		stats = ca.newCacheStats(cacheName)
		ca.caches[cacheName] = stats
	}
	ca.mu.Unlock()
	
	stats.recordHit(key)
	ca.hotKeys.recordAccess(key, true, 0)
	
	// Update metrics collector
	if MetricsEnabled() {
		RecordCacheMetrics(cacheName, true)
	}
}

// RecordMiss records a cache miss
func (ca *CacheAnalytics) RecordMiss(cacheName, key string, loadTime time.Duration) {
	ca.mu.Lock()
	stats, exists := ca.caches[cacheName]
	if !exists {
		stats = ca.newCacheStats(cacheName)
		ca.caches[cacheName] = stats
	}
	ca.mu.Unlock()
	
	stats.recordMiss(key, loadTime)
	ca.hotKeys.recordAccess(key, false, loadTime)
	
	// Update metrics collector
	if MetricsEnabled() {
		RecordCacheMetrics(cacheName, false)
	}
}

// RecordEviction records a cache eviction
func (ca *CacheAnalytics) RecordEviction(cacheName string, count int64, reason string) {
	ca.mu.RLock()
	stats, exists := ca.caches[cacheName]
	ca.mu.RUnlock()
	
	if exists {
		stats.recordEviction(count, reason)
		
		// Update metrics collector
		if MetricsEnabled() {
			mc := GetMetricsCollector()
			mc.RecordCacheEviction(cacheName, count)
		}
	}
}

// UpdateSize updates cache size
func (ca *CacheAnalytics) UpdateSize(cacheName string, size, maxSize int64) {
	ca.mu.Lock()
	stats, exists := ca.caches[cacheName]
	if !exists {
		stats = ca.newCacheStats(cacheName)
		ca.caches[cacheName] = stats
	}
	ca.mu.Unlock()
	
	stats.updateSize(size, maxSize)
	
	// Update metrics collector
	if MetricsEnabled() {
		mc := GetMetricsCollector()
		mc.UpdateCacheSize(cacheName, size)
	}
}

// newCacheStats creates new cache stats
func (ca *CacheAnalytics) newCacheStats(name string) *CacheStats {
	return &CacheStats{
		Name:            name,
		hitsByKey:       make(map[string]int64),
		missesByKey:     make(map[string]int64),
		loadTimeByKey:   make(map[string]time.Duration),
		evictionReasons: make(map[string]int64),
	}
}

// recordHit records a hit in cache stats
func (cs *CacheStats) recordHit(key string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.Hits++
	cs.hitsByKey[key]++
	cs.LastAccessTime = time.Now()
}

// recordMiss records a miss in cache stats
func (cs *CacheStats) recordMiss(key string, loadTime time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.Misses++
	cs.missesByKey[key]++
	cs.TotalLoadTime += loadTime
	cs.loadTimeByKey[key] += loadTime
	cs.LastAccessTime = time.Now()
}

// recordEviction records an eviction
func (cs *CacheStats) recordEviction(count int64, reason string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.Evictions += count
	cs.evictionReasons[reason] += count
}

// updateSize updates cache size
func (cs *CacheStats) updateSize(size, maxSize int64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.Size = size
	cs.MaxSize = maxSize
}

// GetStats returns cache statistics
func (cs *CacheStats) GetStats() CacheStatistics {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	total := cs.Hits + cs.Misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(cs.Hits) / float64(total)
	}
	
	avgLoadTime := time.Duration(0)
	if cs.Misses > 0 {
		avgLoadTime = cs.TotalLoadTime / time.Duration(cs.Misses)
	}
	
	fillRate := float64(0)
	if cs.MaxSize > 0 {
		fillRate = float64(cs.Size) / float64(cs.MaxSize)
	}
	
	return CacheStatistics{
		Name:            cs.Name,
		Hits:            cs.Hits,
		Misses:          cs.Misses,
		Evictions:       cs.Evictions,
		Size:            cs.Size,
		MaxSize:         cs.MaxSize,
		HitRate:         hitRate,
		FillRate:        fillRate,
		AvgLoadTime:     avgLoadTime,
		TotalLoadTime:   cs.TotalLoadTime,
		LastAccessTime:  cs.LastAccessTime,
		EvictionReasons: cs.copyEvictionReasons(),
	}
}

// copyEvictionReasons creates a copy of eviction reasons
func (cs *CacheStats) copyEvictionReasons() map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range cs.evictionReasons {
		copy[k] = v
	}
	return copy
}

// recordAccess records a key access
func (hkt *HotKeyTracker) recordAccess(key string, hit bool, loadTime time.Duration) {
	hkt.mu.Lock()
	defer hkt.mu.Unlock()
	
	info, exists := hkt.keyAccess[key]
	if !exists {
		info = &KeyAccessInfo{
			Key: key,
		}
		hkt.keyAccess[key] = info
	}
	
	info.AccessCount++
	info.LastAccess = time.Now()
	
	if !hit && loadTime > 0 {
		// Update average load time
		totalLoadTime := info.AvgLoadTime * time.Duration(info.AccessCount-1)
		info.AvgLoadTime = (totalLoadTime + loadTime) / time.Duration(info.AccessCount)
	}
	
	// Update hit rate
	if hit {
		info.HitRate = (info.HitRate*float64(info.AccessCount-1) + 1) / float64(info.AccessCount)
	} else {
		info.HitRate = info.HitRate * float64(info.AccessCount-1) / float64(info.AccessCount)
	}
}

// GetHotKeys returns the top N hot keys
func (hkt *HotKeyTracker) GetHotKeys() []*KeyAccessInfo {
	hkt.mu.RLock()
	defer hkt.mu.RUnlock()
	
	// Clean old entries
	cutoff := time.Now().Add(-hkt.window)
	keys := make([]*KeyAccessInfo, 0, len(hkt.keyAccess))
	
	for _, info := range hkt.keyAccess {
		if info.LastAccess.After(cutoff) {
			keys = append(keys, info)
		}
	}
	
	// Sort by access count
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].AccessCount > keys[j].AccessCount
	})
	
	// Return top N
	if len(keys) > hkt.topN {
		keys = keys[:hkt.topN]
	}
	
	return keys
}

// GetAllStats returns statistics for all caches
func (ca *CacheAnalytics) GetAllStats() []CacheStatistics {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	stats := make([]CacheStatistics, 0, len(ca.caches))
	for _, cache := range ca.caches {
		stats = append(stats, cache.GetStats())
	}
	
	// Sort by total operations
	sort.Slice(stats, func(i, j int) bool {
		return (stats[i].Hits + stats[i].Misses) > (stats[j].Hits + stats[j].Misses)
	})
	
	return stats
}

// GetCacheStats returns statistics for a specific cache
func (ca *CacheAnalytics) GetCacheStats(cacheName string) (CacheStatistics, bool) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	if stats, exists := ca.caches[cacheName]; exists {
		return stats.GetStats(), true
	}
	
	return CacheStatistics{}, false
}

// GetHotKeys returns hot keys across all caches
func (ca *CacheAnalytics) GetHotKeys() []*KeyAccessInfo {
	return ca.hotKeys.GetHotKeys()
}

// GetEffectivenessScore calculates cache effectiveness score
func (ca *CacheAnalytics) GetEffectivenessScore() float64 {
	allStats := ca.GetAllStats()
	
	if len(allStats) == 0 {
		return 0
	}
	
	var totalScore float64
	var totalWeight float64
	
	for _, stats := range allStats {
		operations := float64(stats.Hits + stats.Misses)
		if operations == 0 {
			continue
		}
		
		// Score components:
		// - Hit rate (40% weight)
		// - Fill rate efficiency (20% weight)
		// - Eviction rate (20% weight)
		// - Load time efficiency (20% weight)
		
		hitRateScore := stats.HitRate
		fillRateScore := 1.0 - math.Abs(stats.FillRate-0.8) // Optimal around 80%
		
		evictionRate := float64(stats.Evictions) / operations
		evictionScore := 1.0 - math.Min(evictionRate*10, 1.0) // Penalize high eviction
		
		// Assume 50ms is good load time
		loadTimeScore := 1.0
		if stats.AvgLoadTime > 0 {
			loadTimeScore = math.Min(50*time.Millisecond.Seconds()/stats.AvgLoadTime.Seconds(), 1.0)
		}
		
		score := hitRateScore*0.4 + fillRateScore*0.2 + evictionScore*0.2 + loadTimeScore*0.2
		
		totalScore += score * operations
		totalWeight += operations
	}
	
	if totalWeight == 0 {
		return 0
	}
	
	return totalScore / totalWeight
}

// GenerateReport generates a cache analytics report
func (ca *CacheAnalytics) GenerateReport() *CacheAnalyticsReport {
	allStats := ca.GetAllStats()
	hotKeys := ca.GetHotKeys()
	
	report := &CacheAnalyticsReport{
		GeneratedAt:        time.Now(),
		AnalyticsPeriod:    time.Since(ca.startTime),
		CacheStats:         allStats,
		HotKeys:            hotKeys,
		EffectivenessScore: ca.GetEffectivenessScore(),
	}
	
	// Calculate totals
	for _, stats := range allStats {
		report.TotalHits += stats.Hits
		report.TotalMisses += stats.Misses
		report.TotalEvictions += stats.Evictions
		report.TotalSize += stats.Size
		report.TotalMaxSize += stats.MaxSize
	}
	
	if report.TotalHits+report.TotalMisses > 0 {
		report.OverallHitRate = float64(report.TotalHits) / float64(report.TotalHits+report.TotalMisses)
	}
	
	return report
}

// CacheStatistics represents cache performance statistics
type CacheStatistics struct {
	Name            string            `json:"name"`
	Hits            int64             `json:"hits"`
	Misses          int64             `json:"misses"`
	Evictions       int64             `json:"evictions"`
	Size            int64             `json:"size"`
	MaxSize         int64             `json:"max_size"`
	HitRate         float64           `json:"hit_rate"`
	FillRate        float64           `json:"fill_rate"`
	AvgLoadTime     time.Duration     `json:"avg_load_time"`
	TotalLoadTime   time.Duration     `json:"total_load_time"`
	LastAccessTime  time.Time         `json:"last_access_time"`
	EvictionReasons map[string]int64  `json:"eviction_reasons,omitempty"`
}

// CacheAnalyticsReport represents a comprehensive cache report
type CacheAnalyticsReport struct {
	GeneratedAt        time.Time          `json:"generated_at"`
	AnalyticsPeriod    time.Duration      `json:"analytics_period"`
	CacheStats         []CacheStatistics  `json:"cache_stats"`
	HotKeys            []*KeyAccessInfo   `json:"hot_keys"`
	TotalHits          int64              `json:"total_hits"`
	TotalMisses        int64              `json:"total_misses"`
	TotalEvictions     int64              `json:"total_evictions"`
	TotalSize          int64              `json:"total_size"`
	TotalMaxSize       int64              `json:"total_max_size"`
	OverallHitRate     float64            `json:"overall_hit_rate"`
	EffectivenessScore float64            `json:"effectiveness_score"`
}

// Global cache analytics
var globalCacheAnalytics *CacheAnalytics
var cacheAnalyticsOnce sync.Once

// InitializeCacheAnalytics initializes global cache analytics
func InitializeCacheAnalytics() {
	cacheAnalyticsOnce.Do(func() {
		globalCacheAnalytics = NewCacheAnalytics(1 * time.Hour)
	})
}

// GetCacheAnalytics returns the global cache analytics
func GetCacheAnalytics() *CacheAnalytics {
	if globalCacheAnalytics == nil {
		InitializeCacheAnalytics()
	}
	return globalCacheAnalytics
}

