package graft

import (
	"github.com/wayneeseguin/graft/log"
)

// RunPhaseParallel runs a phase with parallel execution support
// For Phase 1, this just delegates to the regular RunPhase
func (ev *Evaluator) RunPhaseParallel(p OperatorPhase) error {
	log.DEBUG("Parallel execution not yet implemented in Phase 1")
	return ev.RunPhase(p)
}

// RunOpsParallel executes operations using parallel execution
// For Phase 1, this just delegates to the regular RunOps
func (ev *Evaluator) RunOpsParallel(ops []*Opcall) error {
	log.DEBUG("Parallel execution not yet implemented in Phase 1")
	return ev.RunOps(ops)
}

// ParallelExecutionStats returns statistics about parallel execution
func ParallelExecutionStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled": false,
		"message": "Parallel execution not yet implemented in Phase 1",
	}
}