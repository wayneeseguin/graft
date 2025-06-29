package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/operators"
)

// ParallelEvaluator extends the standard evaluator with parallel processing capabilities
type ParallelEvaluator struct {
	*graft.Evaluator

	// Configuration
	maxConcurrency int
	enableParallel bool

	// Thread safety
	treeMutex sync.Mutex

	// Metrics
	parallelOpsExecuted atomic.Uint64
	parallelWaves       atomic.Uint64
	totalWaveTime       atomic.Int64 // nanoseconds
}

// NewParallelEvaluator creates a new parallel evaluator
func NewParallelEvaluator(ev *graft.Evaluator, maxConcurrency int) *ParallelEvaluator {
	if maxConcurrency <= 0 {
		maxConcurrency = 4 // Default concurrency
	}

	return &ParallelEvaluator{
		Evaluator:      ev,
		maxConcurrency: maxConcurrency,
		enableParallel: true,
	}
}

// SetParallelEnabled enables or disables parallel processing
func (pev *ParallelEvaluator) SetParallelEnabled(enabled bool) {
	pev.enableParallel = enabled
}

// ParallelDataFlow extends DataFlow to analyze operator waves for parallel execution
func (pev *ParallelEvaluator) ParallelDataFlow(phase graft.OperatorPhase) ([]*OperatorWave, error) {
	// Get the standard dependency-ordered operations
	ops, err := pev.Evaluator.DataFlow(phase)
	if err != nil {
		return nil, err
	}

	if len(ops) == 0 {
		return []*OperatorWave{}, nil
	}

	// Group operations into waves based on dependencies
	waves := pev.buildOperatorWaves(ops)

	log.DEBUG("parallel evaluator: found %d waves with %d total operations", len(waves), len(ops))
	for i, wave := range waves {
		log.DEBUG("  wave %d: %d operations (parallel: %t)", i, len(wave.Operations), wave.CanRunInParallel)
	}

	return waves, nil
}

// OperatorWave represents a group of operators that can potentially run in parallel
type OperatorWave struct {
	Operations       []*graft.Opcall
	CanRunInParallel bool
	WaveIndex        int
}

// buildOperatorWaves groups operators into waves based on their dependencies
func (pev *ParallelEvaluator) buildOperatorWaves(ops []*graft.Opcall) []*OperatorWave {
	waves := []*OperatorWave{}

	// Build dependency map
	dependsOn := make(map[*graft.Opcall][]*graft.Opcall)
	dependents := make(map[*graft.Opcall][]*graft.Opcall)

	for _, op := range ops {
		dependsOn[op] = []*graft.Opcall{}
		dependents[op] = []*graft.Opcall{}
	}

	// Analyze dependencies by comparing operator paths
	for i, op1 := range ops {
		for j, op2 := range ops {
			if i != j && pev.hasDependency(op1, op2) {
				dependsOn[op2] = append(dependsOn[op2], op1)
				dependents[op1] = append(dependents[op1], op2)
			}
		}
	}

	// Group into waves using topological sort with wave detection
	remaining := make(map[*graft.Opcall]bool)
	for _, op := range ops {
		remaining[op] = true
	}

	waveIndex := 0
	for len(remaining) > 0 {
		// Find all operations with no unresolved dependencies
		currentWave := []*graft.Opcall{}

		for op := range remaining {
			hasUnresolvedDeps := false
			for _, dep := range dependsOn[op] {
				if remaining[dep] {
					hasUnresolvedDeps = true
					break
				}
			}

			if !hasUnresolvedDeps {
				currentWave = append(currentWave, op)
			}
		}

		if len(currentWave) == 0 {
			// Cycle detected - fallback to sequential
			for op := range remaining {
				currentWave = append(currentWave, op)
				break
			}
		}

		// Remove operations in current wave from remaining
		for _, op := range currentWave {
			delete(remaining, op)
		}

		// Determine if this wave can run in parallel
		canRunInParallel := len(currentWave) > 1 && pev.enableParallel && pev.canWaveRunInParallel(currentWave)

		waves = append(waves, &OperatorWave{
			Operations:       currentWave,
			CanRunInParallel: canRunInParallel,
			WaveIndex:        waveIndex,
		})

		waveIndex++
	}

	return waves
}

