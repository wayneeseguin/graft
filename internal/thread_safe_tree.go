package internal

import (
	"fmt"
	"sync"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// SafeTree implements ThreadSafeTree interface with proper synchronization
type SafeTree struct {
	data map[interface{}]interface{}
	mu   sync.RWMutex
}

// NewSafeTree creates a new thread-safe tree
func NewSafeTree(data map[interface{}]interface{}) *SafeTree {
	if data == nil {
		data = make(map[interface{}]interface{})
	}
	return &SafeTree{
		data: data,
	}
}

// Find safely retrieves a value from the tree
func (st *SafeTree) Find(path ...string) (interface{}, error) {
	// Track lock wait time
	lockStart := time.Now()
	st.mu.RLock()
	lockWait := time.Since(lockStart)
	defer st.mu.RUnlock()

	// Record metrics if enabled
	if features := GetFeatures(); features.EnableMetrics {
		GetParallelMetrics().RecordLockWait(lockWait)
	}

	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	// Use the existing tree.Find function with proper path construction
	fullPath := path[0]
	for i := 1; i < len(path); i++ {
		fullPath += "." + path[i]
	}

	return tree.Find(st.data, fullPath)
}

// FindAll retrieves all values matching the path pattern
func (st *SafeTree) FindAll(path ...string) ([]interface{}, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	// For now, implement as single Find - can be enhanced later
	val, err := st.Find(path...)
	if err != nil {
		return nil, err
	}

	return []interface{}{val}, nil
}

// Exists checks if a path exists in the tree
func (st *SafeTree) Exists(path ...string) bool {
	_, err := st.Find(path...)
	return err == nil
}

// Copy creates a deep copy of the tree
func (st *SafeTree) Copy() ThreadSafeTree {
	st.mu.RLock()
	defer st.mu.RUnlock()

	copied := deepCopyTree(st.data)
	return NewSafeTree(copied.(map[interface{}]interface{}))
}

// Set safely sets a value at the given path
func (st *SafeTree) Set(value interface{}, path ...string) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	return st.setInternal(value, path...)
}

// setInternal performs the actual set operation (caller must hold lock)
func (st *SafeTree) setInternal(value interface{}, path ...string) error {
	current := st.data

	// Navigate to the parent of the final key
	for i := 0; i < len(path)-1; i++ {
		key := path[i]

		if next, ok := current[key]; ok {
			if nextMap, ok := next.(map[interface{}]interface{}); ok {
				current = nextMap
			} else {
				// Path exists but is not a map - create new map
				newMap := make(map[interface{}]interface{})
				current[key] = newMap
				current = newMap
			}
		} else {
			// Create new map for this path segment
			newMap := make(map[interface{}]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	// Set the final value
	finalKey := path[len(path)-1]
	current[finalKey] = value

	return nil
}

// Delete removes a value at the given path
func (st *SafeTree) Delete(path ...string) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	return st.deleteInternal(path...)
}

// deleteInternal performs the actual delete operation (caller must hold lock)
func (st *SafeTree) deleteInternal(path ...string) error {
	current := st.data

	// Navigate to the parent of the final key
	for i := 0; i < len(path)-1; i++ {
		key := path[i]

		if next, ok := current[key]; ok {
			if nextMap, ok := next.(map[interface{}]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("path not found: %v", path)
			}
		} else {
			return fmt.Errorf("path not found: %v", path)
		}
	}

	// Delete the final key
	finalKey := path[len(path)-1]
	delete(current, finalKey)

	return nil
}

// Replace replaces the entire tree or merges data
func (st *SafeTree) Replace(data map[string]interface{}) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Convert string keys to interface{} keys
	converted := make(map[interface{}]interface{})
	for k, v := range data {
		converted[k] = v
	}

	// Deep merge the new data
	st.data = deepMerge(st.data, converted)

	return nil
}

// Merge merges another ThreadSafeTree into this one
func (st *SafeTree) Merge(other ThreadSafeTree) error {
	if otherSafe, ok := other.(*SafeTree); ok {
		otherSafe.mu.RLock()
		defer otherSafe.mu.RUnlock()

		st.mu.Lock()
		defer st.mu.Unlock()

		st.data = deepMerge(st.data, otherSafe.data)
		return nil
	}

	return fmt.Errorf("cannot merge incompatible tree types")
}

// CompareAndSwap atomically compares and swaps a value
func (st *SafeTree) CompareAndSwap(oldValue, newValue interface{}, path ...string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Get current value
	current := st.data
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		if next, ok := current[key]; ok {
			if nextMap, ok := next.(map[interface{}]interface{}); ok {
				current = nextMap
			} else {
				return false // Path doesn't exist or isn't traversable
			}
		} else {
			return false // Path doesn't exist
		}
	}

	finalKey := path[len(path)-1]
	currentValue, exists := current[finalKey]

	// Compare values
	if !exists && oldValue != nil {
		return false
	}
	if exists && !deepEqual(currentValue, oldValue) {
		return false
	}

	// Swap the value
	current[finalKey] = newValue
	return true
}

// Update atomically updates a value using a function
func (st *SafeTree) Update(fn func(current interface{}) interface{}, path ...string) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Get current value
	fullPath := path[0]
	for i := 1; i < len(path); i++ {
		fullPath += "." + path[i]
	}

	currentValue, _ := tree.Find(st.data, fullPath)
	newValue := fn(currentValue)

	return st.setInternal(newValue, path...)
}

