package spruce

import (
	"context"
	"sync"
	"sync/atomic"
)

// EarlyTerminator manages early termination and conditional evaluation
type EarlyTerminator struct {
	graph           *DependencyGraph
	necessityTracker *NecessityTracker
	canceller       *CancellationManager
	config          *EarlyTermConfig
	metrics         *EarlyTermMetrics
}

// EarlyTermConfig configures early termination behavior
type EarlyTermConfig struct {
	Enabled              bool
	AggressiveMode       bool
	TrackUnusedPaths     bool
	PropagationDelay     int // microseconds
	MaxSkipPercentage    float64
}

// DefaultEarlyTermConfig returns default configuration
func DefaultEarlyTermConfig() *EarlyTermConfig {
	return &EarlyTermConfig{
		Enabled:           true,
		AggressiveMode:    false,
		TrackUnusedPaths:  true,
		PropagationDelay:  10,
		MaxSkipPercentage: 0.5, // Don't skip more than 50% of operations
	}
}

// EarlyTermMetrics tracks early termination metrics
type EarlyTermMetrics struct {
	OperationsSkipped    int64
	OperationsEvaluated  int64
	PathsMarkedUnused    int64
	CancellationsSent    int64
	TimeSaved            int64 // microseconds
}

// NewEarlyTerminator creates a new early terminator
func NewEarlyTerminator(graph *DependencyGraph, config *EarlyTermConfig) *EarlyTerminator {
	if config == nil {
		config = DefaultEarlyTermConfig()
	}
	
	return &EarlyTerminator{
		graph:            graph,
		necessityTracker: NewNecessityTracker(),
		canceller:        NewCancellationManager(),
		config:           config,
		metrics:          &EarlyTermMetrics{},
	}
}

// NecessityTracker tracks whether operations are necessary
type NecessityTracker struct {
	necessary map[string]bool
	used      map[string]bool
	mu        sync.RWMutex
}

// NewNecessityTracker creates a new necessity tracker
func NewNecessityTracker() *NecessityTracker {
	return &NecessityTracker{
		necessary: make(map[string]bool),
		used:      make(map[string]bool),
	}
}

// MarkNecessary marks a node as necessary
func (nt *NecessityTracker) MarkNecessary(nodeID string) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.necessary[nodeID] = true
}

// MarkUsed marks a result as actually used
func (nt *NecessityTracker) MarkUsed(nodeID string) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.used[nodeID] = true
}

// IsNecessary checks if a node is necessary
func (nt *NecessityTracker) IsNecessary(nodeID string) bool {
	nt.mu.RLock()
	defer nt.mu.RUnlock()
	return nt.necessary[nodeID]
}

// IsUsed checks if a result was used
func (nt *NecessityTracker) IsUsed(nodeID string) bool {
	nt.mu.RLock()
	defer nt.mu.RUnlock()
	return nt.used[nodeID]
}

// GetUnusedNodes returns nodes marked necessary but not used
func (nt *NecessityTracker) GetUnusedNodes() []string {
	nt.mu.RLock()
	defer nt.mu.RUnlock()
	
	unused := make([]string, 0)
	for nodeID := range nt.necessary {
		if !nt.used[nodeID] {
			unused = append(unused, nodeID)
		}
	}
	
	return unused
}

// CancellationManager manages operation cancellation
type CancellationManager struct {
	contexts map[string]context.CancelFunc
	mu       sync.RWMutex
}

// NewCancellationManager creates a new cancellation manager
func NewCancellationManager() *CancellationManager {
	return &CancellationManager{
		contexts: make(map[string]context.CancelFunc),
	}
}

// CreateContext creates a cancellable context for a node
func (cm *CancellationManager) CreateContext(ctx context.Context, nodeID string) context.Context {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	nodeCtx, cancel := context.WithCancel(ctx)
	cm.contexts[nodeID] = cancel
	
	return nodeCtx
}

// Cancel cancels a node's context
func (cm *CancellationManager) Cancel(nodeID string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cancel, exists := cm.contexts[nodeID]; exists {
		cancel()
		delete(cm.contexts, nodeID)
		return true
	}
	
	return false
}

// CancelDependents cancels all dependent operations
func (cm *CancellationManager) CancelDependents(nodeID string, graph *DependencyGraph) int {
	dependents := graph.GetDependents(nodeID)
	cancelled := 0
	
	for _, depID := range dependents {
		if cm.Cancel(depID) {
			cancelled++
			// Recursively cancel dependents
			cancelled += cm.CancelDependents(depID, graph)
		}
	}
	
	return cancelled
}

