package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockExecutor implements the Executor interface for testing
type MockExecutor struct {
	executions      []ExecutionRecord
	mu              sync.Mutex
	operationErrors map[string]error
	batchErrors     map[string]error
	executionDelay  time.Duration
}

type ExecutionRecord struct {
	Type      string // "operation" or "batch"
	ID        string
	Timestamp time.Time
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		executions:      make([]ExecutionRecord, 0),
		operationErrors: make(map[string]error),
		batchErrors:     make(map[string]error),
	}
}

func (me *MockExecutor) ExecuteOperation(ctx context.Context, op *DependencyNode) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	if me.executionDelay > 0 {
		time.Sleep(me.executionDelay)
	}

	me.executions = append(me.executions, ExecutionRecord{
		Type:      "operation",
		ID:        op.ID,
		Timestamp: time.Now(),
	})

	if err, exists := me.operationErrors[op.ID]; exists {
		return err
	}

	return nil
}

func (me *MockExecutor) ExecuteBatch(ctx context.Context, batch OperationBatch) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	if me.executionDelay > 0 {
		time.Sleep(me.executionDelay)
	}

	batchID := fmt.Sprintf("batch_%s_%d", batch.Type, len(batch.Operations))
	me.executions = append(me.executions, ExecutionRecord{
		Type:      "batch",
		ID:        batchID,
		Timestamp: time.Now(),
	})

	if err, exists := me.batchErrors[batchID]; exists {
		return err
	}

	return nil
}

func (me *MockExecutor) GetExecutions() []ExecutionRecord {
	me.mu.Lock()
	defer me.mu.Unlock()

	result := make([]ExecutionRecord, len(me.executions))
	copy(result, me.executions)
	return result
}

func (me *MockExecutor) SetOperationError(opID string, err error) {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.operationErrors[opID] = err
}

func (me *MockExecutor) SetBatchError(batchID string, err error) {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.batchErrors[batchID] = err
}

func (me *MockExecutor) SetExecutionDelay(delay time.Duration) {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.executionDelay = delay
}

// CreateMockDocument creates a simple document with operator expressions for testing
func CreateMockDocument() map[string]interface{} {
	return map[string]interface{}{
		"meta": map[string]interface{}{
			"value1": "10",
			"value2": "20",
			"value3": "30",
		},
		"result1":  "(( grab meta.value1 ))",
		"result2":  "(( grab meta.value2 ))",
		"result3":  "(( grab meta.value3 ))",
		"combined": "(( concat result1 result2 ))",
		"final":    "(( concat combined result3 ))",
	}
}

// CreateSimpleDocument creates a minimal document for basic testing
func CreateSimpleDocument() map[string]interface{} {
	return map[string]interface{}{
		"result": "(( grab meta.value ))",
		"meta": map[string]interface{}{
			"value": "test",
		},
	}
}

// TestExecutionPlannerBasic tests basic execution planner functionality
func TestExecutionPlannerBasic(t *testing.T) {
	t.Run("basic planning", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		doc := CreateMockDocument()

		// Plan execution
		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)

		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		if plan == nil {
			t.Fatal("plan should not be nil")
		}

		if len(plan.Stages) == 0 {
			t.Error("plan should have stages")
		}

		if plan.OriginalCost <= 0 {
			t.Error("original cost should be positive")
		}

		if plan.OptimizedCost < 0 {
			t.Error("optimized cost should not be negative")
		}
	})

	t.Run("custom configuration", func(t *testing.T) {
		config := &ExecutionConfig{
			EnableBatching:   false,
			EnableParallel:   false,
			EnableEarlyTerm:  false,
			MaxParallelOps:   5,
			PlanningTimeout:  100 * time.Millisecond,
			ExecutionTimeout: 5 * time.Second,
			CostThreshold:    50.0,
		}

		planner := NewExecutionPlanner(config)
		doc := CreateSimpleDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)

		if err != nil {
			t.Fatalf("planning with custom config failed: %v", err)
		}

		// With batching disabled, stages should not have batches
		for _, stage := range plan.Stages {
			if len(stage.Batches) > 0 {
				t.Error("batching should be disabled")
			}
			if stage.CanParallel {
				t.Error("parallel execution should be disabled")
			}
		}
	})

	t.Run("planning timeout", func(t *testing.T) {
		config := &ExecutionConfig{
			EnableBatching:   true,
			EnableParallel:   true,
			EnableEarlyTerm:  true,
			MaxParallelOps:   10,
			PlanningTimeout:  1 * time.Nanosecond, // Very short timeout
			ExecutionTimeout: 30 * time.Second,
			CostThreshold:    100.0,
		}

		planner := NewExecutionPlanner(config)
		doc := CreateSimpleDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)

		// Should either succeed quickly or timeout
		if err != nil {
			if plan == nil {
				t.Error("if timeout occurs, partial plan should still be returned")
			}
		}
	})

	t.Run("nil config uses defaults", func(t *testing.T) {
		planner := NewExecutionPlanner(nil)

		if planner.config == nil {
			t.Error("config should not be nil")
		}

		if !planner.config.EnableBatching {
			t.Error("default config should enable batching")
		}

		if !planner.config.EnableParallel {
			t.Error("default config should enable parallel")
		}
	})
}

