package spruce

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"
)

// ParamOperator ...
type ParamOperator struct{}

// Setup ...
func (ParamOperator) Setup() error {
	return nil
}

// Phase ...
func (ParamOperator) Phase() OperatorPhase {
	return ParamPhase
}

// Dependencies ...
func (ParamOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, _ []*tree.Cursor) []*tree.Cursor {
	return nil
}

// Run ...
func (ParamOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	// Validate that we have exactly one argument
	if len(args) != 1 {
		return nil, fmt.Errorf("param operator requires exactly one argument")
	}
	
	// Evaluate the argument to get the parameter name
	v, err := args[0].Evaluate(ev.Tree)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate param argument: %s", err)
	}
	
	// Convert the value to string for the parameter name
	paramName := fmt.Sprintf("%v", v)
	
	// The param operator always returns an error - it's meant to fail if not replaced
	// Return the parameter name as the error message to maintain backward compatibility
	return nil, fmt.Errorf("%s", paramName)
}

func init() {
	RegisterOp("param", ParamOperator{})
}
