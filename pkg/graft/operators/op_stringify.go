package operators

import (
	"fmt"

	"github.com/geofffranks/yaml"
	"github.com/starkandwayne/goutils/tree"
)

// StringifyOperator ...
type StringifyOperator struct{}

// Setup ...
func (StringifyOperator) Setup() error {
	return nil
}

// Phase ...
func (StringifyOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (StringifyOperator) Dependencies(ev *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := auto

	for _, arg := range args {
		if arg.Type == OperatorCall {
			// Get dependencies from nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), nil, nil)
				deps = append(deps, nestedDeps...)
			}
		} else if arg.Type == Reference {
			deps = append(deps, arg.Reference)
		}
	}

	return deps
}

// Run ...
func (StringifyOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( stringify ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( stringify ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("stringify operator requires exactly one reference argument")
	}

	// Use ResolveOperatorArgument to support nested expressions
	resolved, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG(" resolution failed\n error: %s", err)
		return nil, err
	}

	var val interface{}

	// Special case for nil values
	if resolved == nil {
		DEBUG(" found nil value")
		val = nil
	} else if str, ok := resolved.(string); ok {
		// Check if it's already a string (literal case)
		DEBUG(" found literal string '%s'", str)
		val = str
	} else {
		// For non-strings, marshal to YAML
		DEBUG("  resolved to a value (could be a map, a list or a scalar)")
		data, err := yaml.Marshal(resolved)
		if err != nil {
			DEBUG("   marshaling failed\n   error: %s", err)
			return nil, fmt.Errorf("Unable to marshal value: %s", err)
		}
		val = string(data)
	}
	DEBUG("")

	return &Response{
		Type:  Replace,
		Value: val,
	}, nil
}

func init() {
	RegisterOp("stringify", StringifyOperator{})
}
