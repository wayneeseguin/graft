package spruce

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"

	. "github.com/geofffranks/spruce/log"
)

// ConcatOperatorEnhanced is an enhanced version that handles nested operator calls
type ConcatOperatorEnhanced struct{}

// Setup ...
func (ConcatOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (ConcatOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (ConcatOperatorEnhanced) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// Include dependencies from nested operator calls
	deps := auto
	
	for _, arg := range args {
		if arg.Type == OperatorCall {
			// Get dependencies from nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), locs, auto)
				deps = append(deps, nestedDeps...)
			}
		}
	}
	
	return deps
}

// Run ...
func (ConcatOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( concat ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( concat ... )) operation at $%s\n", ev.Here)

	var l []string

	if len(args) < 2 {
		return nil, fmt.Errorf("concat operator requires at least two arguments")
	}

	for i, arg := range args {
		// Use the helper to resolve arguments, including nested operators
		v, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
			DEBUG("     [%d]: error was: %s", i, err)
			return nil, err
		}

		// Convert to string
		var stringVal string
		switch val := v.(type) {
		case string:
			stringVal = val
		case []interface{}:
			stringSlice := make([]string, len(val))
			for j, elem := range val {
				stringSlice[j] = fmt.Sprintf("%v", elem)
			}
			stringVal = strings.Join(stringSlice, "")
		default:
			stringVal = fmt.Sprintf("%v", v)
		}

		DEBUG("  arg[%d]: using '%s'", i, stringVal)
		l = append(l, stringVal)
	}

	// Escape the result for shell safety if needed
	var out []string
	for _, s := range l {
		if len(s) == 0 {
			out = append(out, "''")
		} else if strings.Contains(s, "'") {
			out = append(out, fmt.Sprintf(`"%s"`, strings.Replace(s, `"`, `\"`, -1)))
		} else if strings.ContainsAny(s, " \t\n\"$") {
			out = append(out, fmt.Sprintf("'%s'", s))
		} else {
			out = append(out, s)
		}
	}

	DEBUG("  result: %s", ansi.Sprintf("@c{%s}", strings.Join(l, "")))
	return &Response{
		Type:  Replace,
		Value: strings.Join(l, ""),
	}, nil
}

// EnableEnhancedConcat replaces the standard concat operator with the enhanced version
func EnableEnhancedConcat() {
	RegisterOp("concat", ConcatOperatorEnhanced{})
}