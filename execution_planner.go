package graft

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ExecutionPlanner plans and coordinates optimized execution
type ExecutionPlanner struct {
	analyzer      *DependencyAnalyzer
	costEstimator *CostEstimator
	batcher       *OperationBatcher
	config        *ExecutionConfig
	metrics       *ExecutionMetrics
	mu            sync.RWMutex
}

// ExecutionConfig configures execution planning
type ExecutionConfig struct {
	EnableBatching      bool
	EnableParallel      bool
	EnableEarlyTerm     bool
	MaxParallelOps      int
	PlanningTimeout     time.Duration
	ExecutionTimeout    time.Duration
	CostThreshold       float64
}

// DefaultExecutionConfig returns default execution configuration
func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		EnableBatching:   true,
		EnableParallel:   true,
		EnableEarlyTerm:  true,
		MaxParallelOps:   10,
		PlanningTimeout:  50 * time.Millisecond,
		ExecutionTimeout: 30 * time.Second,
		CostThreshold:    100.0,
	}
}

// ExecutionMetrics tracks execution metrics
type ExecutionMetrics struct {
	PlanningTime      time.Duration
	ExecutionTime     time.Duration
	TotalOperations   int
	BatchedOperations int
	ParallelStages    int
	SkippedOperations int
	CostReduction     float64
}

// NewExecutionPlanner creates a new execution planner
func NewExecutionPlanner(config *ExecutionConfig) *ExecutionPlanner {
	if config == nil {
		config = DefaultExecutionConfig()
	}
	
	return &ExecutionPlanner{
		analyzer:      NewDependencyAnalyzer(),
		costEstimator: NewCostEstimator(),
		batcher:       NewOperationBatcher(DefaultBatchConfig()),
		config:        config,
		metrics:       &ExecutionMetrics{},
	}
}

// PlanExecution creates an optimized execution plan
func (ep *ExecutionPlanner) PlanExecution(ctx context.Context, doc interface{}) (*OptimizedExecutionPlan, error) {
	planStart := time.Now()
	defer func() {
		ep.metrics.PlanningTime = time.Since(planStart)
	}()
	
	// Create planning context with timeout
	planCtx, cancel := context.WithTimeout(ctx, ep.config.PlanningTimeout)
	defer cancel()
	
	// Analyze dependencies
	graph, err := ep.analyzer.AnalyzeDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("dependency analysis failed: %v", err)
	}
	
	// Get execution stages
	stages, err := graph.GetExecutionStages()
	if err != nil {
		return nil, fmt.Errorf("stage generation failed: %v", err)
	}
	
	// Create optimized plan
	plan := &OptimizedExecutionPlan{
		Graph:         graph,
		Stages:        make([]OptimizedStage, 0, len(stages)),
		OriginalCost:  ep.calculateTotalCost(graph),
		OptimizedCost: 0,
	}
	
	// Optimize each stage
	for i, stage := range stages {
		select {
		case <-planCtx.Done():
			return plan, fmt.Errorf("planning timeout exceeded")
		default:
			optStage := ep.optimizeStage(stage, i)
			plan.Stages = append(plan.Stages, optStage)
			plan.OptimizedCost += optStage.EstimatedCost
		}
	}
	
	// Calculate metrics
	ep.updateMetrics(plan)
	
	return plan, nil
}

// optimizeStage optimizes a single execution stage
func (ep *ExecutionPlanner) optimizeStage(stage ExecutionStage, stageNum int) OptimizedStage {
	optStage := OptimizedStage{
		StageNumber:   stageNum,
		Operations:    stage.Operations,
		CanParallel:   stage.CanParallel && ep.config.EnableParallel,
		Batches:       make([]OperationBatch, 0),
		EstimatedCost: 0,
	}
	
	// Group operations for batching
	if ep.config.EnableBatching {
		batches := ep.batcher.CreateBatches(stage.Operations)
		optStage.Batches = batches
		
		// Calculate batched cost
		for _, batch := range batches {
			batchCost := ep.costEstimator.EstimateBatchCost(batch.Operations)
			optStage.EstimatedCost += batchCost
		}
	} else {
		// No batching - calculate individual costs
		for _, op := range stage.Operations {
			optStage.EstimatedCost += op.Cost
		}
	}
	
	// Apply parallel execution bonus
	if optStage.CanParallel && len(stage.Operations) > 1 {
		optStage.EstimatedCost = ep.costEstimator.EstimateParallelCost(stage.Operations)
		optStage.ParallelGroups = ep.groupForParallel(stage.Operations)
	}
	
	return optStage
}

// groupForParallel groups operations for parallel execution
func (ep *ExecutionPlanner) groupForParallel(ops []*DependencyNode) []ParallelGroup {
	// Sort by cost descending
	sorted := make([]*DependencyNode, len(ops))
	copy(sorted, ops)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Cost > sorted[j].Cost
	})
	
	// Create groups respecting max parallel ops
	groups := make([]ParallelGroup, 0)
	
	for i := 0; i < len(sorted); i += ep.config.MaxParallelOps {
		end := i + ep.config.MaxParallelOps
		if end > len(sorted) {
			end = len(sorted)
		}
		
		group := ParallelGroup{
			Operations: sorted[i:end],
			MaxCost:    sorted[i].Cost, // First is highest cost
		}
		groups = append(groups, group)
	}
	
	return groups
}

