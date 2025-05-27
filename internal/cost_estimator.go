package internal

import (
	"strings"
	"time"
)

// CostEstimator estimates execution costs for operations
type CostEstimator struct {
	operatorCosts   map[string]float64
	externalCosts   map[string]float64
	cacheMissWeight float64
	parallelBonus   float64
	metricsEnabled  bool
}

// NewCostEstimator creates a new cost estimator
func NewCostEstimator() *CostEstimator {
	return &CostEstimator{
		operatorCosts:   defaultOperatorCosts(),
		externalCosts:   defaultExternalCosts(),
		cacheMissWeight: 2.0,
		parallelBonus:   0.8,
		metricsEnabled:  MetricsEnabled(),
	}
}

// defaultOperatorCosts returns default cost estimates for operators
func defaultOperatorCosts() map[string]float64 {
	return map[string]float64{
		// Simple operators
		"grab":    0.1,
		"param":   0.1,
		"static":  0.1,
		"literal": 0.05,

		// String operators
		"concat":  0.2,
		"join":    0.3,
		"split":   0.2,
		"replace": 0.3,

		// Math operators
		"calc":     0.5,
		"add":      0.1,
		"subtract": 0.1,
		"multiply": 0.1,
		"divide":   0.1,
		"modulo":   0.1,

		// Complex operators
		"static_ips": 2.0,
		"ips":        2.0,
		"cartesian":  3.0,
		"inject":     1.0,
		"defer":      0.5,
		"prune":      0.3,

		// External operators
		"vault":      10.0,
		"file":       5.0,
		"awsparam":   15.0,
		"awssecret":  15.0,
		"env":        0.2,
		"envdefault": 0.2,

		// Default
		"unknown": 1.0,
	}
}

// defaultExternalCosts returns default costs for external systems
func defaultExternalCosts() map[string]float64 {
	return map[string]float64{
		"vault":   50.0,  // 50ms average
		"file":    10.0,  // 10ms for file I/O
		"aws":     100.0, // 100ms for AWS calls
		"network": 30.0,  // 30ms for network calls
		"disk":    5.0,   // 5ms for disk cache
	}
}

// EstimateOperatorCost estimates cost for an operator expression
func (ce *CostEstimator) EstimateOperatorCost(opType string, expr string) float64 {
	baseCost, exists := ce.operatorCosts[opType]
	if !exists {
		baseCost = ce.operatorCosts["unknown"]
	}

	// Adjust for expression complexity
	complexity := ce.estimateComplexity(expr)
	cost := baseCost * complexity

	// Add external cost if applicable
	if ce.isExternalOperator(opType) {
		cost += ce.externalCosts[ce.getExternalSystem(opType)]
	}

	return cost
}

// EstimateNodeCost estimates cost for a dependency node
func (ce *CostEstimator) EstimateNodeCost(node *DependencyNode) float64 {
	if node.Cost > 0 {
		return node.Cost
	}

	// Use actual metrics if available
	if ce.metricsEnabled {
		if avgTime := ce.getAverageExecutionTime(node.OperatorType); avgTime > 0 {
			return avgTime
		}
	}

	// Fall back to estimates
	return ce.EstimateOperatorCost(node.OperatorType, "")
}

// estimateComplexity estimates expression complexity multiplier
func (ce *CostEstimator) estimateComplexity(expr string) float64 {
	complexity := 1.0

	// Adjust for expression length
	if len(expr) > 100 {
		complexity *= 1.2
	}
	if len(expr) > 500 {
		complexity *= 1.5
	}

	// Adjust for nested expressions
	openCount := strings.Count(expr, "((")
	if openCount > 1 {
		complexity *= 1.0 + float64(openCount-1)*0.2
	}

	// Adjust for list operations
	if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
		complexity *= 1.3
	}

	return complexity
}

// isExternalOperator checks if operator requires external calls
func (ce *CostEstimator) isExternalOperator(opType string) bool {
	external := map[string]bool{
		"vault":     true,
		"file":      true,
		"awsparam":  true,
		"awssecret": true,
	}
	return external[opType]
}

// getExternalSystem returns the external system for an operator
func (ce *CostEstimator) getExternalSystem(opType string) string {
	switch opType {
	case "vault":
		return "vault"
	case "file":
		return "file"
	case "awsparam", "awssecret":
		return "aws"
	default:
		return "network"
	}
}

// getAverageExecutionTime gets average execution time from metrics
func (ce *CostEstimator) getAverageExecutionTime(opType string) float64 {
	if !ce.metricsEnabled {
		return 0
	}

	// Get timing aggregator
	aggregator := GetTimingAggregator()
	if aggregator == nil {
		return 0
	}

	// Look up operator timing stats
	stats, exists := aggregator.GetStats("operator_" + opType)
	if !exists || stats.Count == 0 {
		return 0
	}

	// Return mean time in cost units (1 unit = 1ms)
	return float64(stats.MeanTime.Milliseconds())
}

