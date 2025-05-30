package graft

import (
	"fmt"
	"os"
	
	"gopkg.in/yaml.v2"
)

// EvaluateExpr evaluates an expression, including nested operator calls
func EvaluateExpr(e *Expr, ev *Evaluator) (*Response, error) {
	if e == nil {
		return nil, NewExprEvaluationError("nil expression", Position{})
	}
	
	switch e.Type {
	case Literal:
		return &Response{
			Type:  Replace,
			Value: e.Literal,
		}, nil
		
	case Reference:
		v, err := e.Reference.Resolve(ev.Tree)
		if err != nil {
			return nil, WrapError(err, ReferenceError, e.Pos).
				WithContext(fmt.Sprintf("while resolving reference '%s'", e.Reference))
		}
		return &Response{
			Type:  Replace,
			Value: v,
		}, nil
		
	case EnvVar:
		val := os.Getenv(e.Name)
		// Try to unmarshal the value as YAML if it's not empty
		// But only if it looks like it might be structured data (not a plain string)
		if val != "" && (val == "true" || val == "false" || val == "null" || 
			(len(val) > 0 && (val[0] == '{' || val[0] == '[' || val[0] == '-'))) {
			var unmarshalled interface{}
			if err := yaml.Unmarshal([]byte(val), &unmarshalled); err == nil {
				// Only use unmarshalled value if it's not a string
				// (to avoid changing multiline strings)
				if _, isString := unmarshalled.(string); !isString {
					return &Response{
						Type:  Replace,
						Value: unmarshalled,
					}, nil
				}
			}
		}
		// Return as string for plain strings or if unmarshalling fails
		return &Response{
			Type:  Replace,
			Value: val,
		}, nil
		
	case OperatorCall:
		// Evaluate nested operator call
		return evaluateOperatorCall(e, ev)
		
	case LogicalOr:
		// LogicalOr implements fallback behavior in Graft:
		// Try left side first, if it fails (error), try right side
		// This is NOT boolean OR - it returns the actual value, not true/false
		left, err := EvaluateExpr(e.Left, ev)
		if err == nil {
			// Left side succeeded, use its value
			return left, nil
		}
		// Left side failed, try right side
		return EvaluateExpr(e.Right, ev)
		
	default:
		return nil, NewExprEvaluationError(fmt.Sprintf("unknown expression type: %v", e.Type), e.Pos)
	}
}

// looksLikeBooleanExpr checks if an expression looks like it will evaluate to a boolean
func looksLikeBooleanExpr(e *Expr) bool {
	if e == nil {
		return false
	}
	
	switch e.Type {
	case Literal:
		// Check if literal is a boolean value
		_, isBool := e.Literal.(bool)
		return isBool
		
	case OperatorCall:
		// Check if it's a boolean operator
		op := e.Op()
		return op == "&&" || op == "==" || op == "!=" || op == "<" || op == ">" || op == "<=" || op == ">=" || op == "!"
		
	case LogicalOr:
		// Nested || that looks boolean
		return looksLikeBooleanExpr(e.Left) && looksLikeBooleanExpr(e.Right)
		
	default:
		return false
	}
}

// evaluateOperatorCall evaluates a nested operator call expression
func evaluateOperatorCall(e *Expr, ev *Evaluator) (*Response, error) {
	if e.Type != OperatorCall {
		return nil, NewExprEvaluationError("not an operator call expression", e.Pos)
	}
	
	// If we have a Call object, use it directly
	if e.Call != nil {
		return e.Call.Run(ev)
	}
	
	opName := e.Op()
	args := e.Args()
	
	// Get the operator
	op := OperatorFor(opName)
	if op == nil {
		return nil, NewExprOperatorError(fmt.Sprintf("unknown operator: %s", opName), e.Pos)
	}
	
	// Phase checking is handled elsewhere in the pipeline
	// The evaluator doesn't have direct access to the current phase
	
	// Evaluate operator arguments first (unless the operator handles raw expressions)
	evaluatedArgs := make([]*Expr, len(args))
	for i, arg := range args {
		// Some operators (like vault with ||) need raw expressions
		// For now, pass through LogicalOr expressions unchanged
		if arg.Type == LogicalOr {
			evaluatedArgs[i] = arg
		} else if arg.Type == OperatorCall {
			// Recursively evaluate nested operator calls
			resp, err := evaluateOperatorCall(arg, ev)
			if err != nil {
				return nil, WrapError(err, ExprEvaluationError, arg.Pos).
					WithContext(fmt.Sprintf("in nested operator '%s'", arg.Op()))
			}
			// Convert response back to expression
			evaluatedArgs[i] = &Expr{
				Type:    Literal,
				Literal: resp.Value,
			}
		} else {
			evaluatedArgs[i] = arg
		}
	}
	
	// Run the operator
	return op.Run(ev, evaluatedArgs)
}

// EvaluateOperatorArgs is a helper for operators that need to evaluate their arguments
// This handles nested operator calls and other expression types
func EvaluateOperatorArgs(ev *Evaluator, args []*Expr) ([]interface{}, error) {
	results := make([]interface{}, len(args))
	
	for i, arg := range args {
		resp, err := EvaluateExpr(arg, ev)
		if err != nil {
			return nil, err
		}
		results[i] = resp.Value
	}
	
	return results, nil
}