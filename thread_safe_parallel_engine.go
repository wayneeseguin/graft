package graft

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ParallelExecutionEngine provides parallel operator execution using COW trees
type ParallelExecutionEngine struct {
	tree        ThreadSafeTree
	numWorkers  int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	taskQueue   chan *ExecutionTask
	resultQueue chan *ExecutionResult
	metrics     *ParallelMetrics
}

// ExecutionTask represents a task to execute
type ExecutionTask struct {
	ID       string
	Path     []string
	Operator string
	Args     []interface{}
	Priority int
}

// ExecutionResult represents the result of an execution
type ExecutionResult struct {
	TaskID   string
	Value    interface{}
	Error    error
	Duration time.Duration
}

// ParallelMetrics tracks parallel execution metrics
type ParallelMetrics struct {
	tasksQueued    int64
	tasksExecuted  int64
	tasksSucceeded int64
	tasksFailed    int64
	totalDuration  int64 // nanoseconds
}

// NewParallelExecutionEngine creates a new parallel execution engine
func NewParallelExecutionEngine(tree ThreadSafeTree, numWorkers int) *ParallelExecutionEngine {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ParallelExecutionEngine{
		tree:        tree,
		numWorkers:  numWorkers,
		ctx:         ctx,
		cancel:      cancel,
		taskQueue:   make(chan *ExecutionTask, numWorkers*10),
		resultQueue: make(chan *ExecutionResult, numWorkers*10),
		metrics:     &ParallelMetrics{},
	}
}

// Start starts the execution engine workers
func (pee *ParallelExecutionEngine) Start() {
	for i := 0; i < pee.numWorkers; i++ {
		pee.wg.Add(1)
		go pee.worker(i)
	}
}

// Stop stops the execution engine
func (pee *ParallelExecutionEngine) Stop() {
	pee.cancel()
	close(pee.taskQueue)
	pee.wg.Wait()
	close(pee.resultQueue)
}

// Submit submits a task for execution
func (pee *ParallelExecutionEngine) Submit(task *ExecutionTask) error {
	select {
	case pee.taskQueue <- task:
		atomic.AddInt64(&pee.metrics.tasksQueued, 1)
		return nil
	case <-pee.ctx.Done():
		return fmt.Errorf("execution engine stopped")
	}
}

// GetResult retrieves an execution result
func (pee *ParallelExecutionEngine) GetResult() (*ExecutionResult, bool) {
	select {
	case result, ok := <-pee.resultQueue:
		return result, ok
	case <-time.After(time.Millisecond * 10):
		return nil, false
	}
}

// worker is the main worker loop
func (pee *ParallelExecutionEngine) worker(id int) {
	defer pee.wg.Done()
	
	for {
		select {
		case task, ok := <-pee.taskQueue:
			if !ok {
				return // Queue closed
			}
			
			result := pee.executeTask(task)
			
			select {
			case pee.resultQueue <- result:
				// Result sent
			case <-pee.ctx.Done():
				return
			}
			
		case <-pee.ctx.Done():
			return
		}
	}
}

// executeTask executes a single task
func (pee *ParallelExecutionEngine) executeTask(task *ExecutionTask) *ExecutionResult {
	start := time.Now()
	result := &ExecutionResult{
		TaskID: task.ID,
	}
	
	atomic.AddInt64(&pee.metrics.tasksExecuted, 1)
	
	// Execute based on operator type
	switch task.Operator {
	case "set":
		if len(task.Args) > 0 && len(task.Path) > 0 {
			err := pee.tree.Set(task.Args[0], task.Path...)
			result.Error = err
		} else {
			result.Error = fmt.Errorf("invalid arguments for set operation")
		}
		
	case "get":
		if len(task.Path) > 0 {
			value, err := pee.tree.Find(task.Path...)
			result.Value = value
			result.Error = err
		} else {
			result.Error = fmt.Errorf("invalid path for get operation")
		}
		
	case "delete":
		if len(task.Path) > 0 {
			err := pee.tree.Delete(task.Path...)
			result.Error = err
		} else {
			result.Error = fmt.Errorf("invalid path for delete operation")
		}
		
	case "update":
		if len(task.Path) > 0 && len(task.Args) > 0 {
			updateFn, ok := task.Args[0].(func(interface{}) interface{})
			if ok {
				err := pee.tree.Update(updateFn, task.Path...)
				result.Error = err
			} else {
				result.Error = fmt.Errorf("invalid update function")
			}
		} else {
			result.Error = fmt.Errorf("invalid arguments for update operation")
		}
		
	default:
		result.Error = fmt.Errorf("unknown operator: %s", task.Operator)
	}
	
	result.Duration = time.Since(start)
	atomic.AddInt64(&pee.metrics.totalDuration, result.Duration.Nanoseconds())
	
	if result.Error == nil {
		atomic.AddInt64(&pee.metrics.tasksSucceeded, 1)
	} else {
		atomic.AddInt64(&pee.metrics.tasksFailed, 1)
	}
	
	return result
}

// GetMetrics returns current metrics
func (pee *ParallelExecutionEngine) GetMetrics() map[string]int64 {
	return map[string]int64{
		"tasks_queued":    atomic.LoadInt64(&pee.metrics.tasksQueued),
		"tasks_executed":  atomic.LoadInt64(&pee.metrics.tasksExecuted),
		"tasks_succeeded": atomic.LoadInt64(&pee.metrics.tasksSucceeded),
		"tasks_failed":    atomic.LoadInt64(&pee.metrics.tasksFailed),
		"total_duration":  atomic.LoadInt64(&pee.metrics.totalDuration),
	}
}

// ThreadSafeParallelEvaluator evaluates expressions in parallel using COW trees
type ThreadSafeParallelEvaluator struct {
	engine    *ParallelExecutionEngine
	tree      ThreadSafeTree
	taskIDGen int64
}

