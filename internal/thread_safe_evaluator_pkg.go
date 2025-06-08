package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	gotree "github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// ThreadSafeEvaluatorImpl implements ThreadSafeEvaluator interface
type ThreadSafeEvaluatorImpl struct {
	safeTree    ThreadSafeTree
	originalEv  *graft.Evaluator
	mu          sync.RWMutex
	listeners   []EvaluationListener
	progress    *EvaluationProgress
	progressMu  sync.RWMutex
}

// NewThreadSafeEvaluator creates a new thread-safe evaluator
func NewThreadSafeEvaluator(tree ThreadSafeTree) *ThreadSafeEvaluatorImpl {
	// Create a traditional evaluator for backward compatibility
	var rawData map[interface{}]interface{}
	if safeTree, ok := tree.(*SafeTree); ok {
		rawData = safeTree.GetRawDataUnsafe()
	} else {
		rawData = make(map[interface{}]interface{})
	}
	
	originalEv := &graft.Evaluator{
		Tree:     rawData,
		SkipEval: false,
		Here:     &gotree.Cursor{},
		CheckOps: make([]*graft.Opcall, 0),
		Only:     []string{},
	}
	
	// Initialize Deps separately to avoid type issues
	if originalEv.Deps == nil {
		originalEv.Deps = make(map[string][]gotree.Cursor)
	}
	
	return &ThreadSafeEvaluatorImpl{
		safeTree:   tree,
		originalEv: originalEv,
		listeners:  make([]EvaluationListener, 0),
		progress: &EvaluationProgress{
			Total:      0,
			Completed:  0,
			Failed:     0,
			InProgress: 0,
			StartTime:  time.Now(),
		},
	}
}

// Evaluate performs thread-safe evaluation of the tree
func (tse *ThreadSafeEvaluatorImpl) Evaluate(ctx context.Context) error {
	tse.mu.Lock()
	defer tse.mu.Unlock()
	
	// Update the original evaluator's tree with current safe tree data
	if safeTree, ok := tse.safeTree.(*SafeTree); ok {
		tse.originalEv.Tree = safeTree.GetRawData()
	}
	
	// Track progress
	tse.updateProgress(func(p *EvaluationProgress) {
		p.StartTime = time.Now()
		p.InProgress = 1
	})
	
	// Notify listeners
	tse.notifyProgress()
	
	// Perform evaluation using the original evaluator
	err := tse.originalEv.Run([]string{}, []string{})
	
	// Update progress based on result
	tse.updateProgress(func(p *EvaluationProgress) {
		p.InProgress = 0
		if err != nil {
			p.Failed = 1
		} else {
			p.Completed = 1
		}
	})
	
	// Sync changes back to safe tree
	if err == nil {
		if err := tse.syncBackToSafeTree(); err != nil {
			return fmt.Errorf("failed to sync evaluation results: %v", err)
		}
	}
	
	// Final progress notification
	tse.notifyProgress()
	
	return err
}

// EvaluateSubtree performs thread-safe evaluation of a subtree
func (tse *ThreadSafeEvaluatorImpl) EvaluateSubtree(ctx context.Context, path ...string) error {
	tse.mu.Lock()
	defer tse.mu.Unlock()
	
	// This is a simplified implementation
	// In practice, we'd need to handle subtree evaluation more carefully
	return tse.Evaluate(ctx)
}

// ExecuteOperator executes a single operator in a thread-safe manner
func (tse *ThreadSafeEvaluatorImpl) ExecuteOperator(ctx context.Context, op graft.Operator, args []interface{}) (interface{}, error) {
	tse.mu.Lock()
	defer tse.mu.Unlock()
	
	// Convert args to expressions
	exprs := make([]*graft.Expr, len(args))
	for i, arg := range args {
		exprs[i] = &graft.Expr{
			Type:    graft.Literal,
			Literal: arg,
		}
	}
	
	// Execute the operator
	response, err := op.Run(tse.originalEv, exprs)
	if err != nil {
		return nil, err
	}
	
	return response.Value, nil
}

// Progress returns the current evaluation progress
func (tse *ThreadSafeEvaluatorImpl) Progress() EvaluationProgress {
	tse.progressMu.RLock()
	defer tse.progressMu.RUnlock()
	return *tse.progress
}

// Subscribe adds an evaluation listener
func (tse *ThreadSafeEvaluatorImpl) Subscribe(listener EvaluationListener) func() {
	tse.mu.Lock()
	defer tse.mu.Unlock()
	
	tse.listeners = append(tse.listeners, listener)
	
	// Return unsubscribe function
	return func() {
		tse.mu.Lock()
		defer tse.mu.Unlock()
		
		for i, l := range tse.listeners {
			if l == listener {
				tse.listeners = append(tse.listeners[:i], tse.listeners[i+1:]...)
				break
			}
		}
	}
}

// Helper methods

func (tse *ThreadSafeEvaluatorImpl) updateProgress(fn func(p *EvaluationProgress)) {
	tse.progressMu.Lock()
	defer tse.progressMu.Unlock()
	fn(tse.progress)
}

func (tse *ThreadSafeEvaluatorImpl) notifyProgress() {
	progress := tse.Progress()
	for _, listener := range tse.listeners {
		listener.OnProgress(progress)
	}
}

func (tse *ThreadSafeEvaluatorImpl) syncBackToSafeTree() error {
	// Convert the evaluated tree back to our safe tree format
	data := make(map[string]interface{})
	
	// Convert map[interface{}]interface{} to map[string]interface{}
	for k, v := range tse.originalEv.Tree {
		if keyStr, ok := k.(string); ok {
			data[keyStr] = v
		}
	}
	
	// Replace the entire tree content
	return tse.safeTree.Replace(data)
}

