package operators

import (
	"fmt"
	"sort"

	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// KeysOperator handles nested operator calls
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

	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments specified to (( keys ... ))")
	}

	// Collect all keys from all arguments
	keySet := make(map[string]bool)

	for i, arg := range args {
		// Use ResolveOperatorArgument to handle nested expressions
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("arg[%d]: failed to resolve expression to a concrete value", i)
			DEBUG("error was: %s", err)
			return nil, err
		}

		if val == nil {
			DEBUG("arg[%d]: resolved to nil, skipping", i)
			continue
		}

		// Extract keys based on the type
		switch v := val.(type) {
		case map[interface{}]interface{}:
			DEBUG("arg[%d]: extracting keys from map[interface{}]interface{}", i)
			for key := range v {
				keySet[fmt.Sprintf("%v", key)] = true
			}

		case map[string]interface{}:
			DEBUG("arg[%d]: extracting keys from map[string]interface{}", i)
			for key := range v {
				keySet[key] = true
			}

		default:
			DEBUG("arg[%d]: is not a map: %T", i, v)
			// Try to get the original path reference from the argument
			if arg.Type == Reference && arg.Reference != nil {
				return nil, fmt.Errorf("%s is not a map", arg.Reference.String())
			}
			return nil, fmt.Errorf("keys operator only works on maps, argument %d is %T", i, v)
		}
	}

	// Convert to sorted list
	stringKeys := make([]string, 0, len(keySet))
	for key := range keySet {
		stringKeys = append(stringKeys, key)
	}
	sort.Strings(stringKeys)

	// Convert to interface slice
	result := make([]interface{}, len(stringKeys))
	for i, k := range stringKeys {
		result[i] = k
	}

	DEBUG("extracted %d unique keys from %d arguments", len(result), len(args))

	return &Response{
		Type:  Replace,
		Value: result,
	}, nil
}

func init() {
	RegisterOp("keys", KeysOperator{})
}
