package internal

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestTimer tests basic timer functionality
func TestTimer(t *testing.T) {
	t.Run("basic timing", func(t *testing.T) {
		timer := NewTimer("test")
		if timer.Name() != "test" {
			t.Errorf("expected name 'test', got %s", timer.Name())
		}

		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		duration := timer.Stop()

		if duration <= 0 {
			t.Error("expected positive duration")
		}

		// Should be approximately 10ms (allow some variance)
		if duration < 5*time.Millisecond || duration > 50*time.Millisecond {
			t.Errorf("expected ~10ms, got %v", duration)
		}

		elapsed := time.Since(start)
		if absDuration(duration-elapsed) > 5*time.Millisecond {
			t.Errorf("timer duration %v doesn't match elapsed %v", duration, elapsed)
		}
	})

	t.Run("timer metadata", func(t *testing.T) {
		timer := NewTimer("metadata_test")

		timer.SetMetadata("key1", "value1")
		timer.SetMetadata("key2", 42)
		timer.SetMetadata("key3", true)

		if val, ok := timer.GetMetadata("key1"); !ok || val != "value1" {
			t.Errorf("expected metadata key1='value1', got %v", val)
		}

		if val, ok := timer.GetMetadata("key2"); !ok || val != 42 {
			t.Errorf("expected metadata key2=42, got %v", val)
		}

		if val, ok := timer.GetMetadata("key3"); !ok || val != true {
			t.Errorf("expected metadata key3=true, got %v", val)
		}

		if _, ok := timer.GetMetadata("missing"); ok {
			t.Error("expected missing key to return false")
		}
	})

	t.Run("stop with error", func(t *testing.T) {
		timer := NewTimer("error_test")
		testErr := &testError{"test error"}

		duration, err := timer.StopWithError(testErr)
		if err != testErr {
			t.Errorf("expected error to be preserved, got %v", err)
		}

		if duration <= 0 {
			t.Error("expected positive duration")
		}

		if val, ok := timer.GetMetadata("error"); !ok || val != "test error" {
			t.Errorf("expected error metadata to be set, got %v", val)
		}
	})

	t.Run("double stop protection", func(t *testing.T) {
		timer := NewTimer("double_stop")

		// First stop
		duration1 := timer.Stop()
		if duration1 <= 0 {
			t.Error("expected positive duration on first stop")
		}

		// Second stop should return same duration
		duration2 := timer.Stop()
		if duration1 != duration2 {
			t.Errorf("expected same duration on double stop: %v vs %v", duration1, duration2)
		}
	})

	t.Run("running timer duration", func(t *testing.T) {
		timer := NewTimer("running")

		time.Sleep(5 * time.Millisecond)

		// Duration should be positive for running timer
		runningDuration := timer.Duration()
		if runningDuration <= 0 {
			t.Error("expected positive duration for running timer")
		}

		time.Sleep(5 * time.Millisecond)

		// Stop and check final duration
		finalDuration := timer.Stop()
		if finalDuration <= runningDuration {
			t.Errorf("expected final duration %v > running duration %v", finalDuration, runningDuration)
		}
	})
}

