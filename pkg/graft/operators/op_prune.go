package operators

import (
	"fmt"

	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

var keysToPrune []string

func addToPruneListIfNecessary(engine graft.Engine, paths ...string) {
	prunePaths := engine.GetOperatorState().GetKeysToPrune()
	for _, path := range paths {
		if !isIncluded(prunePaths, path) {
			DEBUG("adding '%s' to the list of paths to prune", path)
			engine.GetOperatorState().AddKeyToPrune(path)
		}
	}
}

func isIncluded(list []string, name string) bool {
	for _, entry := range list {
		if entry == name {
			return true
		}
	}

	return false
}

// PruneOperator ...
type PruneOperator struct{}

// Setup ...
func (PruneOperator) Setup() error {
	return nil
}

// Phase ...
func (PruneOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (PruneOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (PruneOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( prune ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( prune ... )) operation at $.%s\n", ev.Here)

	engine := graft.GetEngine(ev)
	addToPruneListIfNecessary(engine, fmt.Sprintf("%s", ev.Here))

	// Get the current value at this location
	val, err := ev.Here.Resolve(ev.Tree)
	if err != nil {
		DEBUG("  failed to resolve current value: %s", err)
		// If we can't resolve the current value, just return a nil replacement
		return &Response{
			Type:  Replace,
			Value: nil,
		}, nil
	}
	
	DEBUG("  current value at %s: %T(%v)", ev.Here, val, val)
	
	// Return the current value unchanged - just mark it for pruning at the end
	// This allows other operators to still reference the value
	return &Response{
		Type:  Replace,
		Value: val,
	}, nil
}

func init() {
	RegisterOp("prune", PruneOperator{})
}