// EstimateBatchCost estimates cost for a batch of operations
func (ce *CostEstimator) EstimateBatchCost(operations []*DependencyNode) float64 {
	if len(operations) == 0 {
		return 0
	}

	// Group by operator type
	groups := make(map[string][]*DependencyNode)
	for _, op := range operations {
		groups[op.OperatorType] = append(groups[op.OperatorType], op)
	}

	totalCost := float64(0)

	for opType, ops := range groups {
		if ce.canBatch(opType) {
			// Batched cost: base + incremental per operation
			baseCost := ce.operatorCosts[opType]
			batchCost := baseCost + float64(len(ops)-1)*baseCost*0.1

			if ce.isExternalOperator(opType) {
				// External calls benefit more from batching
				extCost := ce.externalCosts[ce.getExternalSystem(opType)]
				batchCost = extCost + float64(len(ops)-1)*extCost*0.05
			}

			totalCost += batchCost
		} else {
			// Non-batchable: sum individual costs
			for _, op := range ops {
				totalCost += op.Cost
			}
		}
	}

	return totalCost
}

// canBatch checks if an operator type supports batching
func (ce *CostEstimator) canBatch(opType string) bool {
	batchable := map[string]bool{
		"vault":      true,
		"file":       true,
		"awsparam":   true,
		"awssecret":  true,
		"grab":       false, // Usually can't batch grabs
		"static_ips": false, // Complex operator
	}

	supported, exists := batchable[opType]
	return exists && supported
}

// EstimateParallelCost estimates cost when operations run in parallel
func (ce *CostEstimator) EstimateParallelCost(operations []*DependencyNode) float64 {
	if len(operations) == 0 {
		return 0
	}

	// Parallel cost is the maximum cost of any operation
	maxCost := float64(0)
	for _, op := range operations {
		if op.Cost > maxCost {
			maxCost = op.Cost
		}
	}

	// Apply parallel bonus for coordination overhead
	return maxCost * ce.parallelBonus
}

// UpdateCosts updates cost estimates based on actual execution
func (ce *CostEstimator) UpdateCosts(opType string, actualTime time.Duration) {
	// Exponential moving average
	alpha := 0.2
	actualCost := float64(actualTime.Milliseconds())

	if currentCost, exists := ce.operatorCosts[opType]; exists {
		ce.operatorCosts[opType] = currentCost*(1-alpha) + actualCost*alpha
	} else {
		ce.operatorCosts[opType] = actualCost
	}
}

// GetCostBreakdown returns a cost breakdown for analysis
func (ce *CostEstimator) GetCostBreakdown(node *DependencyNode) CostBreakdown {
	baseCost := ce.operatorCosts[node.OperatorType]
	externalCost := float64(0)

	if ce.isExternalOperator(node.OperatorType) {
		externalCost = ce.externalCosts[ce.getExternalSystem(node.OperatorType)]
	}

	return CostBreakdown{
		BaseCost:      baseCost,
		ExternalCost:  externalCost,
		CacheMissCost: baseCost * ce.cacheMissWeight,
		TotalCost:     node.Cost,
	}
}

// CostBreakdown represents detailed cost breakdown
type CostBreakdown struct {
	BaseCost      float64
	ExternalCost  float64
	CacheMissCost float64
	TotalCost     float64
}

// OptimizeCosts suggests cost optimizations
func (ce *CostEstimator) OptimizeCosts(graph *DependencyGraph) []CostOptimization {
	optimizations := make([]CostOptimization, 0)

	// Analyze all nodes
	nodeIDs, _ := graph.TopologicalSort()
	allNodes := graph.GetNodes()

	// Find expensive operations
	for _, nodeID := range nodeIDs {
		if node, ok := allNodes[nodeID]; ok && node.Cost > 10 { // Expensive threshold
			opt := CostOptimization{
				NodeID:      nodeID,
				CurrentCost: node.Cost,
				Suggestion:  ce.getSuggestion(node),
			}

			// Estimate savings
			if ce.canCache(node.OperatorType) {
				opt.PotentialSavings = node.Cost * 0.9 // 90% savings with cache
				opt.Type = "caching"
			} else if ce.canBatch(node.OperatorType) {
				opt.PotentialSavings = node.Cost * 0.7 // 70% savings with batching
				opt.Type = "batching"
			}

			optimizations = append(optimizations, opt)
		}
	}

	return optimizations
}

// CostOptimization represents a cost optimization suggestion
type CostOptimization struct {
	NodeID           string
	Type             string
	CurrentCost      float64
	PotentialSavings float64
	Suggestion       string
}

// canCache checks if results can be cached
func (ce *CostEstimator) canCache(opType string) bool {
	// Most operators can be cached except volatile ones
	uncacheable := map[string]bool{
		"env":    true, // Environment can change
		"random": true, // Random values
	}
	return !uncacheable[opType]
}

// getSuggestion returns optimization suggestion for a node
func (ce *CostEstimator) getSuggestion(node *DependencyNode) string {
	switch node.OperatorType {
	case "vault":
		return "Consider caching vault responses or batching multiple vault calls"
	case "file":
		return "Consider caching file contents or using memory-mapped files"
	case "awsparam", "awssecret":
		return "Batch AWS parameter/secret fetches and cache results"
	case "static_ips":
		return "Pre-calculate static IPs during initialization"
	default:
		if node.Cost > 50 {
			return "This operation is expensive; consider caching or optimization"
		}
		return "Monitor performance and consider optimization if needed"
	}
}
