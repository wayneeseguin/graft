package spruce

import (
	"strings"
	"sync"
	"time"
)

// ParserMemoizationCache caches parsing results to avoid re-parsing common patterns
type ParserMemoizationCache struct {
	exprCache    map[string]*MemoizedExpr
	tokenCache   map[string][]Token
	mu           sync.RWMutex
	maxSize      int
	ttl          time.Duration
	hits         int64
	misses       int64
	evictions    int64
}

// MemoizedExpr represents a cached expression with metadata
type MemoizedExpr struct {
	Expr      *Expr
	Timestamp time.Time
	HitCount  int64
}

// ParserCacheMetrics holds cache performance metrics
type ParserCacheMetrics struct {
	Hits      int64
	Misses    int64
	HitRate   float64
	Size      int
	Evictions int64
}

// Global parser cache instance
var GlobalParserCache = NewParserMemoizationCache(10000, 30*time.Minute)

// NewParserMemoizationCache creates a new parser memoization cache
func NewParserMemoizationCache(maxSize int, ttl time.Duration) *ParserMemoizationCache {
	return &ParserMemoizationCache{
		exprCache: make(map[string]*MemoizedExpr),
		tokenCache: make(map[string][]Token),
		maxSize:   maxSize,
		ttl:       ttl,
	}
}

// CacheKey generates a cache key for an expression input
func (c *ParserMemoizationCache) CacheKey(input string, operatorRegistry *OperatorRegistry) string {
	// Use optimized key generation
	return DefaultKeyGenerator.GenerateExpressionKey(input, operatorRegistry)
}

// GetExpression retrieves a cached expression
func (c *ParserMemoizationCache) GetExpression(key string) (*Expr, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	cached, exists := c.exprCache[key]
	if !exists {
		c.misses++
		return nil, false
	}
	
	// Check TTL
	if time.Since(cached.Timestamp) > c.ttl {
		// Expired, but don't remove here (would need write lock)
		c.misses++
		return nil, false
	}
	
	c.hits++
	cached.HitCount++
	return cached.Expr, true
}

// SetExpression stores an expression in the cache
func (c *ParserMemoizationCache) SetExpression(key string, expr *Expr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if we need to evict
	if len(c.exprCache) >= c.maxSize {
		c.evictLRU()
	}
	
	c.exprCache[key] = &MemoizedExpr{
		Expr:      expr,
		Timestamp: time.Now(),
		HitCount:  0,
	}
}

// GetTokens retrieves cached tokens
func (c *ParserMemoizationCache) GetTokens(key string) ([]Token, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	tokens, exists := c.tokenCache[key]
	if !exists {
		return nil, false
	}
	
	// Copy tokens to prevent modification
	result := make([]Token, len(tokens))
	copy(result, tokens)
	return result, true
}

// SetTokens stores tokens in the cache
func (c *ParserMemoizationCache) SetTokens(key string, tokens []Token) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if we need to evict from token cache
	if len(c.tokenCache) >= c.maxSize {
		c.evictTokensLRU()
	}
	
	// Store a copy to prevent external modification
	cached := make([]Token, len(tokens))
	copy(cached, tokens)
	c.tokenCache[key] = cached
}

// evictLRU removes the least recently used expression
func (c *ParserMemoizationCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	var lowestHitCount int64 = -1
	
	// Find the oldest entry with lowest hit count
	for key, cached := range c.exprCache {
		if oldestKey == "" || cached.Timestamp.Before(oldestTime) || 
		   (cached.Timestamp.Equal(oldestTime) && (lowestHitCount == -1 || cached.HitCount < lowestHitCount)) {
			oldestKey = key
			oldestTime = cached.Timestamp
			lowestHitCount = cached.HitCount
		}
	}
	
	if oldestKey != "" {
		delete(c.exprCache, oldestKey)
		c.evictions++
	}
}

// evictTokensLRU removes the least recently used tokens
func (c *ParserMemoizationCache) evictTokensLRU() {
	// Simple strategy: remove first key found
	for key := range c.tokenCache {
		delete(c.tokenCache, key)
		break
	}
}

// GetMetrics returns cache performance metrics
func (c *ParserMemoizationCache) GetMetrics() ParserCacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}
	
	return ParserCacheMetrics{
		Hits:      c.hits,
		Misses:    c.misses,
		HitRate:   hitRate,
		Size:      len(c.exprCache),
		Evictions: c.evictions,
	}
}

// Clear removes all cached entries
func (c *ParserMemoizationCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.exprCache = make(map[string]*MemoizedExpr)
	c.tokenCache = make(map[string][]Token)
	c.hits = 0
	c.misses = 0
	c.evictions = 0
}

