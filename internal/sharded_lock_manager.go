package internal

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// ShardedLockManager provides fine-grained locking with multiple shards
type ShardedLockManager struct {
	shards    []shard
	numShards uint32
}

type shard struct {
	mu sync.RWMutex
}

// NewShardedLockManager creates a new sharded lock manager
func NewShardedLockManager(numShards int) *ShardedLockManager {
	if numShards <= 0 {
		numShards = 32 // Default number of shards
	}
	// Ensure numShards doesn't exceed uint32 max
	if numShards > 4294967295 { // Max uint32
		numShards = 4294967295
	}

	shards := make([]shard, numShards)
	return &ShardedLockManager{
		shards:    shards,
		numShards: uint32(numShards), // #nosec G115 - bounds checked above
	}
}

// Lock acquires a lock for the given path
func (slm *ShardedLockManager) Lock(path []string, exclusive bool) func() {
	shardIndex := slm.getShardIndex(path)
	shard := &slm.shards[shardIndex]

	if exclusive {
		shard.mu.Lock()
		return shard.mu.Unlock
	} else {
		shard.mu.RLock()
		return shard.mu.RUnlock
	}
}

// TryLock attempts to acquire a lock with a timeout
func (slm *ShardedLockManager) TryLock(path []string, exclusive bool, timeout time.Duration) (func(), error) {
	shardIndex := slm.getShardIndex(path)
	shard := &slm.shards[shardIndex]

	done := make(chan struct{}, 1)
	var unlock func()

	go func() {
		if exclusive {
			shard.mu.Lock()
			unlock = shard.mu.Unlock
		} else {
			shard.mu.RLock()
			unlock = shard.mu.RUnlock
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		return unlock, nil
	case <-time.After(timeout):
		return nil, &LockTimeoutError{Path: path, Timeout: timeout}
	}
}

// IsLocked checks if a path is currently locked (approximation)
func (slm *ShardedLockManager) IsLocked(path []string) bool {
	shardIndex := slm.getShardIndex(path)
	shard := &slm.shards[shardIndex]

	// Try to acquire and immediately release a read lock
	// If this blocks, the shard is likely write-locked
	done := make(chan bool, 1)
	go func() {
		shard.mu.RLock()
		shard.mu.RUnlock()
		done <- false
	}()

	select {
	case result := <-done:
		return result
	case <-time.After(time.Millisecond):
		return true // Likely locked
	}
}

// getShardIndex determines which shard to use for a given path
func (slm *ShardedLockManager) getShardIndex(path []string) uint32 {
	if len(path) == 0 {
		return 0
	}

	h := fnv.New32a()
	for _, segment := range path {
		_, _ = h.Write([]byte(segment))
		_, _ = h.Write([]byte("."))
	}

	return h.Sum32() % slm.numShards
}

// LockTimeoutError represents a lock timeout error
type LockTimeoutError struct {
	Path    []string
	Timeout time.Duration
}

func (e *LockTimeoutError) Error() string {
	return fmt.Sprintf("failed to acquire lock for path %v within %v", e.Path, e.Timeout)
}

// ShardedSafeTree is a SafeTree that uses sharded locking for better concurrency
type ShardedSafeTree struct {
	data        map[interface{}]interface{}
	lockManager *ShardedLockManager
	rootLock    sync.RWMutex // Protects the root map structure
}

// NewShardedSafeTree creates a new thread-safe tree with sharded locking
func NewShardedSafeTree(data map[interface{}]interface{}, numShards int) *ShardedSafeTree {
	if data == nil {
		data = make(map[interface{}]interface{})
	}

	return &ShardedSafeTree{
		data:        data,
		lockManager: NewShardedLockManager(numShards),
	}
}

// Find safely retrieves a value from the tree using sharded locking
func (sst *ShardedSafeTree) Find(path ...string) (interface{}, error) {
	sst.rootLock.RLock()
	defer sst.rootLock.RUnlock()

	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	fullPath := path[0]
	for i := 1; i < len(path); i++ {
		fullPath += "." + path[i]
	}

	return tree.Find(sst.data, fullPath)
}

// FindAll retrieves all values matching the path pattern
func (sst *ShardedSafeTree) FindAll(path ...string) ([]interface{}, error) {
	sst.rootLock.RLock()
	defer sst.rootLock.RUnlock()

	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	fullPath := path[0]
	for i := 1; i < len(path); i++ {
		fullPath += "." + path[i]
	}

	val, err := tree.Find(sst.data, fullPath)
	if err != nil {
		return nil, err
	}

	return []interface{}{val}, nil
}

// Exists checks if a path exists in the tree
func (sst *ShardedSafeTree) Exists(path ...string) bool {
	_, err := sst.Find(path...)
	return err == nil
}

// Copy creates a deep copy of the tree
func (sst *ShardedSafeTree) Copy() ThreadSafeTree {
	sst.rootLock.RLock()
	defer sst.rootLock.RUnlock()

	copied := deepCopyTree(sst.data)
	return NewShardedSafeTree(copied.(map[interface{}]interface{}), int(sst.lockManager.numShards))
}

// Set safely sets a value at the given path
func (sst *ShardedSafeTree) Set(value interface{}, path ...string) error {
	// Use root lock for write operations to prevent concurrent map modifications
	sst.rootLock.Lock()
	defer sst.rootLock.Unlock()

	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	return sst.setInternal(value, path...)
}

// setInternal performs the actual set operation (caller must hold lock)
func (sst *ShardedSafeTree) setInternal(value interface{}, path ...string) error {
	current := sst.data

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
func (sst *ShardedSafeTree) Delete(path ...string) error {
	sst.rootLock.Lock()
	defer sst.rootLock.Unlock()

	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	return sst.deleteInternal(path...)
}

// deleteInternal performs the actual delete operation (caller must hold lock)
func (sst *ShardedSafeTree) deleteInternal(path ...string) error {
	current := sst.data

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
func (sst *ShardedSafeTree) Replace(data map[string]interface{}) error {
	sst.rootLock.Lock()
	defer sst.rootLock.Unlock()

	// Convert string keys to interface{} keys
	converted := make(map[interface{}]interface{})
	for k, v := range data {
		converted[k] = v
	}

	// Deep merge the new data
	sst.data = deepMerge(sst.data, converted)

	return nil
}

// Merge merges another ThreadSafeTree into this one
func (sst *ShardedSafeTree) Merge(other ThreadSafeTree) error {
	if otherSharded, ok := other.(*ShardedSafeTree); ok {
		// Lock both trees
		sst.rootLock.Lock()
		defer sst.rootLock.Unlock()

		otherSharded.rootLock.RLock()
		defer otherSharded.rootLock.RUnlock()

		sst.data = deepMerge(sst.data, otherSharded.data)
		return nil
	}

	return fmt.Errorf("cannot merge incompatible tree types")
}

// CompareAndSwap atomically compares and swaps a value
func (sst *ShardedSafeTree) CompareAndSwap(oldValue, newValue interface{}, path ...string) bool {
	sst.rootLock.Lock()
	defer sst.rootLock.Unlock()

	// Get current value
	current := sst.data
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
func (sst *ShardedSafeTree) Update(fn func(current interface{}) interface{}, path ...string) error {
	unlock := sst.lockManager.Lock(path, true) // Write lock
	defer unlock()

	// Get current value
	fullPath := path[0]
	for i := 1; i < len(path); i++ {
		fullPath += "." + path[i]
	}

	currentValue, _ := tree.Find(sst.data, fullPath)
	newValue := fn(currentValue)

	return sst.setInternal(newValue, path...)
}

// Transaction executes a function within a transaction
func (sst *ShardedSafeTree) Transaction(fn func(tx TreeTransaction) error) error {
	// For simplicity, use global locking for transactions
	// This could be optimized to only lock affected shards
	unlocks := make([]func(), sst.lockManager.numShards)
	for i := uint32(0); i < sst.lockManager.numShards; i++ {
		shard := &sst.lockManager.shards[i]
		shard.mu.Lock()
		unlocks[i] = shard.mu.Unlock
	}

	defer func() {
		for _, unlock := range unlocks {
			unlock()
		}
	}()

	tx := &shardedTreeTransaction{
		tree:    sst,
		changes: make(map[string]interface{}),
		deletes: make(map[string]bool),
	}

	// Execute transaction function
	err := fn(tx)
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.commitInternal()
}

// shardedTreeTransaction implements TreeTransaction for sharded trees
type shardedTreeTransaction struct {
	tree    *ShardedSafeTree
	changes map[string]interface{}
	deletes map[string]bool
	mu      sync.Mutex
}

// Get retrieves a value within the transaction
func (tx *shardedTreeTransaction) Get(path ...string) (interface{}, error) {
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

	// Get from tree (lock already held by transaction)
	fullPath := path[0]
	for i := 1; i < len(path); i++ {
		fullPath += "." + path[i]
	}

	return tree.Find(tx.tree.data, fullPath)
}

// Set sets a value within the transaction
func (tx *shardedTreeTransaction) Set(value interface{}, path ...string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	pathStr := joinPath(path)
	tx.changes[pathStr] = value
	delete(tx.deletes, pathStr) // Remove from deletes if present

	return nil
}

// Delete marks a path for deletion within the transaction
func (tx *shardedTreeTransaction) Delete(path ...string) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	pathStr := joinPath(path)
	tx.deletes[pathStr] = true
	delete(tx.changes, pathStr) // Remove from changes if present

	return nil
}

// Rollback discards all transaction changes
func (tx *shardedTreeTransaction) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	tx.changes = make(map[string]interface{})
	tx.deletes = make(map[string]bool)

	return nil
}

// Commit applies all transaction changes atomically
func (tx *shardedTreeTransaction) Commit() error {
	// This should not be called directly as the parent Transaction method
	// already holds the necessary locks
	return fmt.Errorf("commit should not be called directly on sharded transaction")
}

// commitInternal applies changes (called by Transaction method that holds locks)
func (tx *shardedTreeTransaction) commitInternal() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

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
