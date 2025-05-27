package operators

import (
	"encoding/base64"
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// Base64DecodeOperator ...
type Base64DecodeOperator struct{}

// Setup ...
func (Base64DecodeOperator) Setup() error {
	return nil
}

// Phase ...
func (Base64DecodeOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (Base64DecodeOperator) Dependencies(ev *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
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
func (Base64DecodeOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( base64-decode ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( base64-decode ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("base64-decode operator requires exactly one string or reference argument")
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
		return nil, ansi.Errorf("@R{tried to base64 decode} @c{%v}@R{, which is not a string scalar}", val)
	}

	DEBUG("  resolved argument to string: %s", contents)

	if decoded, err := base64.StdEncoding.DecodeString(contents); err == nil {
		DEBUG("  resolved (( base64-decode ... )) operation to the string:\n    \"%s\"", string(decoded))
		return &Response{
			Type:  Replace,
			Value: string(decoded),
		}, nil
	} else {
		return nil, fmt.Errorf("unable to base64 decode string %s: %s", contents, err)
	}
}

func init() {
	RegisterOp("base64-decode", Base64DecodeOperator{})
}
