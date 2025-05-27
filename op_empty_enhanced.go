package graft

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// EmptyOperatorEnhanced is an enhanced version that supports nested expressions
type EmptyOperatorEnhanced struct{}

// Setup ...
func (EmptyOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (EmptyOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (EmptyOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (EmptyOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( empty ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( empty ... )) operation at $.%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("empty operator requires exactly one argument")
	}

	// Use ResolveOperatorArgument to handle nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("failed to resolve expression to a concrete value")
		DEBUG("error was: %s", err)
		return nil, err
	}

	// If the argument resolved to a string, it might be a type name
	if typeStr, ok := val.(string); ok {
		DEBUG("argument resolved to string: %s", typeStr)
		
		// Check if it's a type name
		switch strings.ToLower(typeStr) {
		case "hash", "map":
			DEBUG("returning empty hash")
			return &Response{
				Type:  Replace,
				Value: map[interface{}]interface{}{},
			}, nil

		case "array", "list":
			DEBUG("returning empty array")
			return &Response{
				Type:  Replace,
				Value: []interface{}{},
			}, nil

		case "string":
			DEBUG("returning empty string")
			return &Response{
				Type:  Replace,
				Value: "",
			}, nil

		default:
			// If it's not a recognized type, treat it as a request to check if the value is empty
			DEBUG("'%s' is not a recognized type, checking if it's empty", typeStr)
			return &Response{
				Type:  Replace,
				Value: typeStr == "",
			}, nil
		}
	}

	// If the argument resolved to something else, check if it's empty
	DEBUG("checking if resolved value is empty")
	isEmpty := false
	
	switch v := val.(type) {
	case nil:
		isEmpty = true
	case string:
		isEmpty = v == ""
	case []interface{}:
		isEmpty = len(v) == 0
	case map[interface{}]interface{}:
		isEmpty = len(v) == 0
	case map[string]interface{}:
		isEmpty = len(v) == 0
	case int, int64:
		isEmpty = v == 0 || v == int64(0)
	case float64:
		isEmpty = v == 0.0
	case bool:
		isEmpty = !v
	}

	DEBUG("value is empty: %v", isEmpty)
	return &Response{
		Type:  Replace,
		Value: isEmpty,
	}, nil
}

// EnableEnhancedEmpty enables the enhanced empty operator
func EnableEnhancedEmpty() {
	RegisterOp("empty", EmptyOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedEmpty handle it
}