// Transaction executes a function within a transaction
func (st *SafeTree) Transaction(fn func(tx TreeTransaction) error) error {
	tx := &safeTreeTransaction{
		tree:    st,
		changes: make(map[string]interface{}),
		deletes: make(map[string]bool),
	}

	// Execute transaction function
	err := fn(tx)
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit()
}

// GetRawData returns the underlying data (for backward compatibility)
// WARNING: This breaks thread safety guarantees!
func (st *SafeTree) GetRawData() map[interface{}]interface{} {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return deepCopyTree(st.data).(map[interface{}]interface{})
}

// GetRawDataUnsafe returns the underlying data without copying
// WARNING: This breaks thread safety guarantees!
func (st *SafeTree) GetRawDataUnsafe() map[interface{}]interface{} {
	return st.data
}

// safeTreeTransaction implements TreeTransaction
type safeTreeTransaction struct {
	tree    *SafeTree
	changes map[string]interface{}
	deletes map[string]bool
	mu      sync.Mutex
}

// Get retrieves a value within the transaction
func (tx *safeTreeTransaction) Get(path ...string) (interface{}, error) {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	pathStr := joinPath(path)

	// Check if we have a pending change
	if value, ok := tx.changes[pathStr]; ok {
		return value, nil
	}

	// Check if it's deleted
	if tx.deletes[pathStr] {
		return nil, fmt.Errorf("path deleted in transaction: %v", path)
	}

	// Get from tree
	return tx.tree.Find(path...)
}

// Set sets a value within the transaction
func (tx *safeTreeTransaction) Set(value interface{}, path ...string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	pathStr := joinPath(path)
	tx.changes[pathStr] = value
	delete(tx.deletes, pathStr) // Remove from deletes if present

	return nil
}

// Delete marks a path for deletion within the transaction
func (tx *safeTreeTransaction) Delete(path ...string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	pathStr := joinPath(path)
	tx.deletes[pathStr] = true
	delete(tx.changes, pathStr) // Remove from changes if present

	return nil
}

// Rollback discards all transaction changes
func (tx *safeTreeTransaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	tx.changes = make(map[string]interface{})
	tx.deletes = make(map[string]bool)

	return nil
}

// Commit applies all transaction changes atomically
func (tx *safeTreeTransaction) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	tx.tree.mu.Lock()
	defer tx.tree.mu.Unlock()

	// Apply all changes
	for pathStr, value := range tx.changes {
		path := splitPath(pathStr)
		if err := tx.tree.setInternal(value, path...); err != nil {
			return err
		}
	}

	// Apply all deletes
	for pathStr := range tx.deletes {
		path := splitPath(pathStr)
		if err := tx.tree.deleteInternal(path...); err != nil {
			// Ignore delete errors for non-existent paths
		}
	}

	return nil
}

// Helper functions

func deepCopyTree(src interface{}) interface{} {
	switch s := src.(type) {
	case map[interface{}]interface{}:
		dst := make(map[interface{}]interface{})
		for k, v := range s {
			dst[k] = deepCopyTree(v)
		}
		return dst
	case map[string]interface{}:
		dst := make(map[string]interface{})
		for k, v := range s {
			dst[k] = deepCopyTree(v)
		}
		return dst
	case []interface{}:
		dst := make([]interface{}, len(s))
		for i, v := range s {
			dst[i] = deepCopyTree(v)
		}
		return dst
	default:
		return src
	}
}

func deepMerge(dst, src map[interface{}]interface{}) map[interface{}]interface{} {
	if dst == nil {
		return deepCopyTree(src).(map[interface{}]interface{})
	}

	result := deepCopyTree(dst).(map[interface{}]interface{})

	for k, v := range src {
		if dstVal, ok := result[k]; ok {
			if dstMap, ok := dstVal.(map[interface{}]interface{}); ok {
				if srcMap, ok := v.(map[interface{}]interface{}); ok {
					result[k] = deepMerge(dstMap, srcMap)
					continue
				}
			}
		}
		result[k] = deepCopyTree(v)
	}

	return result
}

func deepEqual(a, b interface{}) bool {
	// Simple deep equality check
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch aVal := a.(type) {
	case map[interface{}]interface{}:
		if bVal, ok := b.(map[interface{}]interface{}); ok {
			if len(aVal) != len(bVal) {
				return false
			}
			for k, v := range aVal {
				if bv, exists := bVal[k]; !exists || !deepEqual(v, bv) {
					return false
				}
			}
			return true
		}
	case []interface{}:
		if bVal, ok := b.([]interface{}); ok {
			if len(aVal) != len(bVal) {
				return false
			}
			for i, v := range aVal {
				if !deepEqual(v, bVal[i]) {
					return false
				}
			}
			return true
		}
	default:
		return a == b
	}

	return false
}

func joinPath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	result := path[0]
	for i := 1; i < len(path); i++ {
		result += "." + path[i]
	}
	return result
}

func splitPath(pathStr string) []string {
	if pathStr == "" {
		return []string{}
	}
	// Split on dots
	parts := []string{}
	current := ""
	for _, ch := range pathStr {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
