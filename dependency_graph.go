package graft

import (
	"fmt"
	"sync"
)

// NodeStatus represents the execution status of a node
type NodeStatus string

const (
	StatusPending   NodeStatus = "pending"
	StatusReady     NodeStatus = "ready"
	StatusRunning   NodeStatus = "running"
	StatusCompleted NodeStatus = "completed"
	StatusFailed    NodeStatus = "failed"
	StatusSkipped   NodeStatus = "skipped"
)

// ExecutionStage represents a stage of parallel execution
type ExecutionStage struct {
	Operations  []*DependencyNode
	CanParallel bool
}

// DependencyGraph represents a graph of dependencies between nodes
type DependencyGraph struct {
	nodes     map[string]*DependencyNode
	mu        sync.RWMutex
	sorted    []string
	evaluated map[string]bool
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID           string
	Path         []string
	Dependencies []string
	Dependents   []string
	Value        interface{}
	Expression   string
	Cost         float64
	OperatorType string
	Status       NodeStatus
	Ready        bool
	InProgress   bool
	Completed    bool
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:     make(map[string]*DependencyNode),
		evaluated: make(map[string]bool),
	}
}

// Clear clears the dependency graph
func (dg *DependencyGraph) Clear() {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	dg.nodes = make(map[string]*DependencyNode)
	dg.sorted = nil
	dg.evaluated = make(map[string]bool)
}

// AddNode adds a node to the graph
func (dg *DependencyGraph) AddNode(path string, node *DependencyNode) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	dg.nodes[path] = node
}

// GetNode retrieves a node from the graph
func (dg *DependencyGraph) GetNode(path string) (*DependencyNode, bool) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	node, ok := dg.nodes[path]
	return node, ok
}

// GetNodes returns all nodes in the graph
func (dg *DependencyGraph) GetNodes() map[string]*DependencyNode {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	// Return a copy to prevent concurrent modification
	nodes := make(map[string]*DependencyNode)
	for k, v := range dg.nodes {
		nodes[k] = v
	}
	return nodes
}

// AddDependency adds a dependency between two nodes
func (dg *DependencyGraph) AddDependency(from, to string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	if fromNode, ok := dg.nodes[from]; ok {
		fromNode.Dependencies = append(fromNode.Dependencies, to)
	}
	
	if toNode, ok := dg.nodes[to]; ok {
		toNode.Dependents = append(toNode.Dependents, from)
	}
}

// GetReadyNodes returns nodes that are ready to be processed
func (dg *DependencyGraph) GetReadyNodes() []*DependencyNode {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	var ready []*DependencyNode
	for _, node := range dg.nodes {
		if node.Ready && !node.InProgress && !node.Completed {
			// Check if all dependencies are completed
			allDepsCompleted := true
			for _, dep := range node.Dependencies {
				if depNode, ok := dg.nodes[dep]; ok {
					if !depNode.Completed {
						allDepsCompleted = false
						break
					}
				}
			}
			
			if allDepsCompleted {
				ready = append(ready, node)
			}
		}
	}
	
	return ready
}

// MarkInProgress marks a node as being processed
func (dg *DependencyGraph) MarkInProgress(path string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	if node, ok := dg.nodes[path]; ok {
		node.InProgress = true
	}
}

// MarkCompleted marks a node as completed
func (dg *DependencyGraph) MarkCompleted(path string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	if node, ok := dg.nodes[path]; ok {
		node.InProgress = false
		node.Completed = true
		
		// Update dependents to be ready if all their dependencies are met
		for _, dependent := range node.Dependents {
			if depNode, ok := dg.nodes[dependent]; ok {
				allDepsCompleted := true
				for _, dep := range depNode.Dependencies {
					if d, exists := dg.nodes[dep]; exists && !d.Completed {
						allDepsCompleted = false
						break
					}
				}
				if allDepsCompleted {
					depNode.Ready = true
				}
			}
		}
	}
}

