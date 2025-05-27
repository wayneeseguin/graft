package internal

import (
	"github.com/wayneeseguin/graft/pkg/graft"
)
import (
	"strings"
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCacheKeyGeneration(t *testing.T) {
	Convey("Cache Key Generation", t, func() {
		
		Convey("Basic key generation", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{
				Algorithm: "fnv",
				Compression: false,
				CollisionDetection: false,
			})
			
			key1 := generator.GenerateExpressionKey("grab meta.key", nil)
			key2 := generator.GenerateExpressionKey("grab meta.key", nil)
			key3 := generator.GenerateExpressionKey("grab meta.other", nil)
			
			So(key1, ShouldEqual, key2) // Same input should produce same key
			So(key1, ShouldNotEqual, key3) // Different input should produce different key
			So(len(key1), ShouldBeGreaterThan, 0)
		})
		
		Convey("Algorithm selection", func() {
			fnvGenerator := NewCacheKeyGenerator(CacheKeyConfig{Algorithm: "fnv"})
			sha256Generator := NewCacheKeyGenerator(CacheKeyConfig{Algorithm: "sha256"})
			
			fnvKey := fnvGenerator.GenerateExpressionKey("test", nil)
			sha256Key := sha256Generator.GenerateExpressionKey("test", nil)
			
			So(fnvKey, ShouldNotEqual, sha256Key)
			So(strings.HasPrefix(fnvKey, "fnv_"), ShouldBeTrue)
			So(strings.HasPrefix(sha256Key, "sha_"), ShouldBeTrue)
		})
		
		Convey("Compression", func() {
			compressedGen := NewCacheKeyGenerator(CacheKeyConfig{
				Algorithm: "fnv",
				Compression: true,
			})
			uncompressedGen := NewCacheKeyGenerator(CacheKeyConfig{
				Algorithm: "fnv", 
				Compression: false,
			})
			
			input := "grab meta.environments.production.database.host"
			
			compressedKey := compressedGen.GenerateExpressionKey(input, nil)
			uncompressedKey := uncompressedGen.GenerateExpressionKey(input, nil)
			
			So(compressedKey, ShouldNotEqual, uncompressedKey)
			
			// Test compression patterns
			compressed := compressedGen.compressInput(input)
			So(compressed, ShouldContainSubstring, "m.") // meta. -> m.
			So(compressed, ShouldContainSubstring, "e.") // environments. -> e.
			So(compressed, ShouldContainSubstring, "prod") // production -> prod
			So(compressed, ShouldContainSubstring, "db") // database -> db
		})
		
		Convey("Collision detection", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{
				Algorithm: "fnv",
				CollisionDetection: true,
			})
			
			// Force a collision by using the same hash but different original
			hash := "test_hash"
			original1 := "original1"
			original2 := "original2"
			
			key1 := generator.checkCollision(hash, original1)
			key2 := generator.checkCollision(hash, original2)
			
			So(key1, ShouldEqual, hash) // First should get the original hash
			So(key2, ShouldNotEqual, hash) // Second should get a modified hash
			So(key2, ShouldContainSubstring, hash) // Should contain original hash
			
			stats := generator.GetCollisionStats()
			So(stats["total_keys"], ShouldEqual, 2)
			So(stats["collisions"].(int), ShouldBeGreaterThanOrEqualTo, 1)
		})
		
		Convey("Operator key generation", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{Algorithm: "fnv"})
			
			key1 := generator.GenerateOperatorKey("vault", []string{"secret/path", "key"}, "prod")
			key2 := generator.GenerateOperatorKey("vault", []string{"secret/path", "key"}, "prod")
			key3 := generator.GenerateOperatorKey("vault", []string{"secret/other", "key"}, "prod")
			
			So(key1, ShouldEqual, key2)
			So(key1, ShouldNotEqual, key3)
			So(len(key1), ShouldBeGreaterThan, 0)
		})
		
		Convey("Token key generation", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{Algorithm: "fnv"})
			
			key1 := generator.GenerateTokenKey("grab meta.key")
			key2 := generator.GenerateTokenKey("grab meta.key")
			key3 := generator.GenerateTokenKey("concat a b")
			
			So(key1, ShouldEqual, key2)
			So(key1, ShouldNotEqual, key3)
			So(len(key1), ShouldBeGreaterThan, 0)
		})
		
		Convey("Partial result key generation", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{Algorithm: "fnv"})
			
			key1 := generator.GeneratePartialResultKey("meta.database", "operator", "vault secret/db")
			key2 := generator.GeneratePartialResultKey("meta.database", "operator", "vault secret/db")
			key3 := generator.GeneratePartialResultKey("meta.cache", "operator", "vault secret/cache")
			
			So(key1, ShouldEqual, key2)
			So(key1, ShouldNotEqual, key3)
			So(len(key1), ShouldBeGreaterThan, 0)
		})
		
		Convey("Registry signature", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{Algorithm: "fnv"})
			registry := NewOperatorRegistry()
			
			sig1 := generator.registrySignature(registry)
			sig2 := generator.registrySignature(registry)
			sig3 := generator.registrySignature(nil)
			
			So(sig1, ShouldEqual, sig2)
			So(sig1, ShouldNotEqual, sig3)
			So(sig3, ShouldEqual, "noreg")
			So(len(sig1), ShouldBeGreaterThan, 0)
		})
		
		Convey("Fast key functions", func() {
			key1 := FastExpressionKey("grab meta.test")
			key2 := FastOperatorKey("vault", []string{"secret/path"})
			key3 := FastTokenKey("grab meta.test")
			
			So(len(key1), ShouldBeGreaterThan, 0)
			So(len(key2), ShouldBeGreaterThan, 0)
			So(len(key3), ShouldBeGreaterThan, 0)
			
			// Keys should be different for different types
			So(key1, ShouldNotEqual, key2)
			So(key1, ShouldNotEqual, key3)
		})
		
		Convey("Key consistency", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{
				Algorithm: "fnv",
				Compression: true,
				CollisionDetection: true,
			})
			
			// Generate the same key multiple times
			input := "grab meta.environments.production.database.host"
			keys := make([]string, 100)
			
			for i := 0; i < 100; i++ {
				keys[i] = generator.GenerateExpressionKey(input, nil)
			}
			
			// All keys should be identical
			firstKey := keys[0]
			for i := 1; i < 100; i++ {
				So(keys[i], ShouldEqual, firstKey)
			}
		})
		
		Convey("Stats and reset", func() {
			generator := NewCacheKeyGenerator(CacheKeyConfig{
				Algorithm: "fnv",
				CollisionDetection: true,
			})
			
			// Generate some keys
			generator.GenerateExpressionKey("test1", nil)
			generator.GenerateExpressionKey("test2", nil)
			generator.GenerateOperatorKey("vault", []string{"path"}, "")
			
			stats := generator.GetCollisionStats()
			So(stats["total_keys"], ShouldBeGreaterThan, 0)
			So(stats["algorithm"], ShouldEqual, "fnv")
			So(stats["collision_rate"], ShouldBeGreaterThanOrEqualTo, 0.0)
			
			generator.Reset()
			statsAfter := generator.GetCollisionStats()
			So(statsAfter["total_keys"], ShouldEqual, 0)
		})
	})
}

