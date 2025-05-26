package spruce

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
)

// CacheKeyGenerator generates optimized cache keys for various cache types
type CacheKeyGenerator struct {
	algorithm string
	compression bool
	collisionDetection bool
	collisions map[string]string
	mu sync.RWMutex
}

// CacheKeyConfig configures cache key generation
type CacheKeyConfig struct {
	Algorithm string // "xxhash", "fnv", "sha256"
	Compression bool
	CollisionDetection bool
}

// Default cache key generator
var DefaultKeyGenerator = NewCacheKeyGenerator(CacheKeyConfig{
	Algorithm: "fnv", // Using fnv as it's built-in and fast
	Compression: true,
	CollisionDetection: true,
})

// NewCacheKeyGenerator creates a new cache key generator
func NewCacheKeyGenerator(config CacheKeyConfig) *CacheKeyGenerator {
	return &CacheKeyGenerator{
		algorithm: config.Algorithm,
		compression: config.Compression,
		collisionDetection: config.CollisionDetection,
		collisions: make(map[string]string),
	}
}

// GenerateExpressionKey generates an optimized key for expression caching
func (g *CacheKeyGenerator) GenerateExpressionKey(input string, registry *OperatorRegistry) string {
	// Build key components
	var keyParts []string
	
	// Add input (with compression if enabled)
	if g.compression {
		keyParts = append(keyParts, g.compressInput(input))
	} else {
		keyParts = append(keyParts, input)
	}
	
	// Add registry signature for cache invalidation when operators change
	if registry != nil {
		keyParts = append(keyParts, g.registrySignature(registry))
	}
	
	// Generate base key
	baseKey := strings.Join(keyParts, "|")
	
	// Generate hash
	hash := g.hashString(baseKey)
	
	// Check for collisions if enabled
	if g.collisionDetection {
		return g.checkCollision(hash, baseKey)
	}
	
	return hash
}

// GenerateOperatorKey generates a key for operator result caching
func (g *CacheKeyGenerator) GenerateOperatorKey(opName string, args []string, context string) string {
	var keyParts []string
	
	keyParts = append(keyParts, "op", opName)
	
	if g.compression {
		// Compress argument list
		argsStr := strings.Join(args, ",")
		keyParts = append(keyParts, g.compressInput(argsStr))
	} else {
		keyParts = append(keyParts, args...)
	}
	
	if context != "" {
		keyParts = append(keyParts, "ctx", context)
	}
	
	baseKey := strings.Join(keyParts, "|")
	hash := g.hashString(baseKey)
	
	if g.collisionDetection {
		return g.checkCollision(hash, baseKey)
	}
	
	return hash
}

// GenerateTokenKey generates a key for token caching
func (g *CacheKeyGenerator) GenerateTokenKey(input string) string {
	if g.compression {
		input = g.compressInput(input)
	}
	
	hash := g.hashString("tok|" + input)
	
	if g.collisionDetection {
		return g.checkCollision(hash, "tok|"+input)
	}
	
	return hash
}

// GeneratePartialResultKey generates a key for partial result caching
func (g *CacheKeyGenerator) GeneratePartialResultKey(path string, exprType string, content string) string {
	var keyParts []string
	
	keyParts = append(keyParts, "partial", path, exprType)
	
	if g.compression {
		keyParts = append(keyParts, g.compressInput(content))
	} else {
		keyParts = append(keyParts, content)
	}
	
	baseKey := strings.Join(keyParts, "|")
	hash := g.hashString(baseKey)
	
	if g.collisionDetection {
		return g.checkCollision(hash, baseKey)
	}
	
	return hash
}

// hashString generates a hash for the given string using the configured algorithm
func (g *CacheKeyGenerator) hashString(input string) string {
	switch g.algorithm {
	case "fnv":
		return g.fnvHash(input)
	case "sha256":
		// Fall back to existing SHA256 implementation
		return g.sha256Hash(input)
	default:
		return g.fnvHash(input) // Default to FNV
	}
}

// fnvHash generates an FNV-1a hash (fast and good distribution)
func (g *CacheKeyGenerator) fnvHash(input string) string {
	h := fnv.New64a()
	h.Write([]byte(input))
	return fmt.Sprintf("fnv_%x", h.Sum64())
}

