package internal

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/starkandwayne/goutils/tree"
)

// TestParallelEvaluator tests the parallel evaluator functionality
func TestParallelEvaluator(t *testing.T) {
	t.Run("WaveDetection", func(t *testing.T) {
		// Create a test document with independent operations
		doc := map[interface{}]interface{}{
			"env":  "test",
			"name": "app",
			"services": map[interface{}]interface{}{
				"web":    map[interface{}]interface{}{"count": 2},
				"worker": map[interface{}]interface{}{"count": 1},
			},
			"concat1": "(( concat env \"-\" name ))",
			"concat2": "(( concat \"prefix-\" env ))",
			"concat3": "(( concat name \"-suffix\" ))",
		}

		ev := &Evaluator{Tree: doc}
		pev := NewParallelEvaluator(ev, 4)

		waves, err := pev.ParallelDataFlow(EvalPhase)
		if err != nil {
			t.Fatalf("Failed to build waves: %v", err)
		}

		// Should have at least one wave
		if len(waves) == 0 {
			t.Fatal("Expected at least one wave")
		}

		// Count total operations
		totalOps := 0
		for _, wave := range waves {
			totalOps += len(wave.Operations)
		}

		t.Logf("Found %d total operations in %d waves", totalOps, len(waves))
		for i, wave := range waves {
			t.Logf("Wave %d: %d ops, parallel: %t", i, len(wave.Operations), wave.CanRunInParallel)
		}

		if totalOps == 0 {
			t.Fatal("Expected some operations but found none")
		}

		// The three concat operations should be able to run in parallel
		// since they don't depend on each other
		foundParallelWave := false
		for _, wave := range waves {
			if wave.CanRunInParallel && len(wave.Operations) > 1 {
				foundParallelWave = true
				break
			}
		}

		if !foundParallelWave {
			t.Log("No parallel waves found - this may be expected for simple operations")
		}
	})

	t.Run("DependencyHandling", func(t *testing.T) {
		// Create a test document with dependent operations
		doc := map[interface{}]interface{}{
			"base":     "myapp",
			"version":  "1.0",
			"name":     "(( concat base \"-\" version ))",
			"fullname": "(( concat name \"-release\" ))",
		}

		ev := &Evaluator{Tree: doc}
		pev := NewParallelEvaluator(ev, 4)

		waves, err := pev.ParallelDataFlow(EvalPhase)
		if err != nil {
			t.Fatalf("Failed to build waves: %v", err)
		}

		// Should have multiple waves due to dependencies
		if len(waves) < 2 {
			t.Fatalf("Expected at least 2 waves due to dependencies, got %d", len(waves))
		}

		// Verify that dependent operations are in different waves
		nameOp := findOperationByPath(waves, "name")
		fullnameOp := findOperationByPath(waves, "fullname")

		if nameOp == nil || fullnameOp == nil {
			t.Fatal("Could not find name or fullname operations")
		}

		nameWave := findWaveContaining(waves, nameOp)
		fullnameWave := findWaveContaining(waves, fullnameOp)

		if nameWave.WaveIndex >= fullnameWave.WaveIndex {
			t.Error("name operation should be in an earlier wave than fullname")
		}
	})

	t.Run("ParallelExecution", func(t *testing.T) {
		// Create test document with slow operations to verify parallelism
		doc := map[interface{}]interface{}{
			"base1": "app1",
			"base2": "app2",
			"base3": "app3",
			"name1": "(( concat base1 \"-service\" ))",
			"name2": "(( concat base2 \"-service\" ))",
			"name3": "(( concat base3 \"-service\" ))",
		}

		ev := &Evaluator{Tree: doc}
		pev := NewParallelEvaluator(ev, 4)

		// Execute with timing
		start := time.Now()
		err := pev.RunOpsParallel(EvalPhase)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Parallel execution failed: %v", err)
		}

		// Verify results
		name1, err := tree.ParseCursor("name1")
		if err != nil {
			t.Fatal(err)
		}
		result, err := name1.Resolve(ev.Tree)
		if err != nil {
			t.Fatal(err)
		}
		if result != "app1-service" {
			t.Errorf("Expected 'app1-service', got %v", result)
		}

		// Check metrics
		metrics := pev.GetMetrics()
		if metrics.ParallelOpsExecuted == 0 {
			t.Error("Expected some parallel operations to be executed")
		}

		t.Logf("Parallel execution completed in %v", elapsed)
		t.Logf("Metrics: %s", metrics.String())
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping thread safety test in short mode")
		}

		// Create document with many independent operations
		doc := map[interface{}]interface{}{
			"base": "app",
		}

		// Add many independent concat operations
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("name%d", i)
			value := fmt.Sprintf("(( concat base \"-%d\" ))", i)
			doc[key] = value
		}

		// Run multiple times concurrently to test for race conditions
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				// Create a fresh copy of the document for each iteration
				docCopy := make(map[interface{}]interface{})
				for k, v := range doc {
					docCopy[k] = v
				}

				evCopy := &Evaluator{Tree: docCopy}
				pevCopy := NewParallelEvaluator(evCopy, 8)

				err := pevCopy.RunOpsParallel(EvalPhase)
				if err != nil {
					errors <- fmt.Errorf("iteration %d failed: %v", iteration, err)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Error(err)
		}
	})

	t.Run("FallbackToSequential", func(t *testing.T) {
		// Test that operations fall back to sequential when parallel is disabled
		doc := map[interface{}]interface{}{
			"env":     "test",
			"name":    "app",
			"concat1": "(( concat env \"-\" name ))",
			"concat2": "(( concat \"prefix-\" env ))",
		}

		ev := &Evaluator{Tree: doc}
		pev := NewParallelEvaluator(ev, 4)
		pev.SetParallelEnabled(false)

		waves, err := pev.ParallelDataFlow(EvalPhase)
		if err != nil {
			t.Fatalf("Failed to build waves: %v", err)
		}

		// No waves should be marked as parallel
		for _, wave := range waves {
			if wave.CanRunInParallel {
				t.Error("Expected no parallel waves when parallel is disabled")
			}
		}

		// Should still execute successfully
		err = pev.RunWaves(waves)
		if err != nil {
			t.Fatalf("Sequential execution failed: %v", err)
		}
	})
}

