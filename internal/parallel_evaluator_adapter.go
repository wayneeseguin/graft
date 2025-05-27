package internal

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// ParallelEvaluatorConfig controls parallel execution behavior
type ParallelEvaluatorConfig struct {
	Enabled           bool
	MaxWorkers        int
	MinOpsForParallel int
	Strategy          string // "aggressive", "conservative", "adaptive"
	SafeOperators     map[string]bool
}

// DefaultParallelConfig returns default configuration
func DefaultParallelConfig() *ParallelEvaluatorConfig {
	maxWorkers := runtime.NumCPU()
	if env := os.Getenv("GRAFT_PARALLEL_WORKERS"); env != "" {
		if n, err := strconv.Atoi(env); err == nil && n > 0 {
			maxWorkers = n
		}
	}

	return &ParallelEvaluatorConfig{
		Enabled:           os.Getenv("GRAFT_PARALLEL") == "true",
		MaxWorkers:        maxWorkers,
		MinOpsForParallel: 10, // Don't parallelize small workloads
		Strategy:          "conservative",
		SafeOperators: map[string]bool{
			"grab":              true,
			"concat":            true,
			"base64":            true,
			"cartesian-product": true,
			"keys":              true,
			"empty":             true,
			"join":              true,
			"sort":              true,
			"stringify":         true,
			"base64-decode":     true,
			"load":              true, // File reads can be parallel
			"vault":             true, // External calls benefit from parallelism
			"awsparam":          true,
			"awssecret":         true,
			"file":              true,
		},
	}
}

// ParallelEvaluatorAdapter bridges the existing Evaluator with parallel execution
type ParallelEvaluatorAdapter struct {
	evaluator *Evaluator
	config    *ParallelEvaluatorConfig
	tree      ThreadSafeTree
	engine    *ParallelExecutionEngine

	// Metrics
	totalOps    atomic.Int64
	parallelOps atomic.Int64
	speedup     atomic.Value // float64
}

// NewParallelEvaluatorAdapter creates a new adapter
func NewParallelEvaluatorAdapter(ev *Evaluator, config *ParallelEvaluatorConfig) *ParallelEvaluatorAdapter {
	if config == nil {
		config = DefaultParallelConfig()
	}

	// Create thread-safe tree wrapper
	tree := NewSafeTree(ev.Tree)

	// Create execution engine
	engine := NewParallelExecutionEngine(tree, config.MaxWorkers)
	engine.Start()

	adapter := &ParallelEvaluatorAdapter{
		evaluator: ev,
		config:    config,
		tree:      tree,
		engine:    engine,
	}

	adapter.speedup.Store(1.0)

	return adapter
}

// RunOps executes operations, using parallel execution when beneficial
func (pa *ParallelEvaluatorAdapter) RunOps(ops []*Opcall) error {
	metrics := GetParallelMetrics()

	if !pa.config.Enabled || len(ops) < pa.config.MinOpsForParallel {
		log.DEBUG("Parallel execution disabled or too few ops (%d < %d), using sequential execution",
			len(ops), pa.config.MinOpsForParallel)

		// Execute sequentially with metrics
		start := time.Now()
		err := pa.evaluator.RunOps(ops)
		duration := time.Since(start)

		for range ops {
			metrics.RecordOperation(false, duration/time.Duration(len(ops)), err)
		}

		return err
	}

	// Analyze operations for parallelizability
	groups := pa.analyzeOperations(ops)

	if len(groups.parallel) == 0 {
		log.DEBUG("No parallelizable operations found, using sequential execution")

		start := time.Now()
		err := pa.evaluator.RunOps(ops)
		duration := time.Since(start)

		for range ops {
			metrics.RecordOperation(false, duration/time.Duration(len(ops)), err)
		}

		return err
	}

	log.DEBUG("Running %d operations: %d in parallel, %d sequential",
		len(ops), len(groups.parallel), len(groups.sequential))

	start := time.Now()

	// Execute operations in order
	for _, group := range groups.executionOrder {
		groupStart := time.Now()
		var groupErr error

		if group.canParallel && len(group.ops) > 1 {
			groupErr = pa.runParallelGroup(group.ops)

			// Record parallel operations
			groupDuration := time.Since(groupStart)
			for range group.ops {
				metrics.RecordOperation(true, groupDuration/time.Duration(len(group.ops)), groupErr)
			}
		} else {
			for _, op := range group.ops {
				opStart := time.Now()
				err := pa.evaluator.RunOp(op)
				opDuration := time.Since(opStart)

				metrics.RecordOperation(false, opDuration, err)

				if err != nil {
					groupErr = err
				}
			}
		}

		if groupErr != nil {
			return groupErr
		}
	}

	elapsed := time.Since(start)

	// Calculate speedup
	if groups.estimatedSequentialTime > 0 {
		speedup := float64(groups.estimatedSequentialTime) / float64(elapsed)
		pa.speedup.Store(speedup)
		log.DEBUG("Parallel execution completed in %v (%.2fx speedup)", elapsed, speedup)
	}

	return nil
}

