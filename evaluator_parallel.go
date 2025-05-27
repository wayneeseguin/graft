package spruce

import (
	"os"
	"sync"
	
	. "github.com/geofffranks/spruce/log"
)

var (
	// Global parallel adapter instance (lazily initialized)
	parallelAdapter     *ParallelEvaluatorAdapter
	parallelAdapterOnce sync.Once
	parallelConfig      *ParallelEvaluatorConfig
)

// SetParallelConfig sets the global parallel execution configuration
func SetParallelConfig(config *ParallelEvaluatorConfig) {
	parallelConfig = config
}

// EnableParallelExecution enables parallel execution globally
func EnableParallelExecution(enable bool) {
	if parallelConfig == nil {
		parallelConfig = DefaultParallelConfig()
	}
	parallelConfig.Enabled = enable
}

// RunPhaseParallel runs a phase with parallel execution support
func (ev *Evaluator) RunPhaseParallel(p OperatorPhase) error {
	err := SetupOperators(p)
	if err != nil {
		return err
	}

	ops, err := ev.DataFlow(p)
	if err != nil {
		return err
	}

	// Use parallel execution if enabled
	if shouldUseParallel(ops) {
		return ev.RunOpsParallel(ops)
	}

	return ev.RunOps(ops)
}

// RunOpsParallel executes operations using parallel execution
func (ev *Evaluator) RunOpsParallel(ops []*Opcall) error {
	// Get configuration from feature flags
	features := GetFeatures()
	config := features.GetParallelConfig()

	// Create adapter for this evaluator
	adapter := NewParallelEvaluatorAdapter(ev, config)
	defer adapter.Shutdown()

	// Execute operations
	err := adapter.RunOps(ops)

	// Log metrics if debug enabled or metrics enabled
	if os.Getenv("DEBUG") != "" || features.EnableMetrics {
		metrics := adapter.GetMetrics()
		DEBUG("Parallel execution metrics: %v", metrics)
	}

	return err
}

// shouldUseParallel determines if parallel execution should be used
func shouldUseParallel(ops []*Opcall) bool {
	// Use feature flags system
	features := GetFeatures()
	if !features.IsParallelEnabled() {
		return false
	}

	// Check minimum operation threshold
	if len(ops) < features.ParallelMinOps {
		return false
	}

	return true
}

// ParallelExecutionStats returns statistics about parallel execution
func ParallelExecutionStats() map[string]interface{} {
	if parallelAdapter == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	stats := parallelAdapter.GetMetrics()
	stats["enabled"] = true
	return stats
}