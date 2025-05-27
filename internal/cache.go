package internal

import (
	"github.com/wayneeseguin/graft/pkg/graft"
)
import (
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// CacheItem represents a single cached value
type CacheItem struct {
	Key        string
	Value      interface{}
	ExpiresAt  time.Time
	CreatedAt  time.Time
	HitCount   atomic.Uint64
}

// IsExpired checks if the cache item has expired
func (ci *CacheItem) IsExpired() bool {
	return !ci.ExpiresAt.IsZero() && time.Now().After(ci.ExpiresAt)
}

// ConcurrentCache is a thread-safe cache with sharding for reduced contention
type ConcurrentCache struct {
	shards    []*CacheShard
	shardMask uint32
	ttl       time.Duration
	
	// Global metrics
	hits   atomic.Uint64
	misses atomic.Uint64
	sets   atomic.Uint64
	evicts atomic.Uint64
}

// CacheShard represents a single shard of the cache
type CacheShard struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	capacity int
}

// CacheConfig holds configuration for creating a cache
type CacheConfig struct {
	Shards   int           // Number of shards (must be power of 2)
	Capacity int           // Total capacity across all shards
	TTL      time.Duration // Default TTL for items
}

// NewConcurrentCache creates a new concurrent cache with the given configuration
func NewConcurrentCache(config CacheConfig) *ConcurrentCache {
	if config.Shards == 0 {
		config.Shards = 16 // Default to 16 shards
	}
	
	// Ensure shards is a power of 2
	shards := 1
	for shards < config.Shards {
		shards <<= 1
	}
	
	shardCapacity := config.Capacity / shards
	if shardCapacity < 1 {
		shardCapacity = 1
	}
	
	cache := &ConcurrentCache{
		shards:    make([]*CacheShard, shards),
		shardMask: uint32(shards - 1),
		ttl:       config.TTL,
	}
	
	for i := 0; i < shards; i++ {
		cache.shards[i] = &CacheShard{
			items:    make(map[string]*CacheItem),
			capacity: shardCapacity,
		}
	}
	
	return cache
}

// getShard returns the shard for a given key
func (c *ConcurrentCache) getShard(key string) *CacheShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return c.shards[h.Sum32()&c.shardMask]
}

// Get retrieves a value from the cache
func (c *ConcurrentCache) Get(key string) (interface{}, bool) {
	shard := c.getShard(key)
	
	shard.mu.RLock()
	item, found := shard.items[key]
	if !found {
		shard.mu.RUnlock()
		c.misses.Add(1)
		return nil, false
	}
	
	// Check expiration
	if item.IsExpired() {
		shard.mu.RUnlock()
		// Upgrade to write lock for deletion
		shard.mu.Lock()
		delete(shard.items, key)
		shard.mu.Unlock()
		
		c.misses.Add(1)
		c.evicts.Add(1)
		return nil, false
	}
	
	// Get the value while still holding the lock
	value := item.Value
	shard.mu.RUnlock()
	
	// Update metrics (atomic operations are safe without lock)
	item.HitCount.Add(1)
	c.hits.Add(1)
	
	// Note: We're not updating AccessTime to avoid write contention
	// LRU eviction will still work based on insertion order
	
	return value, true
}

// Set adds or updates a value in the cache
func (c *ConcurrentCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL adds or updates a value in the cache with a specific TTL
func (c *ConcurrentCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	shard := c.getShard(key)
	
	item := &CacheItem{
		Key:        key,
		Value:      value,
		CreatedAt:  time.Now(),
	}
	
	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	}
	
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	// Check capacity and evict if necessary
	if len(shard.items) >= shard.capacity {
		c.evictLRU(shard)
	}
	
	shard.items[key] = item
	c.sets.Add(1)
}

// evictLRU evicts the least recently used item from a shard
// Must be called with shard.mu held
func (c *ConcurrentCache) evictLRU(shard *CacheShard) {
	var lruKey string
	var lruTime time.Time
	var minHits uint64 = ^uint64(0) // Max uint64
	
	// Evict item with oldest creation time and fewest hits
	for key, item := range shard.items {
		hits := item.HitCount.Load()
		if lruKey == "" || hits < minHits || (hits == minHits && item.CreatedAt.Before(lruTime)) {
			lruKey = key
			lruTime = item.CreatedAt
			minHits = hits
		}
	}
	
	if lruKey != "" {
		delete(shard.items, lruKey)
		c.evicts.Add(1)
	}
}

// Delete removes a value from the cache
func (c *ConcurrentCache) Delete(key string) bool {
	shard := c.getShard(key)
	
	shard.mu.Lock()
	_, found := shard.items[key]
	if found {
		delete(shard.items, key)
	}
	shard.mu.Unlock()
	
	return found
}

// Clear removes all items from the cache
func (c *ConcurrentCache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.items = make(map[string]*CacheItem)
		shard.mu.Unlock()
	}
}

// Size returns the total number of items in the cache
func (c *ConcurrentCache) Size() int {
	total := 0
	for _, shard := range c.shards {
		shard.mu.RLock()
		total += len(shard.items)
		shard.mu.RUnlock()
	}
	return total
}

// Metrics returns current cache metrics
func (c *ConcurrentCache) Metrics() CacheMetrics {
	return CacheMetrics{
		Hits:   c.hits.Load(),
		Misses: c.misses.Load(),
		Sets:   c.sets.Load(),
		Evicts: c.evicts.Load(),
		Size:   c.Size(),
		HitRate: c.calculateHitRate(),
	}
}

// calculateHitRate returns the cache hit rate as a percentage
func (c *ConcurrentCache) calculateHitRate() float64 {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	
	if total == 0 {
		return 0.0
	}
	
	return float64(hits) / float64(total) * 100.0
}

// CacheMetrics holds runtime metrics for the cache
type CacheMetrics struct {
	Hits    uint64
	Misses  uint64
	Sets    uint64
	Evicts  uint64
	Size    int
	HitRate float64
}

// String returns a string representation of cache metrics
func (m CacheMetrics) String() string {
	return fmt.Sprintf("Cache Metrics - Hits: %d, Misses: %d, Sets: %d, Evicts: %d, Size: %d, Hit Rate: %.2f%%",
		m.Hits, m.Misses, m.Sets, m.Evicts, m.Size, m.HitRate)
}

// Global caches for different operator types
var (
	// ExpressionCache caches parsed expressions
	ExpressionCache = NewConcurrentCache(CacheConfig{
		Shards:   16,
		Capacity: 10000,
		TTL:      0, // No expiration
	})
	
	// OperatorCache caches operator results
	OperatorCache = NewConcurrentCache(CacheConfig{
		Shards:   32,
		Capacity: 50000,
		TTL:      5 * time.Minute,
	})
	
	// VaultCache caches vault lookups
	VaultCache = NewConcurrentCache(CacheConfig{
		Shards:   16,
		Capacity: 5000,
		TTL:      1 * time.Minute,
	})
	
	// AWSCache caches AWS parameter/secret lookups
	AWSCache = NewConcurrentCache(CacheConfig{
		Shards:   16,
		Capacity: 5000,
		TTL:      5 * time.Minute,
	})
)