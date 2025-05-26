package spruce

import (
	"fmt"
)

// ExprType represents the type of an expression
// NOTE: This extends the existing ExprType enumeration
const (
	// Existing types (already defined in operator.go)
	// Reference ExprType = iota
	// Literal
	// LogicalOr
	// EnvVar
	
	// New type for operator calls
	OperatorCall ExprType = iota + 10 // Start at 10 to avoid conflicts
)

// OperatorExpr represents an operator call expression
// This is a specialized expression type for nested operators
type OperatorExpr struct {
	Type   ExprType
	OpName string   // Operator name (e.g., "grab", "concat", "vault")
	OpArgs []*Expr  // Operator arguments
	Phase  OperatorPhase // Phase when this operator should run
}

// NewOperatorExpr creates a new operator expression
func NewOperatorExpr(name string, args []*Expr) *Expr {
	return &Expr{
		Type: OperatorCall,
		Name: name,
		// We'll store operator args in a special way
		// For now, we use the Name field for the operator name
		// and will extend the Expr struct in the next iteration
	}
}

// IsOperator checks if an expression is an operator call
func (e *Expr) IsOperator() bool {
	return e != nil && e.Type == OperatorCall
}

// IsOperatorNamed checks if an expression is a specific operator
func (e *Expr) IsOperatorNamed(name string) bool {
	return e.IsOperator() && e.Name == name
}

// GetOperatorName returns the operator name if this is an operator expression
func (e *Expr) GetOperatorName() string {
	if e.IsOperator() {
		return e.Name
	}
	return ""
}

// OperatorInfo stores metadata about operators for the enhanced parser
type OperatorInfo struct {
	Name          string
	Precedence    Precedence
	Associativity Associativity
	MinArgs       int
	MaxArgs       int // -1 for unlimited
	Phase         OperatorPhase
}

// Precedence levels for operators
type Precedence int

const (
	PrecedenceLowest Precedence = iota
	PrecedenceTernary           // ? : (lowest precedence)
	PrecedenceOr                // ||
	PrecedenceAnd               // &&
	PrecedenceEquality          // == !=
	PrecedenceComparison        // < > <= >=
	PrecedenceAdditive          // + -
	PrecedenceMultiplicative    // * / %
	PrecedenceUnary             // ! - (future)
	PrecedenceCall              // operator calls
	PrecedenceHighest           // literals, references, parentheses
)

// Associativity rules for operators
type Associativity int

const (
	LeftAssociative Associativity = iota
	RightAssociative
)

// Additional precedence constants to match parser needs
const (
	PrecedenceLogicalOr      = PrecedenceOr
	PrecedenceAddition       = PrecedenceAdditive
	PrecedenceMultiplication = PrecedenceMultiplicative
	PrecedencePostfix        = PrecedenceCall
)

// Additional associativity constants
const (
	AssociativityLeft  = LeftAssociative
	AssociativityRight = RightAssociative
)

// Token is defined in tokenizer_enhanced.go
// TokenEOF is defined in tokenizer_enhanced.go

// OperatorRegistry manages operator metadata
type OperatorRegistry struct {
	operators map[string]*OperatorInfo
}

// NewOperatorRegistry creates a new operator registry
func NewOperatorRegistry() *OperatorRegistry {
	registry := &OperatorRegistry{
		operators: make(map[string]*OperatorInfo),
	}
	
	// Populate with all known operators
	for name, info := range OperatorInfoRegistry {
		infoCopy := info // Copy to avoid pointer issues
		registry.operators[name] = &infoCopy
	}
	
	return registry
}

// Register adds an operator to the registry
func (r *OperatorRegistry) Register(info *OperatorInfo) {
	r.operators[info.Name] = info
}

// Get retrieves operator info
func (r *OperatorRegistry) Get(name string) *OperatorInfo {
	return r.operators[name]
}

