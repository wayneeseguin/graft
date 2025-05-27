package internal

import (
	"github.com/wayneeseguin/graft/pkg/graft"
)
import (
	"bytes"
	"sync"
)

// BufferPool provides a pool of reusable bytes.Buffer instances
// to reduce allocations during string operations
var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer retrieves a buffer from the pool, resetting it for use
func GetBuffer() *bytes.Buffer {
	buf := BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the pool for reuse
func PutBuffer(buf *bytes.Buffer) {
	// Only return reasonably sized buffers to the pool
	// to avoid holding onto excessively large buffers
	if buf.Cap() <= 1024*1024 { // 1MB limit
		BufferPool.Put(buf)
	}
}

// StringBuilderPool provides a pool of string slices for building strings
// This is useful for operators like concat and join that build string arrays
var StringSlicePool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate with reasonable capacity
		slice := make([]string, 0, 16)
		return &slice
	},
}

// GetStringSlice retrieves a string slice from the pool
func GetStringSlice() *[]string {
	slice := StringSlicePool.Get().(*[]string)
	*slice = (*slice)[:0] // Reset length but keep capacity
	return slice
}

// PutStringSlice returns a string slice to the pool
func PutStringSlice(slice *[]string) {
	// Only return reasonably sized slices
	if cap(*slice) <= 1024 {
		StringSlicePool.Put(slice)
	}
}

// TokenPool provides a pool of reusable Token instances
type TokenPool struct {
	pool sync.Pool
}

// Global token pool instance
var tokenPool = &TokenPool{
	pool: sync.Pool{
		New: func() interface{} {
			return &Token{}
		},
	},
}

// GetToken retrieves a token from the pool
func GetToken() *Token {
	token := tokenPool.pool.Get().(*Token)
	token.Reset()
	return token
}

// PutToken returns a token to the pool
func PutToken(token *Token) {
	tokenPool.pool.Put(token)
}

// Reset clears a token for reuse
func (t *Token) Reset() {
	t.Type = 0
	t.Value = ""
	t.Pos = 0
	t.Line = 0
	t.Col = 0
}

// TODO: ASTNodePool - implement when we create a proper AST structure
// Currently, the parser uses Expr directly without an intermediate AST
// This will be implemented in a future optimization phase

// StringInterner provides string deduplication for common strings
type StringInterner struct {
	mu     sync.RWMutex
	intern map[string]string
}

// Global string interner for operator names and common literals
var globalInterner = &StringInterner{
	intern: make(map[string]string, 256),
}

// InternString returns an interned version of the string
func InternString(s string) string {
	return globalInterner.Intern(s)
}

// Intern returns a deduplicated version of the string
func (si *StringInterner) Intern(s string) string {
	// Fast path: check if already interned
	si.mu.RLock()
	if interned, ok := si.intern[s]; ok {
		si.mu.RUnlock()
		return interned
	}
	si.mu.RUnlock()
	
	// Slow path: intern the string
	si.mu.Lock()
	defer si.mu.Unlock()
	
	// Double-check in case another goroutine interned it
	if interned, ok := si.intern[s]; ok {
		return interned
	}
	
	si.intern[s] = s
	return s
}

// PreInternCommonStrings pre-populates the interner with common strings
func PreInternCommonStrings() {
	// Operator names
	operators := []string{
		"grab", "concat", "vault", "static_ips", "calc", "defer",
		"join", "keys", "sort", "prune", "param", "inject",
		"file", "base64", "empty", "load", "stringify", "null",
		"ips", "cartesian-product", "shuffle", "awsparam", "awssecret",
		"vault-try", "ternary", "negate", "base64-decode",
	}
	
	// Common literals
	literals := []string{
		"true", "false", "null", "nil", "",
		"name", "type", "value", "key", "path",
		"0", "1", "-1",
	}
	
	// Boolean operators
	boolOps := []string{
		"and", "or", "not", "&&", "||", "!",
	}
	
	// Intern all common strings
	for _, s := range operators {
		InternString(s)
	}
	for _, s := range literals {
		InternString(s)
	}
	for _, s := range boolOps {
		InternString(s)
	}
}

// Initialize the string interner with common strings at startup
func init() {
	PreInternCommonStrings()
}