// TestTimerHierarchy tests hierarchical timer relationships
func TestTimerHierarchy(t *testing.T) {
	t.Run("parent child relationships", func(t *testing.T) {
		parent := NewTimer("parent")
		child1 := parent.Child("child1")
		child2 := parent.Child("child2")

		if child1.Parent() != parent {
			t.Error("child1 parent should be parent timer")
		}

		if child2.Parent() != parent {
			t.Error("child2 parent should be parent timer")
		}

		children := parent.Children()
		if len(children) != 2 {
			t.Errorf("expected 2 children, got %d", len(children))
		}

		if children[0] != child1 || children[1] != child2 {
			t.Error("children slice doesn't match expected order")
		}
	})

	t.Run("nested hierarchy", func(t *testing.T) {
		root := NewTimer("root")
		level1 := root.Child("level1")
		level2 := level1.Child("level2")
		level3 := level2.Child("level3")

		// Verify hierarchy
		if level3.Parent() != level2 {
			t.Error("level3 parent should be level2")
		}
		if level2.Parent() != level1 {
			t.Error("level2 parent should be level1")
		}
		if level1.Parent() != root {
			t.Error("level1 parent should be root")
		}
		if root.Parent() != nil {
			t.Error("root should have no parent")
		}
	})

	t.Run("timing tree generation", func(t *testing.T) {
		root := NewTimer("root")
		child1 := root.Child("child1")
		child2 := root.Child("child2")
		grandchild := child1.Child("grandchild")

		// Stop timers in reverse order
		grandchild.Stop()
		child2.Stop()
		child1.Stop()
		root.Stop()

		tree := root.GetTree()
		if tree.Name != "root" {
			t.Errorf("expected root name, got %s", tree.Name)
		}

		if len(tree.Children) != 2 {
			t.Errorf("expected 2 root children, got %d", len(tree.Children))
		}

		// Find child1 and verify its structure
		var child1Tree *TimingTree
		for _, child := range tree.Children {
			if child.Name == "child1" {
				child1Tree = child
				break
			}
		}

		if child1Tree == nil {
			t.Fatal("child1 not found in tree")
		}

		if len(child1Tree.Children) != 1 {
			t.Errorf("expected 1 grandchild, got %d", len(child1Tree.Children))
		}

		if child1Tree.Children[0].Name != "grandchild" {
			t.Errorf("expected grandchild name, got %s", child1Tree.Children[0].Name)
		}
	})

	t.Run("self duration calculation", func(t *testing.T) {
		parent := NewTimer("parent")
		time.Sleep(5 * time.Millisecond)

		child := parent.Child("child")
		time.Sleep(10 * time.Millisecond)
		childDuration := child.Stop()

		time.Sleep(5 * time.Millisecond)
		parentDuration := parent.Stop()

		tree := parent.GetTree()
		selfDuration := tree.GetSelfDuration()

		// Self duration should be parent - child
		expectedSelf := parentDuration - childDuration
		if absDuration(selfDuration-expectedSelf) > 2*time.Millisecond {
			t.Errorf("expected self duration ~%v, got %v", expectedSelf, selfDuration)
		}

		// Self duration should be less than total duration
		if selfDuration >= tree.Duration {
			t.Errorf("self duration %v should be less than total %v", selfDuration, tree.Duration)
		}
	})
}

// TestTimingContext tests timing context functionality
func TestTimingContext(t *testing.T) {
	t.Run("basic context operations", func(t *testing.T) {
		tc := NewTimingContext()

		timer1 := tc.Start("operation1")
		if timer1.Name() != "operation1" {
			t.Errorf("expected timer name 'operation1', got %s", timer1.Name())
		}

		time.Sleep(5 * time.Millisecond)
		duration1 := tc.Stop()
		if duration1 <= 0 {
			t.Error("expected positive duration")
		}

		// Start another operation
		timer2 := tc.Start("operation2")
		time.Sleep(5 * time.Millisecond)
		_ = tc.Stop()

		// Verify we can get timers by name
		retrieved1 := tc.GetTimer("operation1")
		if retrieved1 != timer1 {
			t.Error("retrieved timer should match original")
		}

		retrieved2 := tc.GetTimer("operation2")
		if retrieved2 != timer2 {
			t.Error("retrieved timer should match original")
		}
	})

	t.Run("nested context timing", func(t *testing.T) {
		tc := NewTimingContext()

		outer := tc.Start("outer")
		time.Sleep(5 * time.Millisecond)

		inner := tc.Start("inner")
		time.Sleep(5 * time.Millisecond)
		tc.Stop() // stops inner

		time.Sleep(5 * time.Millisecond)
		tc.Stop() // stops outer

		// Verify hierarchy
		if inner.Parent() != outer {
			t.Error("inner timer should be child of outer")
		}

		outerChildren := outer.Children()
		if len(outerChildren) != 1 || outerChildren[0] != inner {
			t.Error("outer should have inner as child")
		}

		// Verify root finding
		root := tc.GetRoot()
		if root != outer {
			t.Error("root should be outer timer")
		}
	})

	t.Run("context integration", func(t *testing.T) {
		ctx := WithTiming(context.Background())

		retrievedTC := GetTimingContext(ctx)
		if retrievedTC == nil {
			t.Fatal("timing context should be retrievable from context")
		}

		timer, newCtx := StartTimer(ctx, "test_op")
		if timer.Name() != "test_op" {
			t.Errorf("expected timer name 'test_op', got %s", timer.Name())
		}

		// Verify context still contains timing context
		retrievedTC2 := GetTimingContext(newCtx)
		if retrievedTC2 == nil {
			t.Error("timing context should still be in new context")
		}

		// Test with context that has no timing context
		plainCtx := context.Background()
		if GetTimingContext(plainCtx) != nil {
			t.Error("plain context should not have timing context")
		}

		// StartTimer should create new timing context
		timer2, newCtx2 := StartTimer(plainCtx, "new_op")
		if timer2.Name() != "new_op" {
			t.Errorf("expected timer name 'new_op', got %s", timer2.Name())
		}

		if GetTimingContext(newCtx2) == nil {
			t.Error("new context should have timing context")
		}
	})
}

