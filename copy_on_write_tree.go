package graft

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// COWNode represents a node in a Copy-on-Write tree
type COWNode struct {
	value    interface{}
	children map[interface{}]*COWNode
	version  int64  // Version for optimistic locking
	isShared int32  // Atomic flag indicating if this node is shared
}

// NewCOWNode creates a new Copy-on-Write node
func NewCOWNode(value interface{}) *COWNode {
	return &COWNode{
		value:    value,
		children: make(map[interface{}]*COWNode),
		version:  0,
		isShared: 0,
	}
}

// markShared atomically marks this node as shared
func (node *COWNode) markShared() {
	atomic.StoreInt32(&node.isShared, 1)
}

// isNodeShared checks if this node is shared
func (node *COWNode) isNodeShared() bool {
	return atomic.LoadInt32(&node.isShared) == 1
}

// clone creates a copy of the node for modification
func (node *COWNode) clone() *COWNode {
	cloned := &COWNode{
		children: make(map[interface{}]*COWNode),
		version:  atomic.AddInt64(&node.version, 1),
		isShared: 0,
	}
	
	// Deep copy the value if it's a map
	switch v := node.value.(type) {
	case map[interface{}]interface{}:
		newMap := make(map[interface{}]interface{})
		for k, val := range v {
			newMap[k] = val
		}
		cloned.value = newMap
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, val := range v {
			newMap[k] = val
		}
		cloned.value = newMap
	default:
		cloned.value = node.value
	}
	
	// Shallow copy of children - they'll be cloned on write
	for k, child := range node.children {
		cloned.children[k] = child
		child.markShared()
	}
	
	return cloned
}

// COWTree implements ThreadSafeTree using Copy-on-Write semantics
type COWTree struct {
	root     *COWNode
	mu       sync.RWMutex
	version  int64
}

// NewCOWTree creates a new Copy-on-Write tree
func NewCOWTree(data map[interface{}]interface{}) *COWTree {
	tree := &COWTree{
		root:    NewCOWNode(nil),
		version: 0,
	}
	
	if data != nil {
		tree.buildFromData(data)
	}
	
	return tree
}

// buildFromData constructs the COW tree from a map
func (cow *COWTree) buildFromData(data map[interface{}]interface{}) {
	cow.mu.Lock()
	defer cow.mu.Unlock()
	
	cow.root = cow.buildNodeFromValue(data)
}

// buildNodeFromValue recursively builds COW nodes from values
func (cow *COWTree) buildNodeFromValue(value interface{}) *COWNode {
	node := NewCOWNode(value)
	
	switch v := value.(type) {
	case map[interface{}]interface{}:
		for k, child := range v {
			node.children[k] = cow.buildNodeFromValue(child)
		}
	case map[string]interface{}:
		for k, child := range v {
			node.children[k] = cow.buildNodeFromValue(child)
		}
	case []interface{}:
		for i, child := range v {
			node.children[i] = cow.buildNodeFromValue(child)
		}
	}
	
	return node
}

// Find safely retrieves a value from the tree
func (cow *COWTree) Find(path ...string) (interface{}, error) {
	cow.mu.RLock()
	defer cow.mu.RUnlock()
	
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	
	return cow.findInNode(cow.root, path...)
}

// findInNode searches for a value in a COW node
func (cow *COWTree) findInNode(node *COWNode, path ...string) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("path not found: %v", path)
	}
	
	if len(path) == 0 {
		return node.value, nil
	}
	
	key := path[0]
	
	// Check if node has the key in its value map
	switch v := node.value.(type) {
	case map[interface{}]interface{}:
		if len(path) == 1 {
			if val, exists := v[key]; exists {
				return val, nil
			}
		}
	case map[string]interface{}:
		if len(path) == 1 {
			if val, exists := v[key]; exists {
				return val, nil
			}
		}
	}
	
	// Look in children
	if childNode, exists := node.children[key]; exists {
		return cow.findInNode(childNode, path[1:]...)
	}
	
	return nil, fmt.Errorf("path not found: %v", path)
}

// FindAll retrieves all values matching the path pattern
func (cow *COWTree) FindAll(path ...string) ([]interface{}, error) {
	value, err := cow.Find(path...)
	if err != nil {
		return nil, err
	}
	return []interface{}{value}, nil
}

// Exists checks if a path exists in the tree
func (cow *COWTree) Exists(path ...string) bool {
	_, err := cow.Find(path...)
	return err == nil
}

// Copy creates a shallow copy of the tree using COW semantics
func (cow *COWTree) Copy() ThreadSafeTree {
	cow.mu.RLock()
	defer cow.mu.RUnlock()
	
	return cow.copyInternal()
}

// copyInternal creates a copy without locking (for use when lock is already held)
func (cow *COWTree) copyInternal() *COWTree {
	// Mark the current root as shared
	cow.root.markShared()
	
	// Create a new tree sharing the same root
	newTree := &COWTree{
		root:    cow.root,
		version: cow.version,
	}
	
	return newTree
}

