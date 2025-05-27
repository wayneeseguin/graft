package graft

import (
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// ParamOperatorEnhanced is an enhanced version that supports nested expressions
type ParamOperatorEnhanced struct{}

// Setup ...
func (ParamOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (ParamOperatorEnhanced) Phase() OperatorPhase {
	return ParamPhase
}

// Dependencies ...
func (ParamOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (ParamOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( param ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( param ... )) operation at $.%s\n", ev.Here)

	if len(args) != 1 {
		return nil, ansi.Errorf("@R{param operator only expects} @c{one argument}")
	}

	// For param operator, we need to be careful about evaluation
	// since it runs in ParamPhase, nested operators might not be available
	// We'll try to resolve but fall back to string representation if needed
	
	var paramName string
	
	// First, try to resolve as a nested expression
	val, err := ResolveOperatorArgument(ev, args[0])
	if err == nil && val != nil {
		paramName = fmt.Sprintf("%v", val)
		DEBUG("resolved param name to: %s", paramName)
	} else {
		// Fall back to direct evaluation
		DEBUG("failed to resolve with ResolveOperatorArgument, trying direct evaluation")
		v, err := args[0].Evaluate(ev.Tree)
		if err != nil {
			DEBUG("direct evaluation also failed")
			return nil, err
		}
		paramName = fmt.Sprintf("%v", v)
		DEBUG("param name from direct evaluation: %s", paramName)
	}

	// Always return an error as param is meant to fail if not replaced
	return nil, ansi.Errorf("@R{unresolved parameter} @c{%s}", paramName)
}

// EnableEnhancedParam enables the enhanced param operator
func EnableEnhancedParam() {
	RegisterOp("param", ParamOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedParam handle it
}