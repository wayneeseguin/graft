package spruce

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/geofffranks/spruce/log"
)

// NegateOperatorEnhanced is an enhanced version that supports nested expressions
type NegateOperatorEnhanced struct{}

// Setup ...
func (NegateOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (NegateOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (NegateOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (NegateOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( negate ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( negate ... )) operation at $.%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("negate operator requires exactly one argument")
	}

	// Use ResolveOperatorArgument to handle nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("failed to resolve expression to a concrete value")
		DEBUG("error was: %s", err)
		return nil, err
	}

	// Handle different types
	switch v := val.(type) {
	case bool:
		DEBUG("negating boolean value: %v", v)
		return &Response{
			Type:  Replace,
			Value: !v,
		}, nil

	case nil:
		DEBUG("negating nil value, returning true")
		return &Response{
			Type:  Replace,
			Value: true,
		}, nil

	case string:
		// Empty string is falsy
		DEBUG("negating string value: '%s'", v)
		return &Response{
			Type:  Replace,
			Value: v == "",
		}, nil

	case int, int64:
		// 0 is falsy
		DEBUG("negating integer value: %v", v)
		return &Response{
			Type:  Replace,
			Value: v == 0 || v == int64(0),
		}, nil

	case float64:
		// 0.0 is falsy
		DEBUG("negating float value: %v", v)
		return &Response{
			Type:  Replace,
			Value: v == 0.0,
		}, nil

	case []interface{}:
		// Empty array is falsy
		DEBUG("negating array value with %d elements", len(v))
		return &Response{
			Type:  Replace,
			Value: len(v) == 0,
		}, nil

	case map[interface{}]interface{}, map[string]interface{}:
		// Empty map is falsy
		size := 0
		switch m := v.(type) {
		case map[interface{}]interface{}:
			size = len(m)
		case map[string]interface{}:
			size = len(m)
		}
		DEBUG("negating map value with %d keys", size)
		return &Response{
			Type:  Replace,
			Value: size == 0,
		}, nil

	default:
		// For any other type, non-nil is truthy
		DEBUG("negating value of type %T", v)
		return &Response{
			Type:  Replace,
			Value: false,
		}, nil
	}
}

// EnableEnhancedNegate enables the enhanced negate operator
func EnableEnhancedNegate() {
	RegisterOp("negate", NegateOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedNegate handle it
}