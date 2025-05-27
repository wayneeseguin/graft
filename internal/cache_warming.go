package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"
	"time"

	"github.com/wayneeseguin/graft/pkg/graft/parser"
)

// CacheWarming implements intelligent cache preloading based on usage patterns
type CacheWarming struct {
	analytics    *UsageAnalytics
	cache        *HierarchicalCache
	config       CacheWarmingConfig
	isWarming    bool
	warmingStats CacheWarmingStats
	mu           sync.RWMutex
}

// CacheWarmingConfig configures cache warming behavior
type CacheWarmingConfig struct {
	Enabled         bool          // Enable cache warming
	Strategy        string        // "frequency", "pattern", "hybrid"
	Background      bool          // Run warming in background
	StartupTimeout  time.Duration // Max time to spend warming during startup
	WarmingInterval time.Duration // Interval for periodic warming
	TopN            int           // Number of top expressions to warm
	MinFrequency    int64         // Minimum frequency to consider for warming
}

// CacheWarmingStats tracks warming performance
type CacheWarmingStats struct {
	LastWarmingTime   time.Time
	WarmingDuration   time.Duration
	ExpressionsWarmed int64
	SuccessfulWarms   int64
	FailedWarms       int64
	mu                sync.RWMutex
}

// UsageAnalytics tracks expression usage patterns for cache warming
type UsageAnalytics struct {
	patterns    map[string]*ExpressionUsage
	mu          sync.RWMutex
	storagePath string
	maxPatterns int
}

// ExpressionUsage tracks usage statistics for an expression
type ExpressionUsage struct {
	Expression   string    `json:"expression"`
	Frequency    int64     `json:"frequency"`
	LastUsed     time.Time `json:"last_used"`
	AverageTime  float64   `json:"average_time"`
	TotalTime    float64   `json:"total_time"`
	Pattern      string    `json:"pattern"`
	OperatorType string    `json:"operator_type"`
}

// NewCacheWarming creates a new cache warming system
func NewCacheWarming(cache *HierarchicalCache, config CacheWarmingConfig) *CacheWarming {
	analytics := NewUsageAnalytics(1000, "/tmp/graft_usage.json")

	cw := &CacheWarming{
		analytics: analytics,
		cache:     cache,
		config:    config,
	}

	// Start periodic warming if enabled
	if config.Enabled && config.WarmingInterval > 0 {
		go cw.periodicWarming()
	}

	return cw
}

// NewUsageAnalytics creates a new usage analytics tracker
func NewUsageAnalytics(maxPatterns int, storagePath string) *UsageAnalytics {
	return &UsageAnalytics{
		patterns:    make(map[string]*ExpressionUsage),
		storagePath: storagePath,
		maxPatterns: maxPatterns,
	}
}

// WarmCache performs cache warming based on usage patterns
func (cw *CacheWarming) WarmCache() error {
	if !cw.config.Enabled {
		return nil
	}

	cw.mu.Lock()
	if cw.isWarming {
		cw.mu.Unlock()
		return fmt.Errorf("cache warming already in progress")
	}
	cw.isWarming = true
	cw.mu.Unlock()

	defer func() {
		cw.mu.Lock()
		cw.isWarming = false
		cw.mu.Unlock()
	}()

	start := time.Now()

	// Load usage patterns
	if err := cw.analytics.LoadFromDisk(); err != nil {
		return fmt.Errorf("failed to load usage patterns: %v", err)
	}

	// Get expressions to warm based on strategy
	expressionsToWarm := cw.getExpressionsToWarm()

	// Warm expressions
	warmed, failed := cw.warmExpressions(expressionsToWarm)

	// Update stats
	duration := time.Since(start)
	cw.updateWarmingStats(duration, int64(len(expressionsToWarm)), warmed, failed)

	return nil
}

// WarmStartup performs cache warming during application startup
func (cw *CacheWarming) WarmStartup() error {
	if !cw.config.Enabled {
		return nil
	}

	// Use a timeout to avoid blocking startup too long
	done := make(chan error, 1)

	go func() {
		done <- cw.WarmCache()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(cw.config.StartupTimeout):
		return fmt.Errorf("cache warming timed out after %v", cw.config.StartupTimeout)
	}
}

