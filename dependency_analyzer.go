package spruce

import (
	"fmt"
	"regexp"
	"strings"
)

// DependencyAnalyzer analyzes expressions to build dependency graphs
type DependencyAnalyzer struct {
	graph         *DependencyGraph
	pathExtractor *PathExtractor
	costEstimator *CostEstimator
}

// PathExtractor extracts referenced paths from expressions
type PathExtractor struct {
	// Matches grab references like (( grab meta.foo.bar ))
	grabPattern *regexp.Regexp
	// Matches concat references like (( concat "prefix" meta.foo ))
	concatPattern *regexp.Regexp
	// Matches any operator with path references
	pathPattern *regexp.Regexp
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer() *DependencyAnalyzer {
	return &DependencyAnalyzer{
		graph:         NewDependencyGraph(),
		pathExtractor: NewPathExtractor(),
		costEstimator: NewCostEstimator(),
	}
}

// NewPathExtractor creates a new path extractor
func NewPathExtractor() *PathExtractor {
	return &PathExtractor{
		grabPattern:   regexp.MustCompile(`\(\(\s*grab\s+([^\s\)]+)\s*\)\)`),
		concatPattern: regexp.MustCompile(`\(\(\s*concat\s+.*?([a-zA-Z_][a-zA-Z0-9_\.]*)`),
		pathPattern:   regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_\.]*`),
	}
}

// AnalyzeDocument analyzes a document and builds dependency graph
func (da *DependencyAnalyzer) AnalyzeDocument(doc interface{}) (*DependencyGraph, error) {
	da.graph.Clear()
	
	// Walk the document tree
	err := da.walkDocument(doc, []string{})
	if err != nil {
		return nil, err
	}
	
	// Estimate costs for all nodes
	da.estimateCosts()
	
	return da.graph, nil
}

// walkDocument recursively walks the document structure
func (da *DependencyAnalyzer) walkDocument(node interface{}, path []string) error {
	switch v := node.(type) {
	case map[interface{}]interface{}:
		for key, value := range v {
			keyStr := fmt.Sprintf("%v", key)
			newPath := append(path, keyStr)
			if err := da.walkDocument(value, newPath); err != nil {
				return err
			}
		}
		
	case map[string]interface{}:
		for key, value := range v {
			newPath := append(path, key)
			if err := da.walkDocument(value, newPath); err != nil {
				return err
			}
		}
		
	case []interface{}:
		for i, value := range v {
			newPath := append(path, fmt.Sprintf("[%d]", i))
			if err := da.walkDocument(value, newPath); err != nil {
				return err
			}
		}
		
	case string:
		// Check if it's an operator expression
		if da.isOperatorExpression(v) {
			return da.analyzeExpression(v, path)
		}
	}
	
	return nil
}

// isOperatorExpression checks if a string is an operator expression
func (da *DependencyAnalyzer) isOperatorExpression(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "((") && strings.HasSuffix(trimmed, "))")
}

// analyzeExpression analyzes an operator expression
func (da *DependencyAnalyzer) analyzeExpression(expr string, path []string) error {
	nodeID := strings.Join(path, ".")
	opType := da.extractOperatorType(expr)
	
	// Add node to graph
	node := da.graph.AddNode(nodeID, expr, opType, path)
	
	// Extract dependencies
	deps := da.extractDependencies(expr, opType)
	
	// Add dependency edges
	for _, dep := range deps {
		depID := da.resolvePathToID(dep, path)
		
		// Ensure dependency node exists
		if _, exists := da.graph.GetNode(depID); !exists {
			// Create placeholder node for external reference
			da.graph.AddNode(depID, nil, "reference", strings.Split(depID, "."))
		}
		
		// Add dependency edge
		if err := da.graph.AddDependency(depID, nodeID); err != nil {
			// Log circular dependency but continue
			fmt.Printf("Warning: %v\n", err)
		}
	}
	
	// Set initial cost estimate
	node.Cost = da.costEstimator.EstimateOperatorCost(opType, expr)
	
	return nil
}

// extractOperatorType extracts the operator type from expression
func (da *DependencyAnalyzer) extractOperatorType(expr string) string {
	expr = strings.TrimSpace(expr)
	expr = strings.TrimPrefix(expr, "((")
	expr = strings.TrimSuffix(expr, "))")
	expr = strings.TrimSpace(expr)
	
	parts := strings.Fields(expr)
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// extractDependencies extracts path dependencies from expression
func (da *DependencyAnalyzer) extractDependencies(expr string, opType string) []string {
	var deps []string
	
	switch opType {
	case "grab":
		// Extract direct path reference
		matches := da.pathExtractor.grabPattern.FindStringSubmatch(expr)
		if len(matches) > 1 {
			deps = append(deps, matches[1])
		}
		
	case "concat":
		// Extract all path references in concat
		matches := da.pathExtractor.pathPattern.FindAllString(expr, -1)
		for _, match := range matches {
			// Skip operator name and string literals
			if match != "concat" && !strings.HasPrefix(match, "\"") {
				deps = append(deps, match)
			}
		}
		
	case "static_ips", "ips":
		// Extract network and job references
		deps = da.extractStaticIPDeps(expr)
		
	default:
		// Generic path extraction
		matches := da.pathExtractor.pathPattern.FindAllString(expr, -1)
		for _, match := range matches {
			// Skip operator name
			if match != opType && da.looksLikePath(match) {
				deps = append(deps, match)
			}
		}
	}
	
	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0, len(deps))
	for _, dep := range deps {
		if !seen[dep] {
			seen[dep] = true
			unique = append(unique, dep)
		}
	}
	
	return unique
}

// looksLikePath checks if a string looks like a path reference
func (da *DependencyAnalyzer) looksLikePath(s string) bool {
	// Skip common keywords and operators
	keywords := map[string]bool{
		"true": true, "false": true, "nil": true, "null": true,
		"or": true, "and": true, "not": true,
	}
	
	if keywords[s] {
		return false
	}
	
	// Check if it contains dots (path separator)
	return strings.Contains(s, ".") || 
		(len(s) > 2 && !strings.HasPrefix(s, "\"") && !strings.HasSuffix(s, "\""))
}

// extractStaticIPDeps extracts dependencies for static_ips operator
func (da *DependencyAnalyzer) extractStaticIPDeps(expr string) []string {
	deps := []string{"networks", "jobs"}
	
	// Look for specific job references
	if strings.Contains(expr, "jobs.") {
		matches := regexp.MustCompile(`jobs\.[a-zA-Z0-9_]+`).FindAllString(expr, -1)
		deps = append(deps, matches...)
	}
	
	return deps
}

// resolvePathToID resolves a path reference to a node ID
func (da *DependencyAnalyzer) resolvePathToID(refPath string, currentPath []string) string {
	// Handle absolute paths
	if !strings.HasPrefix(refPath, ".") {
		return refPath
	}
	
	// Handle relative paths
	parts := strings.Split(refPath, ".")
	basePath := currentPath
	
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		} else if part == ".." {
			if len(basePath) > 0 {
				basePath = basePath[:len(basePath)-1]
			}
		} else {
			basePath = append(basePath, part)
		}
	}
	
	return strings.Join(basePath, ".")
}

// estimateCosts estimates costs for all nodes
func (da *DependencyAnalyzer) estimateCosts() {
	nodes, _ := da.graph.TopologicalSort()
	
	for _, node := range nodes {
		if node.Cost == 0 {
			node.Cost = da.costEstimator.EstimateNodeCost(node)
		}
	}
}

// GetCriticalPath returns the critical path through the graph
func (da *DependencyAnalyzer) GetCriticalPath() ([]*DependencyNode, float64) {
	sorted, err := da.graph.TopologicalSort()
	if err != nil {
		return nil, 0
	}
	
	// Calculate longest path to each node
	longestPath := make(map[string]float64)
	predecessor := make(map[string]*DependencyNode)
	
	for _, node := range sorted {
		maxCost := float64(0)
		var maxPred *DependencyNode
		
		// Find maximum cost path from dependencies
		for depID := range node.Dependencies {
			if dep, ok := da.graph.GetNode(depID); ok {
				cost := longestPath[depID] + dep.Cost
				if cost > maxCost {
					maxCost = cost
					maxPred = dep
				}
			}
		}
		
		longestPath[node.ID] = maxCost + node.Cost
		if maxPred != nil {
			predecessor[node.ID] = maxPred
		}
	}
	
	// Find node with maximum path cost
	var endNode *DependencyNode
	maxTotalCost := float64(0)
	
	for _, node := range sorted {
		if cost := longestPath[node.ID]; cost > maxTotalCost {
			maxTotalCost = cost
			endNode = node
		}
	}
	
	// Reconstruct critical path
	var criticalPath []*DependencyNode
	current := endNode
	
	for current != nil {
		criticalPath = append([]*DependencyNode{current}, criticalPath...)
		current = predecessor[current.ID]
	}
	
	return criticalPath, maxTotalCost
}

// OptimizeExecution analyzes the graph and suggests optimizations
func (da *DependencyAnalyzer) OptimizeExecution() *ExecutionPlan {
	stages, _ := da.graph.GetExecutionStages()
	
	plan := &ExecutionPlan{
		Stages:      make([]PlannedStage, 0, len(stages)),
		TotalCost:   0,
		Parallelism: 0,
	}
	
	for _, stage := range stages {
		plannedStage := PlannedStage{
			Operations:    stage.Operations,
			CanParallel:   stage.CanParallel,
			EstimatedTime: 0,
		}
		
		// Calculate stage time (max of parallel operations)
		maxTime := float64(0)
		for _, op := range stage.Operations {
			if op.Cost > maxTime {
				maxTime = op.Cost
			}
		}
		
		plannedStage.EstimatedTime = maxTime
		plan.TotalCost += maxTime
		
		if len(stage.Operations) > plan.Parallelism {
			plan.Parallelism = len(stage.Operations)
		}
		
		plan.Stages = append(plan.Stages, plannedStage)
	}
	
	return plan
}

// ExecutionPlan represents an optimized execution plan
type ExecutionPlan struct {
	Stages      []PlannedStage
	TotalCost   float64
	Parallelism int
}

// PlannedStage represents a stage in the execution plan
type PlannedStage struct {
	Operations    []*DependencyNode
	CanParallel   bool
	EstimatedTime float64
	BatchGroups   map[string][]*DependencyNode
}