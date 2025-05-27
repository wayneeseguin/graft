package graft

import (
	"os"
	"strings"
	"github.com/wayneeseguin/graft/log"
)

// UseEnhancedParser is a global flag to enable the enhanced parser
var UseEnhancedParser = true

func init() {
	// Initialize based on default value, then check environment variables
	shouldEnableEnhanced := UseEnhancedParser // defaults to true
	
	// Check environment variables to override default
	if env := os.Getenv("GRAFT_LEGACY_PARSER"); env != "" {
		env = strings.ToLower(strings.TrimSpace(env))
		switch env {
		case "1", "true", "yes", "on":
			shouldEnableEnhanced = false
		case "0", "false", "no", "off":
			shouldEnableEnhanced = true
		}
	} else if env := os.Getenv("GRAFT_ENHANCED_PARSER"); env != "" {
		env = strings.ToLower(strings.TrimSpace(env))
		switch env {
		case "1", "true", "yes", "on":
			shouldEnableEnhanced = true
		case "0", "false", "no", "off":
			shouldEnableEnhanced = false
		}
	}
	
	// Apply the final decision
	if shouldEnableEnhanced {
		EnableEnhancedParser()
	} else {
		DisableEnhancedParser()
	}
}

// EnableEnhancedParser enables all enhanced parser features
func EnableEnhancedParser() {
	UseEnhancedParser = true
	
	// Enable enhanced operators
	EnableEnhancedConcat()
	EnableEnhancedGrab()
	EnableEnhancedJoin()
	EnableEnhancedBase64()
	EnableEnhancedBase64Decode()
	EnableEnhancedFile()
	EnableEnhancedKeys()
	EnableEnhancedStringify()
	EnableEnhancedNegate()
	EnableEnhancedEmpty()
	EnableEnhancedNull()
	EnableEnhancedParam()
	EnableEnhancedDefer()
	EnableEnhancedCalc()
	
	// TODO: Phase 2 - Register arithmetic operators
	// RegisterOp("+", AddOperator{})
	// RegisterOp("-", SubtractOperator{})
	// RegisterOp("*", MultiplyOperator{})
	// RegisterOp("/", DivideOperator{})
	// RegisterOp("%", ModuloOperator{})
	
	// TODO: Phase 2 - Register comparison operators
	// RegisterOp("==", ComparisonOperator{op: "=="})
	// RegisterOp("!=", ComparisonOperator{op: "!="})
	// RegisterOp("<", ComparisonOperator{op: "<"})
	// RegisterOp(">", ComparisonOperator{op: ">"})
	// RegisterOp("<=", ComparisonOperator{op: "<="})
	// RegisterOp(">=", ComparisonOperator{op: ">="})
	
	// TODO: Phase 2 - Register boolean operators
	// RegisterOp("&&", BooleanAndOperator{})
	// NOTE: || is not registered as an operator - it's handled specially as alternation in expressions
	// RegisterOp("!", NegationOperator{})
	
	// TODO: Phase 2 - Register ternary operator
	// RegisterOp("?:", TernaryOperator{})
	
	DEBUG("Enhanced parser enabled")
}

// EnableEnhancedGrab enables the enhanced grab operator
func EnableEnhancedGrab() {
	// TODO: Phase 2 - RegisterOp("grab", GrabOperatorEnhanced{})
}

// DisableEnhancedParser disables enhanced parser features
func DisableEnhancedParser() {
	UseEnhancedParser = false
	
	// TODO: Phase 2 - Restore original operators
	// RegisterOp("concat", ConcatOperator{})
	// RegisterOp("grab", GrabOperator{})
	// RegisterOp("join", JoinOperator{})
	// RegisterOp("base64", Base64Operator{})
	// RegisterOp("base64-decode", Base64DecodeOperator{})
	// RegisterOp("file", FileOperator{})
	// RegisterOp("keys", KeysOperator{})
	// RegisterOp("stringify", StringifyOperator{})
	// RegisterOp("negate", NegateOperator{})
	// RegisterOp("empty", EmptyOperator{})
	// Note: there's no original null operator, only enhanced
	// RegisterOp("param", ParamOperator{})
	// RegisterOp("defer", DeferOperator{})
	// RegisterOp("calc", CalcOperator{})
	
	// Unregister arithmetic operators (they don't have original versions)
	delete(OpRegistry, "+")
	delete(OpRegistry, "-")
	delete(OpRegistry, "*")
	delete(OpRegistry, "/")
	delete(OpRegistry, "%")
	
	// Unregister comparison operators
	delete(OpRegistry, "==")
	delete(OpRegistry, "!=")
	delete(OpRegistry, "<")
	delete(OpRegistry, ">")
	delete(OpRegistry, "<=")
	delete(OpRegistry, ">=")
	
	// Unregister boolean operators (except ! which is a standard operator)
	delete(OpRegistry, "&&")
	
	// Unregister ternary operator
	delete(OpRegistry, "?:")
	
	DEBUG("Enhanced parser disabled")
}

// Stub functions for Phase 1 - these will be implemented in later phases
func EnableEnhancedConcat() {}
func EnableEnhancedJoin() {}
func EnableEnhancedBase64() {}
func EnableEnhancedBase64Decode() {}
func EnableEnhancedFile() {}
func EnableEnhancedKeys() {}
func EnableEnhancedStringify() {}
func EnableEnhancedNegate() {}
func EnableEnhancedEmpty() {}
func EnableEnhancedNull() {}
func EnableEnhancedParam() {}
func EnableEnhancedDefer() {}
func EnableEnhancedCalc() {}

// RegisterOp is a helper function to register operators
func RegisterOp(name string, op Operator) {
	OpRegistry[name] = op
}

// Stub operator types for Phase 1
type GrabOperatorEnhanced struct{}
type ConcatOperator struct{}
type GrabOperator struct{}
type JoinOperator struct{}
type Base64Operator struct{}
type Base64DecodeOperator struct{}
type FileOperator struct{}
type KeysOperator struct{}
type StringifyOperator struct{}
type NegateOperator struct{}
type EmptyOperator struct{}
type ParamOperator struct{}
type DeferOperator struct{}
type CalcOperator struct{}
type AddOperator struct{}
type SubtractOperator struct{}
type MultiplyOperator struct{}
type DivideOperator struct{}
type ModuloOperator struct{}
type ComparisonOperator struct{ op string }
type BooleanAndOperator struct{}
type NegationOperator struct{}
type TernaryOperator struct{}

// DEBUG is a helper function for debug logging
func DEBUG(format string, args ...interface{}) {
	log.DEBUG(format, args...)
}

