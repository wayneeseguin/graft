package internal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPClientPool manages a pool of HTTP clients for efficient connection reuse
type HTTPClientPool struct {
	clients chan *http.Client
	factory func() *http.Client
	maxSize int
	created atomic.Int32
	hits    atomic.Uint64
	misses  atomic.Uint64
	mu      sync.RWMutex
	closed  bool
}

// HTTPClientPoolConfig holds configuration for HTTP client pool
type HTTPClientPoolConfig struct {
	MaxClients      int
	IdleTimeout     time.Duration
	ConnectTimeout  time.Duration
	RequestTimeout  time.Duration
	MaxIdleConns    int
	MaxConnsPerHost int
}

// NewHTTPClientPool creates a new HTTP client pool with the given configuration
func NewHTTPClientPool(config HTTPClientPoolConfig) *HTTPClientPool {
	if config.MaxClients <= 0 {
		config.MaxClients = 10
	}

	pool := &HTTPClientPool{
		clients: make(chan *http.Client, config.MaxClients),
		maxSize: config.MaxClients,
	}

	// Client factory function
	pool.factory = func() *http.Client {
		transport := &http.Transport{
			MaxIdleConns:        config.MaxIdleConns,
			MaxIdleConnsPerHost: config.MaxConnsPerHost,
			IdleConnTimeout:     config.IdleTimeout,
			DisableKeepAlives:   false,
		}

		if config.ConnectTimeout > 0 {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				d := &net.Dialer{Timeout: config.ConnectTimeout}
				return d.DialContext(ctx, network, addr)
			}
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   config.RequestTimeout,
		}

		return client
	}

	// Pre-populate pool with initial clients
	initialClients := config.MaxClients / 2
	if initialClients < 1 {
		initialClients = 1
	}

	for i := 0; i < initialClients; i++ {
		pool.clients <- pool.factory()
		pool.created.Add(1)
	}

	return pool
}

// Get retrieves an HTTP client from the pool
func (pool *HTTPClientPool) Get() *http.Client {
	pool.mu.RLock()
	if pool.closed {
		pool.mu.RUnlock()
		pool.misses.Add(1)
		return pool.factory() // Return new client if pool is closed
	}
	pool.mu.RUnlock()

	select {
	case client := <-pool.clients:
		pool.hits.Add(1)
		return client
	default:
		// Pool is empty, create new client if under limit
		maxSize := pool.maxSize
		if maxSize > 2147483647 { // Max int32
			maxSize = 2147483647
		}
		// #nosec G115 - bounds checked above
		if pool.created.Load() < int32(maxSize) {
			pool.created.Add(1)
			pool.misses.Add(1)
			return pool.factory()
		}

		// Wait for a client to become available with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		select {
		case client := <-pool.clients:
			pool.hits.Add(1)
			return client
		case <-ctx.Done():
			pool.misses.Add(1)
			return pool.factory() // Create temporary client
		}
	}
}

// Put returns an HTTP client to the pool
func (pool *HTTPClientPool) Put(client *http.Client) {
	if client == nil {
		return
	}

	pool.mu.RLock()
	if pool.closed {
		pool.mu.RUnlock()
		return
	}
	pool.mu.RUnlock()

	select {
	case pool.clients <- client:
		// Successfully returned to pool
	default:
		// Pool is full, let client be garbage collected
		pool.created.Add(-1)
	}
}

// Close shuts down the pool and releases all clients
func (pool *HTTPClientPool) Close() {
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return
	}
	pool.closed = true
	pool.mu.Unlock()

	// Drain the pool
	close(pool.clients)
	for client := range pool.clients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
}

// Metrics returns pool usage metrics
func (pool *HTTPClientPool) Metrics() HTTPClientPoolMetrics {
	return HTTPClientPoolMetrics{
		MaxSize:   pool.maxSize,
		Created:   int(pool.created.Load()),
		Available: len(pool.clients),
		Hits:      pool.hits.Load(),
		Misses:    pool.misses.Load(),
		HitRate:   pool.calculateHitRate(),
	}
}

// calculateHitRate calculates the cache hit rate as a percentage
func (pool *HTTPClientPool) calculateHitRate() float64 {
	hits := pool.hits.Load()
	misses := pool.misses.Load()
	total := hits + misses

	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total) * 100.0
}