func BenchmarkCacheKeyGeneration(b *testing.B) {
	generator := NewCacheKeyGenerator(CacheKeyConfig{
		Algorithm: "fnv",
		Compression: true,
		CollisionDetection: true,
	})
	
	registry := NewOperatorRegistry()
	
	b.Run("ExpressionKey", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			generator.GenerateExpressionKey("grab meta.environments.production.database.host", registry)
		}
	})
	
	b.Run("OperatorKey", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			generator.GenerateOperatorKey("vault", []string{"secret/path", "key"}, "production")
		}
	})
	
	b.Run("TokenKey", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			generator.GenerateTokenKey("grab meta.environments.production.database.host")
		}
	})
	
	b.Run("FastKeys", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			FastExpressionKey("grab meta.test")
			FastOperatorKey("vault", []string{"secret/path"})
			FastTokenKey("grab meta.test")
		}
	})
	
	b.Run("CompressionComparison", func(b *testing.B) {
		compressedGen := NewCacheKeyGenerator(CacheKeyConfig{
			Algorithm: "fnv",
			Compression: true,
		})
		uncompressedGen := NewCacheKeyGenerator(CacheKeyConfig{
			Algorithm: "fnv",
			Compression: false,
		})
		
		input := "grab meta.environments.production.database.host"
		
		b.ResetTimer()
		
		b.Run("WithCompression", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				compressedGen.GenerateExpressionKey(input, nil)
			}
		})
		
		b.Run("WithoutCompression", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				uncompressedGen.GenerateExpressionKey(input, nil)
			}
		})
	})
}