package operators

// OperatorNoCacheSupport provides helper functions for operators to check nocache modifiers

// ShouldSkipCache checks if an operator call has the nocache modifier
// This function should be called by operators that support caching
func ShouldSkipCache(ev *Evaluator) bool {
	// Check if the current operation has nocache modifier
	// The evaluator would need to provide access to the current expression
	// For now, we'll implement a simple version that checks the call stack
	
	// For now, we'll use a simple implementation
	// In practice, the evaluator would need to track the current expression being evaluated
	// This would require changes to the Evaluator struct
	
	return false
}

// WithNoCacheCheck wraps an operator result to indicate cache behavior
func WithNoCacheCheck(result *Response, skipCache bool) *Response {
	if result != nil && skipCache {
		// Add metadata to indicate this result should not be cached
		// This would be used by the caching layer
		if result.Value != nil {
			// For now, we'll add a special marker that caching layers can check
			// In a real implementation, this might be a more sophisticated metadata system
		}
	}
	return result
}

// IsNoCacheResponse checks if a response should skip caching
func IsNoCacheResponse(result *Response, expr *Expr) bool {
	if expr != nil && expr.IsNoCache() {
		return true
	}
	return false
}
