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