// RecordUsage records usage of an expression for analytics
func (cw *CacheWarming) RecordUsage(expression string, duration time.Duration, operatorType string) {
	if !cw.config.Enabled {
		return
	}

	cw.analytics.RecordUsage(expression, duration, operatorType)
}

// getExpressionsToWarm determines which expressions to warm based on strategy
func (cw *CacheWarming) getExpressionsToWarm() []string {
	patterns := cw.analytics.GetTopPatterns(cw.config.TopN)

	var expressions []string

	switch cw.config.Strategy {
	case "frequency":
		expressions = cw.getFrequencyBasedExpressions(patterns)
	case "pattern":
		expressions = cw.getPatternBasedExpressions(patterns)
	case "hybrid":
		expressions = cw.getHybridExpressions(patterns)
	default:
		expressions = cw.getFrequencyBasedExpressions(patterns)
	}

	return expressions
}

// getFrequencyBasedExpressions selects expressions based on usage frequency
func (cw *CacheWarming) getFrequencyBasedExpressions(patterns []*ExpressionUsage) []string {
	var expressions []string

	for _, pattern := range patterns {
		if pattern.Frequency >= cw.config.MinFrequency {
			expressions = append(expressions, pattern.Expression)
		}
	}

	return expressions
}

// getPatternBasedExpressions selects expressions based on common patterns
func (cw *CacheWarming) getPatternBasedExpressions(patterns []*ExpressionUsage) []string {
	patternGroups := make(map[string][]*ExpressionUsage)

	// Group by pattern
	for _, pattern := range patterns {
		patternGroups[pattern.Pattern] = append(patternGroups[pattern.Pattern], pattern)
	}

	var expressions []string

	// Select representative expressions from each pattern group
	for _, group := range patternGroups {
		if len(group) > 0 {
			// Sort by frequency and take the most frequent
			sort.Slice(group, func(i, j int) bool {
				return group[i].Frequency > group[j].Frequency
			})
			expressions = append(expressions, group[0].Expression)
		}
	}

	return expressions
}

// getHybridExpressions combines frequency and pattern-based selection
func (cw *CacheWarming) getHybridExpressions(patterns []*ExpressionUsage) []string {
	frequencyBased := cw.getFrequencyBasedExpressions(patterns)
	patternBased := cw.getPatternBasedExpressions(patterns)

	// Combine and deduplicate
	expressionSet := make(map[string]bool)
	var expressions []string

	for _, expr := range frequencyBased {
		if !expressionSet[expr] {
			expressionSet[expr] = true
			expressions = append(expressions, expr)
		}
	}

	for _, expr := range patternBased {
		if !expressionSet[expr] {
			expressionSet[expr] = true
			expressions = append(expressions, expr)
		}
	}

	return expressions
}

// warmExpressions preloads expressions into the cache
func (cw *CacheWarming) warmExpressions(expressions []string) (int64, int64) {
	var warmed, failed int64
	registry := parser.NewOperatorRegistry()

	for _, expression := range expressions {
		// Parse and cache the expression
		if expr, err := parser.ParseExpression(expression, registry); err == nil {
			// Store in cache with a generated key
			key := FastExpressionKey(expression)
			cw.cache.Set(key, expr)
			warmed++
		} else {
			failed++
		}

		// Respect startup timeout
		if time.Since(cw.warmingStats.LastWarmingTime) > cw.config.StartupTimeout {
			break
		}
	}

	return warmed, failed
}

// periodicWarming runs cache warming periodically
func (cw *CacheWarming) periodicWarming() {
	ticker := time.NewTicker(cw.config.WarmingInterval)
	defer ticker.Stop()

	for range ticker.C {
		cw.WarmCache()
	}
}

// updateWarmingStats updates warming statistics
func (cw *CacheWarming) updateWarmingStats(duration time.Duration, total, warmed, failed int64) {
	cw.warmingStats.mu.Lock()
	defer cw.warmingStats.mu.Unlock()

	cw.warmingStats.LastWarmingTime = time.Now()
	cw.warmingStats.WarmingDuration = duration
	cw.warmingStats.ExpressionsWarmed = total
	cw.warmingStats.SuccessfulWarms = warmed
	cw.warmingStats.FailedWarms = failed
}