// AnalyzeNecessity analyzes which operations are necessary
func (et *EarlyTerminator) AnalyzeNecessity(outputPaths []string) {
	if !et.config.Enabled {
		// Mark all as necessary if disabled
		nodeIDs, _ := et.graph.TopologicalSort()
		for _, nodeID := range nodeIDs {
			et.necessityTracker.MarkNecessary(nodeID)
		}
		return
	}
	
	// Mark output paths as necessary
	for _, path := range outputPaths {
		et.necessityTracker.MarkNecessary(path)
	}
	
	// Propagate necessity through dependencies
	et.propagateNecessity()
	
	// Mark unnecessary nodes for skipping
	et.markUnnecessaryNodes()
}

// propagateNecessity propagates necessity through the graph
func (et *EarlyTerminator) propagateNecessity() {
	// Get nodes in reverse topological order
	sortedIDs, _ := et.graph.TopologicalSort()
	allNodes := et.graph.GetNodes()
	
	// Process from outputs backward
	for i := len(sortedIDs) - 1; i >= 0; i-- {
		nodeID := sortedIDs[i]
		
		if et.necessityTracker.IsNecessary(nodeID) {
			// Mark all dependencies as necessary
			if node, ok := allNodes[nodeID]; ok {
				for _, depID := range node.Dependencies {
					et.necessityTracker.MarkNecessary(depID)
				}
			}
		}
	}
}

// markUnnecessaryNodes marks nodes that can be skipped
func (et *EarlyTerminator) markUnnecessaryNodes() {
	nodeIDs, _ := et.graph.TopologicalSort()
	allNodes := et.graph.GetNodes()
	totalNodes := len(nodeIDs)
	skippable := 0
	
	for _, nodeID := range nodeIDs {
		if !et.necessityTracker.IsNecessary(nodeID) {
			// Check if we're within skip limit
			skipPercentage := float64(skippable) / float64(totalNodes)
			if skipPercentage < et.config.MaxSkipPercentage {
				if node, ok := allNodes[nodeID]; ok {
					node.Status = StatusSkipped
					skippable++
					atomic.AddInt64(&et.metrics.OperationsSkipped, 1)
				}
			}
		}
	}
}

// ShouldEvaluate checks if a node should be evaluated
func (et *EarlyTerminator) ShouldEvaluate(nodeID string) bool {
	if !et.config.Enabled {
		return true
	}
	
	node, exists := et.graph.GetNode(nodeID)
	if !exists {
		return true
	}
	
	// Check if already marked for skipping
	if node.Status == StatusSkipped {
		return false
	}
	
	// Check if necessary
	if !et.necessityTracker.IsNecessary(nodeID) {
		node.Status = StatusSkipped
		atomic.AddInt64(&et.metrics.OperationsSkipped, 1)
		return false
	}
	
	// In aggressive mode, check if any dependent will use the result
	if et.config.AggressiveMode {
		return et.hasNecessaryDependents(nodeID)
	}
	
	return true
}

// hasNecessaryDependents checks if any dependent is necessary
func (et *EarlyTerminator) hasNecessaryDependents(nodeID string) bool {
	dependents := et.graph.GetDependents(nodeID)
	
	for _, depID := range dependents {
		if et.necessityTracker.IsNecessary(depID) {
			return true
		}
	}
	
	return false
}

// OnNodeCompleted handles node completion for early termination
func (et *EarlyTerminator) OnNodeCompleted(nodeID string, result interface{}, err error) {
	if !et.config.Enabled {
		return
	}
	
	atomic.AddInt64(&et.metrics.OperationsEvaluated, 1)
	
	// If node failed, cancel dependents
	if err != nil {
		cancelled := et.canceller.CancelDependents(nodeID, et.graph)
		atomic.AddInt64(&et.metrics.CancellationsSent, int64(cancelled))
		return
	}
	
	// Check if result is actually used
	if et.config.TrackUnusedPaths {
		// This would be called when result is accessed
		et.necessityTracker.MarkUsed(nodeID)
	}
	
	// In aggressive mode, check if we can skip dependents
	if et.config.AggressiveMode {
		et.checkDependentNecessity(nodeID, result)
	}
}

// checkDependentNecessity checks if dependents are still necessary
func (et *EarlyTerminator) checkDependentNecessity(nodeID string, result interface{}) {
	// Check if result indicates dependents can be skipped
	// For example, if result is nil or empty
	
	if et.isSkippableResult(result) {
		dependents := et.graph.GetDependents(nodeID)
		allNodes := et.graph.GetNodes()
		
		for _, depID := range dependents {
			if dep, ok := allNodes[depID]; ok {
				if et.canSkipDependent(dep, result) {
					dep.Status = StatusSkipped
					atomic.AddInt64(&et.metrics.OperationsSkipped, 1)
					
					// Cancel if already running
					if et.canceller.Cancel(depID) {
						atomic.AddInt64(&et.metrics.CancellationsSent, 1)
					}
				}
			}
		}
	}
}

