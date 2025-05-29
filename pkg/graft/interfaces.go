package graft

import (
	"fmt"
	"github.com/starkandwayne/goutils/tree"
)

// Action represents the type of action an operator should take
type Action int

const (
	// Replace the current value
	Replace Action = iota
	// Inject into the parent structure
	Inject
)

// OperatorPhase represents when an operator runs
type OperatorPhase int

const (
	// MergePhase runs during document merging
	MergePhase OperatorPhase = iota
	// EvalPhase runs during evaluation
	EvalPhase
	// ParamPhase runs during parameter resolution
	ParamPhase
)

// Response from an operator execution
type Response struct {
	Type  Action
	Value interface{}
}

// Expr represents a parsed expression
type Expr struct {
	Type      ExprType
	Operator  string
	Name      string
	Left      *Expr
	Right     *Expr
	Literal   interface{}
	Reference *tree.Cursor
	Call      *Opcall
	Pos       Position
}

// ExprType represents the type of expression
type ExprType int

const (
	// Literal value
	Literal ExprType = iota
	// Reference to another part of the document
	Reference
	// List expression
	List
	// Or expression (||)
	Or
	// Negate expression (!)
	Negate
	// Addition operator
	Addition
	// Subtraction operator
	Subtraction
	// Multiplication operator
	Multiplication
	// Division operator  
	Division
	// Modulo operator
	Modulo
	// Comparison operators
	Equal
	NotEqual
	LessThan
	LessThanOrEqual
	GreaterThan
	GreaterThanOrEqual
	// Logical operators
	LogicalAnd
	LogicalOr
	// RegexpMatch operator
	RegexpMatch
	// EnvVar reference
	EnvVar
	// BoshVariable reference
	BoshVar
	// OperatorCall represents a nested operator call
	OperatorCall
	// VaultGroup represents a () grouping expression for vault sub-operators
	VaultGroup
	// VaultChoice represents a | choice expression for vault sub-operators
	VaultChoice
)

// Operator interface that all operators must implement
type Operator interface {
	// Setup performs any necessary initialization
	Setup() error

	// Run evaluates the operator with given arguments
	Run(ev *Evaluator, args []*Expr) (*Response, error)

	// Dependencies returns paths this operator depends on
	Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor

	// Phase returns when this operator should run
	Phase() OperatorPhase
}

// Opcall represents an operator call
type Opcall struct {
	src       string
	where     *tree.Cursor
	canonical *tree.Cursor
	op        Operator
	args      []*Expr
}

// Args returns the arguments for this operator call
func (op *Opcall) Args() []*Expr {
	return op.args
}

// Canonical returns the canonical cursor for this operator call
func (op *Opcall) Canonical() *tree.Cursor {
	return op.canonical
}

// Operator returns the operator for this call
func (op *Opcall) Operator() Operator {
	return op.op
}

// Where returns the cursor location for this operator call
func (op *Opcall) Where() *tree.Cursor {
	return op.where
}

// Src returns the source string for this operator call
func (op *Opcall) Src() string {
	return op.src
}

// Dependencies returns the dependencies for this operator call
func (op *Opcall) Dependencies(ev *Evaluator, locs []*tree.Cursor) []*tree.Cursor {
	l := []*tree.Cursor{}
	for _, arg := range op.args {
		if arg != nil {
			for _, c := range arg.Dependencies(ev, locs) {
				l = append(l, c)
			}
		}
	}
	return op.op.Dependencies(ev, op.args, locs, l)
}

// Run executes this operator call
func (op *Opcall) Run(ev *Evaluator) (*Response, error) {
	was := ev.Here
	ev.Here = op.where
	r, err := op.op.Run(ev, op.args)
	ev.Here = was

	if err != nil {
		return nil, fmt.Errorf("$.%s: %s", op.where, err)
	}
	return r, nil
}

// IsOperator checks if an expression is an operator call
func (e *Expr) IsOperator() bool {
	return e != nil && e.Type == OperatorCall
}