// operationGroups represents analyzed operation groups
type operationGroups struct {
	parallel                []*Opcall
	sequential              []*Opcall
	executionOrder          []operationGroup
	estimatedSequentialTime time.Duration
}

// operationGroup represents a group of operations that can be executed together
type operationGroup struct {
	ops         []*Opcall
	canParallel bool
}

// analyzeOperations groups operations for optimal execution
func (pa *ParallelEvaluatorAdapter) analyzeOperations(ops []*Opcall) operationGroups {
	groups := operationGroups{
		parallel:   make([]*Opcall, 0),
		sequential: make([]*Opcall, 0),
	}

	// Build dependency map from evaluator
	deps := make(map[string][]string)
	opsByPath := make(map[string]*Opcall)

	for _, op := range ops {
		path := op.Where().String()
		opsByPath[path] = op

		// Extract dependencies from evaluator's Deps map
		if cursors, ok := pa.evaluator.Deps[path]; ok {
			depPaths := make([]string, len(cursors))
			for i, cursor := range cursors {
				depPaths[i] = cursor.String()
			}
			deps[path] = depPaths
		}
	}

	// Group operations by execution waves
	processed := make(map[string]bool)

	for len(processed) < len(ops) {
		group := operationGroup{
			ops:         make([]*Opcall, 0),
			canParallel: true,
		}

		// Find operations that can execute in this wave
		for _, op := range ops {
			path := op.Where().String()
			if processed[path] {
				continue
			}

			// Check if all dependencies are satisfied
			canExecute := true
			for _, dep := range deps[path] {
				if !processed[dep] {
					canExecute = false
					break
				}
			}

			if canExecute {
				group.ops = append(group.ops, op)

				// Check if operator is safe for parallel execution
				if !pa.isOperatorSafe(op) {
					group.canParallel = false
				}
			}
		}

		// Mark operations as processed
		for _, op := range group.ops {
			processed[op.Where().String()] = true
		}

		// Add to appropriate category
		if group.canParallel && len(group.ops) > 1 {
			groups.parallel = append(groups.parallel, group.ops...)
		} else {
			groups.sequential = append(groups.sequential, group.ops...)
		}

		groups.executionOrder = append(groups.executionOrder, group)

		// Estimate sequential time (rough approximation)
		for range group.ops {
			// Simple estimation based on operation
			groups.estimatedSequentialTime += 10 * time.Millisecond
		}
	}

	return groups
}

