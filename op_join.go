package spruce

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"

	. "github.com/geofffranks/spruce/log"
)

// JoinOperator is invoked with (( join <separator> <lists/strings>... )) and
// joins lists and strings into one string, separated by <separator>
type JoinOperator struct{}

// Setup ...
func (JoinOperator) Setup() error {
	return nil
}

// Phase ...
func (JoinOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns the nodes that (( join ... )) requires to be resolved
// before its evaluation. Returns no dependencies on error, because who cares
// about eval order if Run is going to bomb out anyway.
func (JoinOperator) Dependencies(ev *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	DEBUG("Calculating dependencies for (( join ... ))")
	deps := auto
	if len(args) < 2 {
		DEBUG("Not enough arguments to (( join ... ))")
		return deps
	}

	//skip the separator arg
	for _, arg := range args[1:] {
		if arg.Type == Literal {
			continue
		}
		if arg.Type == OperatorCall {
			// Get dependencies from nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), nil, nil)
				deps = append(deps, nestedDeps...)
			}
			continue
		}
		if arg.Type == Reference {
			// For references, we need to check if they point to lists
			list, err := arg.Reference.Resolve(ev.Tree)
			if err != nil {
				DEBUG("Could not retrieve object at path '%s'", arg.String())
				deps = append(deps, arg.Reference)
				continue
			}
			//must be a list or a string
			switch list.(type) {
			case []interface{}:
				//add .* to the end of the cursor so we can glob all the elements
				globCursor, err := tree.ParseCursor(fmt.Sprintf("%s.*", arg.Reference.String()))
				if err != nil {
					DEBUG("Could not parse cursor with '.*' appended. This is a BUG")
					deps = append(deps, arg.Reference)
					continue
				}
				//have the cursor library get all the subelements for us
				subElements, err := globCursor.Glob(ev.Tree)
				if err != nil {
					DEBUG("Could not retrieve subelements at path '%s'. This may be a BUG.", arg.String())
					deps = append(deps, arg.Reference)
					continue
				}
				deps = append(deps, subElements...)
			default:
				deps = append(deps, arg.Reference)
			}
		}
	}

	DEBUG("Dependencies for (( join ... )):")
	for i, dep := range deps {
		DEBUG("\t#%d %s", i, dep.String())
	}
	return deps
}

// Run ...
func (JoinOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( join ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( join ... )) operation at $%s\n", ev.Here)

	if len(args) == 0 {
		DEBUG("  no arguments supplied to (( join ... )) operation.")
		return nil, ansi.Errorf("no arguments specified to @c{(( join ... ))}")
	}

	if len(args) == 1 {
		DEBUG("  too few arguments supplied to (( join ... )) operation.")
		return nil, ansi.Errorf("too few arguments supplied to @c{(( join ... ))}")
	}

	var separator string
	var list []string

	for i, arg := range args {
		if i == 0 { // argument #0: separator
			// The original join operator only accepts literals for separator
			if arg.Type != Literal {
				DEBUG("     [%d]: unsupported type for join operator separator argument: '%v'", i, arg)
				return nil, fmt.Errorf("join operator only accepts literal argument for the separator")
			}
			separator = fmt.Sprintf("%v", arg.Literal)
			DEBUG("     [%d]: list separator will be: %s", i, separator)

		} else { // argument #1..n: list, or literal
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

			switch v := val.(type) {
			case []interface{}:
				DEBUG("     [%d]: resolved to a list", i)
				for idx, entry := range v {
					switch entry.(type) {
					case []interface{}:
						DEBUG("     [%d]: entry #%d in list is a list (not a literal)", i, idx)
						return nil, ansi.Errorf("entry #%d in list is not compatible for @c{(( join ... ))}", idx)

					case map[interface{}]interface{}, map[string]interface{}:
						DEBUG("     [%d]: entry #%d in list is a map (not a literal)", i, idx)
						return nil, ansi.Errorf("entry #%d in list is not compatible for @c{(( join ... ))}", idx)

					default:
						list = append(list, fmt.Sprintf("%v", entry))
					}
				}

			case map[interface{}]interface{}, map[string]interface{}:
				DEBUG("     [%d]: resolved to a map (not a list or a literal)", i)
				return nil, ansi.Errorf("referenced entry is not a list or string for @c{(( join ... ))}")

			default:
				DEBUG("     [%d]: resolved to a literal", i)
				list = append(list, fmt.Sprintf("%v", v))
			}
		}
	}

	// finally, join and return
	DEBUG("  joined list: %s", strings.Join(list, separator))
	return &Response{
		Type:  Replace,
		Value: strings.Join(list, separator),
	}, nil
}

func init() {
	RegisterOp("join", JoinOperator{})
}