// TopologicalSort performs a topological sort on the graph
func (dg *DependencyGraph) TopologicalSort() ([]string, error) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	
	if dg.sorted != nil {
		return dg.sorted, nil
	}
	
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	sorted := make([]string, 0, len(dg.nodes))
	
	var visit func(string) error
	visit = func(path string) error {
		if visiting[path] {
			return fmt.Errorf("circular dependency detected at %s", path)
		}
		
		if visited[path] {
			return nil
		}
		
		visiting[path] = true
		
		if node, ok := dg.nodes[path]; ok {
			for _, dep := range node.Dependencies {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		
		visiting[path] = false
		visited[path] = true
		sorted = append(sorted, path)
		
		return nil
	}
	
	for path := range dg.nodes {
		if err := visit(path); err != nil {
			return nil, err
		}
	}
	
	dg.sorted = sorted
	return sorted, nil
}

// GetDependencyWaves returns nodes grouped by execution waves
func (dg *DependencyGraph) GetDependencyWaves() ([][]string, error) {
	// First do topological sort to detect cycles
	if _, err := dg.TopologicalSort(); err != nil {
		return nil, err
	}
	
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	waves := [][]string{}
	processed := make(map[string]bool)
	
	for len(processed) < len(dg.nodes) {
		wave := []string{}
		
		// Find all nodes that can be processed in this wave
		for path, node := range dg.nodes {
			if processed[path] {
				continue
			}
			
			// Check if all dependencies have been processed
			canProcess := true
			for _, dep := range node.Dependencies {
				if !processed[dep] {
					canProcess = false
					break
				}
			}
			
			if canProcess {
				wave = append(wave, path)
			}
		}
		
		if len(wave) == 0 {
			// This shouldn't happen after topological sort
			return nil, fmt.Errorf("unable to make progress in dependency resolution")
		}
		
		// Mark wave nodes as processed
		for _, path := range wave {
			processed[path] = true
		}
		
		waves = append(waves, wave)
	}
	
	return waves, nil
}

// HasCycles checks if the graph has cycles
func (dg *DependencyGraph) HasCycles() bool {
	_, err := dg.TopologicalSort()
	return err != nil
}

// Size returns the number of nodes in the graph
func (dg *DependencyGraph) Size() int {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	return len(dg.nodes)
}

// GetDependents returns all dependents for a given node
func (dg *DependencyGraph) GetDependents(nodeID string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	if node, ok := dg.nodes[nodeID]; ok {
		return node.Dependents
	}
	return nil
}

// GetStatistics returns statistics about the graph
func (dg *DependencyGraph) GetStatistics() map[string]int {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	stats := map[string]int{
		"total_nodes":      len(dg.nodes),
		"ready_nodes":      0,
		"in_progress":      0,
		"completed":        0,
		"total_edges":      0,
		"max_dependencies": 0,
		"max_dependents":   0,
	}
	
	for _, node := range dg.nodes {
		if node.Ready && !node.InProgress && !node.Completed {
			stats["ready_nodes"]++
		}
		if node.InProgress {
			stats["in_progress"]++
		}
		if node.Completed {
			stats["completed"]++
		}
		
		stats["total_edges"] += len(node.Dependencies)
		
		if len(node.Dependencies) > stats["max_dependencies"] {
			stats["max_dependencies"] = len(node.Dependencies)
		}
		if len(node.Dependents) > stats["max_dependents"] {
			stats["max_dependents"] = len(node.Dependents)
		}
	}
	
	return stats
}

// GetExecutionStages returns operations grouped by execution stages
func (dg *DependencyGraph) GetExecutionStages() ([]ExecutionStage, error) {
	waves, err := dg.GetDependencyWaves()
	if err != nil {
		return nil, err
	}
	
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	
	stages := make([]ExecutionStage, len(waves))
	for i, wave := range waves {
		stage := ExecutionStage{
			Operations:  make([]*DependencyNode, len(wave)),
			CanParallel: true, // All operations in a wave can run in parallel
		}
		
		for j, path := range wave {
			if node, ok := dg.nodes[path]; ok {
				stage.Operations[j] = node
			}
		}
		
		stages[i] = stage
	}
	
	return stages, nil
}