// NewThreadSafeParallelEvaluator creates a new thread-safe parallel evaluator
func NewThreadSafeParallelEvaluator(tree ThreadSafeTree, numWorkers int) *ThreadSafeParallelEvaluator {
	engine := NewParallelExecutionEngine(tree, numWorkers)
	engine.Start()
	
	return &ThreadSafeParallelEvaluator{
		engine: engine,
		tree:   tree,
	}
}

// EvaluateParallel evaluates multiple operations in parallel
func (pe *ThreadSafeParallelEvaluator) EvaluateParallel(operations []Operation) ([]OperationResult, error) {
	results := make([]OperationResult, len(operations))
	resultMap := make(map[string]int)
	
	// Submit all operations
	for i, op := range operations {
		taskID := fmt.Sprintf("task-%d-%d", atomic.AddInt64(&pe.taskIDGen, 1), i)
		resultMap[taskID] = i
		
		task := &ExecutionTask{
			ID:       taskID,
			Path:     op.Path,
			Operator: op.Type,
			Args:     op.Args,
			Priority: op.Priority,
		}
		
		if err := pe.engine.Submit(task); err != nil {
			return nil, fmt.Errorf("failed to submit task %s: %v", taskID, err)
		}
	}
	
	// Collect results
	collected := 0
	timeout := time.NewTimer(time.Second * 10)
	defer timeout.Stop()
	
	for collected < len(operations) {
		select {
		case <-timeout.C:
			return nil, fmt.Errorf("timeout waiting for results")
			
		default:
			result, ok := pe.engine.GetResult()
			if !ok {
				time.Sleep(time.Millisecond)
				continue
			}
			
			if idx, exists := resultMap[result.TaskID]; exists {
				results[idx] = OperationResult{
					Operation: operations[idx],
					Value:     result.Value,
					Error:     result.Error,
					Duration:  result.Duration,
				}
				collected++
			}
		}
	}
	
	return results, nil
}

// Stop stops the parallel evaluator
func (pe *ThreadSafeParallelEvaluator) Stop() {
	pe.engine.Stop()
}

// Operation represents an operation to execute
type Operation struct {
	Type     string
	Path     []string
	Args     []interface{}
	Priority int
}

// OperationResult represents the result of an operation
type OperationResult struct {
	Operation Operation
	Value     interface{}
	Error     error
	Duration  time.Duration
}

// ParallelBatchProcessor processes batches of operations in parallel
type ParallelBatchProcessor struct {
	evaluator *ThreadSafeParallelEvaluator
	batchSize int
}

// NewParallelBatchProcessor creates a new batch processor
func NewParallelBatchProcessor(tree ThreadSafeTree, numWorkers, batchSize int) *ParallelBatchProcessor {
	return &ParallelBatchProcessor{
		evaluator: NewThreadSafeParallelEvaluator(tree, numWorkers),
		batchSize: batchSize,
	}
}

// ProcessBatch processes a batch of operations
func (pbp *ParallelBatchProcessor) ProcessBatch(operations []Operation) ([]OperationResult, error) {
	if len(operations) <= pbp.batchSize {
		return pbp.evaluator.EvaluateParallel(operations)
	}
	
	// Process in batches
	allResults := make([]OperationResult, 0, len(operations))
	
	for i := 0; i < len(operations); i += pbp.batchSize {
		end := i + pbp.batchSize
		if end > len(operations) {
			end = len(operations)
		}
		
		batch := operations[i:end]
		results, err := pbp.evaluator.EvaluateParallel(batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d failed: %v", i/pbp.batchSize, err)
		}
		
		allResults = append(allResults, results...)
	}
	
	return allResults, nil
}

// Stop stops the batch processor
func (pbp *ParallelBatchProcessor) Stop() {
	pbp.evaluator.Stop()
}

// COWParallelExecutor executes operations in parallel using COW tree snapshots
type COWParallelExecutor struct {
	baseTree  ThreadSafeTree
	executors []*ThreadSafeParallelEvaluator
}

// NewCOWParallelExecutor creates a new COW parallel executor
func NewCOWParallelExecutor(tree ThreadSafeTree, numExecutors int) *COWParallelExecutor {
	executors := make([]*ThreadSafeParallelEvaluator, numExecutors)
	
	for i := 0; i < numExecutors; i++ {
		// Each executor gets its own COW snapshot
		snapshot := tree.Copy()
		executors[i] = NewThreadSafeParallelEvaluator(snapshot, 1)
	}
	
	return &COWParallelExecutor{
		baseTree:  tree,
		executors: executors,
	}
}

// ExecuteIsolated executes operations in isolation using COW snapshots
func (cpe *COWParallelExecutor) ExecuteIsolated(operations [][]Operation) ([][]OperationResult, error) {
	if len(operations) > len(cpe.executors) {
		return nil, fmt.Errorf("too many operation groups: %d > %d executors", len(operations), len(cpe.executors))
	}
	
	results := make([][]OperationResult, len(operations))
	errChan := make(chan error, len(operations))
	
	var wg sync.WaitGroup
	
	for i, ops := range operations {
		wg.Add(1)
		go func(idx int, operations []Operation) {
			defer wg.Done()
			
			res, err := cpe.executors[idx].EvaluateParallel(operations)
			if err != nil {
				errChan <- err
				return
			}
			
			results[idx] = res
		}(i, ops)
	}
	
	wg.Wait()
	close(errChan)
	
	// Check for errors
	for err := range errChan {
		return nil, err
	}
	
	return results, nil
}

// Stop stops all executors
func (cpe *COWParallelExecutor) Stop() {
	for _, executor := range cpe.executors {
		executor.Stop()
	}
}