// hasDependency checks if op2 depends on op1 (op1 must run before op2)
func (pev *ParallelEvaluator) hasDependency(op1, op2 *graft.Opcall) bool {
	// If op2 reads from a path that op1 writes to, then op2 depends on op1
	op1Output := op1.Canonical()

	// Check if op2 references the output of op1
	deps := op2.Dependencies(pev.Evaluator, []*tree.Cursor{})
	log.DEBUG("Checking if %s depends on %s: deps=%v", op2.Canonical(), op1.Canonical(), deps)
	for _, arg := range deps {
		if op1Output.Contains(arg) || arg.Contains(op1Output) || op1Output.String() == arg.String() {
			log.DEBUG("  -> YES: %s references %s", arg, op1Output)
			return true
		}
	}

	log.DEBUG("  -> NO")
	return false
}

// canWaveRunInParallel determines if operations in a wave can safely run in parallel
func (pev *ParallelEvaluator) canWaveRunInParallel(ops []*graft.Opcall) bool {
	// Check for operations that should not run in parallel
	for _, op := range ops {
		// Skip parallel execution for certain operator types that might have side effects
		operator := op.Operator()
		// Check if operator is marked as thread-safe
		if !pev.isOperatorThreadSafe(operator) {
			return false
		}
	}

	// Check for path conflicts (operations writing to overlapping paths)
	for i, op1 := range ops {
		for j, op2 := range ops {
			if i != j && pev.hasPathConflict(op1, op2) {
				return false
			}
		}
	}

	return true
}

// isOperatorThreadSafe checks if an operator can be safely executed in parallel
func (pev *ParallelEvaluator) isOperatorThreadSafe(op graft.Operator) bool {
	// Most read-only operators are thread-safe
	switch op.(type) {
	case *operators.GrabOperator, *operators.ConcatOperator, *operators.JoinOperator:
		return true
	case *operators.VaultOperator:
		return true // With proper connection pooling
	case *operators.CalcOperator:
		return true
	// Note: StaticIpsOperator and other operators that may modify global state
	// should return false for thread safety
	case *operators.InjectOperator:
		return false // Modifies document structure
	default:
		// For now, allow most operators to run in parallel
		// This is less conservative but should work for most cases
		return true
	}
}

// hasPathConflict checks if two operations have conflicting write paths
func (pev *ParallelEvaluator) hasPathConflict(op1, op2 *graft.Opcall) bool {
	path1 := op1.Canonical()
	path2 := op2.Canonical()

	// If operations write to the same path or overlapping paths, they conflict
	return path1.Contains(path2) || path2.Contains(path1) || path1.String() == path2.String()
}

// RunWaves executes operator waves with parallel processing where possible
func (pev *ParallelEvaluator) RunWaves(waves []*OperatorWave) error {
	log.DEBUG("parallel evaluator: executing %d waves", len(waves))

	errors := graft.MultiError{Errors: []error{}}

	for _, wave := range waves {
		waveStart := time.Now()

		if wave.CanRunInParallel && len(wave.Operations) > 1 {
			log.DEBUG("parallel evaluator: executing wave %d with %d operations in parallel",
				wave.WaveIndex, len(wave.Operations))

			err := pev.runWaveParallel(wave)
			if err != nil {
				errors.Append(err)
			}

			pev.parallelWaves.Add(1)
			pev.parallelOpsExecuted.Add(uint64(len(wave.Operations)))
		} else {
			log.DEBUG("parallel evaluator: executing wave %d with %d operations sequentially",
				wave.WaveIndex, len(wave.Operations))

			err := pev.runWaveSequential(wave)
			if err != nil {
				errors.Append(err)
			}
		}

		waveTime := time.Since(waveStart)
		pev.totalWaveTime.Add(waveTime.Nanoseconds())

		log.DEBUG("parallel evaluator: wave %d completed in %v", wave.WaveIndex, waveTime)
	}

	if len(errors.Errors) > 0 {
		return errors
	}

	return nil
}