// IsOperatorNamed checks if an expression is a specific operator
func (e *Expr) IsOperatorNamed(name string) bool {
	return e.IsOperator() && e.Operator == name
}

// GetOperatorName returns the operator name if this is an operator expression
func (e *Expr) GetOperatorName() string {
	if e.IsOperator() {
		return e.Operator
	}
	return ""
}

// Op returns the operator name for compatibility
func (e *Expr) Op() string {
	return e.Operator
}

// Args returns the arguments for an operator call expression
func (e *Expr) Args() []*Expr {
	if e.Call != nil {
		return e.Call.Args()
	}
	// For binary operators, return left and right as args
	if e.Left != nil && e.Right != nil {
		return []*Expr{e.Left, e.Right}
	}
	if e.Left != nil {
		return []*Expr{e.Left}
	}
	return nil
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
		if e.Call != nil {
			deps = append(deps, e.Call.Dependencies(ev, locs)...)
		}
	}
	
	// Check left and right expressions
	if e.Left != nil {
		deps = append(deps, e.Left.Dependencies(ev, locs)...)
	}
	if e.Right != nil {
		deps = append(deps, e.Right.Dependencies(ev, locs)...)
	}
	
	return deps
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
		return fmt.Sprintf("%s(...)", e.Operator)
	case EnvVar:
		return fmt.Sprintf("$%s", e.Name)
	case BoshVar:
		return fmt.Sprintf("((%s))", e.Name)
	case LogicalOr:
		return fmt.Sprintf("(%s || %s)", e.Left.String(), e.Right.String())
	case LogicalAnd:
		return fmt.Sprintf("(%s && %s)", e.Left.String(), e.Right.String())
	case Addition:
		return fmt.Sprintf("(%s + %s)", e.Left.String(), e.Right.String())
	case Subtraction:
		return fmt.Sprintf("(%s - %s)", e.Left.String(), e.Right.String())
	case Multiplication:
		return fmt.Sprintf("(%s * %s)", e.Left.String(), e.Right.String())
	case Division:
		return fmt.Sprintf("(%s / %s)", e.Left.String(), e.Right.String())
	case Modulo:
		return fmt.Sprintf("(%s %% %s)", e.Left.String(), e.Right.String())
	case Equal:
		return fmt.Sprintf("(%s == %s)", e.Left.String(), e.Right.String())
	case NotEqual:
		return fmt.Sprintf("(%s != %s)", e.Left.String(), e.Right.String())
	case LessThan:
		return fmt.Sprintf("(%s < %s)", e.Left.String(), e.Right.String())
	case LessThanOrEqual:
		return fmt.Sprintf("(%s <= %s)", e.Left.String(), e.Right.String())
	case GreaterThan:
		return fmt.Sprintf("(%s > %s)", e.Left.String(), e.Right.String())
	case GreaterThanOrEqual:
		return fmt.Sprintf("(%s >= %s)", e.Left.String(), e.Right.String())
	case Negate:
		return fmt.Sprintf("!%s", e.Left.String())
	default:
		return fmt.Sprintf("<unknown type %d>", e.Type)
	}
}

// SetArgs sets the arguments for an operator call expression
func (e *Expr) SetArgs(args []*Expr) {
	if e.Call != nil {
		e.Call.args = args
	}
}

// Evaluate evaluates the expression against the given tree
func (e *Expr) Evaluate(tree interface{}) (interface{}, error) {
	switch e.Type {
	case Literal:
		return e.Literal, nil
	case Reference:
		if e.Reference != nil {
			return e.Reference.Resolve(tree)
		}
		return nil, fmt.Errorf("nil reference")
	case EnvVar:
		// TODO: Implement environment variable lookup
		return nil, fmt.Errorf("environment variable evaluation not implemented")
	case OperatorCall:
		// TODO: Implement operator call evaluation
		return nil, fmt.Errorf("operator call evaluation not implemented")
	default:
		return nil, fmt.Errorf("unsupported expression type for evaluation: %d", e.Type)
	}
}
