package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// MockOperator implements graft.Operator for testing
type MockOperator struct {
	name       string
	runFunc    func(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error)
	execCount  int64
	concurrent bool
}

func (m *MockOperator) Setup() error {
	return nil
}

func (m *MockOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	atomic.AddInt64(&m.execCount, 1)
	if m.runFunc != nil {
		return m.runFunc(ev, args)
	}
	return &graft.Response{
		Type:  graft.Replace,
		Value: fmt.Sprintf("mock_result_%s", m.name),
	}, nil
}

func (m *MockOperator) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

func (m *MockOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

func (m *MockOperator) GetExecutionCount() int64 {
	return atomic.LoadInt64(&m.execCount)
}

// TestThreadSafeEvaluatorImpl_Creation tests evaluator creation
func TestThreadSafeEvaluatorImpl_Creation(t *testing.T) {
	tests := []struct {
		name     string
		treeData map[interface{}]interface{}
	}{
		{
			name:     "create with empty tree",
			treeData: make(map[interface{}]interface{}),
		},
		{
			name: "create with data",
			treeData: map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"nested": "value2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewSafeTree(tt.treeData)
			evaluator := NewThreadSafeEvaluator(tree)

			if evaluator == nil {
				t.Fatal("expected evaluator to be created")
			}
			if evaluator.safeTree == nil {
				t.Error("expected safe tree to be set")
			}
			if evaluator.originalEv == nil {
				t.Error("expected original evaluator to be set")
			}
			if evaluator.progress == nil {
				t.Error("expected progress to be initialized")
			}
		})
	}
}

// TestThreadSafeEvaluatorImpl_Evaluate tests the evaluate functionality
func TestThreadSafeEvaluatorImpl_Evaluate(t *testing.T) {
	t.Run("successful evaluation", func(t *testing.T) {
		data := map[interface{}]interface{}{
			"test":   "value",
			"number": 42,
		}
		tree := NewSafeTree(data)
		evaluator := NewThreadSafeEvaluator(tree)

		ctx := context.Background()
		err := evaluator.Evaluate(ctx)

		if err != nil {
			t.Errorf("expected successful evaluation, got error: %v", err)
		}

		progress := evaluator.Progress()
		if progress.Completed != 1 {
			t.Errorf("expected 1 completed, got %d", progress.Completed)
		}
		if progress.Failed != 0 {
			t.Errorf("expected 0 failed, got %d", progress.Failed)
		}
	})

	t.Run("concurrent evaluations", func(t *testing.T) {
		data := map[interface{}]interface{}{
			"shared": "resource",
		}
		tree := NewSafeTree(data)
		evaluator := NewThreadSafeEvaluator(tree)

		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make([]error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ctx := context.Background()
				errors[idx] = evaluator.Evaluate(ctx)
			}(i)
		}

		wg.Wait()

		// Check for errors
		for i, err := range errors {
			if err != nil {
				t.Errorf("goroutine %d failed with error: %v", i, err)
			}
		}
	})
}