// Set safely sets a value at the given path
func (cow *COWTree) Set(value interface{}, path ...string) error {
	cow.mu.Lock()
	defer cow.mu.Unlock()
	
	return cow.setInternal(value, path...)
}

// setInternal sets a value without locking (for use when lock is already held)
func (cow *COWTree) setInternal(value interface{}, path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}
	
	// Clone the root if it's shared
	if cow.root.isNodeShared() {
		cow.root = cow.root.clone()
	}
	
	err := cow.setInternalNode(cow.root, value, path...)
	if err == nil {
		atomic.AddInt64(&cow.version, 1)
	}
	return err
}

// setInternal performs the actual set operation with COW semantics
func (cow *COWTree) setInternalNode(node *COWNode, value interface{}, path ...string) error {
	if len(path) == 1 {
		// Base case: set the value
		key := path[0]
		
		// Ensure node has a map value
		if node.value == nil {
			node.value = make(map[interface{}]interface{})
		}
		
		// Convert value to map if needed
		var nodeMap map[interface{}]interface{}
		switch v := node.value.(type) {
		case map[interface{}]interface{}:
			nodeMap = v
		case map[string]interface{}:
			// Convert to interface{} map
			nodeMap = make(map[interface{}]interface{})
			for k, val := range v {
				nodeMap[k] = val
			}
			node.value = nodeMap
		default:
			// Replace with new map
			nodeMap = make(map[interface{}]interface{})
			node.value = nodeMap
		}
		
		// Set the value
		nodeMap[key] = value
		
		// Create or update child node
		if childNode, exists := node.children[key]; exists {
			if childNode.isNodeShared() {
				node.children[key] = childNode.clone()
			}
			node.children[key].value = value
		} else {
			node.children[key] = NewCOWNode(value)
		}
		
		return nil
	}
	
	// Recursive case: navigate deeper
	key := path[0]
	remainingPath := path[1:]
	
	var childNode *COWNode
	if existing, exists := node.children[key]; exists {
		if existing.isNodeShared() {
			childNode = existing.clone()
			node.children[key] = childNode
		} else {
			childNode = existing
		}
	} else {
		// Create new intermediate node
		childNode = NewCOWNode(make(map[interface{}]interface{}))
		node.children[key] = childNode
		
		// Update parent's value map
		if node.value == nil {
			node.value = make(map[interface{}]interface{})
		}
		if nodeMap, ok := node.value.(map[interface{}]interface{}); ok {
			nodeMap[key] = childNode.value
		}
	}
	
	return cow.setInternalNode(childNode, value, remainingPath...)
}

// Delete removes a value at the given path
func (cow *COWTree) Delete(path ...string) error {
	cow.mu.Lock()
	defer cow.mu.Unlock()
	
	return cow.deleteInternal(path...)
}

// deleteInternal deletes without locking (for use when lock is already held)
func (cow *COWTree) deleteInternal(path ...string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}
	
	// Clone the root if it's shared
	if cow.root.isNodeShared() {
		cow.root = cow.root.clone()
	}
	
	err := cow.deleteInternalNode(cow.root, path...)
	if err == nil {
		atomic.AddInt64(&cow.version, 1)
	}
	return err
}

// deleteInternal performs the actual delete operation with COW semantics
func (cow *COWTree) deleteInternalNode(node *COWNode, path ...string) error {
	if len(path) == 1 {
		// Base case: delete the key
		key := path[0]
		
		// Remove from children
		delete(node.children, key)
		
		// Remove from value map
		if nodeMap, ok := node.value.(map[interface{}]interface{}); ok {
			delete(nodeMap, key)
		}
		
		return nil
	}
	
	// Recursive case
	key := path[0]
	remainingPath := path[1:]
	
	childNode, exists := node.children[key]
	if !exists {
		return fmt.Errorf("path not found: %v", path)
	}
	
	// Clone child if shared
	if childNode.isNodeShared() {
		childNode = childNode.clone()
		node.children[key] = childNode
	}
	
	return cow.deleteInternalNode(childNode, remainingPath...)
}

// Replace replaces the entire tree data
func (cow *COWTree) Replace(data map[string]interface{}) error {
	cow.mu.Lock()
	defer cow.mu.Unlock()
	
	// Convert to interface{} map
	converted := make(map[interface{}]interface{})
	for k, v := range data {
		converted[k] = v
	}
	
	// Merge with existing data
	existingData := cow.toMapInterface()
	mergedData := deepMerge(existingData, converted)
	
	// Rebuild tree
	cow.root = cow.buildNodeFromValue(mergedData)
	atomic.AddInt64(&cow.version, 1)
	
	return nil
}

// Merge merges another ThreadSafeTree into this one
func (cow *COWTree) Merge(other ThreadSafeTree) error {
	if otherCOW, ok := other.(*COWTree); ok {
		cow.mu.Lock()
		defer cow.mu.Unlock()
		
		otherCOW.mu.RLock()
		defer otherCOW.mu.RUnlock()
		
		// Convert both trees to maps and merge
		thisData := cow.toMapInterface()
		otherData := otherCOW.toMapInterface()
		mergedData := deepMerge(thisData, otherData)
		
		// Rebuild tree
		cow.root = cow.buildNodeFromValue(mergedData)
		atomic.AddInt64(&cow.version, 1)
		
		return nil
	}
	
	return fmt.Errorf("cannot merge incompatible tree types")
}