// isSkippableResult checks if a result indicates dependents can be skipped
func (et *EarlyTerminator) isSkippableResult(result interface{}) bool {
	switch v := result.(type) {
	case nil:
		return true
	case bool:
		return !v // false results might skip dependent operations
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

// canSkipDependent checks if a dependent can be skipped based on result
func (et *EarlyTerminator) canSkipDependent(dep *DependencyNode, result interface{}) bool {
	// Operation-specific logic
	switch dep.OperatorType {
	case "concat":
		// Skip concat if input is empty
		return et.isSkippableResult(result)
	case "grab":
		// Skip grab if source doesn't exist
		return result == nil
	case "ips":
		// Skip IP allocation if no instances
		return et.isSkippableResult(result)
	default:
		// Conservative: don't skip unknown operators
		return false
	}
}

// CreateNodeContext creates a cancellable context for a node
func (et *EarlyTerminator) CreateNodeContext(ctx context.Context, nodeID string) context.Context {
	if !et.config.Enabled {
		return ctx
	}
	
	return et.canceller.CreateContext(ctx, nodeID)
}

// GetMetrics returns early termination metrics
func (et *EarlyTerminator) GetMetrics() EarlyTermMetrics {
	return EarlyTermMetrics{
		OperationsSkipped:   atomic.LoadInt64(&et.metrics.OperationsSkipped),
		OperationsEvaluated: atomic.LoadInt64(&et.metrics.OperationsEvaluated),
		PathsMarkedUnused:   atomic.LoadInt64(&et.metrics.PathsMarkedUnused),
		CancellationsSent:   atomic.LoadInt64(&et.metrics.CancellationsSent),
		TimeSaved:          atomic.LoadInt64(&et.metrics.TimeSaved),
	}
}

// AnalyzeUnusedPaths analyzes paths that were marked necessary but not used
func (et *EarlyTerminator) AnalyzeUnusedPaths() []string {
	unused := et.necessityTracker.GetUnusedNodes()
	atomic.StoreInt64(&et.metrics.PathsMarkedUnused, int64(len(unused)))
	return unused
}

// Reset resets the early terminator for a new execution
func (et *EarlyTerminator) Reset() {
	et.necessityTracker = NewNecessityTracker()
	et.canceller = NewCancellationManager()
	et.metrics = &EarlyTermMetrics{}
}

// ConditionalEvaluator handles conditional evaluation logic
type ConditionalEvaluator struct {
	conditions map[string]EvalCondition
	mu         sync.RWMutex
}

// EvalCondition represents an evaluation condition
type EvalCondition func(dependencies map[string]interface{}) bool

// NewConditionalEvaluator creates a new conditional evaluator
func NewConditionalEvaluator() *ConditionalEvaluator {
	return &ConditionalEvaluator{
		conditions: make(map[string]EvalCondition),
	}
}

// RegisterCondition registers a condition for a node
func (ce *ConditionalEvaluator) RegisterCondition(nodeID string, condition EvalCondition) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.conditions[nodeID] = condition
}

// ShouldEvaluate checks if a node should be evaluated based on conditions
func (ce *ConditionalEvaluator) ShouldEvaluate(nodeID string, dependencies map[string]interface{}) bool {
	ce.mu.RLock()
	condition, exists := ce.conditions[nodeID]
	ce.mu.RUnlock()
	
	if !exists {
		return true // No condition means always evaluate
	}
	
	return condition(dependencies)
}

// Example condition functions

// OnlyIfNotEmpty evaluates only if dependency is not empty
func OnlyIfNotEmpty(depName string) EvalCondition {
	return func(deps map[string]interface{}) bool {
		val, exists := deps[depName]
		if !exists {
			return false
		}
		
		switch v := val.(type) {
		case string:
			return v != ""
		case []interface{}:
			return len(v) > 0
		case map[string]interface{}:
			return len(v) > 0
		default:
			return val != nil
		}
	}
}

// OnlyIfTrue evaluates only if dependency is true
func OnlyIfTrue(depName string) EvalCondition {
	return func(deps map[string]interface{}) bool {
		val, exists := deps[depName]
		if !exists {
			return false
		}
		
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
		
		return false
	}
}