package graft

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// JoinOperatorEnhanced is an enhanced version that supports nested expressions
type JoinOperatorEnhanced struct{}

// Setup ...
func (JoinOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (JoinOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns the nodes that (( join ... )) requires to be resolved
func (JoinOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// For the enhanced version, we can't pre-calculate all dependencies
	// because nested expressions might compute paths dynamically
	return auto
}

// Run ...
func (JoinOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
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
			// Use ResolveOperatorArgument to handle nested expressions
			sep, err := ResolveOperatorArgument(ev, arg)
			if err != nil {
				DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
				return nil, err
			}

			if sep == nil {
				DEBUG("     [%d]: separator resolved to nil", i)
				return nil, fmt.Errorf("join operator separator cannot be nil")
			}

			DEBUG("     [%d]: list separator will be: %v", i, sep)
			separator = fmt.Sprintf("%v", sep)

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

			case map[interface{}]interface{}, map[string]interface{}:
				DEBUG("     [%d]: resolved to a map (not a list or a literal)", i)
				return nil, ansi.Errorf("join operator cannot join maps, only lists and literals")

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

// EnableEnhancedJoin enables the enhanced join operator
func EnableEnhancedJoin() {
	RegisterOp("join", JoinOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedJoin handle it
}