// CompareAndSwap atomically compares and swaps a value
func (cow *COWTree) CompareAndSwap(oldValue, newValue interface{}, path ...string) bool {
	cow.mu.Lock()
	defer cow.mu.Unlock()
	
	// Get current value
	currentValue, err := cow.Find(path...)
	if err != nil && oldValue != nil {
		return false
	}
	
	// Compare values
	if !deepEqual(currentValue, oldValue) {
		return false
	}
	
	// Perform the swap
	err = cow.Set(newValue, path...)
	return err == nil
}

// Update atomically updates a value using a function
func (cow *COWTree) Update(fn func(current interface{}) interface{}, path ...string) error {
	cow.mu.Lock()
	defer cow.mu.Unlock()
	
	currentValue, _ := cow.Find(path...)
	newValue := fn(currentValue)
	
	return cow.Set(newValue, path...)
}

// Transaction executes a function within a transaction
func (cow *COWTree) Transaction(fn func(tx TreeTransaction) error) error {
	cow.mu.Lock()
	defer cow.mu.Unlock()
	
	// Create a transaction that operates on a copy
	tx := &cowTreeTransaction{
		tree:     cow,
		snapshot: cow.copyInternal(),
		changes:  make(map[string]interface{}),
		deletes:  make(map[string]bool),
	}
	
	// Execute transaction function
	err := fn(tx)
	if err != nil {
		return err
	}
	
	// Commit the transaction
	return tx.commitInternal()
}

// toMapInterface converts the COW tree back to a map[interface{}]interface{}
func (cow *COWTree) toMapInterface() map[interface{}]interface{} {
	if cow.root == nil {
		return make(map[interface{}]interface{})
	}
	
	return cow.nodeToMap(cow.root)
}

// nodeToMap recursively converts a COW node to a map
func (cow *COWTree) nodeToMap(node *COWNode) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})
	
	switch v := node.value.(type) {
	case map[interface{}]interface{}:
		for k, val := range v {
			if childNode, exists := node.children[k]; exists {
				if childNode.value != nil {
					switch childNode.value.(type) {
					case map[interface{}]interface{}, map[string]interface{}:
						result[k] = cow.nodeToMap(childNode)
					default:
						result[k] = childNode.value
					}
				} else {
					result[k] = val
				}
			} else {
				result[k] = val
			}
		}
	case map[string]interface{}:
		for k, val := range v {
			if childNode, exists := node.children[k]; exists {
				if childNode.value != nil {
					switch childNode.value.(type) {
					case map[interface{}]interface{}, map[string]interface{}:
						result[k] = cow.nodeToMap(childNode)
					default:
						result[k] = childNode.value
					}
				} else {
					result[k] = val
				}
			} else {
				result[k] = val
			}
		}
	}
	
	return result
}

// GetVersion returns the current version of the tree
func (cow *COWTree) GetVersion() int64 {
	cow.mu.RLock()
	defer cow.mu.RUnlock()
	return cow.version
}

// cowTreeTransaction implements TreeTransaction for COW trees
type cowTreeTransaction struct {
	tree     *COWTree
	snapshot *COWTree
	changes  map[string]interface{}
	deletes  map[string]bool
	mu       sync.Mutex
}

// Get retrieves a value within the transaction
func (tx *cowTreeTransaction) Get(path ...string) (interface{}, error) {
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
	
	// Get from snapshot
	return tx.snapshot.Find(path...)
}

// Set sets a value within the transaction
func (tx *cowTreeTransaction) Set(value interface{}, path ...string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	
	pathStr := joinPath(path)
	tx.changes[pathStr] = value
	delete(tx.deletes, pathStr)
	
	return nil
}

// Delete marks a path for deletion within the transaction
func (tx *cowTreeTransaction) Delete(path ...string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	
	pathStr := joinPath(path)
	tx.deletes[pathStr] = true
	delete(tx.changes, pathStr)
	
	return nil
}

// Rollback discards all transaction changes
func (tx *cowTreeTransaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	
	tx.changes = make(map[string]interface{})
	tx.deletes = make(map[string]bool)
	
	return nil
}

// Commit applies all transaction changes
func (tx *cowTreeTransaction) Commit() error {
	return tx.commitInternal()
}

// commitInternal applies changes (called by Transaction method)
func (tx *cowTreeTransaction) commitInternal() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	
	// Apply all changes to the original tree
	for pathStr, value := range tx.changes {
		path := splitPath(pathStr)
		if err := tx.tree.setInternal(value, path...); err != nil {
			return err
		}
	}
	
	// Apply all deletes
	for pathStr := range tx.deletes {
		path := splitPath(pathStr)
		tx.tree.deleteInternal(path...) // Ignore errors for non-existent paths
	}
	
	return nil
}

