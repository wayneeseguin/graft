package spruce

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OperationBatcher batches similar operations for efficient execution
type OperationBatcher struct {
	config    *BatchConfig
	collector *BatchCollector
	executor  *BatchExecutor
	metrics   *BatchMetrics
	mu        sync.RWMutex
}

// BatchConfig configures batching behavior
type BatchConfig struct {
	MaxBatchSize     int
	MaxWaitTime      time.Duration
	MinBatchSize     int
	BatchByType      bool
	BatchByTarget    bool
	EnableMetrics    bool
}

// DefaultBatchConfig returns default batch configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		MaxBatchSize:  50,
		MaxWaitTime:   100 * time.Millisecond,
		MinBatchSize:  2,
		BatchByType:   true,
		BatchByTarget: true,
		EnableMetrics: true,
	}
}

// BatchMetrics tracks batching performance
type BatchMetrics struct {
	TotalBatches      int64
	TotalOperations   int64
	AverageBatchSize  float64
	BatchHitRate      float64
	TimeSaved         time.Duration
	mu                sync.RWMutex
}

// OperationBatch represents a batch of operations
type OperationBatch struct {
	ID         string
	Type       string
	Target     string
	Operations []*DependencyNode
	CreatedAt  time.Time
	ExecuteAt  time.Time
	Result     map[string]interface{}
	Error      error
}

// BatchCollector collects operations into batches
type BatchCollector struct {
	pendingBatches map[string]*PendingBatch
	config         *BatchConfig
	mu             sync.Mutex
	flushTimer     *time.Timer
}

// PendingBatch represents a batch being collected
type PendingBatch struct {
	batch     *OperationBatch
	readyChan chan struct{}
	mu        sync.Mutex
}

// NewOperationBatcher creates a new operation batcher
func NewOperationBatcher(config *BatchConfig) *OperationBatcher {
	if config == nil {
		config = DefaultBatchConfig()
	}
	
	return &OperationBatcher{
		config:    config,
		collector: NewBatchCollector(config),
		executor:  NewBatchExecutor(),
		metrics:   &BatchMetrics{},
	}
}

// NewBatchCollector creates a new batch collector
func NewBatchCollector(config *BatchConfig) *BatchCollector {
	return &BatchCollector{
		pendingBatches: make(map[string]*PendingBatch),
		config:         config,
	}
}

// CreateBatches creates batches from a list of operations
func (ob *OperationBatcher) CreateBatches(operations []*DependencyNode) []OperationBatch {
	// Group operations by batch key
	groups := ob.groupOperations(operations)
	
	// Create batches from groups
	batches := make([]OperationBatch, 0, len(groups))
	
	for key, ops := range groups {
		// Skip if below minimum batch size
		if len(ops) < ob.config.MinBatchSize && len(ops) > 1 {
			continue
		}
		
		// Split large groups into multiple batches
		for i := 0; i < len(ops); i += ob.config.MaxBatchSize {
			end := i + ob.config.MaxBatchSize
			if end > len(ops) {
				end = len(ops)
			}
			
			batch := OperationBatch{
				ID:         fmt.Sprintf("batch_%s_%d", key, i/ob.config.MaxBatchSize),
				Type:       ops[i].OperatorType,
				Target:     ob.extractTarget(ops[i]),
				Operations: ops[i:end],
				CreatedAt:  time.Now(),
			}
			
			batches = append(batches, batch)
		}
	}
	
	// Update metrics
	if ob.config.EnableMetrics {
		ob.updateBatchMetrics(batches, operations)
	}
	
	return batches
}

// groupOperations groups operations by batch key
func (ob *OperationBatcher) groupOperations(operations []*DependencyNode) map[string][]*DependencyNode {
	groups := make(map[string][]*DependencyNode)
	
	for _, op := range operations {
		if !ob.canBatch(op) {
			// Non-batchable operations go in their own group
			key := fmt.Sprintf("single_%s", op.ID)
			groups[key] = []*DependencyNode{op}
			continue
		}
		
		key := ob.getBatchKey(op)
		groups[key] = append(groups[key], op)
	}
	
	return groups
}

// canBatch determines if an operation can be batched
func (ob *OperationBatcher) canBatch(op *DependencyNode) bool {
	// Check if operator type supports batching
	batchableOps := map[string]bool{
		"vault":      true,
		"file":       true,
		"awsparam":   true,
		"awssecret":  true,
		"grab":       false, // Grabs are usually unique
		"static_ips": false, // Complex operations
		"defer":      false, // Order-sensitive
	}
	
	supported, exists := batchableOps[op.OperatorType]
	return exists && supported
}

