package spruce

import (
	"context"
	"fmt"
	"strings"
	"sync"
	
	"github.com/cloudfoundry-community/vaultkv"
)

// VaultWorkerPool is the global worker pool for Vault operations
var (
	vaultWorkerPool     *WorkerPool
	vaultWorkerPoolOnce sync.Once
)

// GetVaultWorkerPool returns the global Vault worker pool, creating it if necessary
func GetVaultWorkerPool() *WorkerPool {
	vaultWorkerPoolOnce.Do(func() {
		vaultWorkerPool = NewWorkerPool(WorkerPoolConfig{
			Name:      "vault",
			Workers:   10,
			QueueSize: 100,
			RateLimit: 50, // 50 requests per second
		})
	})
	return vaultWorkerPool
}

// VaultTask represents a Vault lookup operation
type VaultTask struct {
	id       string
	path     string
	key      string
	client   *vaultkv.KV
	useCache bool
}

// NewVaultTask creates a new Vault task
func NewVaultTask(path, key string, client *vaultkv.KV) *VaultTask {
	id := fmt.Sprintf("vault:%s:%s", path, key)
	return &VaultTask{
		id:       id,
		path:     path,
		key:      key,
		client:   client,
		useCache: true,
	}
}

// ID returns the unique identifier for this task
func (t *VaultTask) ID() string {
	return t.id
}

// Execute performs the Vault lookup
func (t *VaultTask) Execute(ctx context.Context) (interface{}, error) {
	// Check cache first if enabled
	if t.useCache {
		cacheKey := fmt.Sprintf("%s:%s", t.path, t.key)
		if value, found := VaultCache.Get(cacheKey); found {
			return value, nil
		}
	}
	
	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Perform Vault lookup
	var secret map[string]interface{}
	_, err := t.client.Get(t.path, &secret, nil)
	if err != nil {
		return nil, fmt.Errorf("vault lookup failed for %s: %w", t.path, err)
	}
	
	// Extract the specific key if provided
	var result interface{}
	if t.key != "" {
		value, ok := secret[t.key]
		if !ok {
				return nil, fmt.Errorf("vault path %s does not contain key %s", t.path, t.key)
		}
		result = value
	} else {
		result = secret
	}
	
	// Cache the result
	if t.useCache {
		cacheKey := fmt.Sprintf("%s:%s", t.path, t.key)
		VaultCache.Set(cacheKey, result)
	}
	
	return result, nil
}

// VaultBatchTask represents multiple Vault lookups that can be batched
type VaultBatchTask struct {
	id       string
	requests []VaultRequest
	client   *vaultkv.KV
}

// VaultRequest represents a single request in a batch
type VaultRequest struct {
	Path string
	Key  string
}

// NewVaultBatchTask creates a new batch Vault task
func NewVaultBatchTask(requests []VaultRequest, client *vaultkv.KV) *VaultBatchTask {
	// Generate ID from all paths
	paths := make([]string, len(requests))
	for i, req := range requests {
		paths[i] = fmt.Sprintf("%s:%s", req.Path, req.Key)
	}
	id := fmt.Sprintf("vault-batch:%s", strings.Join(paths, ","))
	
	return &VaultBatchTask{
		id:       id,
		requests: requests,
		client:   client,
	}
}

// ID returns the unique identifier for this batch task
func (t *VaultBatchTask) ID() string {
	return t.id
}

// Execute performs multiple Vault lookups
func (t *VaultBatchTask) Execute(ctx context.Context) (interface{}, error) {
	results := make(map[string]interface{})
	
	// Group requests by path to minimize Vault calls
	pathRequests := make(map[string][]VaultRequest)
	for _, req := range t.requests {
		pathRequests[req.Path] = append(pathRequests[req.Path], req)
	}
	
	// Fetch each unique path
	for path, requests := range pathRequests {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// Check cache for the entire path
		var secret map[string]interface{}
		cacheKey := fmt.Sprintf("%s:", path)
		if cached, found := VaultCache.Get(cacheKey); found {
			secret = cached.(map[string]interface{})
		} else {
			// Fetch from Vault
			secret = make(map[string]interface{})
			_, err := t.client.Get(path, &secret, nil)
			if err != nil {
				return nil, fmt.Errorf("vault lookup failed for %s: %w", path, err)
			}
			
			// Cache the entire secret
			VaultCache.Set(cacheKey, secret)
		}
		
		// Extract requested keys
		for _, req := range requests {
			key := fmt.Sprintf("%s:%s", req.Path, req.Key)
			if req.Key != "" {
				value, ok := secret[req.Key]
				if !ok {
					return nil, fmt.Errorf("vault path %s does not contain key %s", req.Path, req.Key)
				}
				results[key] = value
			} else {
				results[key] = secret
			}
		}
	}
	
	return results, nil
}

// VaultTaskExecutor provides a high-level interface for executing Vault tasks
type VaultTaskExecutor struct {
	pool   *WorkerPool
	client *vaultkv.KV
}

// NewVaultTaskExecutor creates a new Vault task executor
func NewVaultTaskExecutor(client *vaultkv.KV) *VaultTaskExecutor {
	return &VaultTaskExecutor{
		pool:   GetVaultWorkerPool(),
		client: client,
	}
}

// Lookup performs a single Vault lookup
func (e *VaultTaskExecutor) Lookup(path, key string) (interface{}, error) {
	task := NewVaultTask(path, key, e.client)
	return e.pool.SubmitAndWait(task)
}

// BatchLookup performs multiple Vault lookups efficiently
func (e *VaultTaskExecutor) BatchLookup(requests []VaultRequest) (map[string]interface{}, error) {
	if len(requests) == 0 {
		return map[string]interface{}{}, nil
	}
	
	// For small batches, use individual tasks
	if len(requests) <= 3 {
		results := make(map[string]interface{})
		for _, req := range requests {
			value, err := e.Lookup(req.Path, req.Key)
			if err != nil {
				return nil, err
			}
			key := fmt.Sprintf("%s:%s", req.Path, req.Key)
			results[key] = value
		}
		return results, nil
	}
	
	// For larger batches, use batch task
	task := NewVaultBatchTask(requests, e.client)
	result, err := e.pool.SubmitAndWait(task)
	if err != nil {
		return nil, err
	}
	
	return result.(map[string]interface{}), nil
}

// Shutdown gracefully shuts down the Vault worker pool
func (e *VaultTaskExecutor) Shutdown() {
	e.pool.Shutdown()
}