// CleanupExpired removes expired entries
func (c *ParserMemoizationCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, cached := range c.exprCache {
		if now.Sub(cached.Timestamp) > c.ttl {
			delete(c.exprCache, key)
		}
	}
}

// MemoizedEnhancedParser wraps EnhancedParser with memoization
type MemoizedEnhancedParser struct {
	*EnhancedParser
	cache *ParserMemoizationCache
	input string
}

// NewMemoizedEnhancedParser creates a parser with memoization
func NewMemoizedEnhancedParser(input string, registry *OperatorRegistry) *MemoizedEnhancedParser {
	// Check cache for tokens first
	cache := GlobalParserCache
	key := cache.CacheKey(input, registry)
	
	var tokens []Token
	if cachedTokens, found := cache.GetTokens(key); found {
		tokens = cachedTokens
	} else {
		// Tokenize and cache
		tokenizer := NewEnhancedTokenizer(input)
		tokens = tokenizer.Tokenize()
		cache.SetTokens(key, tokens)
	}
	
	parser := NewEnhancedParser(tokens, registry)
	
	return &MemoizedEnhancedParser{
		EnhancedParser: parser,
		cache:          cache,
		input:          input,
	}
}

// Parse parses with memoization
func (mp *MemoizedEnhancedParser) Parse() (*Expr, error) {
	// Generate cache key
	key := mp.cache.CacheKey(mp.input, mp.registry)
	
	// Check cache first
	if expr, found := mp.cache.GetExpression(key); found {
		return expr, nil
	}
	
	// Parse normally
	expr, err := mp.EnhancedParser.Parse()
	if err != nil {
		return nil, err
	}
	
	// Cache successful result
	mp.cache.SetExpression(key, expr)
	
	return expr, nil
}

// WithMemoization configures an existing parser to use memoization
func (p *EnhancedParser) WithMemoization(input string) *MemoizedEnhancedParser {
	return &MemoizedEnhancedParser{
		EnhancedParser: p,
		cache:          GlobalParserCache,
		input:          input,
	}
}

// Common expression pattern detection for better caching
type ExpressionPattern struct {
	Pattern     string
	Frequency   int64
	LastSeen    time.Time
	CacheHits   int64
}

// PatternTracker tracks common expression patterns for cache optimization
type PatternTracker struct {
	patterns map[string]*ExpressionPattern
	mu       sync.RWMutex
}

// GlobalPatternTracker tracks parsing patterns
var GlobalPatternTracker = &PatternTracker{
	patterns: make(map[string]*ExpressionPattern),
}

// RecordPattern records an expression pattern
func (pt *PatternTracker) RecordPattern(input string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	// Normalize pattern (remove specific values, keep structure)
	pattern := pt.normalizePattern(input)
	
	if existing, found := pt.patterns[pattern]; found {
		existing.Frequency++
		existing.LastSeen = time.Now()
	} else {
		pt.patterns[pattern] = &ExpressionPattern{
			Pattern:   pattern,
			Frequency: 1,
			LastSeen:  time.Now(),
		}
	}
}

// normalizePattern creates a normalized pattern for caching
func (pt *PatternTracker) normalizePattern(input string) string {
	// Simple pattern normalization: replace quoted strings and numbers with placeholders
	normalized := input
	
	// Replace quoted strings
	normalized = strings.ReplaceAll(normalized, `"[^"]*"`, `"STRING"`)
	
	// Replace numbers (simple regex-like replacement)
	for i := 0; i < len(normalized); i++ {
		if normalized[i] >= '0' && normalized[i] <= '9' {
			start := i
			for i < len(normalized) && ((normalized[i] >= '0' && normalized[i] <= '9') || normalized[i] == '.') {
				i++
			}
			if start != i {
				normalized = normalized[:start] + "NUM" + normalized[i:]
				i = start + 3 // Length of "NUM"
			}
		}
	}
	
	return normalized
}

// GetTopPatterns returns the most frequently used patterns
func (pt *PatternTracker) GetTopPatterns(limit int) []*ExpressionPattern {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	
	patterns := make([]*ExpressionPattern, 0, len(pt.patterns))
	for _, pattern := range pt.patterns {
		patterns = append(patterns, pattern)
	}
	
	// Simple bubble sort by frequency (good enough for small sets)
	for i := 0; i < len(patterns)-1; i++ {
		for j := 0; j < len(patterns)-i-1; j++ {
			if patterns[j].Frequency < patterns[j+1].Frequency {
				patterns[j], patterns[j+1] = patterns[j+1], patterns[j]
			}
		}
	}
	
	if limit > 0 && limit < len(patterns) {
		patterns = patterns[:limit]
	}
	
	return patterns
}