// TestThreadSafeEvaluatorImpl_ExecuteOperator tests operator execution
func TestThreadSafeEvaluatorImpl_ExecuteOperator(t *testing.T) {
	tree := NewSafeTree(make(map[interface{}]interface{}))
	evaluator := NewThreadSafeEvaluator(tree)

	t.Run("execute mock operator", func(t *testing.T) {
		mockOp := &MockOperator{
			name: "test_op",
			runFunc: func(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
				return &graft.Response{
					Type:  graft.Replace,
					Value: "executed",
				}, nil
			},
		}

		ctx := context.Background()
		result, err := evaluator.ExecuteOperator(ctx, mockOp, []interface{}{"arg1", "arg2"})

		if err != nil {
			t.Errorf("expected successful execution, got error: %v", err)
		}
		if result != "executed" {
			t.Errorf("expected 'executed' result, got %v", result)
		}
		if mockOp.GetExecutionCount() != 1 {
			t.Errorf("expected operator to be executed once, got %d", mockOp.GetExecutionCount())
		}
	})

	t.Run("execute operator with error", func(t *testing.T) {
		mockOp := &MockOperator{
			name: "error_op",
			runFunc: func(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
				return nil, fmt.Errorf("operator error")
			},
		}

		ctx := context.Background()
		_, err := evaluator.ExecuteOperator(ctx, mockOp, []interface{}{})

		if err == nil {
			t.Error("expected error from operator")
		}
		if err.Error() != "operator error" {
			t.Errorf("expected 'operator error', got %v", err)
		}
	})

	t.Run("concurrent operator execution", func(t *testing.T) {
		mockOp := &MockOperator{
			name:       "concurrent_op",
			concurrent: true,
		}

		const numGoroutines = 20
		var wg sync.WaitGroup
		results := make([]interface{}, numGoroutines)
		errors := make([]error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ctx := context.Background()
				results[idx], errors[idx] = evaluator.ExecuteOperator(ctx, mockOp, []interface{}{idx})
			}(i)
		}

		wg.Wait()

		// Check results
		for i, err := range errors {
			if err != nil {
				t.Errorf("goroutine %d failed: %v", i, err)
			}
		}

		expectedExecCount := int64(numGoroutines)
		if mockOp.GetExecutionCount() != expectedExecCount {
			t.Errorf("expected %d executions, got %d", expectedExecCount, mockOp.GetExecutionCount())
		}
	})
}

// TestThreadSafeEvaluatorImpl_Progress tests progress tracking
func TestThreadSafeEvaluatorImpl_Progress(t *testing.T) {
	tree := NewSafeTree(make(map[interface{}]interface{}))
	evaluator := NewThreadSafeEvaluator(tree)

	t.Run("initial progress", func(t *testing.T) {
		progress := evaluator.Progress()

		if progress.Total != 0 {
			t.Errorf("expected 0 total, got %d", progress.Total)
		}
		if progress.Completed != 0 {
			t.Errorf("expected 0 completed, got %d", progress.Completed)
		}
		if progress.Failed != 0 {
			t.Errorf("expected 0 failed, got %d", progress.Failed)
		}
		if progress.InProgress != 0 {
			t.Errorf("expected 0 in progress, got %d", progress.InProgress)
		}
	})

	t.Run("concurrent progress access", func(t *testing.T) {
		const numReaders = 10
		var wg sync.WaitGroup

		// Start readers
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = evaluator.Progress()
					time.Sleep(time.Microsecond)
				}
			}()
		}

		// Simulate progress updates
		go func() {
			for j := 0; j < 50; j++ {
				evaluator.updateProgress(func(p *EvaluationProgress) {
					p.Completed++
				})
				time.Sleep(time.Microsecond * 10)
			}
		}()

		wg.Wait()
	})
}