// HTTPClientPoolMetrics holds metrics for HTTP client pool
type HTTPClientPoolMetrics struct {
	MaxSize   int
	Created   int
	Available int
	Hits      uint64
	Misses    uint64
	HitRate   float64
}

// String returns a string representation of the metrics
func (m HTTPClientPoolMetrics) String() string {
	return fmt.Sprintf("HTTP Pool - Max: %d, Created: %d, Available: %d, Hits: %d, Misses: %d, Hit Rate: %.2f%%",
		m.MaxSize, m.Created, m.Available, m.Hits, m.Misses, m.HitRate)
}

// VaultClientPool manages a pool of Vault clients
type VaultClientPool struct {
	clients chan *VaultClientWrapper
	factory func() (*VaultClientWrapper, error)
	config  VaultClientPoolConfig
	maxSize int
	created atomic.Int32
	hits    atomic.Uint64
	misses  atomic.Uint64
	mu      sync.RWMutex
	closed  bool
}

// VaultClientWrapper wraps a vault client with metadata
type VaultClientWrapper struct {
	Client    interface{} // *vaultkv.KV - using interface{} to avoid import issues
	CreatedAt time.Time
	LastUsed  atomic.Value // time.Time
	UseCount  atomic.Uint64
}

// UpdateLastUsed updates the last used timestamp
func (vcw *VaultClientWrapper) UpdateLastUsed() {
	vcw.LastUsed.Store(time.Now())
	vcw.UseCount.Add(1)
}

// GetLastUsed returns the last used timestamp
func (vcw *VaultClientWrapper) GetLastUsed() time.Time {
	if t, ok := vcw.LastUsed.Load().(time.Time); ok {
		return t
	}
	return vcw.CreatedAt
}

// VaultClientPoolConfig holds configuration for Vault client pool
type VaultClientPoolConfig struct {
	MaxClients   int
	IdleTimeout  time.Duration
	MaxIdleTime  time.Duration
	ReuseClients bool
}

// NewVaultClientPool creates a new Vault client pool
func NewVaultClientPool(config VaultClientPoolConfig, factory func() (interface{}, error)) *VaultClientPool {
	if config.MaxClients <= 0 {
		config.MaxClients = 5
	}

	pool := &VaultClientPool{
		clients: make(chan *VaultClientWrapper, config.MaxClients),
		config:  config,
		maxSize: config.MaxClients,
	}

	// Wrapper factory
	pool.factory = func() (*VaultClientWrapper, error) {
		client, err := factory()
		if err != nil {
			return nil, err
		}

		wrapper := &VaultClientWrapper{
			Client:    client,
			CreatedAt: time.Now(),
		}
		wrapper.LastUsed.Store(time.Now())

		return wrapper, nil
	}

	return pool
}

// Get retrieves a Vault client from the pool
func (pool *VaultClientPool) Get() (interface{}, error) {
	pool.mu.RLock()
	if pool.closed {
		pool.mu.RUnlock()
		wrapper, err := pool.factory()
		if err != nil {
			pool.misses.Add(1)
			return nil, err
		}
		return wrapper.Client, nil
	}
	pool.mu.RUnlock()

	// Try to get from pool first
	select {
	case wrapper := <-pool.clients:
		// Check if client is still valid
		if pool.config.MaxIdleTime > 0 && time.Since(wrapper.GetLastUsed()) > pool.config.MaxIdleTime {
			// Client has been idle too long, create new one
			pool.created.Add(-1)
			return pool.createNewClient()
		}

		wrapper.UpdateLastUsed()
		pool.hits.Add(1)
		return wrapper.Client, nil
	default:
		// Pool is empty, create new client if under limit
		return pool.createNewClient()
	}
}

// createNewClient creates a new client, respecting the pool limit
func (pool *VaultClientPool) createNewClient() (interface{}, error) {
	maxSize := pool.maxSize
	if maxSize > 2147483647 { // Max int32
		maxSize = 2147483647
	}
	// #nosec G115 - bounds checked above
	if pool.created.Load() < int32(maxSize) {
		wrapper, err := pool.factory()
		if err != nil {
			pool.misses.Add(1)
			return nil, err
		}
		pool.created.Add(1)
		pool.misses.Add(1)
		return wrapper.Client, nil
	}

	// Pool is at capacity, wait for a client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	select {
	case wrapper := <-pool.clients:
		wrapper.UpdateLastUsed()
		pool.hits.Add(1)
		return wrapper.Client, nil
	case <-ctx.Done():
		// Timeout, create temporary client
		wrapper, err := pool.factory()
		if err != nil {
			pool.misses.Add(1)
			return nil, err
		}
		pool.misses.Add(1)
		return wrapper.Client, nil
	}
}

