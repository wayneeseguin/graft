package parser

import (
	"fmt"
	"unsafe"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// OperatorRegistry holds operator metadata for the parser
type OperatorRegistry struct {
	operators map[string]*OperatorInfo
}

// NewOperatorRegistry creates a new operator registry
func NewOperatorRegistry() *OperatorRegistry {
	return &OperatorRegistry{
		operators: make(map[string]*OperatorInfo),
	}
}

// Register adds an operator to the registry
func (r *OperatorRegistry) Register(info *OperatorInfo) {
	r.operators[info.Name] = info
}

// Get returns operator info for a given name
func (r *OperatorRegistry) Get(name string) (*OperatorInfo, bool) {
	info, ok := r.operators[name]
	return info, ok
}

// Count returns the number of registered operators
func (r *OperatorRegistry) Count() int {
	return len(r.operators)
}

// Names returns a slice of all registered operator names
func (r *OperatorRegistry) Names() []string {
	names := make([]string, 0, len(r.operators))
	for name := range r.operators {
		names = append(names, name)
	}
	return names
}

// ErrorRecoveryContext manages error recovery during parsing
type ErrorRecoveryContext struct {
	errors []error
	maxErrors int
}

// NewErrorRecoveryContext creates a new error recovery context
func NewErrorRecoveryContext(maxErrors int) *ErrorRecoveryContext {
	return &ErrorRecoveryContext{
		errors: []error{},
		maxErrors: maxErrors,
	}
}

// StopOnFirst returns true if we should stop on first error
func (e *ErrorRecoveryContext) StopOnFirst() bool {
	return e.maxErrors == 1
}

// MaxErrors returns the maximum number of errors
func (e *ErrorRecoveryContext) MaxErrors() int {
	return e.maxErrors
}

// RecordError records an error
func (e *ErrorRecoveryContext) RecordError(err error) bool {
	e.errors = append(e.errors, err)
	return len(e.errors) < e.maxErrors
}

// HasErrors returns true if there are any errors
func (e *ErrorRecoveryContext) HasErrors() bool {
	return len(e.errors) > 0
}

// GetError returns the first error or a multi-error
func (e *ErrorRecoveryContext) GetError() error {
	if len(e.errors) == 0 {
		return nil
	}
	if len(e.errors) == 1 {
		return e.errors[0]
	}
	// Return multi-error
	return e.errors[0] // Phase 1: Just return first error
}

// evaluateOperatorCall evaluates a nested operator call expression
func evaluateOperatorCall(expr *graft.Expr, ev *graft.Evaluator) (*graft.Response, error) {
	// Phase 1: Simple stub implementation
	// In later phases, this will properly evaluate nested operator calls
	return &graft.Response{
		Type: graft.Replace,
		Value: nil,
	}, nil
}

// Additional helper functions for the parser package

// OperatorFor returns the operator for the given name
func OperatorFor(name string) graft.Operator {
	return graft.OperatorFor(name)
}

// EvaluateExpr evaluates an expression
func EvaluateExpr(expr *graft.Expr, ev *graft.Evaluator) (*graft.Response, error) {
	// Phase 1: Delegate to graft package
	return graft.EvaluateExpr(expr, ev)
}

// DEBUG logs debug messages
func DEBUG(format string, args ...interface{}) {
	graft.DEBUG(format, args...)
}

// NullOperator type alias
type NullOperator = graft.NullOperator

// Operator type alias
type Operator = graft.Operator

// Opcall type alias
type Opcall = graft.Opcall

// Response type alias
type Response = graft.Response

// Evaluator type alias
type Evaluator = graft.Evaluator

// Expr type alias
type Expr = graft.Expr

// OperatorPhase type alias
type OperatorPhase = graft.OperatorPhase

// ExprError type alias
type ExprError = graft.ExprError

// Position type alias
type Position = graft.Position

// OpRegistry reference
var OpRegistry = graft.OpRegistry

// Constants
const (
	SyntaxError = graft.SyntaxError
	Reference = graft.Reference
	LogicalOr = graft.LogicalOr
	OperatorCall = graft.OperatorCall
	Literal = graft.Literal
	EnvVar = graft.EnvVar
	EvalPhase = graft.EvalPhase
	VaultGroup = graft.VaultGroup
	VaultChoice = graft.VaultChoice
)

// Precedence levels for operators
type Precedence int

const (
	PrecedenceNone Precedence = iota
	PrecedenceOr              // ||
	PrecedenceAnd             // &&
	PrecedenceEquality        // == !=
	PrecedenceComparison      // < > <= >=
	PrecedenceAddition        // + -
	PrecedenceMultiplication  // * / %
	PrecedenceUnary           // ! -
	PrecedencePostfix         // function calls
)

// Aliases for compatibility
const (
	PrecedenceAdditive = PrecedenceAddition
	PrecedenceMultiplicative = PrecedenceMultiplication
	PrecedenceLowest = PrecedenceNone
	PrecedenceTernary = PrecedenceOr // Ternary has same precedence as Or
)

// Associativity represents operator associativity
type Associativity int

const (
	LeftAssociative Associativity = iota
	RightAssociative
	AssociativityLeft = LeftAssociative // Alias
)


// OperatorInfo holds metadata about an operator
type OperatorInfo struct {
	Name         string
	Precedence   Precedence
	Associativity Associativity
	MinArgs      int
	MaxArgs      int
	Phase        graft.OperatorPhase
}

// WarningError represents a non-fatal error that should be treated as a warning
type WarningError interface {
	error
	Warn()
}

// ReduceExpr reduces an expression (stub for Phase 1)
func ReduceExpr(e *Expr) (*Expr, error) {
	// Phase 1: Just return self
	return e, nil
}

// createFullOperatorRegistry creates a registry with all operators
func createFullOperatorRegistry() *OperatorRegistry {
	return createOperatorRegistry()
}

// parseOpcallWithParser uses the parser to parse an operator call
func parseOpcallWithParser(phase graft.OperatorPhase, src string) (*graft.Opcall, error) {
	// Phase 1: Delegate to ParseOpcall
	return ParseOpcall(phase, src)
}


// NewOperatorCall creates a new operator call expression
func NewOperatorCall(op string, args []*graft.Expr) *graft.Expr {
	return NewOperatorCallWithPos(op, args, graft.Position{})
}

// NewOperatorCallWithPos creates a new operator call expression with position
func NewOperatorCallWithPos(op string, args []*graft.Expr, pos graft.Position) *graft.Expr {
	expr := &graft.Expr{
		Type:     graft.OperatorCall,
		Name:     op,
		Operator: op,  // Set both Name and Operator for compatibility
		Pos:      pos,
	}
	
	// Create a minimal Opcall structure to store args
	// We use reflection to set the private args field
	opcall := &graft.Opcall{}
	
	// Use a type assertion and struct literal to create opcall with args
	// This is a workaround since we can't access private fields directly
	type opcallWithArgs struct {
		src       string
		where     *tree.Cursor
		canonical *tree.Cursor
		op        graft.Operator
		args      []*graft.Expr
	}
	
	// Create a new opcall with args set
	opcallPtr := (*opcallWithArgs)(unsafe.Pointer(opcall)) // #nosec G103 - required for efficient struct field access
	opcallPtr.args = args
	
	expr.Call = opcall
	
	// Also set Left/Right for backward compatibility
	if len(args) >= 1 {
		expr.Left = args[0]
	}
	if len(args) >= 2 {
		expr.Right = args[1]
	}
	
	return expr
}

// NewSyntaxError creates a new syntax error
func NewSyntaxError(msg string, pos graft.Position) error {
	return &graft.ExprError{
		Type:     graft.SyntaxError,
		Message:  msg,
		Position: pos,
	}
}

// IsRegisteredOperator checks if an operator is registered
func IsRegisteredOperator(name string) bool {
	_, ok := graft.OpRegistry[name]
	return ok
}

// DefaultKeyGenerator generates a default cache key
func DefaultKeyGenerator(phase graft.OperatorPhase, src string) string {
	return fmt.Sprintf("%d:%s", phase, src)
}

// SetModifierExpr sets a modifier on an expression
func SetModifierExpr(e *Expr, modifier string) {
	// Phase 1: No-op
}

// InternString is a string interning function
func InternString(s string) string {
	// Phase 1: Just return the string as-is
	return s
}