// TestThreadSafeEvaluatorImpl_Subscribe tests listener subscription
func TestThreadSafeEvaluatorImpl_Subscribe(t *testing.T) {
	tree := NewSafeTree(make(map[interface{}]interface{}))
	evaluator := NewThreadSafeEvaluator(tree)

	t.Run("subscribe and notify", func(t *testing.T) {
		progressCount := int64(0)

		listener := &SimpleEvaluationListener{
			OnProgressFunc: func(progress EvaluationProgress) {
				atomic.AddInt64(&progressCount, 1)
			},
		}

		unsubscribe := evaluator.Subscribe(listener)

		// Trigger progress notification
		evaluator.notifyProgress()

		// Wait a bit for async notification
		time.Sleep(time.Millisecond * 10)

		if atomic.LoadInt64(&progressCount) != 1 {
			t.Errorf("expected 1 progress notification, got %d", progressCount)
		}

		// Unsubscribe
		unsubscribe()

		// Trigger another notification
		evaluator.notifyProgress()
		time.Sleep(time.Millisecond * 10)

		// Count should not increase
		if atomic.LoadInt64(&progressCount) != 1 {
			t.Errorf("expected no new notifications after unsubscribe, got %d", progressCount)
		}
	})

	t.Run("multiple listeners", func(t *testing.T) {
		counts := make([]int64, 3)
		listeners := make([]EvaluationListener, 3)
		unsubscribes := make([]func(), 3)

		for i := range listeners {
			idx := i
			listeners[i] = &SimpleEvaluationListener{
				OnProgressFunc: func(progress EvaluationProgress) {
					atomic.AddInt64(&counts[idx], 1)
				},
			}
			unsubscribes[i] = evaluator.Subscribe(listeners[i])
		}

		// Notify all
		evaluator.notifyProgress()
		time.Sleep(time.Millisecond * 10)

		for i, count := range counts {
			if atomic.LoadInt64(&count) != 1 {
				t.Errorf("listener %d: expected 1 notification, got %d", i, count)
			}
		}

		// Unsubscribe middle listener
		unsubscribes[1]()

		// Notify again
		evaluator.notifyProgress()
		time.Sleep(time.Millisecond * 10)

		expected := []int64{2, 1, 2}
		for i, count := range counts {
			if atomic.LoadInt64(&count) != expected[i] {
				t.Errorf("listener %d: expected %d notifications, got %d", i, expected[i], count)
			}
		}
	})
}

// TestThreadSafeOperatorAdapter tests the operator adapter
func TestThreadSafeOperatorAdapter(t *testing.T) {
	t.Run("basic operation", func(t *testing.T) {
		mockOp := &MockOperator{name: "base_op"}
		adapter := NewThreadSafeOperatorAdapter(mockOp)

		ev := &graft.Evaluator{}
		args := []*graft.Expr{
			{Type: graft.Literal, Literal: "test"},
		}

		result, err := adapter.Run(ev, args)

		if err != nil {
			t.Errorf("expected successful run, got error: %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if result.Type != graft.Replace {
			t.Errorf("expected Replace type, got %v", result.Type)
		}
	})

	t.Run("caching for read-only operators", func(t *testing.T) {
		callCount := int64(0)
		mockOp := &MockOperator{
			name: "cached_op",
			runFunc: func(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
				atomic.AddInt64(&callCount, 1)
				return &graft.Response{
					Type:  graft.Replace,
					Value: "cached_result",
				}, nil
			},
		}

		adapter := NewThreadSafeOperatorAdapter(mockOp)
		// Override cache TTL for testing
		adapter.cacheTTL = time.Hour

		ev := &graft.Evaluator{}
		args := []*graft.Expr{
			{Type: graft.Literal, Literal: "test"},
		}

		// First call
		result1, err1 := adapter.Run(ev, args)
		if err1 != nil {
			t.Errorf("first call failed: %v", err1)
		}

		// Second call (should use cache if operator is read-only)
		result2, err2 := adapter.Run(ev, args)
		if err2 != nil {
			t.Errorf("second call failed: %v", err2)
		}

		if result1.Value != result2.Value {
			t.Error("expected same results from cached calls")
		}
	})

	t.Run("concurrent execution", func(t *testing.T) {
		mockOp := &MockOperator{
			name: "concurrent_adapter_op",
			runFunc: func(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
				// Simulate some work
				time.Sleep(time.Millisecond)
				return &graft.Response{
					Type:  graft.Replace,
					Value: fmt.Sprintf("result_%v", args),
				}, nil
			},
		}

		adapter := NewThreadSafeOperatorAdapter(mockOp)

		const numGoroutines = 20
		var wg sync.WaitGroup
		results := make([]*graft.Response, numGoroutines)
		errors := make([]error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ev := &graft.Evaluator{}
				args := []*graft.Expr{
					{Type: graft.Literal, Literal: idx},
				}

				results[idx], errors[idx] = adapter.Run(ev, args)
			}(i)
		}

		wg.Wait()

		// Verify all operations completed
		for i, err := range errors {
			if err != nil {
				t.Errorf("goroutine %d failed: %v", i, err)
			}
			if results[i] == nil {
				t.Errorf("goroutine %d got nil result", i)
			}
		}
	})
}

