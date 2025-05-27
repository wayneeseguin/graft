package parser

import (
	"fmt"
	"strings"
	
	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// ParseOpcallCompat provides backward compatibility while allowing enhanced parser usage
func ParseOpcallCompat(phase graft.OperatorPhase, src string) (*graft.Opcall, error) {
	// Only parse strings that look like operator expressions
	if !strings.HasPrefix(strings.TrimSpace(src), "((") || !strings.HasSuffix(strings.TrimSpace(src), "))") {
		log.DEBUG("ParseOpcallCompat: '%s' does not look like an operator expression", src)
		return nil, nil
	}
	
	log.DEBUG("ParseOpcallCompat: checking '%s' in phase %v", src, phase)
	
	// Check if we should use the enhanced parser
	if UseEnhancedParser || shouldUseEnhancedParser(src) {
		log.DEBUG("ParseOpcallCompat: Using enhanced parser for '%s'", src)
		// Try enhanced parser first
		result, err := parseOpcallWithEnhancedParser(phase, src)
		if err != nil {
			// If there's an error, fall back to original parser
			log.DEBUG("Enhanced parser error for '%s': %v, falling back to original", src, err)
			return ParseOpcall(phase, src)
		}
		if result != nil {
			log.DEBUG("Enhanced parser succeeded for '%s'", src)
			return result, nil
		}
		// If enhanced parser returned nil, fall back to original parser
		// This ensures backward compatibility during transition
		log.DEBUG("Enhanced parser returned nil for '%s', falling back to original", src)
	}
	
	// Use original parser
	log.DEBUG("ParseOpcallCompat: Using original parser for '%s'", src)
	return ParseOpcall(phase, src)
}

// shouldUseEnhancedParser determines if a given expression would benefit from the enhanced parser
func shouldUseEnhancedParser(src string) bool {
	// Use enhanced parser for expressions that contain nested operators
	// This is a heuristic based on common patterns
	
	// Strip outer (( )) if present for better pattern matching
	inner := strings.TrimSpace(src)
	if strings.HasPrefix(inner, "((") && strings.HasSuffix(inner, "))") {
		inner = strings.TrimSpace(inner[2:len(inner)-2])
	}
	
	// Check for nested operator patterns like "(grab foo.bar)"
	// Don't trigger on simple "vault || grab" patterns which work fine with the old parser
	if strings.Contains(inner, "(grab ") || strings.Contains(inner, "(concat ") {
		return true
	}
	if strings.Contains(inner, " (") && strings.Contains(inner, ")") {
		// Has parenthesized sub-expressions
		return true
	}
	
	// Check for arithmetic operators
	if strings.ContainsAny(inner, "+-*/") {
		return true
	}
	
	// Check for parentheses (grouping)
	if strings.Contains(inner, "(") && strings.Contains(inner, ")") {
		return true
	}
	
	return false
}

// WrapOperatorForNestedCalls wraps an operator to handle nested operator call expressions
// This allows existing operators to work with the enhanced parser without modification
func WrapOperatorForNestedCalls(op graft.Operator) graft.Operator {
	return &NestedCallWrapper{wrapped: op}
}

// NestedCallWrapper wraps an operator to evaluate nested calls in arguments
type NestedCallWrapper struct {
	wrapped graft.Operator
}

func (w *NestedCallWrapper) Setup() error {
	return w.wrapped.Setup()
}

func (w *NestedCallWrapper) Phase() graft.OperatorPhase {
	return w.wrapped.Phase()
}

func (w *NestedCallWrapper) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// For dependencies, we need to extract dependencies from nested calls too
	deps := w.wrapped.Dependencies(ev, args, locs, auto)
	
	// Add dependencies from nested operator calls
	for _, arg := range args {
		if arg.Type == OperatorCall {
			// Get the nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(*NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), locs, auto)
				deps = append(deps, nestedDeps...)
			}
		}
	}
	
	return deps
}

func (w *NestedCallWrapper) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	// Pre-process arguments to evaluate nested operator calls
	processedArgs := make([]*graft.Expr, len(args))
	
	for i, arg := range args {
		if arg.Type == graft.OperatorCall {
			// Evaluate the nested operator call
			resp, err := evaluateOperatorCall(arg, ev)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate nested operator call: %v", err)
			}
			// Replace with the result
			processedArgs[i] = &graft.Expr{
				Type:    graft.Literal,
				Literal: resp.Value,
			}
		} else {
			processedArgs[i] = arg
		}
	}
	
	// Call the wrapped operator with processed arguments
	return w.wrapped.Run(ev, processedArgs)
}

// MigrateArgify provides a migration path from the old argify to the enhanced parser
func MigrateArgify(phase OperatorPhase) func(string) ([]*Expr, error) {
	return func(src string) ([]*Expr, error) {
		if UseEnhancedParser || shouldUseEnhancedParser(src) {
			return argifyEnhanced(phase, src)
		}
		// This would call the original argify, but it's defined inside ParseOpcall
		// For now, return an error to indicate we need to refactor
		return nil, fmt.Errorf("argify migration not yet implemented")
	}
}