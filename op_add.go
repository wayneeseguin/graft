package graft

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// AddOperator implements the + operator
type AddOperator struct{}

// Setup ...
func (AddOperator) Setup() error {
	return nil
}

// Phase ...
func (AddOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (AddOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (AddOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( + ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( + ... )) operation at $.%s\n", ev.Here)

	if len(args) != 2 {
		return nil, fmt.Errorf("+ operator requires exactly two arguments, got %d", len(args))
	}

	// Resolve arguments - support nested expressions
	left, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, fmt.Errorf("failed to resolve left operand: %v", err)
	}

	right, err := ResolveOperatorArgument(ev, args[1])
	if err != nil {
		return nil, fmt.Errorf("failed to resolve right operand: %v", err)
	}

	DEBUG("left operand: %v (%T)", left, left)
	DEBUG("right operand: %v (%T)", right, right)

	// Special case: string concatenation
	leftStr, leftIsStr := left.(string)
	rightStr, rightIsStr := right.(string)
	if leftIsStr && rightIsStr {
		DEBUG("performing string concatenation")
		return &Response{
			Type:  Replace,
			Value: leftStr + rightStr,
		}, nil
	}

	// Special case: if one is string and other is not, convert to string and concatenate
	if leftIsStr || rightIsStr {
		DEBUG("performing string concatenation with type coercion")
		leftAsStr, err := AsString(left)
		if err != nil {
			return nil, fmt.Errorf("cannot convert left operand to string: %v", err)
		}
		rightAsStr, err := AsString(right)
		if err != nil {
			return nil, fmt.Errorf("cannot convert right operand to string: %v", err)
		}
		return &Response{
			Type:  Replace,
			Value: leftAsStr + rightAsStr,
		}, nil
	}

	// Numeric addition
	result, err := performArithmetic(left, right, "+")
	if err != nil {
		return nil, err
	}

	// Check for overflow if result is float
	if f, ok := result.(float64); ok {
		if err := checkNumericOverflow(f); err != nil {
			return nil, err
		}
	}

	DEBUG("addition result: %v (%T)", result, result)
	return &Response{
		Type:  Replace,
		Value: result,
	}, nil
}

func init() {
	RegisterOp("+", AddOperator{})
}