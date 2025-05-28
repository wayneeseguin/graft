package operators

import (
	"os"
	"sync"

	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/parser"
)

// Type aliases for convenience
type (
	Evaluator     = graft.Evaluator
	Expr          = graft.Expr
	Response      = graft.Response
	OperatorPhase = graft.OperatorPhase
	ExprType      = graft.ExprType
	Operator      = graft.Operator
)

// Constants
const (
	EvalPhase    = graft.EvalPhase
	MergePhase   = graft.MergePhase
	ParamPhase   = graft.ParamPhase
	Literal      = graft.Literal
	Reference    = graft.Reference
	EnvVar       = graft.EnvVar
	Replace      = graft.Replace
	Inject       = graft.Inject
	LogicalOr    = graft.LogicalOr
	OperatorCall = graft.OperatorCall
)

// Helper functions
func DEBUG(format string, args ...interface{}) {
	graft.DEBUG(format, args...)
}

func TRACE(format string, args ...interface{}) {
	graft.TRACE(format, args...)
}

// OperatorFor returns the operator for the given name
func OperatorFor(name string) Operator {
	return graft.OperatorFor(name)
}

// ResolveEnv resolves environment variables in a slice of strings
func ResolveEnv(nodes []string) []string {
	// Phase 1: Simple implementation
	for i, node := range nodes {
		if len(node) > 2 && node[0] == '$' {
			if val := os.Getenv(node[1:]); val != "" {
				nodes[i] = val
			}
		}
	}
	return nodes
}

// OpRegistry reference
var OpRegistry = graft.OpRegistry

// EvaluateExpr evaluates an expression
func EvaluateExpr(expr *Expr, ev *Evaluator) (*Response, error) {
	return graft.EvaluateExpr(expr, ev)
}

// EvaluateOperatorArgs evaluates operator arguments and returns a list of values
func EvaluateOperatorArgs(ev *Evaluator, args []*Expr) ([]interface{}, error) {
	values := make([]interface{}, len(args))
	for i, arg := range args {
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	return values, nil
}

// String pool for performance
var stringPool = &sync.Pool{
	New: func() interface{} {
		return make([]string, 0, 10)
	},
}

// GetStringSlice gets a string slice from the pool
func GetStringSlice() *[]string {
	s := stringPool.Get().([]string)[:0]
	return &s
}

// PutStringSlice returns a string slice to the pool
func PutStringSlice(s *[]string) {
	if s == nil || cap(*s) > 100 { // Don't pool very large slices
		return
	}
	stringPool.Put(*s)
}

// DefaultKeyGenerator generates keys
func DefaultKeyGenerator() func() (string, error) {
	return graft.DefaultKeyGenerator()
}

// Helper functions for merge operations

// Merge merges two data structures
func Merge(dst, src interface{}) error {
	return graft.Merge(dst, src)
}

// DebugOn returns true if debugging is enabled
func DebugOn() bool {
	return graft.DebugOn()
}

// UseEnhancedParser returns true if enhanced parser is enabled
func UseEnhancedParser() bool {
	// Enhanced parser is now the default
	return true
}

// ParseOpcallEnhanced parses operator call with enhanced parser
func ParseOpcallEnhanced(phase OperatorPhase, src string, ev *Evaluator) (*graft.Opcall, error) {
	return graft.ParseOpcallEnhanced(phase, src, ev)
}

// Type aliases for error handling
type WarningError = graft.WarningError

// Constants
const OperatorCallCall = graft.OperatorCall

// Parser-related type aliases and functions
type OperatorInfo = parser.OperatorInfo
type Precedence = parser.Precedence
type Associativity = parser.Associativity

const (
	PrecedencePostfix        = parser.PrecedencePostfix
	PrecedenceOr             = parser.PrecedenceOr
	PrecedenceAnd            = parser.PrecedenceAnd
	PrecedenceEquality       = parser.PrecedenceEquality
	PrecedenceComparison     = parser.PrecedenceComparison
	PrecedenceAdditive       = parser.PrecedenceAddition
	PrecedenceMultiplicative = parser.PrecedenceMultiplication
	PrecedenceTernary        = parser.PrecedenceTernary
	PrecedenceUnary          = parser.PrecedenceUnary
	RightAssociative         = parser.RightAssociative
	LeftAssociative          = parser.LeftAssociative
)

func NewOperatorRegistry() *parser.OperatorRegistry {
	return parser.NewOperatorRegistry()
}

// Additional imports that operators commonly need
// These are provided here so operators don't need to import them individually
var (
	// tree package is used by all operators for Dependencies method
	_ = tree.Cursor{}
)
