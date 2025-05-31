package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"
)

// NegationOperator implements logical negation (!)
type NegationOperator struct{}

// Setup initializes the operator
func (NegationOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (NegationOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns operator dependencies
func (NegationOperator) Dependencies(_ *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the negation
func (NegationOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("! operator requires exactly 1 argument, got %d", len(args))
	}

	// Evaluate the argument
	resp, err := EvaluateExpr(args[0], ev)
	if err != nil {
		return nil, err
	}

	// Return the negation of the truthiness
	return &Response{
		Type:  Replace,
		Value: !isTruthy(resp.Value),
	}, nil
}

// Register negation operator
// NOTE: Negation operator is now registered in op_boolean.go to avoid conflicts
