package operators

import (
	"fmt"

	"github.com/wayneeseguin/graft/internal/utils/tree"

	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// TypeAwareTernaryOperator implements the ternary conditional operator (? :) with type awareness
type TypeAwareTernaryOperator struct{}

// Setup initializes the operator
func (TypeAwareTernaryOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (TypeAwareTernaryOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies returns operator dependencies
func (TypeAwareTernaryOperator) Dependencies(_ *graft.Evaluator, _ []*graft.Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// We can't know which branch will be taken until evaluation time,
	// so we need to include dependencies from all branches
	return auto
}

// Run executes the ternary operator with type-aware truthiness evaluation
func (TypeAwareTernaryOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	log.DEBUG("TypeAwareTernaryOperator.Run called")
	log.DEBUG("running (( ?: ... )) operation at $.%s", ev.Here)
	defer log.DEBUG("done with (( ?: ... )) operation at $.%s\n", ev.Here)

	if len(args) != 3 {
		return nil, fmt.Errorf("?: operator requires exactly 3 arguments (condition, true_value, false_value), got %d", len(args))
	}

	// Evaluate the condition
	condition, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate ternary condition: %v", err)
	}

	log.DEBUG("  condition = %v (type: %T)", condition, condition)

	// Use type-aware truthiness evaluation
	conditionTruthy := IsTruthy(condition)
	log.DEBUG("  condition is truthy: %v", conditionTruthy)

	// Short-circuit evaluation: only evaluate the branch we need
	if conditionTruthy {
		// Condition is truthy, evaluate and return true branch
		log.DEBUG("  evaluating true branch")
		result, err := ResolveOperatorArgument(ev, args[1])
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate true branch: %v", err)
		}
		return &graft.Response{
			Type:  graft.Replace,
			Value: result,
		}, nil
	} else {
		// Condition is falsy, evaluate and return false branch
		log.DEBUG("  evaluating false branch")
		result, err := ResolveOperatorArgument(ev, args[2])
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate false branch: %v", err)
		}
		return &graft.Response{
			Type:  graft.Replace,
			Value: result,
		}, nil
	}
}