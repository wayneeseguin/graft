package operators

import (
	"fmt"

	"github.com/wayneeseguin/graft/internal/utils/tree"

	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// ComparisonOperatorBase provides common functionality for comparison operators
type ComparisonOperatorBase struct {
	op       string
	registry *TypeRegistry
}

// NewComparisonOperatorBase creates a new comparison operator base
func NewComparisonOperatorBase(op string) *ComparisonOperatorBase {
	return &ComparisonOperatorBase{
		op:       op,
		registry: GetGlobalTypeRegistry(),
	}
}

// Setup initializes the operator
func (c *ComparisonOperatorBase) Setup() error {
	return nil
}

// Phase returns the evaluation phase
func (c *ComparisonOperatorBase) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies returns the operator dependencies
func (c *ComparisonOperatorBase) Dependencies(_ *graft.Evaluator, _ []*graft.Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run executes the comparison operation using type handlers
func (c *ComparisonOperatorBase) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	log.DEBUG("ComparisonOperatorBase.Run called for operator %s", c.op)
	log.DEBUG("running (( %s ... )) operation at $.%s", c.op, ev.Here)
	defer log.DEBUG("done with (( %s ... )) operation at $.%s\n", c.op, ev.Here)

	if len(args) != 2 {
		return nil, fmt.Errorf("%s operator requires exactly two arguments", c.op)
	}

	// Resolve arguments
	left, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, err
	}

	right, err := ResolveOperatorArgument(ev, args[1])
	if err != nil {
		return nil, err
	}

	log.DEBUG("  [0] = %v (type: %T)", left, left)
	log.DEBUG("  [1] = %v (type: %T)", right, right)

	// Get operand types
	leftType := GetOperandType(left)
	rightType := GetOperandType(right)

	// Find appropriate handler
	handler := c.registry.FindHandler(leftType, rightType)
	if handler == nil {
		// Fall back to the old comparison logic for backward compatibility
		log.DEBUG("No type handler found for %s and %s, falling back to legacy comparison", leftType, rightType)
		result, err := c.performLegacyComparison(left, right)
		if err != nil {
			return nil, err
		}
		return &graft.Response{
			Type:  graft.Replace,
			Value: result,
		}, nil
	}

	// Execute operation based on operator type
	var result bool
	switch c.op {
	case "==":
		result, err = handler.Equal(left, right)
	case "!=":
		result, err = handler.NotEqual(left, right)
	case "<":
		result, err = handler.Less(left, right)
	case ">":
		result, err = handler.Greater(left, right)
	case "<=":
		result, err = handler.LessOrEqual(left, right)
	case ">=":
		result, err = handler.GreaterOrEqual(left, right)
	default:
		return nil, fmt.Errorf("unknown comparison operator: %s", c.op)
	}

	if err != nil {
		return nil, err
	}

	log.DEBUG("ComparisonOperatorBase.Run returning result: %v", result)
	return &graft.Response{
		Type:  graft.Replace,
		Value: result,
	}, nil
}

// performLegacyComparison performs comparison using the old logic
func (c *ComparisonOperatorBase) performLegacyComparison(left, right interface{}) (bool, error) {
	// Use the original ComparisonOperator logic
	op := &ComparisonOperator{op: c.op}

	switch c.op {
	case "==":
		return op.equal(left, right), nil
	case "!=":
		return !op.equal(left, right), nil
	case "<", ">", "<=", ">=":
		cmp, err := op.compare(left, right)
		if err != nil {
			return false, fmt.Errorf("cannot compare %v and %v: %v", left, right, err)
		}
		switch c.op {
		case "<":
			return cmp < 0, nil
		case ">":
			return cmp > 0, nil
		case "<=":
			return cmp <= 0, nil
		case ">=":
			return cmp >= 0, nil
		}
	}

	return false, fmt.Errorf("unknown comparison operator: %s", c.op)
}

// TypeAwareEqualOperator implements the == operator with type awareness
type TypeAwareEqualOperator struct {
	*ComparisonOperatorBase
}

// NewTypeAwareEqualOperator creates a new type-aware equal operator
func NewTypeAwareEqualOperator() *TypeAwareEqualOperator {
	return &TypeAwareEqualOperator{
		ComparisonOperatorBase: NewComparisonOperatorBase("=="),
	}
}

// TypeAwareNotEqualOperator implements the != operator with type awareness
type TypeAwareNotEqualOperator struct {
	*ComparisonOperatorBase
}

// NewTypeAwareNotEqualOperator creates a new type-aware not equal operator
func NewTypeAwareNotEqualOperator() *TypeAwareNotEqualOperator {
	return &TypeAwareNotEqualOperator{
		ComparisonOperatorBase: NewComparisonOperatorBase("!="),
	}
}

// TypeAwareLessOperator implements the < operator with type awareness
type TypeAwareLessOperator struct {
	*ComparisonOperatorBase
}

// NewTypeAwareLessOperator creates a new type-aware less operator
func NewTypeAwareLessOperator() *TypeAwareLessOperator {
	return &TypeAwareLessOperator{
		ComparisonOperatorBase: NewComparisonOperatorBase("<"),
	}
}

// TypeAwareGreaterOperator implements the > operator with type awareness
type TypeAwareGreaterOperator struct {
	*ComparisonOperatorBase
}

// NewTypeAwareGreaterOperator creates a new type-aware greater operator
func NewTypeAwareGreaterOperator() *TypeAwareGreaterOperator {
	return &TypeAwareGreaterOperator{
		ComparisonOperatorBase: NewComparisonOperatorBase(">"),
	}
}

// TypeAwareLessOrEqualOperator implements the <= operator with type awareness
type TypeAwareLessOrEqualOperator struct {
	*ComparisonOperatorBase
}

// NewTypeAwareLessOrEqualOperator creates a new type-aware less or equal operator
func NewTypeAwareLessOrEqualOperator() *TypeAwareLessOrEqualOperator {
	return &TypeAwareLessOrEqualOperator{
		ComparisonOperatorBase: NewComparisonOperatorBase("<="),
	}
}

// TypeAwareGreaterOrEqualOperator implements the >= operator with type awareness
type TypeAwareGreaterOrEqualOperator struct {
	*ComparisonOperatorBase
}

// NewTypeAwareGreaterOrEqualOperator creates a new type-aware greater or equal operator
func NewTypeAwareGreaterOrEqualOperator() *TypeAwareGreaterOrEqualOperator {
	return &TypeAwareGreaterOrEqualOperator{
		ComparisonOperatorBase: NewComparisonOperatorBase(">="),
	}
}