// isOperatorSafe checks if an operator is safe for parallel execution
func (pa *ParallelEvaluatorAdapter) isOperatorSafe(op *Opcall) bool {
	// Get operator type from registry
	opType := pa.getOperatorType(op)
	if opType == "" {
		return false // Unknown operator, be conservative
	}

	// Check configuration whitelist
	if safe, ok := pa.config.SafeOperators[opType]; ok {
		return safe
	}

	// Conservative strategy: only whitelist is safe
	if pa.config.Strategy == "conservative" {
		return false
	}

	// Aggressive strategy: assume safe unless proven otherwise
	if pa.config.Strategy == "aggressive" {
		// Still exclude known unsafe operations
		switch opType {
		case "static_ips", "inject", "merge", "prune":
			return false
		}
		return true
	}

	// Adaptive strategy: analyze operator behavior
	// TODO: Implement adaptive analysis based on metrics

	return false
}

// getOperatorType returns the type name of an operator
func (pa *ParallelEvaluatorAdapter) getOperatorType(op *Opcall) string {
	if op.Operator() == nil {
		return ""
	}

	// Check the operator registry to find the type
	for name, registeredOp := range graft.OpRegistry {
		if registeredOp == op.Operator() {
			return name
		}
	}

	// Fallback: try to extract from source string
	if op.Src() != "" {
		// Extract operator name from (( op args )) format
		re := regexp.MustCompile(`\(\(\s*(\w+)`)
		matches := re.FindStringSubmatch(op.Src())
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// runParallelGroup executes a group of operations in parallel
func (pa *ParallelEvaluatorAdapter) runParallelGroup(ops []*Opcall) error {
	pa.totalOps.Add(int64(len(ops)))
	pa.parallelOps.Add(int64(len(ops)))

	metrics := GetParallelMetrics()

	// Track concurrency
	metrics.RecordConcurrency(int32(len(ops)))
	defer metrics.RecordConcurrency(-int32(len(ops)))

	// Convert Opcalls to Operations for parallel engine
	operations := make([]Operation, len(ops))
	for i, op := range ops {
		operations[i] = pa.opcallToOperation(op)
	}

	// Create evaluator for this batch
	evaluator := NewThreadSafeParallelEvaluator(pa.tree, pa.config.MaxWorkers)
	defer evaluator.Stop()

	// Execute in parallel
	results, err := evaluator.EvaluateParallel(operations)
	if err != nil {
		return fmt.Errorf("parallel execution failed: %v", err)
	}

	// Process results
	var errors []error
	for i, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("operation at %s failed: %v",
				ops[i].Where().String(), result.Error))
		} else {
			// Apply the result back to the original tree
			if err := pa.applyResult(ops[i], result.Value); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return graft.MultiError{Errors: errors}
	}

	return nil
}

// opcallToOperation converts an Opcall to an Operation for parallel execution
func (pa *ParallelEvaluatorAdapter) opcallToOperation(op *Opcall) Operation {
	return Operation{
		Type:     "opcall",
		Path:     strings.Split(op.Where().String(), "."),
		Args:     []interface{}{op},
		Priority: 1, // TODO: Implement priority based on operator type
	}
}

// applyResult applies the result of a parallel operation back to the tree
func (pa *ParallelEvaluatorAdapter) applyResult(op *Opcall, result interface{}) error {
	// For now, we rely on the operator having updated the tree directly
	// In future, we could make this more explicit
	return nil
}

// GetMetrics returns execution metrics
func (pa *ParallelEvaluatorAdapter) GetMetrics() map[string]interface{} {
	total := pa.totalOps.Load()
	parallel := pa.parallelOps.Load()

	return map[string]interface{}{
		"total_operations":      total,
		"parallel_operations":   parallel,
		"sequential_operations": total - parallel,
		"parallel_percentage":   float64(parallel) / float64(total) * 100,
		"speedup_factor":        pa.speedup.Load(),
		"engine_metrics":        pa.engine.GetMetrics(),
	}
}

// Shutdown gracefully shuts down the parallel evaluator
func (pa *ParallelEvaluatorAdapter) Shutdown() {
	pa.engine.Stop()
}
