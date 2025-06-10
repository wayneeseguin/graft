package operators

import (
	"fmt"
	"reflect"

	"github.com/wayneeseguin/graft/internal/utils/tree"

	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// BooleanOperatorBase provides common functionality for boolean operators
type BooleanOperatorBase struct {
	op           string
	shortCircuit bool // Whether to use short-circuit evaluation
}

// NewBooleanOperatorBase creates a new boolean operator base
func NewBooleanOperatorBase(op string, shortCircuit bool) *BooleanOperatorBase {
	return &BooleanOperatorBase{
		op:           op,
		shortCircuit: shortCircuit,
	}
}

// Setup initializes the operator
func (b *BooleanOperatorBase) Setup() error {
	return nil
}

// Phase returns the evaluation phase
func (b *BooleanOperatorBase) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies returns the operator dependencies
func (b *BooleanOperatorBase) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0, len(auto))
	deps = append(deps, auto...)

	// Collect dependencies from all arguments
	// This ensures we catch dependencies in nested expressions
	for _, arg := range args {
		if arg != nil {
			deps = append(deps, arg.Dependencies(ev, locs)...)
		}
	}

	return deps
}

// Run executes the boolean operation
func (b *BooleanOperatorBase) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	log.DEBUG("BooleanOperatorBase.Run called for operator %s", b.op)
	log.DEBUG("running (( %s ... )) operation at $.%s", b.op, ev.Here)
	defer log.DEBUG("done with (( %s ... )) operation at $.%s\n", b.op, ev.Here)

	switch b.op {
	case "&&":
		return b.runAnd(ev, args)
	case "||":
		return b.runOr(ev, args)
	case "!":
		return b.runNot(ev, args)
	default:
		return nil, fmt.Errorf("unknown boolean operator: %s", b.op)
	}
}

// runAnd implements logical AND with short-circuit evaluation
func (b *BooleanOperatorBase) runAnd(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("&& operator requires exactly 2 arguments, got %d", len(args))
	}

	// Evaluate left operand
	left, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, err
	}

	leftTruthy := IsTruthy(left)
	log.DEBUG("  left = %v (truthy: %v)", left, leftTruthy)

	// Short-circuit: if left is false, return false without evaluating right
	if !leftTruthy {
		return &graft.Response{
			Type:  graft.Replace,
			Value: false,
		}, nil
	}

	// Left is truthy, evaluate right
	right, err := ResolveOperatorArgument(ev, args[1])
	if err != nil {
		return nil, err
	}

	rightTruthy := IsTruthy(right)
	log.DEBUG("  right = %v (truthy: %v)", right, rightTruthy)

	return &graft.Response{
		Type:  graft.Replace,
		Value: rightTruthy,
	}, nil
}

// runOr implements logical OR with short-circuit evaluation
func (b *BooleanOperatorBase) runOr(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("|| operator requires exactly 2 arguments, got %d", len(args))
	}

	// Evaluate left operand
	left, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, err
	}

	leftTruthy := IsTruthy(left)
	log.DEBUG("  left = %v (truthy: %v)", left, leftTruthy)

	// Short-circuit: if left is true, return true without evaluating right
	if leftTruthy {
		return &graft.Response{
			Type:  graft.Replace,
			Value: true,
		}, nil
	}

	// Left is falsy, evaluate right
	right, err := ResolveOperatorArgument(ev, args[1])
	if err != nil {
		return nil, err
	}

	rightTruthy := IsTruthy(right)
	log.DEBUG("  right = %v (truthy: %v)", right, rightTruthy)

	return &graft.Response{
		Type:  graft.Replace,
		Value: rightTruthy,
	}, nil
}

// runNot implements logical NOT
func (b *BooleanOperatorBase) runNot(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("! operator requires exactly 1 argument, got %d", len(args))
	}

	// Evaluate the operand
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, err
	}

	truthy := IsTruthy(val)
	log.DEBUG("  operand = %v (truthy: %v)", val, truthy)

	return &graft.Response{
		Type:  graft.Replace,
		Value: !truthy,
	}, nil
}

// IsTruthy determines if a value is truthy according to Graft's rules
// false, nil, 0, "", [], {} are falsy
// Everything else is truthy
func IsTruthy(v interface{}) bool {
	if v == nil {
		return false
	}

	// Check for boolean
	if b, ok := v.(bool); ok {
		return b
	}

	// Check for numeric zero
	switch num := v.(type) {
	case int:
		return num != 0
	case int8:
		return num != 0
	case int16:
		return num != 0
	case int32:
		return num != 0
	case int64:
		return num != 0
	case uint:
		return num != 0
	case uint8:
		return num != 0
	case uint16:
		return num != 0
	case uint32:
		return num != 0
	case uint64:
		return num != 0
	case float32:
		return num != 0
	case float64:
		return num != 0
	}

	// Check for empty string
	if s, ok := v.(string); ok {
		return s != ""
	}

	// Check for empty slice/array/map
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() > 0
	}

	// Everything else is truthy
	return true
}

// TypeAwareAndOperator implements the && operator with type awareness
type TypeAwareAndOperator struct {
	*BooleanOperatorBase
}

// NewTypeAwareAndOperator creates a new type-aware AND operator
func NewTypeAwareAndOperator() *TypeAwareAndOperator {
	return &TypeAwareAndOperator{
		BooleanOperatorBase: NewBooleanOperatorBase("&&", true),
	}
}

// TypeAwareOrOperator implements the || operator with type awareness
type TypeAwareOrOperator struct {
	*BooleanOperatorBase
}

// NewTypeAwareOrOperator creates a new type-aware OR operator
func NewTypeAwareOrOperator() *TypeAwareOrOperator {
	return &TypeAwareOrOperator{
		BooleanOperatorBase: NewBooleanOperatorBase("||", true),
	}
}

// TypeAwareNotOperator implements the ! operator with type awareness
type TypeAwareNotOperator struct {
	*BooleanOperatorBase
}

// NewTypeAwareNotOperator creates a new type-aware NOT operator
func NewTypeAwareNotOperator() *TypeAwareNotOperator {
	return &TypeAwareNotOperator{
		BooleanOperatorBase: NewBooleanOperatorBase("!", false),
	}
}
