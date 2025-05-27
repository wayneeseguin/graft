package graft

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// ShuffleOperator ...
type ShuffleOperator struct{}

// Setup ...
func (ShuffleOperator) Setup() error {
	return nil
}

// Phase ...
func (ShuffleOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (ShuffleOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (ShuffleOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( shuffle ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( shuffle ... )) operation at $%s\n", ev.Here)

	var vals []interface{}

	for i, arg := range args {
		// Use ResolveOperatorArgument to support nested expressions
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
			return nil, err
		}

		if val == nil {
			DEBUG("  arg[%d]: resolved to nil", i)
			return nil, fmt.Errorf("shuffle operator argument cannot be nil")
		}

		switch v := val.(type) {
		case []interface{}:
			DEBUG("  arg[%d]: found list value", i)
			for _, thing := range v {
				vals = append(vals, thing)
			}

		case map[interface{}]interface{}, map[string]interface{}:
			DEBUG("     [%d]: resolved to a map; error!", i)
			return nil, fmt.Errorf("shuffle only accepts arrays and scalar values")

		default:
			DEBUG("  arg[%d]: found scalar value '%v'", i, val)
			vals = append(vals, val)
		}
		DEBUG("")
	}

	return &Response{
		Type:  Replace,
		Value: shuffle(vals),
	}, nil
}

func init() {
	RegisterOp("shuffle", ShuffleOperator{})
}

func shuffle(l []interface{}) []interface{} {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(l), func(i, j int) { l[i], l[j] = l[j], l[i] })
	return l
}