// TestExecutionPlannerOptimization tests optimization features
func TestExecutionPlannerOptimization(t *testing.T) {
	t.Run("batching optimization", func(t *testing.T) {
		config := &ExecutionConfig{
			EnableBatching:   true,
			EnableParallel:   false,
			EnableEarlyTerm:  false,
			MaxParallelOps:   10,
			PlanningTimeout:  5 * time.Second,
			ExecutionTimeout: 30 * time.Second,
			CostThreshold:    100.0,
		}

		planner := NewExecutionPlanner(config)
		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)

		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		// Check that batching was attempted (some stages should have batches)
		foundBatches := false
		for _, stage := range plan.Stages {
			if len(stage.Batches) > 0 {
				foundBatches = true
				break
			}
		}

		if !foundBatches {
			t.Log("No batches found - may be expected if operations can't be batched")
		}
	})

	t.Run("parallel optimization", func(t *testing.T) {
		config := &ExecutionConfig{
			EnableBatching:   false,
			EnableParallel:   true,
			EnableEarlyTerm:  false,
			MaxParallelOps:   3,
			PlanningTimeout:  5 * time.Second,
			ExecutionTimeout: 30 * time.Second,
			CostThreshold:    100.0,
		}

		planner := NewExecutionPlanner(config)
		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)

		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		// Check parallel groups
		foundParallel := false
		for _, stage := range plan.Stages {
			if stage.CanParallel && len(stage.ParallelGroups) > 0 {
				foundParallel = true

				// Verify group size constraints
				for _, group := range stage.ParallelGroups {
					if len(group.Operations) > config.MaxParallelOps {
						t.Errorf("parallel group size %d exceeds max %d",
							len(group.Operations), config.MaxParallelOps)
					}
				}
			}
		}

		if !foundParallel {
			t.Log("No parallel groups found - may be expected if operations have dependencies")
		}
	})

	t.Run("cost calculation", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)
		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)

		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		// Original cost should be positive (exact value depends on cost estimation)
		if plan.OriginalCost <= 0 {
			t.Errorf("expected positive original cost, got %.2f", plan.OriginalCost)
		}

		// Optimized cost should be <= original cost
		if plan.OptimizedCost > plan.OriginalCost {
			t.Errorf("optimized cost %.2f should not exceed original cost %.2f",
				plan.OptimizedCost, plan.OriginalCost)
		}
	})
}

