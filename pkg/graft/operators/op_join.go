package operators

import (
	"fmt"
	"sort"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// JoinOperator handles nested operator calls
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
func (JoinOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// For the  version, we can't pre-calculate all dependencies
	// because nested expressions might compute paths dynamically
	return auto
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
	list := GetStringSlice()
	defer PutStringSlice(list)

	for i, arg := range args {
		if i == 0 { // argument #0: separator
			// The separator must be a literal
			if arg.Type != Literal {
				DEBUG("     [%d]: separator is not a literal", i)
				return nil, ansi.Errorf("join operator only accepts literal argument for the separator")
			}

			DEBUG("     [%d]: list separator will be: %v", i, arg.Literal)
			separator = fmt.Sprintf("%v", arg.Literal)

		} else { // argument #1..n: list, literal, or expression
			// Use ResolveOperatorArgument to handle nested expressions
			val, err := ResolveOperatorArgument(ev, arg)
			if err != nil {
				DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
				return nil, err
			}

			if val == nil {
				DEBUG("     [%d]: argument resolved to nil, skipping", i)
				continue
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
						*list = append(*list, fmt.Sprintf("%v", entry))
					}
				}

			case map[interface{}]interface{}:
				DEBUG("     [%d]: resolved to a map", i)
				// Sort keys for consistent output
				keys := make([]string, 0, len(v))
				for k := range v {
					keys = append(keys, fmt.Sprintf("%v", k))
				}
				sort.Strings(keys)
				
				// Join key:value pairs
				for _, k := range keys {
					pair := fmt.Sprintf("%s:%v", k, v[k])
					*list = append(*list, pair)
				}
				
			case map[string]interface{}:
				DEBUG("     [%d]: resolved to a map", i)
				// Sort keys for consistent output
				keys := make([]string, 0, len(v))
				for k := range v {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				
				// Join key:value pairs
				for _, k := range keys {
					pair := fmt.Sprintf("%s:%v", k, v[k])
					*list = append(*list, pair)
				}

			default:
				DEBUG("     [%d]: resolved to a literal value", i)
				*list = append(*list, fmt.Sprintf("%v", v))
			}
		}
	}

	// finally, join and return
	DEBUG("  joined list: %s", strings.Join(*list, separator))
	return &Response{
		Type:  Replace,
		Value: strings.Join(*list, separator),
	}, nil
}

func init() {
	RegisterOp("join", JoinOperator{})
}