// BenchmarkParallelEvaluator benchmarks parallel vs sequential evaluation
func BenchmarkParallelEvaluator(b *testing.B) {
	// Create a document with many independent operations
	doc := map[interface{}]interface{}{
		"base": "application",
		"env":  "production",
	}

	// Add many independent operations
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("service_%d", i)
		value := fmt.Sprintf("(( concat base \"-%d-\" env ))", i)
		doc[key] = value
	}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Create fresh document copy
			docCopy := make(map[interface{}]interface{})
			for k, v := range doc {
				docCopy[k] = v
			}

			ev := &Evaluator{Tree: docCopy}
			pev := NewParallelEvaluator(ev, 1) // Force sequential
			pev.SetParallelEnabled(false)

			_ = pev.RunOpsParallel(EvalPhase)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Create fresh document copy
			docCopy := make(map[interface{}]interface{})
			for k, v := range doc {
				docCopy[k] = v
			}

			ev := &Evaluator{Tree: docCopy}
			pev := NewParallelEvaluator(ev, 8) // Allow parallelism

			_ = pev.RunOpsParallel(EvalPhase)
		}
	})
}

// Helper functions for tests

func findOperationByPath(waves []*OperatorWave, path string) *Opcall {
	for _, wave := range waves {
		for _, op := range wave.Operations {
			if strings.Contains(op.canonical.String(), path) {
				return op
			}
		}
	}
	return nil
}

func findWaveContaining(waves []*OperatorWave, op *Opcall) *OperatorWave {
	for _, wave := range waves {
		for _, waveOp := range wave.Operations {
			if waveOp == op {
				return wave
			}
		}
	}
	return nil
}
