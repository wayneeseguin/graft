package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"
)

// LogicalOrOperator implements true logical OR (not fallback)
// This is separate from FallbackOperator which implements fallback/coalesce behavior
type LogicalOrOperator struct{}

// Setup initializes the operator
func (LogicalOrOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (LogicalOrOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns operator dependencies
func (LogicalOrOperator) Dependencies(_ *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the logical OR with short-circuit evaluation
func (LogicalOrOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("|| operator requires exactly 2 arguments, got %d", len(args))
	}

	// Short-circuit evaluation: evaluate left first
	leftResp, err := EvaluateExpr(args[0], ev)
	if err != nil {
		return nil, err
	}

	if isTruthy(leftResp.Value) {
		// Left is truthy, return true without evaluating right
		return &Response{
			Type:  Replace,
			Value: true,
		}, nil
	}

	// Left is falsy, evaluate right
	rightResp, err := EvaluateExpr(args[1], ev)
	if err != nil {
		return nil, err
	}

	return &Response{
		Type:  Replace,
		Value: isTruthy(rightResp.Value),
	}, nil
}

// Note: We don't register this operator in init() yet to avoid conflicts
// with the existing FallbackOperator. This will be done when we refactor
// the operator registration.