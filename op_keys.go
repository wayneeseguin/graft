package graft

import (
	"fmt"
	"sort"

	"github.com/starkandwayne/goutils/ansi"

	. "github.com/wayneeseguin/graft/log"
	"github.com/starkandwayne/goutils/tree"
)

// KeysOperator ...
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
	defer DEBUG("done with (( keys ... )) operation at $%s\n", ev.Here)

	var vals []string

	for i, arg := range args {
		// The keys operator traditionally only accepts references, not literals or expressions
		if arg.Type == Literal {
			DEBUG("  arg[%d]: found literal value", i)
			DEBUG("           (keys operator only handles references to other parts of the YAML tree)")
			return nil, fmt.Errorf("keys operator only accepts key reference arguments")
		}

		// Use ResolveOperatorArgument to support nested expressions
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
			// Wrap error to maintain backward compatibility
			if arg.Type == Reference {
				return nil, fmt.Errorf("Unable to resolve `%s`: %s", arg.Reference, err)
			}
			return nil, err
		}

		// Check if the resolved value is a map
		switch m := val.(type) {
		case map[interface{}]interface{}:
			DEBUG("     [%d]: resolved to a map; extracting keys", i)
			for k := range m {
				vals = append(vals, k.(string))
			}
		case map[string]interface{}:
			DEBUG("     [%d]: resolved to a map; extracting keys", i)
			for k := range m {
				vals = append(vals, k)
			}
		default:
			DEBUG("     [%d]: resolved to something that is not a map.  that is unacceptable.", i)
			if arg.Type == Reference {
				return nil, ansi.Errorf("@c{%s} @R{is not a map}", arg.Reference)
			}
			return nil, ansi.Errorf("@R{argument is not a map}")
		}
		DEBUG("")
	}

	switch len(args) {
	case 0:
		DEBUG("  no arguments supplied to (( keys ... )) operation.  oops.")
		return nil, ansi.Errorf("no arguments specified to @c{(( keys ... ))}")

	default:
		sort.Strings(vals)
		return &Response{
			Type:  Replace,
			Value: vals,
		}, nil
	}
}

func init() {
	RegisterOp("keys", KeysOperator{})
}
