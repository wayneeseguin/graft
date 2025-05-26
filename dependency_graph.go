package spruce

import (
	"fmt"
	"strings"
	"sync"
)

// DependencyGraph represents dependencies between expressions
type DependencyGraph struct {
	mu    sync.RWMutex
	nodes map[string]*DependencyNode
	edges map[string]map[string]bool // from -> to -> exists
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID           string
	Expression   interface{}
	OperatorType string
	Path         []string
	Cost         float64
	Dependencies map[string]bool
	Dependents   map[string]bool
	Status       ExecutionStatus
	Result       interface{}
	Error        error
	mu           sync.RWMutex
}

// ExecutionStatus represents the execution state of a node
type ExecutionStatus int

const (
	StatusPending ExecutionStatus = iota
	StatusReady
	StatusExecuting
	StatusCompleted
	StatusFailed
	StatusSkipped
)

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
		edges: make(map[string]map[string]bool),
	}
}

// AddNode adds a node to the graph
func (g *DependencyGraph) AddNode(id string, expr interface{}, opType string, path []string) *DependencyNode {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if node, exists := g.nodes[id]; exists {
		return node
	}
	
	node := &DependencyNode{
		ID:           id,
		Expression:   expr,
		OperatorType: opType,
		Path:         path,
		Dependencies: make(map[string]bool),
		Dependents:   make(map[string]bool),
		Status:       StatusPending,
	}
	
	g.nodes[id] = node
	g.edges[id] = make(map[string]bool)
	
	return node
}

// AddDependency adds a dependency edge from -> to
func (g *DependencyGraph) AddDependency(fromID, toID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	fromNode, fromExists := g.nodes[fromID]
	toNode, toExists := g.nodes[toID]
	
	if !fromExists {
		return fmt.Errorf("node %s not found", fromID)
	}
	if !toExists {
		return fmt.Errorf("node %s not found", toID)
	}
	
	// Check for circular dependency
	if g.wouldCreateCycle(fromID, toID) {
		return fmt.Errorf("circular dependency detected: %s -> %s", fromID, toID)
	}
	
	// Add edge
	if g.edges[fromID] == nil {
		g.edges[fromID] = make(map[string]bool)
	}
	g.edges[fromID][toID] = true
	
	// Update node references
	fromNode.Dependents[toID] = true
	toNode.Dependencies[fromID] = true
	
	return nil
}

// wouldCreateCycle checks if adding an edge would create a cycle
func (g *DependencyGraph) wouldCreateCycle(from, to string) bool {
	// Check if there's already a path from 'to' to 'from'
	visited := make(map[string]bool)
	return g.hasPath(to, from, visited)
}

// hasPath checks if there's a path from start to end
func (g *DependencyGraph) hasPath(start, end string, visited map[string]bool) bool {
	if start == end {
		return true
	}
	
	if visited[start] {
		return false
	}
	visited[start] = true
	
	if edges, ok := g.edges[start]; ok {
		for next := range edges {
			if g.hasPath(next, end, visited) {
				return true
			}
		}
	}
	
	return false
}

// TopologicalSort returns nodes in topological order
func (g *DependencyGraph) TopologicalSort() ([]*DependencyNode, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	// Count incoming edges
	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = 0
	}
	
	for _, edges := range g.edges {
		for to := range edges {
			inDegree[to]++
		}
	}
	
	// Find nodes with no dependencies
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	
	// Process nodes
	var sorted []*DependencyNode
	processed := 0
	
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		
		node := g.nodes[current]
		sorted = append(sorted, node)
		processed++
		
		// Update degrees of dependent nodes
		if edges, ok := g.edges[current]; ok {
			for dependent := range edges {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}
	}
	
	if processed != len(g.nodes) {
		return nil, fmt.Errorf("circular dependency detected: processed %d of %d nodes", processed, len(g.nodes))
	}
	
	return sorted, nil
}

// GetExecutionStages returns nodes grouped by execution stages
func (g *DependencyGraph) GetExecutionStages() ([]ExecutionStage, error) {
	sorted, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}
	
	stages := make([]ExecutionStage, 0)
	nodeStage := make(map[string]int)
	
	for _, node := range sorted {
		maxDepStage := -1
		
		// Find maximum stage of dependencies
		for depID := range node.Dependencies {
			if stage, ok := nodeStage[depID]; ok && stage > maxDepStage {
				maxDepStage = stage
			}
		}
		
		// Assign to next stage
		stageIndex := maxDepStage + 1
		nodeStage[node.ID] = stageIndex
		
		// Ensure we have enough stages
		for len(stages) <= stageIndex {
			stages = append(stages, ExecutionStage{
				Operations:  make([]*DependencyNode, 0),
				CanParallel: true,
			})
		}
		
		stages[stageIndex].Operations = append(stages[stageIndex].Operations, node)
	}
	
	return stages, nil
}

