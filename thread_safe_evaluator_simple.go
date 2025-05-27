package spruce

import (
	"context"
	"fmt"
	"sync"
)

// ThreadSafeEvaluatorSimple is a simplified thread-safe evaluator
type ThreadSafeEvaluatorSimple struct {
	safeTree ThreadSafeTree
	mu       sync.RWMutex
}

// NewThreadSafeEvaluatorSimple creates a simplified thread-safe evaluator
func NewThreadSafeEvaluatorSimple(tree ThreadSafeTree) *ThreadSafeEvaluatorSimple {
	return &ThreadSafeEvaluatorSimple{
		safeTree: tree,
	}
}

// Evaluate performs basic thread-safe evaluation
func (tse *ThreadSafeEvaluatorSimple) Evaluate(ctx context.Context) error {
	tse.mu.Lock()
	defer tse.mu.Unlock()
	
	// For now, this is a placeholder that demonstrates thread safety
	// In Phase 4-5, we'll implement the full evaluation logic
	
	// Simulate some evaluation work
	return nil
}

// GetTree returns the underlying thread-safe tree
func (tse *ThreadSafeEvaluatorSimple) GetTree() ThreadSafeTree {
	tse.mu.RLock()
	defer tse.mu.RUnlock()
	return tse.safeTree
}

// SetValue safely sets a value in the tree
func (tse *ThreadSafeEvaluatorSimple) SetValue(value interface{}, path ...string) error {
	return tse.safeTree.Set(value, path...)
}

// GetValue safely gets a value from the tree
func (tse *ThreadSafeEvaluatorSimple) GetValue(path ...string) (interface{}, error) {
	return tse.safeTree.Find(path...)
}

// MigrationHelperSimple provides basic migration utilities
type MigrationHelperSimple struct {
	originalData map[interface{}]interface{}
	safeTree     ThreadSafeTree
}

// NewMigrationHelperSimple creates a simple migration helper
func NewMigrationHelperSimple(data map[interface{}]interface{}) *MigrationHelperSimple {
	safeTree := NewSafeTree(data)
	return &MigrationHelperSimple{
		originalData: data,
		safeTree:     safeTree,
	}
}

// GetThreadSafeTree returns the thread-safe tree
func (mh *MigrationHelperSimple) GetThreadSafeTree() ThreadSafeTree {
	return mh.safeTree
}

// GetThreadSafeEvaluator returns a simple thread-safe evaluator
func (mh *MigrationHelperSimple) GetThreadSafeEvaluator() *ThreadSafeEvaluatorSimple {
	return NewThreadSafeEvaluatorSimple(mh.safeTree)
}

// UpdateFromEvaluator updates the safe tree with data from a traditional evaluator
func (mh *MigrationHelperSimple) UpdateFromEvaluator(ev *Evaluator) error {
	// Convert evaluator tree to safe tree format
	data := make(map[string]interface{})
	for k, v := range ev.Tree {
		if keyStr, ok := k.(string); ok {
			data[keyStr] = v
		}
	}
	
	return mh.safeTree.Replace(data)
}

// ExportToEvaluator creates a traditional evaluator with current safe tree data
func (mh *MigrationHelperSimple) ExportToEvaluator() (*Evaluator, error) {
	if safeTree, ok := mh.safeTree.(*SafeTree); ok {
		rawData := safeTree.GetRawData()
		
		ev := &Evaluator{
			Tree:     rawData,
			SkipEval: false,
			CheckOps: make([]*Opcall, 0),
			Only:     []string{},
		}
		
		return ev, nil
	}
	
	return nil, fmt.Errorf("unsupported tree type for export")
}