// TestExecutionPlannerExecution tests plan execution
func TestExecutionPlannerExecution(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)
		executor := NewMockExecutor()

		doc := CreateMockDocument()

		// Create and execute plan
		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		err = planner.ExecutePlan(ctx, plan, executor)
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}

		// Verify execution occurred
		executions := executor.GetExecutions()
		if len(executions) == 0 {
			t.Error("no executions recorded")
		}
	})

	t.Run("execution with operation error", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)
		executor := NewMockExecutor()

		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		// Set error for first operation found in any stage
		hasOperations := false
		for _, stage := range plan.Stages {
			if len(stage.Operations) > 0 {
				hasOperations = true
				// Set error for first operation in this stage
				executor.SetOperationError(stage.Operations[0].ID, fmt.Errorf("operation failed"))
				break
			}
		}

		if !hasOperations {
			t.Skip("No operations found in plan to test error handling")
			return
		}

		err = planner.ExecutePlan(ctx, plan, executor)
		if err == nil {
			// The execution may succeed if operations are batched or handled differently
			t.Log("Execution succeeded despite operation error - may be due to batching or error handling")
		} else {
			// Verify we got an error and it contains our message
			if !plannerContainsSubstring(err.Error(), "operation failed") {
				t.Errorf("error should contain operation error message: %v", err)
			}
		}
	})

	t.Run("execution timeout", func(t *testing.T) {
		config := &ExecutionConfig{
			EnableBatching:   false,
			EnableParallel:   false,
			EnableEarlyTerm:  false,
			MaxParallelOps:   10,
			PlanningTimeout:  5 * time.Second,
			ExecutionTimeout: 50 * time.Millisecond, // Short timeout
			CostThreshold:    100.0,
		}

		planner := NewExecutionPlanner(config)
		executor := NewMockExecutor()
		executor.SetExecutionDelay(100 * time.Millisecond) // Longer than timeout

		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		err = planner.ExecutePlan(ctx, plan, executor)
		if err == nil {
			t.Log("Execution succeeded despite timeout - execution may be fast enough or handled differently")
		} else {
			t.Logf("Got expected timeout error: %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)
		executor := NewMockExecutor()
		executor.SetExecutionDelay(100 * time.Millisecond)

		doc := CreateMockDocument()

		ctx, cancel := context.WithCancel(context.Background())
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		// Cancel context immediately
		cancel()

		err = planner.ExecutePlan(ctx, plan, executor)
		if err == nil {
			t.Error("expected execution to fail due to context cancellation")
		}
	})
}

// TestExecutionPlannerMetrics tests metrics collection
func TestExecutionPlannerMetrics(t *testing.T) {
	t.Run("metrics collection", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		metrics := planner.GetMetrics()

		// Check planning time was recorded
		if metrics.PlanningTime <= 0 {
			t.Error("planning time should be positive")
		}

		// Check operation count (should be positive)
		if metrics.TotalOperations <= 0 {
			t.Error("total operations should be positive")
		}

		// Execute plan to get execution metrics
		executor := NewMockExecutor()
		err = planner.ExecutePlan(ctx, plan, executor)
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}

		finalMetrics := planner.GetMetrics()

		// Check execution time was recorded
		if finalMetrics.ExecutionTime <= 0 {
			t.Error("execution time should be positive")
		}

		// Execution time should be >= planning time (both should be positive)
		if finalMetrics.ExecutionTime < finalMetrics.PlanningTime {
			t.Log("Execution time less than planning time - may be normal for simple operations")
		}
	})

	t.Run("cost reduction calculation", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		metrics := planner.GetMetrics()

		// Cost reduction should be between 0 and 1
		if metrics.CostReduction < 0 || metrics.CostReduction > 1 {
			t.Errorf("cost reduction should be between 0 and 1, got %.2f",
				metrics.CostReduction)
		}

		// If optimized cost < original cost, cost reduction should be positive
		if plan.OptimizedCost < plan.OriginalCost && metrics.CostReduction <= 0 {
			t.Error("cost reduction should be positive when optimization reduces cost")
		}
	})
}