// getBatchKey generates a key for grouping operations
func (ob *OperationBatcher) getBatchKey(op *DependencyNode) string {
	key := ""
	
	if ob.config.BatchByType {
		key += op.OperatorType
	}
	
	if ob.config.BatchByTarget {
		target := ob.extractTarget(op)
		if target != "" {
			key += "_" + target
		}
	}
	
	return key
}

// extractTarget extracts the target system/path from an operation
func (ob *OperationBatcher) extractTarget(op *DependencyNode) string {
	switch op.OperatorType {
	case "vault":
		// Extract vault path prefix
		if _, ok := op.Expression.(string); ok {
			// Simple extraction - could be enhanced
			return "vault"
		}
	case "file":
		// Extract file directory
		if _, ok := op.Expression.(string); ok {
			return "file"
		}
	case "awsparam", "awssecret":
		return "aws"
	}
	
	return ""
}

// CollectOperation adds an operation to a pending batch
func (ob *OperationBatcher) CollectOperation(ctx context.Context, op *DependencyNode) <-chan *OperationBatch {
	return ob.collector.Collect(ctx, op)
}

// Collect adds an operation to a pending batch
func (bc *BatchCollector) Collect(ctx context.Context, op *DependencyNode) <-chan *OperationBatch {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	key := bc.getBatchKey(op)
	
	// Get or create pending batch
	pending, exists := bc.pendingBatches[key]
	if !exists {
		pending = &PendingBatch{
			batch: &OperationBatch{
				ID:         fmt.Sprintf("batch_%s_%d", key, time.Now().UnixNano()),
				Type:       op.OperatorType,
				CreatedAt:  time.Now(),
				Operations: make([]*DependencyNode, 0, bc.config.MaxBatchSize),
			},
			readyChan: make(chan struct{}),
		}
		bc.pendingBatches[key] = pending
		
		// Start flush timer
		bc.scheduleFlush(key, pending)
	}
	
	// Add operation to batch
	pending.mu.Lock()
	pending.batch.Operations = append(pending.batch.Operations, op)
	shouldFlush := len(pending.batch.Operations) >= bc.config.MaxBatchSize
	pending.mu.Unlock()
	
	// Flush if batch is full
	if shouldFlush {
		bc.flushBatch(key)
	}
	
	// Return channel that will receive the batch when ready
	resultChan := make(chan *OperationBatch, 1)
	go func() {
		select {
		case <-ctx.Done():
			resultChan <- &OperationBatch{Error: ctx.Err()}
		case <-pending.readyChan:
			resultChan <- pending.batch
		}
		close(resultChan)
	}()
	
	return resultChan
}

// getBatchKey generates batch key for collector
func (bc *BatchCollector) getBatchKey(op *DependencyNode) string {
	// Simplified - delegates to batcher logic
	return op.OperatorType
}

// scheduleFlush schedules a batch flush after max wait time
func (bc *BatchCollector) scheduleFlush(key string, pending *PendingBatch) {
	time.AfterFunc(bc.config.MaxWaitTime, func() {
		bc.mu.Lock()
		defer bc.mu.Unlock()
		
		// Check if batch still exists
		if current, exists := bc.pendingBatches[key]; exists && current == pending {
			bc.flushBatch(key)
		}
	})
}

// flushBatch flushes a pending batch
func (bc *BatchCollector) flushBatch(key string) {
	pending, exists := bc.pendingBatches[key]
	if !exists {
		return
	}
	
	delete(bc.pendingBatches, key)
	
	pending.mu.Lock()
	pending.batch.ExecuteAt = time.Now()
	pending.mu.Unlock()
	
	// Signal batch is ready
	close(pending.readyChan)
}

// updateBatchMetrics updates batching metrics
func (ob *OperationBatcher) updateBatchMetrics(batches []OperationBatch, originalOps []*DependencyNode) {
	ob.metrics.mu.Lock()
	defer ob.metrics.mu.Unlock()
	
	ob.metrics.TotalBatches += int64(len(batches))
	ob.metrics.TotalOperations += int64(len(originalOps))
	
	// Calculate average batch size
	totalInBatches := 0
	for _, batch := range batches {
		if len(batch.Operations) > 1 {
			totalInBatches += len(batch.Operations)
		}
	}
	
	if len(batches) > 0 {
		ob.metrics.AverageBatchSize = float64(totalInBatches) / float64(len(batches))
	}
	
	// Calculate batch hit rate
	if len(originalOps) > 0 {
		ob.metrics.BatchHitRate = float64(totalInBatches) / float64(len(originalOps))
	}
}