// OperatorInfoRegistry contains metadata for all operators
// This will be populated during initialization
var OperatorInfoRegistry = map[string]OperatorInfo{
	"vault-try": {
		Name:       "vault-try",
		Precedence: PrecedenceCall,
		MinArgs:    2, // at least one path + default
		MaxArgs:    -1,
		Phase:      EvalPhase,
	},
	"grab": {
		Name:       "grab",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"concat": {
		Name:       "concat",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    -1,
		Phase:      EvalPhase,
	},
	"static_ips": {
		Name:       "static_ips",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    -1,
		Phase:      EvalPhase,
	},
	"join": {
		Name:       "join",
		Precedence: PrecedenceCall,
		MinArgs:    2, // separator and list
		MaxArgs:    -1,
		Phase:      EvalPhase,
	},
	"base64": {
		Name:       "base64",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"base64-decode": {
		Name:       "base64-decode",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"file": {
		Name:       "file",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    2,
		Phase:      EvalPhase,
	},
	"keys": {
		Name:       "keys",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"stringify": {
		Name:       "stringify",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"negate": {
		Name:       "negate",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"empty": {
		Name:       "empty",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"null": {
		Name:       "null",
		Precedence: PrecedenceCall,
		MinArgs:    0,
		MaxArgs:    1,
		Phase:      EvalPhase,
	},
	"defer": {
		Name:       "defer",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      MergePhase,
	},
	"param": {
		Name:       "param",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    1,
		Phase:      ParamPhase,
	},
	"vault": {
		Name:       "vault",
		Precedence: PrecedenceCall,
		MinArgs:    1,
		MaxArgs:    2,
		Phase:      EvalPhase,
	},
	// Arithmetic operators
	"+": {
		Name:          "+",
		Precedence:    PrecedenceAdditive,
		Associativity: LeftAssociative,
		MinArgs:       2,
		MaxArgs:       2,
		Phase:         EvalPhase,
	},
	"-": {
		Name:          "-",
		Precedence:    PrecedenceAdditive,
		Associativity: LeftAssociative,
		MinArgs:       2,
		MaxArgs:       2,
		Phase:         EvalPhase,
	},
	"*": {
		Name:          "*",
		Precedence:    PrecedenceMultiplicative,
		Associativity: LeftAssociative,
		MinArgs:       2,
		MaxArgs:       2,
		Phase:         EvalPhase,
	},
	"/": {
		Name:          "/",
		Precedence:    PrecedenceMultiplicative,
		Associativity: LeftAssociative,
		MinArgs:       2,
		MaxArgs:       2,
		Phase:         EvalPhase,
	},
	"%": {
		Name:          "%",
		Precedence:    PrecedenceMultiplicative,
		Associativity: LeftAssociative,
		MinArgs:       2,
		MaxArgs:       2,
		Phase:         EvalPhase,
	},
}

// GetOperatorInfo retrieves operator metadata
func GetOperatorInfo(name string) (OperatorInfo, bool) {
	info, ok := OperatorInfoRegistry[name]
	return info, ok
}

// IsRegisteredOperator checks if a name is a registered operator
func IsRegisteredOperator(name string) bool {
	_, ok := OperatorInfoRegistry[name]
	return ok
}

// ValidateOperatorArgs validates argument count for an operator
func ValidateOperatorArgs(opName string, argCount int) error {
	info, ok := GetOperatorInfo(opName)
	if !ok {
		return fmt.Errorf("unknown operator: %s", opName)
	}
	
	if argCount < info.MinArgs {
		return fmt.Errorf("operator %s requires at least %d arguments, got %d", 
			opName, info.MinArgs, argCount)
	}
	
	if info.MaxArgs != -1 && argCount > info.MaxArgs {
		return fmt.Errorf("operator %s accepts at most %d arguments, got %d", 
			opName, info.MaxArgs, argCount)
	}
	
	return nil
}

// String representation for debugging (extends existing String method)
func (e *Expr) StringEnhanced() string {
	switch e.Type {
	case OperatorCall:
		return fmt.Sprintf("%s(...)", e.Name)
	default:
		// Fall back to existing String() method
		return e.String()
	}
}