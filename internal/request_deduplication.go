package internal

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// RequestResult represents the result of a deduplicated request
type RequestResult struct {
	Value interface{}
	Err   error
}

// PendingRequest represents a request that is currently being processed
type PendingRequest struct {
	resultChan chan RequestResult
	waiters    []chan RequestResult
	started    time.Time
	completed  bool
	mu         sync.Mutex
}

// AddWaiter adds a new waiter to the pending request
func (pr *PendingRequest) AddWaiter() <-chan RequestResult {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	waiter := make(chan RequestResult, 1)
	pr.waiters = append(pr.waiters, waiter)
	return waiter
}

// Complete completes the pending request and notifies all waiters
func (pr *PendingRequest) Complete(result RequestResult) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	if pr.completed {
		return // Already completed
	}
	
	pr.completed = true
	
	// Send result to primary channel
	select {
	case pr.resultChan <- result:
	default:
		// Channel might be closed or full
	}
	
	// Send result to all waiters
	for _, waiter := range pr.waiters {
		select {
		case waiter <- result:
		default:
			// Waiter might have timed out
		}
	}
}

// RequestDeduplicator manages request deduplication to prevent duplicate work
type RequestDeduplicator struct {
	pending   map[string]*PendingRequest
	mu        sync.RWMutex
	timeout   time.Duration
	
	// Metrics
	hits      atomic.Uint64
	misses    atomic.Uint64
	timeouts  atomic.Uint64
	errors    atomic.Uint64
}

// RequestDeduplicatorConfig holds configuration for request deduplication
type RequestDeduplicatorConfig struct {
	Timeout        time.Duration
	CleanupInterval time.Duration
}

// NewRequestDeduplicator creates a new request deduplicator
func NewRequestDeduplicator(config RequestDeduplicatorConfig) *RequestDeduplicator {
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	
	rd := &RequestDeduplicator{
		pending: make(map[string]*PendingRequest),
		timeout: config.Timeout,
	}
	
	// Start cleanup goroutine
	go rd.cleanupLoop(config.CleanupInterval)
	
	return rd
}

// Deduplicate deduplicates a request based on the given key
func (rd *RequestDeduplicator) Deduplicate(key string, fn func() (interface{}, error)) <-chan RequestResult {
	return rd.DeduplicateWithContext(context.Background(), key, fn)
}

// DeduplicateWithContext deduplicates a request with context support
func (rd *RequestDeduplicator) DeduplicateWithContext(ctx context.Context, key string, fn func() (interface{}, error)) <-chan RequestResult {
	// Check if request is already in progress
	rd.mu.RLock()
	if pending, exists := rd.pending[key]; exists {
		rd.mu.RUnlock()
		rd.hits.Add(1)
		
		// Request is already in progress, add waiter
		return pending.AddWaiter()
	}
	rd.mu.RUnlock()
	
	// Start new request
	rd.mu.Lock()
	
	// Double-check in case another goroutine started it
	if pending, exists := rd.pending[key]; exists {
		rd.mu.Unlock()
		rd.hits.Add(1)
		return pending.AddWaiter()
	}
	
	// Create new pending request
	pending := &PendingRequest{
		resultChan: make(chan RequestResult, 1),
		waiters:    []chan RequestResult{},
		started:    time.Now(),
	}
	rd.pending[key] = pending
	rd.mu.Unlock()
	
	rd.misses.Add(1)
	
	// Execute the request in a separate goroutine
	go func() {
		defer func() {
			// Clean up pending request
			rd.mu.Lock()
			delete(rd.pending, key)
			rd.mu.Unlock()
		}()
		
		// Create context with timeout
		reqCtx, cancel := context.WithTimeout(ctx, rd.timeout)
		defer cancel()
		
		// Channel to receive the result
		resultChan := make(chan RequestResult, 1)
		
		// Execute the function in a separate goroutine
		go func() {
			value, err := fn()
			resultChan <- RequestResult{Value: value, Err: err}
		}()
		
		// Wait for result or timeout
		select {
		case result := <-resultChan:
			if result.Err != nil {
				rd.errors.Add(1)
			}
			pending.Complete(result)
		case <-reqCtx.Done():
			rd.timeouts.Add(1)
			pending.Complete(RequestResult{
				Err: fmt.Errorf("request timeout after %v", rd.timeout),
			})
		}
	}()
	
	return pending.resultChan
}

