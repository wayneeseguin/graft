package internal

import (
	"sync"
	"time"

	"github.com/wayneeseguin/graft/internal/cache"
)

var (
	globalCacheAnalytics *cache.CacheAnalytics
	cacheAnalyticsMu     sync.RWMutex
)

// InitializeCacheAnalytics initializes the global cache analytics instance
func InitializeCacheAnalytics() {
	cacheAnalyticsMu.Lock()
	defer cacheAnalyticsMu.Unlock()
	globalCacheAnalytics = cache.NewCacheAnalytics(24 * time.Hour)
}

// GetCacheAnalytics returns the global cache analytics instance
func GetCacheAnalytics() *cache.CacheAnalytics {
	cacheAnalyticsMu.RLock()
	defer cacheAnalyticsMu.RUnlock()
	return globalCacheAnalytics
}