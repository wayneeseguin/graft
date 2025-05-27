package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"
)

// TernaryOperator implements the ternary conditional operator (? :)
type TernaryOperator struct{}

// Setup initializes the operator
func (TernaryOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (TernaryOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns operator dependencies
func (TernaryOperator) Dependencies(_ *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	// Only need dependencies from the condition and the branch that will be taken
	// But we don't know which branch yet, so include all
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the ternary operator
func (TernaryOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("?: operator requires exactly 3 arguments (condition, true_value, false_value), got %d", len(args))
	}

	// Evaluate the condition
	condResp, err := EvaluateExpr(args[0], ev)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate ternary condition: %v", err)
	}

	// Short-circuit evaluation: only evaluate the branch we need
	if isTruthy(condResp.Value) {
		// Condition is truthy, evaluate and return true branch
		return EvaluateExpr(args[1], ev)
	} else {
		// Condition is falsy, evaluate and return false branch
		return EvaluateExpr(args[2], ev)
	}
}

// Register ternary operator
func init() {
	RegisterOp("?:", TernaryOperator{})
}
