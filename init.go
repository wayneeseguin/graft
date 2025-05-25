package spruce

import (
	"os"
	"strings"
	
	. "github.com/geofffranks/spruce/log"
)

func init() {
	// Initialize based on default value, then check environment variables
	shouldEnableEnhanced := UseEnhancedParser // defaults to true from parser_integration.go
	
	// Check environment variables to override default
	if env := os.Getenv("SPRUCE_LEGACY_PARSER"); env != "" {
		env = strings.ToLower(strings.TrimSpace(env))
		switch env {
		case "1", "true", "yes", "on":
			shouldEnableEnhanced = false
		case "0", "false", "no", "off":
			shouldEnableEnhanced = true
		}
	} else if env := os.Getenv("SPRUCE_ENHANCED_PARSER"); env != "" {
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
	
	// Register arithmetic operators
	RegisterOp("+", AddOperator{})
	RegisterOp("-", SubtractOperator{})
	RegisterOp("*", MultiplyOperator{})
	RegisterOp("/", DivideOperator{})
	RegisterOp("%", ModuloOperator{})
	
	// Register comparison operators
	RegisterOp("==", ComparisonOperator{op: "=="})
	RegisterOp("!=", ComparisonOperator{op: "!="})
	RegisterOp("<", ComparisonOperator{op: "<"})
	RegisterOp(">", ComparisonOperator{op: ">"})
	RegisterOp("<=", ComparisonOperator{op: "<="})
	RegisterOp(">=", ComparisonOperator{op: ">="})
	
	// Register boolean operators
	RegisterOp("&&", BooleanAndOperator{})
	// NOTE: || is not registered as an operator - it's handled specially as alternation in expressions
	RegisterOp("!", NegationOperator{})
	
	// Register ternary operator
	RegisterOp("?:", TernaryOperator{})
	
	DEBUG("Enhanced parser enabled")
}

// EnableEnhancedGrab enables the enhanced grab operator
func EnableEnhancedGrab() {
	RegisterOp("grab", GrabOperatorEnhanced{})
}

// DisableEnhancedParser disables enhanced parser features
func DisableEnhancedParser() {
	UseEnhancedParser = false
	
	// Restore original operators
	RegisterOp("concat", ConcatOperator{})
	RegisterOp("grab", GrabOperator{})
	RegisterOp("join", JoinOperator{})
	RegisterOp("base64", Base64Operator{})
	RegisterOp("base64-decode", Base64DecodeOperator{})
	RegisterOp("file", FileOperator{})
	RegisterOp("keys", KeysOperator{})
	RegisterOp("stringify", StringifyOperator{})
	RegisterOp("negate", NegateOperator{})
	RegisterOp("empty", EmptyOperator{})
	// Note: there's no original null operator, only enhanced
	RegisterOp("param", ParamOperator{})
	RegisterOp("defer", DeferOperator{})
	RegisterOp("calc", CalcOperator{})
	
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

