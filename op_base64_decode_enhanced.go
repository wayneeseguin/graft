package graft

import (
	"encoding/base64"
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// Base64DecodeOperatorEnhanced is an enhanced version that supports nested expressions
type Base64DecodeOperatorEnhanced struct{}

// Setup ...
func (Base64DecodeOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (Base64DecodeOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (Base64DecodeOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (Base64DecodeOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( base64-decode ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( base64-decode ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("base64-decode operator requires exactly one string or reference argument")
	}

	// Use ResolveOperatorArgument to handle nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("  arg[0]: failed to resolve expression to a concrete value")
		DEBUG("     [0]: error was: %s", err)
		return nil, err
	}

	if val == nil {
		return nil, ansi.Errorf("@R{base64-decode operator argument resolved to nil}")
	}

	// Convert to string
	var contents string
	switch v := val.(type) {
	case string:
		DEBUG("  resolved to string: '%s'", v)
		contents = v
	default:
		DEBUG("  resolved to non-string: %T = %v", v, v)
		return nil, ansi.Errorf("@R{tried to base64 decode} @c{%v}@R{, which is not a string}", v)
	}

	decoded, err := base64.StdEncoding.DecodeString(contents)
	if err != nil {
		return nil, ansi.Errorf("@R{base64 decoding failed:} @c{%s}", err)
	}

	DEBUG("  resolved (( base64-decode ... )) operation to the string:\n    \"%s\"", string(decoded))

	return &Response{
		Type:  Replace,
		Value: string(decoded),
	}, nil
}

// EnableEnhancedBase64Decode enables the enhanced base64-decode operator
func EnableEnhancedBase64Decode() {
	RegisterOp("base64-decode", Base64DecodeOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedBase64Decode handle it
}