// TestMigrationHelper tests the migration helper functionality
func TestMigrationHelper(t *testing.T) {
	t.Run("create migration helper", func(t *testing.T) {
		data := map[interface{}]interface{}{
			"key": "value",
		}

		helper := NewMigrationHelper(data)

		if helper == nil {
			t.Fatal("expected helper to be created")
		}
		if helper.safeTree == nil {
			t.Error("expected safe tree to be created")
		}
		if helper.tsEval == nil {
			t.Error("expected thread-safe evaluator to be created")
		}
	})

	t.Run("get thread-safe evaluator", func(t *testing.T) {
		data := map[interface{}]interface{}{}
		helper := NewMigrationHelper(data)

		tsEval := helper.GetThreadSafeEvaluator()

		if tsEval == nil {
			t.Fatal("expected thread-safe evaluator")
		}
		if tsEval != helper.tsEval {
			t.Error("expected same evaluator instance")
		}
	})

	t.Run("migrate existing evaluator", func(t *testing.T) {
		data := map[interface{}]interface{}{}
		helper := NewMigrationHelper(data)

		// Create original evaluator with custom settings
		originalEv := &graft.Evaluator{
			Tree:     data,
			SkipEval: true,
			CheckOps: []*graft.Opcall{}, // Empty for testing
			Only:     []string{"path1", "path2"},
		}

		migratedEval := helper.MigrateEvaluator(originalEv)

		if migratedEval == nil {
			t.Fatal("expected migrated evaluator")
		}
		if migratedEval.originalEv.SkipEval != true {
			t.Error("expected SkipEval to be preserved")
		}
		if len(migratedEval.originalEv.Only) != 2 {
			t.Error("expected Only paths to be preserved")
		}
	})

	t.Run("get compatible evaluator", func(t *testing.T) {
		data := map[interface{}]interface{}{}
		helper := NewMigrationHelper(data)

		compatEval := helper.GetCompatibleEvaluator()

		if compatEval == nil {
			t.Fatal("expected compatible evaluator")
		}
		if compatEval != helper.tsEval.originalEv {
			t.Error("expected original evaluator from thread-safe wrapper")
		}
	})
}

// Benchmarks

func BenchmarkThreadSafeEvaluator_Evaluate(b *testing.B) {
	data := map[interface{}]interface{}{
		"test": "value",
		"nested": map[interface{}]interface{}{
			"key": "value",
		},
	}
	tree := NewSafeTree(data)
	evaluator := NewThreadSafeEvaluator(tree)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = evaluator.Evaluate(ctx)
	}
}

func BenchmarkThreadSafeEvaluator_ConcurrentEvaluate(b *testing.B) {
	data := map[interface{}]interface{}{
		"test": "value",
	}
	tree := NewSafeTree(data)
	evaluator := NewThreadSafeEvaluator(tree)
	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = evaluator.Evaluate(ctx)
		}
	})
}

func BenchmarkThreadSafeOperatorAdapter_Run(b *testing.B) {
	mockOp := &MockOperator{name: "bench_op"}
	adapter := NewThreadSafeOperatorAdapter(mockOp)
	ev := &graft.Evaluator{}
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: "test"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = adapter.Run(ev, args)
	}
}

func BenchmarkThreadSafeOperatorAdapter_ConcurrentRun(b *testing.B) {
	mockOp := &MockOperator{name: "bench_concurrent_op"}
	adapter := NewThreadSafeOperatorAdapter(mockOp)
	ev := &graft.Evaluator{}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			args := []*graft.Expr{
				{Type: graft.Literal, Literal: i},
			}
			_, _ = adapter.Run(ev, args)
			i++
		}
	})
}