// GetWarmingStats returns cache warming statistics
func (cw *CacheWarming) GetWarmingStats() CacheWarmingStats {
	cw.warmingStats.mu.RLock()
	defer cw.warmingStats.mu.RUnlock()

	// Return a copy without the mutex
	return CacheWarmingStats{
		LastWarmingTime:   cw.warmingStats.LastWarmingTime,
		WarmingDuration:   cw.warmingStats.WarmingDuration,
		ExpressionsWarmed: cw.warmingStats.ExpressionsWarmed,
		SuccessfulWarms:   cw.warmingStats.SuccessfulWarms,
		FailedWarms:       cw.warmingStats.FailedWarms,
	}
}

// UsageAnalytics methods

// RecordUsage records usage of an expression
func (ua *UsageAnalytics) RecordUsage(expression string, duration time.Duration, operatorType string) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	pattern := ua.normalizePattern(expression)

	if usage, exists := ua.patterns[expression]; exists {
		usage.Frequency++
		usage.LastUsed = time.Now()
		usage.TotalTime += duration.Seconds()
		usage.AverageTime = usage.TotalTime / float64(usage.Frequency)
	} else {
		// Check if we need to evict old patterns
		if len(ua.patterns) >= ua.maxPatterns {
			ua.evictOldestPattern()
		}

		ua.patterns[expression] = &ExpressionUsage{
			Expression:   expression,
			Frequency:    1,
			LastUsed:     time.Now(),
			AverageTime:  duration.Seconds(),
			TotalTime:    duration.Seconds(),
			Pattern:      pattern,
			OperatorType: operatorType,
		}
	}
}

// GetTopPatterns returns the most frequently used patterns
func (ua *UsageAnalytics) GetTopPatterns(limit int) []*ExpressionUsage {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	patterns := make([]*ExpressionUsage, 0, len(ua.patterns))
	for _, pattern := range ua.patterns {
		patterns = append(patterns, pattern)
	}

	// Sort by frequency (descending)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	if limit > 0 && limit < len(patterns) {
		patterns = patterns[:limit]
	}

	return patterns
}

// normalizePattern creates a normalized pattern from an expression
func (ua *UsageAnalytics) normalizePattern(expression string) string {
	// For now, just return the expression as-is
	// TODO: Implement pattern normalization
	return expression
}

// evictOldestPattern removes the least recently used pattern
func (ua *UsageAnalytics) evictOldestPattern() {
	var oldestKey string
	var oldestTime time.Time

	for key, usage := range ua.patterns {
		if oldestKey == "" || usage.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = usage.LastUsed
		}
	}

	if oldestKey != "" {
		delete(ua.patterns, oldestKey)
	}
}

// LoadFromDisk loads usage patterns from persistent storage
func (ua *UsageAnalytics) LoadFromDisk() error {
	if ua.storagePath == "" {
		return nil
	}

	data, err := ioutil.ReadFile(ua.storagePath)
	if err != nil {
		return nil // File doesn't exist, start fresh
	}

	var patterns map[string]*ExpressionUsage
	if err := json.Unmarshal(data, &patterns); err != nil {
		return fmt.Errorf("failed to parse usage patterns: %v", err)
	}

	ua.mu.Lock()
	defer ua.mu.Unlock()
	ua.patterns = patterns

	return nil
}

// SaveToDisk saves usage patterns to persistent storage
func (ua *UsageAnalytics) SaveToDisk() error {
	if ua.storagePath == "" {
		return nil
	}

	ua.mu.RLock()
	defer ua.mu.RUnlock()

	data, err := json.Marshal(ua.patterns)
	if err != nil {
		return fmt.Errorf("failed to marshal usage patterns: %v", err)
	}

	return ioutil.WriteFile(ua.storagePath, data, 0644)
}

// DefaultCacheWarmingConfig provides sensible defaults
func DefaultCacheWarmingConfig() CacheWarmingConfig {
	return CacheWarmingConfig{
		Enabled:         true,
		Strategy:        "hybrid",
		Background:      true,
		StartupTimeout:  30 * time.Second,
		WarmingInterval: 5 * time.Minute,
		TopN:            50,
		MinFrequency:    3,
	}
}

// Global cache warming instance
var GlobalCacheWarming *CacheWarming

// InitializeGlobalCacheWarming initializes the global cache warming system
func InitializeGlobalCacheWarming(cache *HierarchicalCache, config CacheWarmingConfig) {
	GlobalCacheWarming = NewCacheWarming(cache, config)
}