// cleanupLoop periodically cleans up old pending requests
func (rd *RequestDeduplicator) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for range ticker.C {
		rd.cleanup()
	}
}

// cleanup removes old completed or timed out requests
func (rd *RequestDeduplicator) cleanup() {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	
	cutoff := time.Now().Add(-rd.timeout * 2) // Keep for 2x timeout duration
	
	for key, pending := range rd.pending {
		if pending.started.Before(cutoff) {
			delete(rd.pending, key)
		}
	}
}

// GetMetrics returns deduplication metrics
func (rd *RequestDeduplicator) GetMetrics() RequestDeduplicationMetrics {
	rd.mu.RLock()
	pendingCount := len(rd.pending)
	rd.mu.RUnlock()
	
	return RequestDeduplicationMetrics{
		Hits:      rd.hits.Load(),
		Misses:    rd.misses.Load(),
		Timeouts:  rd.timeouts.Load(),
		Errors:    rd.errors.Load(),
		Pending:   pendingCount,
		HitRate:   rd.calculateHitRate(),
	}
}

// calculateHitRate calculates the hit rate as a percentage
func (rd *RequestDeduplicator) calculateHitRate() float64 {
	hits := rd.hits.Load()
	misses := rd.misses.Load()
	total := hits + misses
	
	if total == 0 {
		return 0.0
	}
	
	return float64(hits) / float64(total) * 100.0
}

// RequestDeduplicationMetrics holds metrics for request deduplication
type RequestDeduplicationMetrics struct {
	Hits     uint64
	Misses   uint64
	Timeouts uint64
	Errors   uint64
	Pending  int
	HitRate  float64
}

// String returns a string representation of the metrics
func (m RequestDeduplicationMetrics) String() string {
	return fmt.Sprintf("Deduplication - Hits: %d, Misses: %d, Timeouts: %d, Errors: %d, Pending: %d, Hit Rate: %.2f%%",
		m.Hits, m.Misses, m.Timeouts, m.Errors, m.Pending, m.HitRate)
}

// KeyBuilder helps build consistent keys for deduplication
type KeyBuilder struct {
	prefix string
}

// NewKeyBuilder creates a new key builder with the given prefix
func NewKeyBuilder(prefix string) *KeyBuilder {
	return &KeyBuilder{prefix: prefix}
}

// BuildKey builds a deduplication key from the given components
func (kb *KeyBuilder) BuildKey(components ...string) string {
	hasher := sha256.New()
	hasher.Write([]byte(kb.prefix))
	
	for _, component := range components {
		hasher.Write([]byte(":"))
		hasher.Write([]byte(component))
	}
	
	return fmt.Sprintf("%s:%x", kb.prefix, hasher.Sum(nil)[:8])
}

// VaultKeyBuilder builds keys for Vault operations
var VaultKeyBuilder = NewKeyBuilder("vault")

// AWSKeyBuilder builds keys for AWS operations
var AWSKeyBuilder = NewKeyBuilder("aws")

// FileKeyBuilder builds keys for file operations
var FileKeyBuilder = NewKeyBuilder("file")

// Global deduplicators for different operation types
var (
	VaultDeduplicator *RequestDeduplicator
	AWSDeduplicator   *RequestDeduplicator
	FileDeduplicator  *RequestDeduplicator
	
	deduplicatorInitOnce sync.Once
)

// InitializeDeduplicators initializes the global request deduplicators
func InitializeDeduplicators() {
	deduplicatorInitOnce.Do(func() {
		config := RequestDeduplicatorConfig{
			Timeout:         30 * time.Second,
			CleanupInterval: 5 * time.Minute,
		}
		
		VaultDeduplicator = NewRequestDeduplicator(config)
		AWSDeduplicator = NewRequestDeduplicator(config)
		FileDeduplicator = NewRequestDeduplicator(config)
	})
}

// GetDeduplicationMetrics returns metrics for all deduplicators
func GetDeduplicationMetrics() map[string]RequestDeduplicationMetrics {
	if VaultDeduplicator == nil {
		InitializeDeduplicators()
	}
	
	return map[string]RequestDeduplicationMetrics{
		"vault": VaultDeduplicator.GetMetrics(),
		"aws":   AWSDeduplicator.GetMetrics(),
		"file":  FileDeduplicator.GetMetrics(),
	}
}