// Put returns a Vault client to the pool
func (pool *VaultClientPool) Put(client interface{}) {
	if client == nil || !pool.config.ReuseClients {
		return
	}

	pool.mu.RLock()
	if pool.closed {
		pool.mu.RUnlock()
		return
	}
	pool.mu.RUnlock()

	wrapper := &VaultClientWrapper{
		Client:    client,
		CreatedAt: time.Now(),
	}
	wrapper.LastUsed.Store(time.Now())

	select {
	case pool.clients <- wrapper:
		// Successfully returned to pool
	default:
		// Pool is full, let client be garbage collected
		pool.created.Add(-1)
	}
}

// Close shuts down the Vault client pool
func (pool *VaultClientPool) Close() {
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return
	}
	pool.closed = true
	pool.mu.Unlock()

	// Drain the pool
	close(pool.clients)
	for range pool.clients {
		// Just drain, vault clients don't need explicit cleanup
	}
}

// Metrics returns pool usage metrics
func (pool *VaultClientPool) Metrics() VaultClientPoolMetrics {
	return VaultClientPoolMetrics{
		MaxSize:   pool.maxSize,
		Created:   int(pool.created.Load()),
		Available: len(pool.clients),
		Hits:      pool.hits.Load(),
		Misses:    pool.misses.Load(),
		HitRate:   pool.calculateHitRate(),
	}
}

// calculateHitRate calculates the cache hit rate as a percentage
func (pool *VaultClientPool) calculateHitRate() float64 {
	hits := pool.hits.Load()
	misses := pool.misses.Load()
	total := hits + misses

	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total) * 100.0
}

// VaultClientPoolMetrics holds metrics for Vault client pool
type VaultClientPoolMetrics struct {
	MaxSize   int
	Created   int
	Available int
	Hits      uint64
	Misses    uint64
	HitRate   float64
}

// String returns a string representation of the metrics
func (m VaultClientPoolMetrics) String() string {
	return fmt.Sprintf("Vault Pool - Max: %d, Created: %d, Available: %d, Hits: %d, Misses: %d, Hit Rate: %.2f%%",
		m.MaxSize, m.Created, m.Available, m.Hits, m.Misses, m.HitRate)
}

// Global pools for different services
var (
	// HTTPClientPool for general HTTP operations
	DefaultHTTPPool *HTTPClientPool

	// VaultClientPool for Vault operations
	DefaultVaultPool *VaultClientPool

	poolInitOnce sync.Once
)

// InitializePools initializes the global connection pools
func InitializePools() {
	poolInitOnce.Do(func() {
		// Initialize HTTP pool
		DefaultHTTPPool = NewHTTPClientPool(HTTPClientPoolConfig{
			MaxClients:      20,
			IdleTimeout:     90 * time.Second,
			ConnectTimeout:  10 * time.Second,
			RequestTimeout:  30 * time.Second,
			MaxIdleConns:    100,
			MaxConnsPerHost: 10,
		})

		// Note: Vault pool will be initialized when first vault client is created
		// since it requires vault configuration
	})
}

// GetHTTPClient returns an HTTP client from the global pool
func GetHTTPClient() *http.Client {
	if DefaultHTTPPool == nil {
		InitializePools()
	}
	return DefaultHTTPPool.Get()
}

// PutHTTPClient returns an HTTP client to the global pool
func PutHTTPClient(client *http.Client) {
	if DefaultHTTPPool != nil {
		DefaultHTTPPool.Put(client)
	}
}

// GetConnectionPoolMetrics returns metrics for all connection pools
func GetConnectionPoolMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	if DefaultHTTPPool != nil {
		metrics["http"] = DefaultHTTPPool.Metrics()
	}

	if DefaultVaultPool != nil {
		metrics["vault"] = DefaultVaultPool.Metrics()
	}

	return metrics
}