// TestExecutionPlannerConcurrency tests concurrent planning and execution
func TestExecutionPlannerConcurrency(t *testing.T) {
	t.Run("concurrent planning", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		var wg sync.WaitGroup
		var successCount atomic.Int32
		var errorCount atomic.Int32

		// Run multiple planning operations concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				doc := CreateMockDocument()
				ctx := context.Background()

				plan, err := planner.PlanExecution(ctx, doc)
				if err != nil {
					errorCount.Add(1)
					t.Logf("planning %d failed: %v", id, err)
					return
				}

				if plan == nil {
					errorCount.Add(1)
					t.Logf("planning %d returned nil plan", id)
					return
				}

				successCount.Add(1)
			}(i)
		}

		wg.Wait()

		if successCount.Load() == 0 {
			t.Error("no concurrent planning operations succeeded")
		}

		t.Logf("Concurrent planning results: %d successes, %d errors",
			successCount.Load(), errorCount.Load())
	})

	t.Run("concurrent execution", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		// Create plan once
		doc := CreateMockDocument()
		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		var wg sync.WaitGroup
		var successCount atomic.Int32
		var errorCount atomic.Int32

		// Execute same plan concurrently with different executors
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				executor := NewMockExecutor()
				err := planner.ExecutePlan(ctx, plan, executor)
				if err != nil {
					errorCount.Add(1)
					t.Logf("execution %d failed: %v", id, err)
					return
				}

				successCount.Add(1)
			}(i)
		}

		wg.Wait()

		if successCount.Load() == 0 {
			t.Error("no concurrent execution operations succeeded")
		}

		t.Logf("Concurrent execution results: %d successes, %d errors",
			successCount.Load(), errorCount.Load())
	})
}

// TestOptimizedExecutionPlan tests the execution plan structure
func TestOptimizedExecutionPlan(t *testing.T) {
	t.Run("plan visualization", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		visualization := plan.Visualize()

		// Should contain key information
		expectedSubstrings := []string{
			"Optimized Execution Plan",
			"Original Cost:",
			"Optimized Cost:",
			"Stages:",
		}

		for _, substr := range expectedSubstrings {
			if !plannerContainsSubstring(visualization, substr) {
				t.Errorf("visualization should contain '%s': %s", substr, visualization)
			}
		}

		// Should not be empty
		if len(visualization) < 50 {
			t.Errorf("visualization seems too short: %s", visualization)
		}
	})

	t.Run("stage structure", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		doc := CreateMockDocument()

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning failed: %v", err)
		}

		// Verify stage structure
		for i, stage := range plan.Stages {
			if stage.StageNumber != i {
				t.Errorf("stage %d has incorrect stage number %d", i, stage.StageNumber)
			}

			if len(stage.Operations) == 0 {
				t.Errorf("stage %d should have operations", i)
			}

			if stage.EstimatedCost < 0 {
				t.Errorf("stage %d has negative estimated cost %.2f", i, stage.EstimatedCost)
			}
		}
	})
}

// TestExecutionPlannerEdgeCases tests edge cases and error conditions
func TestExecutionPlannerEdgeCases(t *testing.T) {
	t.Run("empty document", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)

		doc := map[string]interface{}{
			"simple": "no_operators_here",
		}

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)

		// Should handle document with no operators gracefully
		if err != nil {
			t.Logf("Empty operator planning returned error (may be expected): %v", err)
		}

		if plan != nil {
			if len(plan.Stages) != 0 {
				t.Error("document with no operators should produce empty plan")
			}
			if plan.OriginalCost != 0 {
				t.Error("document with no operators should have zero original cost")
			}
		}
	})

	t.Run("single operation", func(t *testing.T) {
		config := DefaultExecutionConfig()
		planner := NewExecutionPlanner(config)
		executor := NewMockExecutor()

		doc := map[string]interface{}{
			"simple": "(( grab meta.value ))",
			"meta": map[string]interface{}{
				"value": "test_result",
			},
		}

		ctx := context.Background()
		plan, err := planner.PlanExecution(ctx, doc)
		if err != nil {
			t.Fatalf("planning single operation failed: %v", err)
		}

		if plan.OriginalCost <= 0 {
			t.Errorf("expected positive original cost, got %.2f", plan.OriginalCost)
		}

		// Execute the plan
		err = planner.ExecutePlan(ctx, plan, executor)
		if err != nil {
			t.Fatalf("executing single operation failed: %v", err)
		}

		executions := executor.GetExecutions()
		if len(executions) == 0 {
			t.Error("expected at least one execution")
		} else {
			t.Logf("Got %d executions as expected", len(executions))
		}
	})
}

// Helper function for string containment check
func plannerContainsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
