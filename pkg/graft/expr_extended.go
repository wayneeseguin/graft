package graft

import (
	"fmt"
	"github.com/starkandwayne/goutils/tree"
)

// ExtendedExpr adds new fields to support operator calls
// This extends the functionality of the base Expr struct
// without modifying the original definition


// InternString is a string interning function for performance
func InternString(s string) string {
	// For Phase 1, just return the string as-is
	// In later phases, we can implement actual string interning
	return s
}

// Op field for operator name in OperatorCall expressions
func (e *Expr) Op() string {
	if e.Type == OperatorCall {
		return e.Name
	}
	return ""
}

// SetOp sets the operator name for OperatorCall expressions
func (e *Expr) SetOp(op string) {
	if e.Type == OperatorCall {
		// Intern operator names
		e.Name = InternString(op)
	}
}

// Args returns the arguments for an OperatorCall expression
// For now, we'll use Left and Right to store arguments in a linked list
func (e *Expr) Args() []*Expr {
	if e.Type != OperatorCall {
		return nil
	}
	
	// If we have arguments stored in Left as a special ArgsList expression
	if e.Left != nil && e.Left.Type == ArgsList {
		return e.Left.extractArgs()
	}
	
	return nil
}

// SetArgs sets the arguments for an OperatorCall expression
func (e *Expr) SetArgs(args []*Expr) {
	if e.Type == OperatorCall {
		e.Left = &Expr{
			Type: ArgsList,
		}
		e.Left.storeArgs(args)
	}
}

// ArgsList is a special expression type used internally to store operator arguments
const ArgsList ExprType = 999

// extractArgs extracts arguments from an ArgsList expression
func (e *Expr) extractArgs() []*Expr {
	if e.Type != ArgsList {
		return nil
	}
	
	var args []*Expr
	current := e
	
	// Traverse the linked list structure
	for current != nil && current.Type == ArgsList {
		if current.Left != nil {
			args = append(args, current.Left)
		}
		current = current.Right
	}
	
	return args
}

// storeArgs stores arguments in an ArgsList expression
func (e *Expr) storeArgs(args []*Expr) {
	if e.Type != ArgsList || len(args) == 0 {
		return
	}
	
	// Store first argument
	e.Left = args[0]
	
	// Create linked list for remaining arguments
	current := e
	for i := 1; i < len(args); i++ {
		current.Right = &Expr{
			Type: ArgsList,
			Left: args[i],
		}
		current = current.Right
	}
}

// Dependencies returns the dependencies for this expression
func (e *Expr) Dependencies(ev *Evaluator, locs []*tree.Cursor) []*tree.Cursor {
	deps := []*tree.Cursor{}
	
	switch e.Type {
	case Reference:
		if e.Reference != nil {
			deps = append(deps, e.Reference)
		}
		
	case OperatorCall:
		// Get dependencies from operator arguments
		for _, arg := range e.Args() {
			deps = append(deps, arg.Dependencies(ev, locs)...)
		}
		
		// If this is an Opcall, get operator-specific dependencies
		if e.Call != nil {
			deps = append(deps, e.Call.Dependencies(ev, locs)...)
		}
		
	case LogicalOr:
		if e.Left != nil {
			deps = append(deps, e.Left.Dependencies(ev, locs)...)
		}
		if e.Right != nil {
			deps = append(deps, e.Right.Dependencies(ev, locs)...)
		}
		
	case List:
		// For list expressions, get dependencies from all elements
		if e.Left != nil {
			deps = append(deps, e.Left.Dependencies(ev, locs)...)
		}
		if e.Right != nil {
			deps = append(deps, e.Right.Dependencies(ev, locs)...)
		}
	}
	
	return deps
}

// Evaluate evaluates this expression
func (e *Expr) Evaluate(tree interface{}) (interface{}, error) {
	// Simple evaluation for Phase 1
	switch e.Type {
	case Literal:
		return e.Literal, nil
	case Reference:
		if e.Reference != nil {
			return e.Reference.Resolve(tree)
		}
		return nil, nil
	default:
		return nil, nil
	}
}

// String returns a string representation of the expression
func (e *Expr) String() string {
	if e == nil {
		return "<nil>"
	}
	switch e.Type {
	case Literal:
		return fmt.Sprintf("%v", e.Literal)
	case Reference:
		if e.Reference != nil {
			return e.Reference.String()
		}
		return "<nil reference>"
	case OperatorCall:
		return fmt.Sprintf("(( %s ... ))", e.Op())
	default:
		return fmt.Sprintf("<expr type=%d>", e.Type)
	}
}