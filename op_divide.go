package spruce

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/geofffranks/spruce/log"
)

// DivideOperator implements the / operator
type DivideOperator struct{}

// Setup ...
func (DivideOperator) Setup() error {
	return nil
}

// Phase ...
func (DivideOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (DivideOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (DivideOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( / ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( / ... )) operation at $.%s\n", ev.Here)

	if len(args) != 2 {
		return nil, fmt.Errorf("/ operator requires exactly two arguments, got %d", len(args))
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

	// Division only works with numbers and always returns float64
	result, err := performArithmetic(left, right, "/")
	if err != nil {
		return nil, err
	}

	// Check for overflow if result is float
	if f, ok := result.(float64); ok {
		if err := checkNumericOverflow(f); err != nil {
			return nil, err
		}
	}

	DEBUG("division result: %v (%T)", result, result)
	return &Response{
		Type:  Replace,
		Value: result,
	}, nil
}

func init() {
	RegisterOp("/", DivideOperator{})
}