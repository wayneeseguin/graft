package spruce

import (
	"fmt"
	"strings"

	. "github.com/geofffranks/spruce/log"
	"github.com/starkandwayne/goutils/tree"
)

// VaultTryOperator provides a way to try multiple vault paths with a fallback default
// Syntax: (( vault-try "secret/prod:password" "secret/dev:password" "default-password" ))
// The last argument is always treated as the default value
type VaultTryOperator struct{}

// Setup initializes the operator
func (VaultTryOperator) Setup() error {
	return nil
}

// Phase identifies when this operator runs - during evaluation
func (VaultTryOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns the dependencies for this operator
func (VaultTryOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run executes the vault-try operator
func (o VaultTryOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( vault-try ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( vault-try ... )) operation at $.%s\n", ev.Here)

	// Minimum 2 arguments: at least one vault path and a default
	if len(args) < 2 {
		return nil, fmt.Errorf("vault-try operator requires at least 2 arguments (one or more vault paths, followed by a default value)")
	}

	// The last argument is always the default
	vaultPaths := args[:len(args)-1]
	defaultExpr := args[len(args)-1]

	// Try each vault path in order
	for i, pathExpr := range vaultPaths {
		DEBUG("vault-try: attempting path %d of %d", i+1, len(vaultPaths))
		
		response, err := o.tryVaultPath(ev, pathExpr)
		if err == nil {
			// Success! Return the value
			DEBUG("vault-try: path %d succeeded", i+1)
			return response, nil
		}
		
		// Log the error but continue to next path
		DEBUG("vault-try: path %d failed: %s", i+1, err)
		
		// If it's not a "not found" error on the last path, we might want to warn
		if i == len(vaultPaths)-1 && !isVaultNotFound(err) {
			DEBUG("vault-try: last path failed with non-404 error: %s", err)
		}
	}

	// All vault paths failed, use the default value
	DEBUG("vault-try: all paths failed, evaluating default value")
	defaultValue, err := ResolveOperatorArgument(ev, defaultExpr)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate default value: %s", err)
	}

	return &Response{
		Type:  Replace,
		Value: defaultValue,
	}, nil
}

// tryVaultPath attempts to retrieve a secret from a single vault path expression
func (o VaultTryOperator) tryVaultPath(ev *Evaluator, pathExpr *Expr) (*Response, error) {
	// Resolve the path expression to a string
	path, err := o.resolvePathExpression(ev, pathExpr)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve vault path: %s", err)
	}

	// Validate the path format
	if err := o.validateVaultPath(path); err != nil {
		return nil, err
	}

	// Track this vault reference
	if refs, found := VaultRefs[path]; !found {
		VaultRefs[path] = []string{ev.Here.String()}
	} else {
		VaultRefs[path] = append(refs, ev.Here.String())
	}

	// Use the shared vault infrastructure
	vaultOp := VaultOperator{}
	secret, err := vaultOp.performVaultLookup(path)
	if err != nil {
		return nil, err
	}

	return &Response{
		Type:  Replace,
		Value: secret,
	}, nil
}

// resolvePathExpression resolves an expression to a vault path string
func (o VaultTryOperator) resolvePathExpression(ev *Evaluator, expr *Expr) (string, error) {
	// Handle nested operator calls
	if expr.Type == OperatorCall {
		val, err := ResolveOperatorArgument(ev, expr)
		if err != nil {
			return "", err
		}
		if val == nil {
			return "", fmt.Errorf("vault path cannot be nil")
		}
		path := fmt.Sprintf("%v", val)
		if path == "" {
			return "", fmt.Errorf("vault path cannot be empty")
		}
		return path, nil
	}
	
	resolved, err := expr.Resolve(ev.Tree)
	if err != nil {
		return "", err
	}

	switch resolved.Type {
	case Literal:
		if resolved.Literal == nil {
			return "", fmt.Errorf("vault path cannot be nil")
		}
		path := fmt.Sprintf("%v", resolved.Literal)
		if path == "" {
			return "", fmt.Errorf("vault path cannot be empty")
		}
		return path, nil

	case Reference:
		value, err := resolved.Reference.Resolve(ev.Tree)
		if err != nil {
			return "", fmt.Errorf("unable to resolve reference %s: %s", resolved.Reference, err)
		}
		
		// Ensure it's a string scalar
		switch v := value.(type) {
		case string:
			if v == "" {
				return "", fmt.Errorf("vault path cannot be empty")
			}
			return v, nil
		case int, int64, float64, bool:
			return fmt.Sprintf("%v", v), nil
		default:
			return "", fmt.Errorf("vault path must be a string scalar, not %T", v)
		}

	default:
		return "", fmt.Errorf("vault-try expects string literals or references to strings")
	}
}

// validateVaultPath checks if a path looks like a valid vault path
func (o VaultTryOperator) validateVaultPath(path string) error {
	if !strings.Contains(path, ":") {
		return fmt.Errorf("invalid vault path '%s': must be in the form 'path/to/secret:key'", path)
	}
	
	parts := strings.Split(path, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid vault path '%s': must be in the form 'path/to/secret:key'", path)
	}
	
	return nil
}

func init() {
	RegisterOp("vault-try", VaultTryOperator{})
}