// ThreadSafeOperatorAdapter wraps an operator for thread-safe execution
type ThreadSafeOperatorAdapter struct {
	operator   graft.Operator
	mu         sync.Mutex
	cache      map[string]interface{}
	cacheTTL   time.Duration
	cacheTime  map[string]time.Time
}

// NewThreadSafeOperatorAdapter creates a new thread-safe operator adapter
func NewThreadSafeOperatorAdapter(op graft.Operator) *ThreadSafeOperatorAdapter {
	return &ThreadSafeOperatorAdapter{
		operator:  op,
		cache:     make(map[string]interface{}),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  5 * time.Minute,
	}
}

// Run executes the operator in a thread-safe manner with caching
func (toa *ThreadSafeOperatorAdapter) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	// Generate cache key
	cacheKey := toa.generateCacheKey(args)
	
	// Check cache first (for read-only operators)
	if toa.isReadOnlyOperator() {
		toa.mu.Lock()
		if cached, found := toa.cache[cacheKey]; found {
			if cacheTime, ok := toa.cacheTime[cacheKey]; ok {
				if time.Since(cacheTime) < toa.cacheTTL {
					toa.mu.Unlock()
					return &graft.Response{
						Type:  graft.Replace,
						Value: cached,
					}, nil
				}
			}
		}
		toa.mu.Unlock()
	}
	
	// Execute the operator
	response, err := toa.operator.Run(ev, args)
	
	// Cache successful results for read-only operators
	if err == nil && toa.isReadOnlyOperator() {
		toa.mu.Lock()
		toa.cache[cacheKey] = response.Value
		toa.cacheTime[cacheKey] = time.Now()
		toa.mu.Unlock()
	}
	
	return response, err
}

func (toa *ThreadSafeOperatorAdapter) generateCacheKey(args []*graft.Expr) string {
	// Simple cache key generation - in practice, this would be more sophisticated
	return fmt.Sprintf("%v", args)
}

func (toa *ThreadSafeOperatorAdapter) isReadOnlyOperator() bool {
	// Define which operators are read-only and safe to cache
	readOnlyOps := map[string]bool{
		"*GrabOperator":     true,
		"*ConcatOperator":   true,
		"*Base64Operator":   true,
		"*KeysOperator":     true,
		"*EmptyOperator":    true,
		"*JoinOperator":     true,
		"*SortOperator":     true,
		"*StringifyOperator": true,
	}
	
	// This is a simplification - in practice, we'd need a better way 
	// to identify operator types
	return readOnlyOps[fmt.Sprintf("%T", toa.operator)]
}

// MigrationHelper provides utilities for migrating to thread-safe evaluators
type MigrationHelper struct {
	safeTree ThreadSafeTree
	tsEval   *ThreadSafeEvaluatorImpl
}

// NewMigrationHelper creates a helper for migrating existing code
func NewMigrationHelper(data map[interface{}]interface{}) *MigrationHelper {
	safeTree := NewSafeTree(data)
	tsEval := NewThreadSafeEvaluator(safeTree)
	
	return &MigrationHelper{
		safeTree: safeTree,
		tsEval:   tsEval,
	}
}

// GetThreadSafeEvaluator returns a thread-safe evaluator
func (mh *MigrationHelper) GetThreadSafeEvaluator() *ThreadSafeEvaluatorImpl {
	return mh.tsEval
}

// MigrateEvaluator converts an existing evaluator to thread-safe
func (mh *MigrationHelper) MigrateEvaluator(originalEv *graft.Evaluator) *ThreadSafeEvaluatorImpl {
	// Update the internal evaluator with original's settings
	mh.tsEval.originalEv.Deps = originalEv.Deps
	mh.tsEval.originalEv.SkipEval = originalEv.SkipEval
	mh.tsEval.originalEv.CheckOps = originalEv.CheckOps
	mh.tsEval.originalEv.Only = originalEv.Only
	
	return mh.tsEval
}

// GetThreadSafeTree returns the underlying thread-safe tree
func (mh *MigrationHelper) GetThreadSafeTree() ThreadSafeTree {
	return mh.safeTree
}

// GetCompatibleEvaluator returns an evaluator compatible with existing code
func (mh *MigrationHelper) GetCompatibleEvaluator() *graft.Evaluator {
	return mh.tsEval.originalEv
}

// SimpleEvaluationListener provides a basic implementation of EvaluationListener
type SimpleEvaluationListener struct {
	OnOperatorStartFunc    func(path []string, operator string)
	OnOperatorCompleteFunc func(path []string, operator string, result interface{}, err error)
	OnProgressFunc         func(progress EvaluationProgress)
}

// OnOperatorStart is called when an operator starts
func (sel *SimpleEvaluationListener) OnOperatorStart(path []string, operator string) {
	if sel.OnOperatorStartFunc != nil {
		sel.OnOperatorStartFunc(path, operator)
	}
}

// OnOperatorComplete is called when an operator completes
func (sel *SimpleEvaluationListener) OnOperatorComplete(path []string, operator string, result interface{}, err error) {
	if sel.OnOperatorCompleteFunc != nil {
		sel.OnOperatorCompleteFunc(path, operator, result, err)
	}
}

// OnProgress is called when evaluation progress changes
func (sel *SimpleEvaluationListener) OnProgress(progress EvaluationProgress) {
	if sel.OnProgressFunc != nil {
		sel.OnProgressFunc(progress)
	}
}