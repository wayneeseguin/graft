package graft

import (
	"context"
	"sync"
	"time"
)

// ThreadSafeTree defines the interface for a thread-safe tree implementation
type ThreadSafeTree interface {
	// Read operations (safe for concurrent access)
	Find(path ...string) (interface{}, error)
	FindAll(path ...string) ([]interface{}, error)
	Exists(path ...string) bool
	Copy() ThreadSafeTree
	
	// Write operations (synchronized)
	Set(value interface{}, path ...string) error
	Delete(path ...string) error
	Replace(data map[string]interface{}) error
	Merge(other ThreadSafeTree) error
	
	// Atomic operations
	CompareAndSwap(oldValue, newValue interface{}, path ...string) bool
	Update(fn func(current interface{}) interface{}, path ...string) error
	
	// Bulk operations
	Transaction(fn func(tx TreeTransaction) error) error
}

// TreeTransaction represents a transaction on the tree
type TreeTransaction interface {
	Get(path ...string) (interface{}, error)
	Set(value interface{}, path ...string) error
	Delete(path ...string) error
	Rollback() error
	Commit() error
}

// ThreadSafeEvaluator defines the interface for thread-safe evaluation
type ThreadSafeEvaluator interface {
	// Evaluation operations
	Evaluate(ctx context.Context) error
	EvaluateSubtree(ctx context.Context, path ...string) error
	
	// Operator execution
	ExecuteOperator(ctx context.Context, op Operator, args []interface{}) (interface{}, error)
	
	// Progress monitoring
	Progress() EvaluationProgress
	Subscribe(listener EvaluationListener) func()
}

// EvaluationProgress tracks the progress of evaluation
type EvaluationProgress struct {
	Total      int
	Completed  int
	Failed     int
	InProgress int
	StartTime  time.Time
}

// EvaluationListener receives evaluation events
type EvaluationListener interface {
	OnOperatorStart(path []string, operator string)
	OnOperatorComplete(path []string, operator string, result interface{}, err error)
	OnProgress(progress EvaluationProgress)
}

// ConcurrencyControl manages concurrency limits and synchronization
type ConcurrencyControl interface {
	// Resource acquisition
	Acquire(ctx context.Context, resource string) error
	Release(resource string)
	
	// Rate limiting
	Wait(ctx context.Context) error
	
	// Deadlock detection
	CheckDeadlock() error
}

// LockStrategy defines different locking strategies
type LockStrategy int

const (
	GlobalLock LockStrategy = iota
	ShardedLocks
	PathBasedLocks
	OptimisticLocking
)

// LockManager manages locks for tree operations
type LockManager interface {
	Lock(path []string, exclusive bool) func()
	TryLock(path []string, exclusive bool, timeout time.Duration) (func(), error)
	IsLocked(path []string) bool
}

// ThreadSafeOperator wraps an operator for thread-safe execution
type ThreadSafeOperator struct {
	operator Operator
	mu       sync.RWMutex
}

// NewThreadSafeOperator creates a thread-safe wrapper for an operator
func NewThreadSafeOperator(op Operator) *ThreadSafeOperator {
	return &ThreadSafeOperator{operator: op}
}

// Run safely executes the operator
func (tso *ThreadSafeOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	// Most operators are read-only, use RLock by default
	if tso.isWriteOperator() {
		tso.mu.Lock()
		defer tso.mu.Unlock()
	} else {
		tso.mu.RLock()
		defer tso.mu.RUnlock()
	}
	
	return tso.operator.Run(ev, args)
}

// isWriteOperator determines if the operator modifies state
func (tso *ThreadSafeOperator) isWriteOperator() bool {
	// These operators modify the tree structure
	writeOperators := map[OperatorPhase]bool{
		MergePhase:  true,
		EvalPhase:   true, // Some eval operators modify state
	}
	
	return writeOperators[tso.operator.Phase()]
}

// ThreadSafeOperatorRegistry provides thread-safe operator registration
type ThreadSafeOperatorRegistry struct {
	operators map[string]*ThreadSafeOperator
	mu        sync.RWMutex
}

// NewThreadSafeOperatorRegistry creates a new thread-safe operator registry
func NewThreadSafeOperatorRegistry() *ThreadSafeOperatorRegistry {
	return &ThreadSafeOperatorRegistry{
		operators: make(map[string]*ThreadSafeOperator),
	}
}

// Register adds an operator to the registry
func (or *ThreadSafeOperatorRegistry) Register(name string, op Operator) {
	or.mu.Lock()
	defer or.mu.Unlock()
	
	or.operators[name] = NewThreadSafeOperator(op)
}

// Get retrieves an operator from the registry
func (or *ThreadSafeOperatorRegistry) Get(name string) (*ThreadSafeOperator, bool) {
	or.mu.RLock()
	defer or.mu.RUnlock()
	
	op, ok := or.operators[name]
	return op, ok
}

// EvaluationContext provides context for concurrent evaluation
type EvaluationContext struct {
	ctx      context.Context
	tree     ThreadSafeTree
	registry *ThreadSafeOperatorRegistry
	control  ConcurrencyControl
	locks    LockManager
	progress *EvaluationProgress
	mu       sync.RWMutex
}

// NewEvaluationContext creates a new evaluation context
func NewEvaluationContext(ctx context.Context, tree ThreadSafeTree) *EvaluationContext {
	return &EvaluationContext{
		ctx:      ctx,
		tree:     tree,
		registry: NewThreadSafeOperatorRegistry(),
		progress: &EvaluationProgress{
			StartTime: time.Now(),
		},
	}
}

// UpdateProgress atomically updates evaluation progress
func (ec *EvaluationContext) UpdateProgress(fn func(p *EvaluationProgress)) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	fn(ec.progress)
}

// GetProgress returns a copy of current progress
func (ec *EvaluationContext) GetProgress() EvaluationProgress {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return *ec.progress
}