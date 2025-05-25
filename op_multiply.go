package spruce

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/geofffranks/spruce/log"
)

// MultiplyOperator implements the * operator
type MultiplyOperator struct{}

// Setup ...
func (MultiplyOperator) Setup() error {
	return nil
}

// Phase ...
func (MultiplyOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (MultiplyOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (MultiplyOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( * ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( * ... )) operation at $.%s\n", ev.Here)

	if len(args) != 2 {
		return nil, fmt.Errorf("* operator requires exactly two arguments, got %d", len(args))
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

	// Special case: string repetition (string * int)
	if str, ok := left.(string); ok {
		if count, err := toNumeric(right); err == nil {
			if intCount, ok := count.(int64); ok && intCount >= 0 {
				DEBUG("performing string repetition")
				if intCount == 0 {
					return &Response{Type: Replace, Value: ""}, nil
				}
				if intCount > 10000 {
					return nil, fmt.Errorf("string repetition count too large: %d", intCount)
				}
				return &Response{
					Type:  Replace,
					Value: strings.Repeat(str, int(intCount)),
				}, nil
			}
		}
		return nil, fmt.Errorf("cannot multiply string by non-integer or negative value")
	}

	// Special case: int * string
	if str, ok := right.(string); ok {
		if count, err := toNumeric(left); err == nil {
			if intCount, ok := count.(int64); ok && intCount >= 0 {
				DEBUG("performing string repetition (reversed)")
				if intCount == 0 {
					return &Response{Type: Replace, Value: ""}, nil
				}
				if intCount > 10000 {
					return nil, fmt.Errorf("string repetition count too large: %d", intCount)
				}
				return &Response{
					Type:  Replace,
					Value: strings.Repeat(str, int(intCount)),
				}, nil
			}
		}
		return nil, fmt.Errorf("cannot multiply non-integer or negative value by string")
	}

	// Numeric multiplication
	result, err := performArithmetic(left, right, "*")
	if err != nil {
		return nil, err
	}

	// Check for overflow if result is float
	if f, ok := result.(float64); ok {
		if err := checkNumericOverflow(f); err != nil {
			return nil, err
		}
	}

	DEBUG("multiplication result: %v (%T)", result, result)
	return &Response{
		Type:  Replace,
		Value: result,
	}, nil
}

func init() {
	RegisterOp("*", MultiplyOperator{})
}