package operators

import (
	"encoding/base64"
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// Base64Operator ...
type Base64Operator struct{}

// Setup ...
func (Base64Operator) Setup() error {
	return nil
}

// Phase ...
func (Base64Operator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (Base64Operator) Dependencies(ev *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
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
func (Base64Operator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( base64 ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( base64 ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("base64 operator requires exactly one string or reference argument")
	}

	// Use ResolveOperatorArgument to support nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("  arg[0]: failed to resolve expression to a concrete value")
		DEBUG("     [0]: error was: %s", err)
		return nil, err
	}

	// Check if it's a string - original behavior only accepts strings
	contents, ok := val.(string)
	if !ok {
		DEBUG("  arg[0]: %v is not a string scalar", val)
		return nil, ansi.Errorf("@R{tried to base64 encode} @c{%v}@R{, which is not a string scalar}", val)
	}

	DEBUG("  resolved argument to string: %s", contents)

	encoded := base64.StdEncoding.EncodeToString([]byte(contents))
	DEBUG("  resolved (( base64 ... )) operation to the string:\n    \"%s\"", string(encoded))

	return &Response{
		Type:  Replace,
		Value: string(encoded),
	}, nil
}

func init() {
	RegisterOp("base64", Base64Operator{})
}