// GetNode returns a node by ID
func (g *DependencyGraph) GetNode(id string) (*DependencyNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	node, exists := g.nodes[id]
	return node, exists
}

// GetDependencies returns direct dependencies of a node
func (g *DependencyGraph) GetDependencies(id string) []*DependencyNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	node, exists := g.nodes[id]
	if !exists {
		return nil
	}
	
	deps := make([]*DependencyNode, 0, len(node.Dependencies))
	for depID := range node.Dependencies {
		if dep, ok := g.nodes[depID]; ok {
			deps = append(deps, dep)
		}
	}
	
	return deps
}

// GetDependents returns nodes that depend on this node
func (g *DependencyGraph) GetDependents(id string) []*DependencyNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	node, exists := g.nodes[id]
	if !exists {
		return nil
	}
	
	deps := make([]*DependencyNode, 0, len(node.Dependents))
	for depID := range node.Dependents {
		if dep, ok := g.nodes[depID]; ok {
			deps = append(deps, dep)
		}
	}
	
	return deps
}

// MarkCompleted marks a node as completed with result
func (g *DependencyGraph) MarkCompleted(id string, result interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if node, exists := g.nodes[id]; exists {
		node.mu.Lock()
		node.Status = StatusCompleted
		node.Result = result
		node.mu.Unlock()
	}
}

// MarkFailed marks a node as failed with error
func (g *DependencyGraph) MarkFailed(id string, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if node, exists := g.nodes[id]; exists {
		node.mu.Lock()
		node.Status = StatusFailed
		node.Error = err
		node.mu.Unlock()
	}
}

// CanExecute checks if a node is ready to execute
func (g *DependencyGraph) CanExecute(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	node, exists := g.nodes[id]
	if !exists || node.Status != StatusPending {
		return false
	}
	
	// Check if all dependencies are completed
	for depID := range node.Dependencies {
		if dep, ok := g.nodes[depID]; ok {
			dep.mu.RLock()
			status := dep.Status
			dep.mu.RUnlock()
			
			if status != StatusCompleted && status != StatusSkipped {
				return false
			}
		}
	}
	
	return true
}

// GetReadyNodes returns all nodes ready for execution
func (g *DependencyGraph) GetReadyNodes() []*DependencyNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	ready := make([]*DependencyNode, 0)
	
	for id, node := range g.nodes {
		if g.CanExecute(id) {
			ready = append(ready, node)
		}
	}
	
	return ready
}

// Size returns the number of nodes in the graph
func (g *DependencyGraph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// Clear removes all nodes and edges
func (g *DependencyGraph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	g.nodes = make(map[string]*DependencyNode)
	g.edges = make(map[string]map[string]bool)
}

// Visualize returns a string representation of the graph
func (g *DependencyGraph) Visualize() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	var sb strings.Builder
	sb.WriteString("Dependency Graph:\n")
	
	// Sort nodes for consistent output
	sorted, _ := g.TopologicalSort()
	
	for _, node := range sorted {
		sb.WriteString(fmt.Sprintf("  %s [%s] (status: %v)\n", 
			node.ID, node.OperatorType, node.Status))
		
		if len(node.Dependencies) > 0 {
			sb.WriteString("    Dependencies: ")
			first := true
			for depID := range node.Dependencies {
				if !first {
					sb.WriteString(", ")
				}
				sb.WriteString(depID)
				first = false
			}
			sb.WriteString("\n")
		}
		
		if len(node.Dependents) > 0 {
			sb.WriteString("    Dependents: ")
			first := true
			for depID := range node.Dependents {
				if !first {
					sb.WriteString(", ")
				}
				sb.WriteString(depID)
				first = false
			}
			sb.WriteString("\n")
		}
	}
	
	return sb.String()
}

// ExecutionStage represents a group of operations that can run in parallel
type ExecutionStage struct {
	Operations    []*DependencyNode
	CanParallel   bool
	EstimatedTime float64
}

// String returns execution status as string
func (s ExecutionStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusReady:
		return "ready"
	case StatusExecuting:
		return "executing"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}