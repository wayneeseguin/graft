package spruce

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/geofffranks/spruce/log"
)

// ModuloOperator implements the % operator
type ModuloOperator struct{}

// Setup ...
func (ModuloOperator) Setup() error {
	return nil
}

// Phase ...
func (ModuloOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (ModuloOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (ModuloOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( %% ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( %% ... )) operation at $.%s\n", ev.Here)

	if len(args) != 2 {
		return nil, fmt.Errorf("%% operator requires exactly two arguments, got %d", len(args))
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

	// Modulo only works with integers
	leftInt, rightInt, err := promoteToInt(left, right)
	if err != nil {
		return nil, fmt.Errorf("modulo operator requires integer operands: %v", err)
	}

	if rightInt == 0 {
		return nil, fmt.Errorf("modulo by zero")
	}

	result := leftInt % rightInt
	DEBUG("modulo result: %v", result)

	return &Response{
		Type:  Replace,
		Value: result,
	}, nil
}

func init() {
	RegisterOp("%", ModuloOperator{})
}