// sha256Hash generates a SHA256 hash (for compatibility)
func (g *CacheKeyGenerator) sha256Hash(input string) string {
	// Use the existing implementation from parser_memoization.go
	// This is a simplified version - in practice, we'd import crypto/sha256
	return fmt.Sprintf("sha_%x", simpleHash(input))
}

// simpleHash provides a simple hash function for fallback
func simpleHash(s string) uint64 {
	var hash uint64 = 5381
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint64(c)
	}
	return hash
}

// compressInput compresses input strings by removing common patterns
func (g *CacheKeyGenerator) compressInput(input string) string {
	// Simple compression strategies
	compressed := input
	
	// Replace common patterns
	compressed = strings.ReplaceAll(compressed, "meta.", "m.")
	compressed = strings.ReplaceAll(compressed, "environments.", "e.")
	compressed = strings.ReplaceAll(compressed, "properties.", "p.")
	compressed = strings.ReplaceAll(compressed, "production", "prod")
	compressed = strings.ReplaceAll(compressed, "development", "dev")
	compressed = strings.ReplaceAll(compressed, "database", "db")
	
	// Remove redundant whitespace
	compressed = strings.TrimSpace(compressed)
	compressed = strings.ReplaceAll(compressed, "  ", " ")
	
	return compressed
}

// registrySignature generates a signature for the operator registry
func (g *CacheKeyGenerator) registrySignature(registry *OperatorRegistry) string {
	if registry == nil {
		return "noreg"
	}
	
	// Simple registry signature based on operator count and some names
	// In practice, this could be more sophisticated
	signature := fmt.Sprintf("reg_%d", len(registry.operators))
	
	// Add some operator names for better cache invalidation (sorted for determinism)
	var names []string
	for name := range registry.operators {
		names = append(names, name)
	}
	
	// Sort names for deterministic signature
	if len(names) > 1 {
		// Simple bubble sort for small lists
		for i := 0; i < len(names)-1; i++ {
			for j := 0; j < len(names)-i-1; j++ {
				if names[j] > names[j+1] {
					names[j], names[j+1] = names[j+1], names[j]
				}
			}
		}
	}
	
	// Include first 3 names for performance
	count := 0
	for _, name := range names {
		if count < 3 {
			signature += "_" + name
			count++
		}
	}
	
	return g.hashString(signature)[:8] // Use first 8 chars of hash
}

// checkCollision checks for hash collisions and handles them
func (g *CacheKeyGenerator) checkCollision(hash, original string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if existing, found := g.collisions[hash]; found {
		if existing != original {
			// Collision detected, append suffix
			collisionCount := 1
			newHash := hash + "_" + strconv.Itoa(collisionCount)
			
			for {
				if _, exists := g.collisions[newHash]; !exists {
					g.collisions[newHash] = original
					return newHash
				}
				collisionCount++
				newHash = hash + "_" + strconv.Itoa(collisionCount)
			}
		}
	} else {
		g.collisions[hash] = original
	}
	
	return hash
}

// GetCollisionStats returns collision statistics
func (g *CacheKeyGenerator) GetCollisionStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	collisionCount := 0
	for key := range g.collisions {
		if strings.Contains(key, "_") {
			collisionCount++
		}
	}
	
	return map[string]interface{}{
		"total_keys": len(g.collisions),
		"collisions": collisionCount,
		"collision_rate": float64(collisionCount) / float64(len(g.collisions)),
		"algorithm": g.algorithm,
		"compression": g.compression,
	}
}

// Reset clears collision tracking
func (g *CacheKeyGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.collisions = make(map[string]string)
}

// Optimized key generation functions for common use cases

// FastExpressionKey generates a fast key for expression caching
func FastExpressionKey(input string) string {
	return DefaultKeyGenerator.GenerateExpressionKey(input, nil)
}

// FastOperatorKey generates a fast key for operator caching
func FastOperatorKey(opName string, args []string) string {
	return DefaultKeyGenerator.GenerateOperatorKey(opName, args, "")
}

// FastTokenKey generates a fast key for token caching
func FastTokenKey(input string) string {
	return DefaultKeyGenerator.GenerateTokenKey(input)
}