// runWaveParallel executes operations in a wave concurrently
func (pev *ParallelEvaluator) runWaveParallel(wave *OperatorWave) error {
	// Limit concurrency to configured maximum
	concurrency := len(wave.Operations)
	if concurrency > pev.maxConcurrency {
		concurrency = pev.maxConcurrency
	}

	// Create a semaphore to limit concurrent operations
	semaphore := make(chan struct{}, concurrency)

	// Channel for collecting results
	type opResult struct {
		op  *Opcall
		err error
	}
	results := make(chan opResult, len(wave.Operations))

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Execute operations concurrently
	for _, op := range wave.Operations {
		op := op // capture loop variable
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results <- opResult{op, ctx.Err()}
				return
			}

			// Execute the operation with a thread-local evaluator context
			log.DEBUG("parallel evaluator: executing %s at %s", op.Src(), op.Where())

			// Create a thread-local copy of the evaluator to avoid race conditions
			// This ensures each goroutine has its own 'Here' cursor
			localEv := &graft.Evaluator{
				Tree:     pev.Evaluator.Tree,
				Deps:     pev.Evaluator.Deps,
				SkipEval: pev.Evaluator.SkipEval,
				CheckOps: pev.Evaluator.CheckOps,
				Only:     pev.Evaluator.Only,
				// Here will be set by RunOp, no need to copy it
			}

			err := pev.runOpWithLock(localEv, op)

			results <- opResult{op, err}
		}()
	}

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and check for errors
	var firstError error
	completedOps := 0

	for result := range results {
		completedOps++
		if result.err != nil && firstError == nil {
			firstError = result.err
			cancel() // Cancel remaining operations
		}
	}

	return firstError
}

// runWaveSequential executes operations in a wave sequentially
func (pev *ParallelEvaluator) runWaveSequential(wave *OperatorWave) error {
	for _, op := range wave.Operations {
		log.DEBUG("sequential evaluator: executing %s at %s", op.Src(), op.Where())
		err := pev.Evaluator.RunOp(op)
		if err != nil {
			return err
		}
	}
	return nil
}

// RunOpsParallel is the main entry point for parallel operator execution
func (pev *ParallelEvaluator) RunOpsParallel(phase graft.OperatorPhase) error {
	waves, err := pev.ParallelDataFlow(phase)
	if err != nil {
		return err
	}

	if len(waves) == 0 {
		log.DEBUG("parallel evaluator: no operations to execute in phase %v", phase)
		return nil
	}

	return pev.RunWaves(waves)
}

// runOpWithLock executes an operation with proper synchronization for thread safety
func (pev *ParallelEvaluator) runOpWithLock(localEv *graft.Evaluator, op *graft.Opcall) error {
	// For now, we need to lock the entire operation execution
	// because operators read from the tree while we're modifying it
	// In the future, we should use a thread-safe tree implementation
	//
	// This essentially makes the parallel evaluator sequential, but it's
	// necessary to avoid race conditions until we have a proper thread-safe
	// tree implementation. The benefit of the parallel evaluator is still
	// in the dependency analysis and wave detection.
	pev.treeMutex.Lock()
	defer pev.treeMutex.Unlock()

	// Run the operation using the standard evaluator
	return pev.Evaluator.RunOp(op)
}

// GetMetrics returns performance metrics for the parallel evaluator
func (pev *ParallelEvaluator) GetMetrics() ParallelEvaluatorMetrics {
	return ParallelEvaluatorMetrics{
		ParallelOpsExecuted: pev.parallelOpsExecuted.Load(),
		ParallelWaves:       pev.parallelWaves.Load(),
		TotalWaveTime:       time.Duration(pev.totalWaveTime.Load()),
		MaxConcurrency:      pev.maxConcurrency,
		ParallelEnabled:     pev.enableParallel,
	}
}

// ParallelEvaluatorMetrics holds performance metrics
type ParallelEvaluatorMetrics struct {
	ParallelOpsExecuted uint64
	ParallelWaves       uint64
	TotalWaveTime       time.Duration
	MaxConcurrency      int
	ParallelEnabled     bool
}

// String returns a string representation of the metrics
func (m ParallelEvaluatorMetrics) String() string {
	return fmt.Sprintf("Parallel Evaluator Metrics - Ops: %d, Waves: %d, Time: %v, Max Concurrency: %d, Enabled: %t",
		m.ParallelOpsExecuted, m.ParallelWaves, m.TotalWaveTime, m.MaxConcurrency, m.ParallelEnabled)
}