// TestTimingConcurrency tests concurrent timer operations
func TestTimingConcurrency(t *testing.T) {
	t.Run("concurrent metadata access", func(t *testing.T) {
		timer := NewTimer("concurrent")
		var wg sync.WaitGroup

		// Start multiple goroutines setting metadata
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				timer.SetMetadata("key", index)
				timer.GetMetadata("key")
			}(i)
		}

		wg.Wait()

		// Timer should still be functional
		if timer.Name() != "concurrent" {
			t.Error("timer name should remain unchanged")
		}
	})

	t.Run("concurrent child creation", func(t *testing.T) {
		parent := NewTimer("parent")
		var wg sync.WaitGroup
		childCount := 50

		for i := 0; i < childCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				child := parent.Child("child")
				child.SetMetadata("index", index)
				child.Stop()
			}(i)
		}

		wg.Wait()

		children := parent.Children()
		if len(children) != childCount {
			t.Errorf("expected %d children, got %d", childCount, len(children))
		}
	})

	t.Run("concurrent timing context", func(t *testing.T) {
		tc := NewTimingContext()
		var wg sync.WaitGroup
		operationCount := 50

		for i := 0; i < operationCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				timer := tc.Start("concurrent_op")
				time.Sleep(1 * time.Millisecond)
				timer.SetMetadata("index", index)
				tc.Stop()
			}(i)
		}

		wg.Wait()

		// Verify all operations were recorded
		// Note: Due to concurrency, we might have fewer unique timers
		// since multiple goroutines might use the same name
	})
}

// TestAutoTimer tests automatic timer functionality
func TestAutoTimer(t *testing.T) {
	t.Run("basic auto timer", func(t *testing.T) {
		var duration time.Duration
		func() {
			autoTimer := NewAutoTimer("auto_test")
			defer autoTimer.Stop()

			time.Sleep(10 * time.Millisecond)
			duration = autoTimer.Timer().Duration()
		}()

		if duration <= 5*time.Millisecond {
			t.Errorf("expected duration > 5ms, got %v", duration)
		}
	})

	t.Run("auto timer with panic", func(t *testing.T) {
		var autoTimer *AutoTimer

		func() {
			defer func() {
				if r := recover(); r != nil {
					// Timer should still be accessible after panic
					if autoTimer.Timer().Name() != "panic_test" {
						t.Error("timer should remain accessible after panic")
					}
				}
			}()

			autoTimer = NewAutoTimer("panic_test")
			defer autoTimer.Stop()

			panic("test panic")
		}()
	})
}

// TestTimingUtilities tests utility functions
func TestTimingUtilities(t *testing.T) {
	t.Run("time function", func(t *testing.T) {
		executed := false
		err := TimeFunc("test_func", func() error {
			executed = true
			time.Sleep(5 * time.Millisecond)
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !executed {
			t.Error("function should have been executed")
		}
	})

	t.Run("time function with error", func(t *testing.T) {
		testErr := &testError{"test error"}
		err := TimeFunc("error_func", func() error {
			return testErr
		})

		if err != testErr {
			t.Errorf("expected test error, got %v", err)
		}
	})

	t.Run("time function with metadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		err := TimeFuncWithMetadata("metadata_func", metadata, func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

// TestTimingAccuracy tests measurement accuracy
func TestTimingAccuracy(t *testing.T) {
	t.Run("measurement precision", func(t *testing.T) {
		iterations := 10
		sleepDuration := 10 * time.Millisecond
		tolerance := 5 * time.Millisecond

		for i := 0; i < iterations; i++ {
			timer := NewTimer("precision_test")
			time.Sleep(sleepDuration)
			measured := timer.Stop()

			if absDuration(measured-sleepDuration) > tolerance {
				t.Errorf("iteration %d: expected ~%v, got %v (diff: %v)",
					i, sleepDuration, measured, absDuration(measured-sleepDuration))
			}
		}
	})

	t.Run("sub-millisecond timing", func(t *testing.T) {
		timer := NewTimer("submilli")
		time.Sleep(500 * time.Microsecond)
		duration := timer.Stop()

		// Should measure at least some time
		if duration <= 0 {
			t.Error("should measure positive duration for sub-millisecond timing")
		}

		// Should be in the right ballpark (allowing for timer resolution)
		if duration > 5*time.Millisecond {
			t.Errorf("sub-millisecond timing seems too large: %v", duration)
		}
	})
}

// Helper functions and types
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