// GetMetrics returns batching metrics
func (ob *OperationBatcher) GetMetrics() BatchMetrics {
	ob.metrics.mu.RLock()
	defer ob.metrics.mu.RUnlock()
	
	// Return a copy without the mutex
	return BatchMetrics{
		TotalBatches:     ob.metrics.TotalBatches,
		TotalOperations:  ob.metrics.TotalOperations,
		AverageBatchSize: ob.metrics.AverageBatchSize,
		BatchHitRate:     ob.metrics.BatchHitRate,
		TimeSaved:        ob.metrics.TimeSaved,
	}
}

// BatchExecutor handles batch execution
type BatchExecutor struct {
	strategies map[string]BatchStrategy
	mu         sync.RWMutex
}

// BatchStrategy defines how to execute a batch of operations
type BatchStrategy interface {
	ExecuteBatch(ctx context.Context, batch *OperationBatch) error
	CanHandle(opType string) bool
}

// NewBatchExecutor creates a new batch executor
func NewBatchExecutor() *BatchExecutor {
	executor := &BatchExecutor{
		strategies: make(map[string]BatchStrategy),
	}
	
	// Register default strategies
	executor.RegisterStrategy("vault", &VaultBatchStrategy{})
	executor.RegisterStrategy("file", &FileBatchStrategy{})
	executor.RegisterStrategy("aws", &AWSBatchStrategy{})
	
	return executor
}

// RegisterStrategy registers a batch execution strategy
func (be *BatchExecutor) RegisterStrategy(name string, strategy BatchStrategy) {
	be.mu.Lock()
	defer be.mu.Unlock()
	be.strategies[name] = strategy
}

// ExecuteBatch executes a batch of operations
func (be *BatchExecutor) ExecuteBatch(ctx context.Context, batch *OperationBatch) error {
	be.mu.RLock()
	strategy, exists := be.strategies[batch.Type]
	be.mu.RUnlock()
	
	if !exists {
		// Fall back to sequential execution
		return be.executeSequential(ctx, batch)
	}
	
	return strategy.ExecuteBatch(ctx, batch)
}

// executeSequential executes operations sequentially
func (be *BatchExecutor) executeSequential(ctx context.Context, batch *OperationBatch) error {
	results := make(map[string]interface{})
	
	for _, op := range batch.Operations {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Execute individual operation
			// This would integrate with actual operator execution
			results[op.ID] = fmt.Sprintf("result_%s", op.ID)
		}
	}
	
	batch.Result = results
	return nil
}

// Example batch strategies

// VaultBatchStrategy batches vault operations
type VaultBatchStrategy struct{}

func (s *VaultBatchStrategy) CanHandle(opType string) bool {
	return opType == "vault"
}

func (s *VaultBatchStrategy) ExecuteBatch(ctx context.Context, batch *OperationBatch) error {
	// In real implementation, this would:
	// 1. Extract all vault paths from operations
	// 2. Make a single vault request for multiple paths
	// 3. Distribute results back to operations
	
	results := make(map[string]interface{})
	for _, op := range batch.Operations {
		results[op.ID] = fmt.Sprintf("vault_result_%s", op.ID)
	}
	
	batch.Result = results
	return nil
}

// FileBatchStrategy batches file operations
type FileBatchStrategy struct{}

func (s *FileBatchStrategy) CanHandle(opType string) bool {
	return opType == "file"
}

func (s *FileBatchStrategy) ExecuteBatch(ctx context.Context, batch *OperationBatch) error {
	// In real implementation, this would:
	// 1. Group files by directory
	// 2. Read multiple files in one go
	// 3. Cache file handles if appropriate
	
	results := make(map[string]interface{})
	for _, op := range batch.Operations {
		results[op.ID] = fmt.Sprintf("file_result_%s", op.ID)
	}
	
	batch.Result = results
	return nil
}

// AWSBatchStrategy batches AWS operations
type AWSBatchStrategy struct{}

func (s *AWSBatchStrategy) CanHandle(opType string) bool {
	return opType == "awsparam" || opType == "awssecret"
}

func (s *AWSBatchStrategy) ExecuteBatch(ctx context.Context, batch *OperationBatch) error {
	// In real implementation, this would:
	// 1. Use AWS batch APIs (GetParameters, BatchGetSecretValue)
	// 2. Handle partial failures
	// 3. Implement retry logic
	
	results := make(map[string]interface{})
	for _, op := range batch.Operations {
		results[op.ID] = fmt.Sprintf("aws_result_%s", op.ID)
	}
	
	batch.Result = results
	return nil
}