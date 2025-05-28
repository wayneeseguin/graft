package operators

import (
	"fmt"
	"sort"

	"github.com/starkandwayne/goutils/tree"
)

// KeysOperator is an enhanced version that supports nested expressions
type KeysOperator struct{}

// Setup ...
func (KeysOperator) Setup() error {
	return nil
}

// Phase ...
func (KeysOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (KeysOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (KeysOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( keys ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( keys ... )) operation at $.%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("keys operator requires exactly one argument")
	}

	// Use ResolveOperatorArgument to handle nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("failed to resolve expression to a concrete value")
		DEBUG("error was: %s", err)
		return nil, err
	}

	if val == nil {
		return nil, fmt.Errorf("keys operator argument resolved to nil")
	}

	// Extract keys based on the type
	var keys []interface{}

	switch v := val.(type) {
	case map[interface{}]interface{}:
		DEBUG("extracting keys from map[interface{}]interface{}")
		for key := range v {
			keys = append(keys, fmt.Sprintf("%v", key))
		}

	case map[string]interface{}:
		DEBUG("extracting keys from map[string]interface{}")
		for key := range v {
			keys = append(keys, key)
		}

	default:
		DEBUG("argument is not a map: %T", v)
		return nil, fmt.Errorf("keys operator only works on maps, got %T", v)
	}

	// Sort the keys for consistent output
	stringKeys := make([]string, len(keys))
	for i, k := range keys {
		stringKeys[i] = fmt.Sprintf("%v", k)
	}
	sort.Strings(stringKeys)

	// Convert back to interface slice
	result := make([]interface{}, len(stringKeys))
	for i, k := range stringKeys {
		result[i] = k
	}

	DEBUG("extracted %d keys", len(result))

	return &Response{
		Type:  Replace,
		Value: result,
	}, nil
}

func init() {
	RegisterOp("keys", KeysOperator{})
}