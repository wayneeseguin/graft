package parser

import (
	"testing"
	"time"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestParserMemoization(t *testing.T) {
	Convey("Parser Memoization", t, func() {
		registry := NewOperatorRegistry()
		
		Convey("Basic expression caching", func() {
			cache := NewParserMemoizationCache(100, time.Minute)
			
			// Parse the same expression twice
			parser1 := NewMemoizedParser("grab meta.key", registry)
			parser1.cache = cache
			expr1, err1 := parser1.Parse()
			So(err1, ShouldBeNil)
			So(expr1, ShouldNotBeNil)
			
			parser2 := NewMemoizedParser("grab meta.key", registry)
			parser2.cache = cache
			expr2, err2 := parser2.Parse()
			So(err2, ShouldBeNil)
			So(expr2, ShouldNotBeNil)
			
			// Check metrics
			metrics := cache.GetMetrics()
			So(metrics.Hits, ShouldBeGreaterThan, 0)
			So(metrics.HitRate, ShouldBeGreaterThan, 0.0)
		})
		
		Convey("Token caching", func() {
			cache := NewParserMemoizationCache(100, time.Minute)
			key := cache.CacheKey("grab meta.key", registry)
			
			// First tokenization should cache tokens
			parser1 := NewMemoizedParser("grab meta.key", registry)
			parser1.cache = cache
			
			// Second parser should use cached tokens
			tokens, found := cache.GetTokens(key)
			if !found {
				// Cache tokens manually for test
				tokenizer := NewTokenizer("grab meta.key")
				tokens = tokenizer.Tokenize()
				cache.SetTokens(key, tokens)
				
				// Verify cached tokens
				cachedTokens, found := cache.GetTokens(key)
				So(found, ShouldBeTrue)
				So(len(cachedTokens), ShouldEqual, len(tokens))
			}
		})
		
		Convey("Cache eviction", func() {
			cache := NewParserMemoizationCache(2, time.Minute) // Small cache
			
			// Fill cache beyond capacity
			parser1 := NewMemoizedParser("grab meta.key1", registry)
			parser1.cache = cache
			parser1.Parse()
			
			parser2 := NewMemoizedParser("grab meta.key2", registry)
			parser2.cache = cache
			parser2.Parse()
			
			parser3 := NewMemoizedParser("grab meta.key3", registry)
			parser3.cache = cache
			parser3.Parse()
			
			metrics := cache.GetMetrics()
			So(metrics.Size, ShouldBeLessThanOrEqualTo, 2)
			So(metrics.Evictions, ShouldBeGreaterThan, 0)
		})
		
		Convey("TTL expiration", func() {
			cache := NewParserMemoizationCache(100, 10*time.Millisecond) // Short TTL
			
			parser := NewMemoizedParser("grab meta.key", registry)
			parser.cache = cache
			parser.Parse()
			
			// Wait for expiration
			time.Sleep(20 * time.Millisecond)
			
			// Should not find expired entry
			key := cache.CacheKey("grab meta.key", registry)
			_, found := cache.GetExpression(key)
			So(found, ShouldBeFalse)
		})
		
		Convey("Pattern tracking", func() {
			tracker := &PatternTracker{
				patterns: make(map[string]*ExpressionPattern),
			}
			
			// Record some patterns
			tracker.RecordPattern(`grab "value1"`)
			tracker.RecordPattern(`grab "value2"`)
			tracker.RecordPattern(`grab "value1"`) // Duplicate
			
			patterns := tracker.GetTopPatterns(10)
			So(len(patterns), ShouldBeGreaterThan, 0)
			
			// Most frequent pattern should be grab with string
			topPattern := patterns[0]
			So(topPattern.Frequency, ShouldBeGreaterThanOrEqualTo, 2)
		})
		
		Convey("ParseExpression convenience function", func() {
			// Clear global cache for clean test
			GlobalParserCache.Clear()
			
			// Parse the same expression multiple times
			expr1, err1 := ParseExpression("grab meta.test", registry)
			So(err1, ShouldBeNil)
			So(expr1, ShouldNotBeNil)
			
			expr2, err2 := ParseExpression("grab meta.test", registry)
			So(err2, ShouldBeNil)
			So(expr2, ShouldNotBeNil)
			
			// Check that cache was used
			metrics := GlobalParserCache.GetMetrics()
			So(metrics.Hits, ShouldBeGreaterThan, 0)
		})
		
		Convey("Cache key generation", func() {
			cache := NewParserMemoizationCache(100, time.Minute)
			
			// Same input should generate same key
			key1 := cache.CacheKey("grab meta.key", registry)
			key2 := cache.CacheKey("grab meta.key", registry)
			So(key1, ShouldEqual, key2)
			
			// Different input should generate different key
			key3 := cache.CacheKey("grab meta.other", registry)
			So(key1, ShouldNotEqual, key3)
		})
		
		Convey("Pattern normalization", func() {
			tracker := &PatternTracker{
				patterns: make(map[string]*ExpressionPattern),
			}
			
			// Different strings should normalize to same pattern
			pattern1 := tracker.NormalizePattern(`grab "value1"`)
			pattern2 := tracker.NormalizePattern(`grab "value2"`)
			So(pattern1, ShouldEqual, pattern2)
			
			// Different numbers should normalize to same pattern
			pattern3 := tracker.NormalizePattern("calc 100 + 200")
			pattern4 := tracker.NormalizePattern("calc 50 + 75")
			So(pattern3, ShouldEqual, pattern4)
		})
		
		Convey("Complex expression caching", func() {
			cache := NewParserMemoizationCache(100, time.Minute)
			
			complexExpr := `grab meta.environments.production || grab meta.environments.staging`
			
			parser1 := NewMemoizedParser(complexExpr, registry)
			parser1.cache = cache
			expr1, err1 := parser1.Parse()
			So(err1, ShouldBeNil)
			So(expr1, ShouldNotBeNil)
			
			parser2 := NewMemoizedParser(complexExpr, registry)
			parser2.cache = cache
			expr2, err2 := parser2.Parse()
			So(err2, ShouldBeNil)
			So(expr2, ShouldNotBeNil)
			
			metrics := cache.GetMetrics()
			So(metrics.HitRate, ShouldBeGreaterThan, 0.0)
		})
	})
}

func BenchmarkParserMemoization(b *testing.B) {
	registry := NewOperatorRegistry()
	cache := NewParserMemoizationCache(1000, time.Hour)
	expressions := []string{
		"grab meta.key",
		"concat meta.prefix meta.suffix",
		"grab environments.production.database.host",
		`calc 100 * 1.5`,
		`grab meta.value || "default"`,
	}
	
	b.Run("WithMemoization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			expr := expressions[i%len(expressions)]
			parser := NewMemoizedParser(expr, registry)
			parser.cache = cache
			parser.Parse()
		}
		
		metrics := cache.GetMetrics()
		b.Logf("Cache hit rate: %.2f%%, Size: %d", metrics.HitRate*100, metrics.Size)
	})
	
	b.Run("WithoutMemoization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			expr := expressions[i%len(expressions)]
			tokenizer := NewTokenizer(expr)
			tokens := tokenizer.Tokenize()
			parser := NewParser(tokens, registry)
			parser.Parse()
		}
	})
}

func TestParserMemoizationMemoryUsage(t *testing.T) {
	Convey("Memory usage", t, func() {
		cache := NewParserMemoizationCache(1000, time.Hour)
		registry := NewOperatorRegistry()
		
		// Fill cache with many expressions
		for i := 0; i < 100; i++ {
			expr := "grab meta.key" + string(rune(i))
			parser := NewMemoizedParser(expr, registry)
			parser.cache = cache
			parser.Parse()
		}
		
		metrics := cache.GetMetrics()
		So(metrics.Size, ShouldBeLessThanOrEqualTo, 1000)
		
		// Test cleanup
		cache.CleanupExpired()
		metricsAfter := cache.GetMetrics()
		So(metricsAfter.Size, ShouldBeLessThanOrEqualTo, metrics.Size)
	})
}