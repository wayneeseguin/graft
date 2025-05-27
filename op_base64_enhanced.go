package graft

import (
	"encoding/base64"
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// Base64OperatorEnhanced is an enhanced version that supports nested expressions
type Base64OperatorEnhanced struct{}

// Setup ...
func (Base64OperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (Base64OperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (Base64OperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (Base64OperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( base64 ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( base64 ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("base64 operator requires exactly one string or reference argument, got %d arguments", len(args))
	}

	// Use ResolveOperatorArgument to handle nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("  arg[0]: failed to resolve expression to a concrete value")
		DEBUG("     [0]: error was: %s", err)
		return nil, err
	}

	if val == nil {
		return nil, ansi.Errorf("@R{base64 operator argument resolved to nil}")
	}

	// Convert to string
	var contents string
	switch v := val.(type) {
	case string:
		DEBUG("  resolved to string: '%s'", v)
		contents = v
	default:
		DEBUG("  resolved to non-string: %T = %v", v, v)
		// For non-string scalars, convert to string representation
		contents = fmt.Sprintf("%v", v)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(contents))
	DEBUG("  resolved (( base64 ... )) operation to the string:\n    \"%s\"", encoded)

	return &Response{
		Type:  Replace,
		Value: encoded,
	}, nil
}

// EnableEnhancedBase64 enables the enhanced base64 operator
func EnableEnhancedBase64() {
	RegisterOp("base64", Base64OperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedBase64 handle it
}