// calculateTotalCost calculates total cost without optimization
func (ep *ExecutionPlanner) calculateTotalCost(graph *DependencyGraph) float64 {
	totalCost := float64(0)
	nodeIDs, _ := graph.TopologicalSort()
	allNodes := graph.GetNodes()
	
	for _, nodeID := range nodeIDs {
		if node, ok := allNodes[nodeID]; ok {
			totalCost += node.Cost
		}
	}
	
	return totalCost
}

// updateMetrics updates execution metrics
func (ep *ExecutionPlanner) updateMetrics(plan *OptimizedExecutionPlan) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	
	ep.metrics.TotalOperations = plan.Graph.Size()
	ep.metrics.ParallelStages = 0
	ep.metrics.BatchedOperations = 0
	
	for _, stage := range plan.Stages {
		if stage.CanParallel {
			ep.metrics.ParallelStages++
		}
		
		for _, batch := range stage.Batches {
			if len(batch.Operations) > 1 {
				ep.metrics.BatchedOperations += len(batch.Operations)
			}
		}
	}
	
	if plan.OriginalCost > 0 {
		ep.metrics.CostReduction = (plan.OriginalCost - plan.OptimizedCost) / plan.OriginalCost
	}
}

// ExecutePlan executes an optimized plan
func (ep *ExecutionPlanner) ExecutePlan(ctx context.Context, plan *OptimizedExecutionPlan, executor Executor) error {
	execStart := time.Now()
	defer func() {
		ep.metrics.ExecutionTime = time.Since(execStart)
	}()
	
	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, ep.config.ExecutionTimeout)
	defer cancel()
	
	// Execute stages in order
	for _, stage := range plan.Stages {
		if err := ep.executeStage(execCtx, stage, executor); err != nil {
			return fmt.Errorf("stage %d execution failed: %v", stage.StageNumber, err)
		}
	}
	
	return nil
}

// executeStage executes a single optimized stage
func (ep *ExecutionPlanner) executeStage(ctx context.Context, stage OptimizedStage, executor Executor) error {
	if stage.CanParallel && len(stage.ParallelGroups) > 0 {
		// Execute parallel groups
		for _, group := range stage.ParallelGroups {
			if err := ep.executeParallelGroup(ctx, group, executor); err != nil {
				return err
			}
		}
	} else if len(stage.Batches) > 0 {
		// Execute batches
		for _, batch := range stage.Batches {
			if err := executor.ExecuteBatch(ctx, batch); err != nil {
				return err
			}
		}
	} else {
		// Execute operations sequentially
		for _, op := range stage.Operations {
			if err := executor.ExecuteOperation(ctx, op); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// executeParallelGroup executes operations in parallel
func (ep *ExecutionPlanner) executeParallelGroup(ctx context.Context, group ParallelGroup, executor Executor) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(group.Operations))
	
	for _, op := range group.Operations {
		wg.Add(1)
		go func(operation *DependencyNode) {
			defer wg.Done()
			if err := executor.ExecuteOperation(ctx, operation); err != nil {
				errChan <- err
			}
		}(op)
	}
	
	// Wait for completion
	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	case <-doneChan:
		return nil
	}
}

// GetMetrics returns execution metrics
func (ep *ExecutionPlanner) GetMetrics() ExecutionMetrics {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return *ep.metrics
}

// OptimizedExecutionPlan represents an optimized execution plan
type OptimizedExecutionPlan struct {
	Graph         *DependencyGraph
	Stages        []OptimizedStage
	OriginalCost  float64
	OptimizedCost float64
	CriticalPath  []*DependencyNode
}

// OptimizedStage represents an optimized execution stage
type OptimizedStage struct {
	StageNumber    int
	Operations     []*DependencyNode
	CanParallel    bool
	ParallelGroups []ParallelGroup
	Batches        []OperationBatch
	EstimatedCost  float64
}

// ParallelGroup represents operations that can run in parallel
type ParallelGroup struct {
	Operations []*DependencyNode
	MaxCost    float64
}

// Executor interface for operation execution
type Executor interface {
	ExecuteOperation(ctx context.Context, op *DependencyNode) error
	ExecuteBatch(ctx context.Context, batch OperationBatch) error
}

// Visualize returns a string representation of the plan
func (plan *OptimizedExecutionPlan) Visualize() string {
	var result string
	
	result += fmt.Sprintf("=== Optimized Execution Plan ===\n")
	result += fmt.Sprintf("Original Cost: %.2f\n", plan.OriginalCost)
	result += fmt.Sprintf("Optimized Cost: %.2f (%.1f%% reduction)\n", 
		plan.OptimizedCost, 
		(plan.OriginalCost-plan.OptimizedCost)/plan.OriginalCost*100)
	result += fmt.Sprintf("Stages: %d\n\n", len(plan.Stages))
	
	for _, stage := range plan.Stages {
		result += fmt.Sprintf("Stage %d (Cost: %.2f):\n", stage.StageNumber, stage.EstimatedCost)
		
		if stage.CanParallel {
			result += "  Parallel Execution:\n"
			for i, group := range stage.ParallelGroups {
				result += fmt.Sprintf("    Group %d (%d ops, max cost: %.2f)\n", 
					i, len(group.Operations), group.MaxCost)
			}
		}
		
		if len(stage.Batches) > 0 {
			result += "  Batches:\n"
			for i, batch := range stage.Batches {
				result += fmt.Sprintf("    Batch %d: %s (%d ops)\n", 
					i, batch.Type, len(batch.Operations))
			}
		}
		
		result += "\n"
	}
	
	return result
}