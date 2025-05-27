package operators

import (
	"fmt"
	"strings"
)

// vaultArgProcessor handles argument processing for vault operator with LogicalOr support
type vaultArgProcessor struct {
	args         []*Expr
	hasDefault   bool
	defaultExpr  *Expr
	defaultIndex int
}

// newVaultArgProcessor creates a processor that extracts defaults from any position
func newVaultArgProcessor(args []*Expr) *vaultArgProcessor {
	processor := &vaultArgProcessor{
		args:         make([]*Expr, len(args)),
		hasDefault:   false,
		defaultIndex: -1,
	}

	// Copy args and extract any LogicalOr
	for i, arg := range args {
		if arg.Type == LogicalOr {
			processor.hasDefault = true
			processor.defaultExpr = arg.Right
			processor.defaultIndex = i
			// Use the left side of LogicalOr for vault path
			processor.args[i] = arg.Left
		} else {
			processor.args[i] = arg
		}
	}

	return processor
}

// resolveToString resolves an expression and converts it to a string
func (p *vaultArgProcessor) resolveToString(ev *Evaluator, expr *Expr) (string, error) {
	// Use ResolveOperatorArgument to support nested expressions
	value, err := ResolveOperatorArgument(ev, expr)
	if err != nil {
		// Maintain backward compatibility with error messages
		if expr.Type == Reference {
			return "", fmt.Errorf("Unable to resolve `%s`: %s", expr.Reference, err)
		}
		return "", err
	}

	if value == nil {
		return "", fmt.Errorf("cannot use nil as vault path component")
	}

	// Convert resolved value to string with vault-specific error messages
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int64, float32, float64, bool:
		return fmt.Sprintf("%v", v), nil
	case map[interface{}]interface{}, map[string]interface{}:
		if expr.Type == Reference {
			return "", fmt.Errorf("$.%s is a map; only scalars are supported for vault paths", expr.Reference)
		}
		return "", fmt.Errorf("value is a map; only scalars are supported for vault paths")
	case []interface{}:
		if expr.Type == Reference {
			return "", fmt.Errorf("$.%s is a list; only scalars are supported for vault paths", expr.Reference)
		}
		return "", fmt.Errorf("value is a list; only scalars are supported for vault paths")
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// buildVaultPath resolves all arguments and concatenates them into a vault path
func (p *vaultArgProcessor) buildVaultPath(ev *Evaluator) (string, error) {
	parts := make([]string, 0, len(p.args))

	for i, arg := range p.args {
		DEBUG("  processing arg[%d] for concatenation", i)

		part, err := p.resolveToString(ev, arg)
		if err != nil {
			DEBUG("    failed to resolve arg[%d]: %s", i, err)
			return "", err
		}

		DEBUG("    resolved to: '%s'", part)
		parts = append(parts, part)
	}

	path := strings.Join(parts, "")
	DEBUG("  final concatenated path: '%s'", path)

	return path, nil
}

// evaluateDefault evaluates the default expression if one exists
func (p *vaultArgProcessor) evaluateDefault(ev *Evaluator) (interface{}, error) {
	if !p.hasDefault || p.defaultExpr == nil {
		return nil, fmt.Errorf("no default value available")
	}

	DEBUG("  evaluating default expression")
	// Use ResolveOperatorArgument to support nested expressions in defaults
	value, err := ResolveOperatorArgument(ev, p.defaultExpr)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate default value: %s", err)
	}

	return value, nil
}

// vaultError represents an error that may have a default fallback
type vaultError struct {
	Err        error
	HasDefault bool
	IsNotFound bool
}

func (e *vaultError) Error() string {
	return e.Err.Error()
}

// isVaultNotFound checks if an error indicates a missing secret
func isVaultNotFound(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "404") ||
		strings.Contains(errMsg, "secret not found")
}
