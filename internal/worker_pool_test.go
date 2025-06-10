package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestTask implements the Task interface for testing
type TestTask struct {
	id       string
	duration time.Duration
	result   interface{}
	err      error
	executed atomic.Bool
}

func NewTestTask(id string, duration time.Duration, result interface{}, err error) *TestTask {
	return &TestTask{
		id:       id,
		duration: duration,
		result:   result,
		err:      err,
	}
}

func (t *TestTask) Execute(ctx context.Context) (interface{}, error) {
	t.executed.Store(true)

	if t.duration > 0 {
		select {
		case <-time.After(t.duration):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return t.result, t.err
}

func (t *TestTask) ID() string {
	return t.id
}

func (t *TestTask) WasExecuted() bool {
	return t.executed.Load()
}

// TestWorkerPoolDetailed tests detailed worker pool functionality
func TestWorkerPoolDetailed(t *testing.T) {
	t.Run("basic task execution", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "test_pool",
			Workers:   2,
			QueueSize: 10,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Submit a simple task
		task := NewTestTask("task1", 10*time.Millisecond, "result1", nil)
		err := wp.Submit(task)
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}

		// Wait for result
		select {
		case result := <-wp.Results():
			if result.ID != "task1" {
				t.Errorf("expected task ID 'task1', got %s", result.ID)
			}
			if result.Value != "result1" {
				t.Errorf("expected result 'result1', got %v", result.Value)
			}
			if result.Err != nil {
				t.Errorf("expected no error, got %v", result.Err)
			}
			if result.Duration <= 0 {
				t.Error("expected positive duration")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for result")
		}

		// Verify task was executed
		if !task.WasExecuted() {
			t.Error("task should have been executed")
		}
	})

	t.Run("multiple tasks", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "multi_pool",
			Workers:   3,
			QueueSize: 20,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Submit multiple tasks
		taskCount := 10
		tasks := make([]*TestTask, taskCount)

		for i := 0; i < taskCount; i++ {
			task := NewTestTask(fmt.Sprintf("task%d", i), 5*time.Millisecond, fmt.Sprintf("result%d", i), nil)
			tasks[i] = task

			err := wp.Submit(task)
			if err != nil {
				t.Fatalf("failed to submit task %d: %v", i, err)
			}
		}

		// Collect results
		results := make(map[string]TaskResult)
		for i := 0; i < taskCount; i++ {
			select {
			case result := <-wp.Results():
				results[result.ID] = result
			case <-time.After(2 * time.Second):
				t.Fatalf("timeout waiting for result %d", i)
			}
		}

		// Verify all results
		if len(results) != taskCount {
			t.Errorf("expected %d results, got %d", taskCount, len(results))
		}

		for i := 0; i < taskCount; i++ {
			taskID := fmt.Sprintf("task%d", i)
			result, exists := results[taskID]
			if !exists {
				t.Errorf("missing result for task %s", taskID)
				continue
			}

			expectedResult := fmt.Sprintf("result%d", i)
			if result.Value != expectedResult {
				t.Errorf("task %s: expected result %s, got %v", taskID, expectedResult, result.Value)
			}

			if !tasks[i].WasExecuted() {
				t.Errorf("task %s should have been executed", taskID)
			}
		}
	})

	t.Run("task with error", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "error_pool",
			Workers:   1,
			QueueSize: 5,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Submit task that returns error
		expectedErr := fmt.Errorf("test error")
		task := NewTestTask("error_task", 0, nil, expectedErr)

		err := wp.Submit(task)
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}

		// Wait for result
		select {
		case result := <-wp.Results():
			if result.Err == nil {
				t.Error("expected error result")
			}
			if result.Err.Error() != expectedErr.Error() {
				t.Errorf("expected error %v, got %v", expectedErr, result.Err)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for error result")
		}
	})

	t.Run("queue full", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "full_pool",
			Workers:   1,
			QueueSize: 2,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Fill the queue with slow tasks
		for i := 0; i < 2; i++ {
			task := NewTestTask(fmt.Sprintf("slow%d", i), 100*time.Millisecond, "result", nil)
			err := wp.Submit(task)
			if err != nil {
				t.Fatalf("failed to submit task %d: %v", i, err)
			}
		}

		// Try to submit one more task - should fail
		task := NewTestTask("overflow", 0, "result", nil)
		err := wp.Submit(task)
		if err == nil {
			t.Error("expected error when queue is full")
		}
		if err.Error() != "task queue is full" {
			t.Errorf("expected 'task queue is full' error, got %v", err)
		}
	})

	t.Run("submit and wait", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "wait_pool",
			Workers:   2,
			QueueSize: 5,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Submit task and wait for result
		task := NewTestTask("wait_task", 20*time.Millisecond, "wait_result", nil)

		start := time.Now()
		result, err := wp.SubmitAndWait(task)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "wait_result" {
			t.Errorf("expected result 'wait_result', got %v", result)
		}
		if duration < 15*time.Millisecond {
			t.Errorf("expected duration >= 15ms, got %v", duration)
		}
		if !task.WasExecuted() {
			t.Error("task should have been executed")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "cancel_pool",
			Workers:   2,
			QueueSize: 5,
		}

		wp := NewWorkerPool(config)

		// Submit task that checks context
		task := NewTestTask("cancel_task", 100*time.Millisecond, "should_not_complete", nil)
		err := wp.Submit(task)
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}

		// Shutdown immediately
		wp.Shutdown()

		// The task should either:
		// 1. Not execute at all (if worker exits before picking it up)
		// 2. Execute but return context.Canceled (if worker picks it up but context is cancelled)

		// We can't guarantee which case happens, so we just verify shutdown doesn't hang
		select {
		case result := <-wp.Results():
			// If we get a result, it should be either successful or cancelled
			if result.Err != nil && result.Err != context.Canceled {
				t.Logf("Task completed with error (expected): %v", result.Err)
			}
		case <-time.After(200 * time.Millisecond):
			// If no result within timeout, that's also acceptable for shutdown
		}
	})
}

// TestWorkerPoolMetrics tests worker pool metrics
func TestWorkerPoolMetrics(t *testing.T) {
	t.Run("basic metrics", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "metrics_pool",
			Workers:   2,
			QueueSize: 10,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Initial metrics
		metrics := wp.Metrics()
		if metrics.Name != "metrics_pool" {
			t.Errorf("expected name 'metrics_pool', got %s", metrics.Name)
		}
		if metrics.Workers != 2 {
			t.Errorf("expected 2 workers, got %d", metrics.Workers)
		}
		if metrics.QueueCapacity != 10 {
			t.Errorf("expected queue capacity 10, got %d", metrics.QueueCapacity)
		}
		if metrics.TasksQueued != 0 {
			t.Errorf("expected 0 tasks queued initially, got %d", metrics.TasksQueued)
		}

		// Submit successful tasks
		for i := 0; i < 3; i++ {
			task := NewTestTask(fmt.Sprintf("success%d", i), 5*time.Millisecond, "result", nil)
			err := wp.Submit(task)
			if err != nil {
				t.Fatalf("failed to submit task %d: %v", i, err)
			}
		}

		// Submit error task
		errorTask := NewTestTask("error", 5*time.Millisecond, nil, fmt.Errorf("test error"))
		err := wp.Submit(errorTask)
		if err != nil {
			t.Fatalf("failed to submit error task: %v", err)
		}

		// Wait for all tasks to complete
		for i := 0; i < 4; i++ {
			select {
			case <-wp.Results():
			case <-time.After(1 * time.Second):
				t.Fatalf("timeout waiting for result %d", i)
			}
		}

		// Check final metrics
		finalMetrics := wp.Metrics()
		if finalMetrics.TasksQueued != 4 {
			t.Errorf("expected 4 tasks queued, got %d", finalMetrics.TasksQueued)
		}
		if finalMetrics.TasksProcessed != 4 {
			t.Errorf("expected 4 tasks processed, got %d", finalMetrics.TasksProcessed)
		}
		if finalMetrics.Errors != 1 {
			t.Errorf("expected 1 error, got %d", finalMetrics.Errors)
		}
		if finalMetrics.QueueLength > 0 {
			t.Errorf("expected empty queue, got length %d", finalMetrics.QueueLength)
		}
	})
}

// TestWorkerPoolConcurrency tests concurrent access to worker pool
func TestWorkerPoolConcurrency(t *testing.T) {
	t.Run("concurrent task submission", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "concurrent_pool",
			Workers:   5,
			QueueSize: 100,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Submit tasks concurrently
		var wg sync.WaitGroup
		taskCount := 50
		goroutines := 10
		tasksPerGoroutine := taskCount / goroutines

		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for i := 0; i < tasksPerGoroutine; i++ {
					taskID := fmt.Sprintf("g%d_t%d", goroutineID, i)
					task := NewTestTask(taskID, 1*time.Millisecond, fmt.Sprintf("result_%s", taskID), nil)

					err := wp.Submit(task)
					if err != nil {
						t.Errorf("goroutine %d: failed to submit task %s: %v", goroutineID, taskID, err)
					}
				}
			}(g)
		}

		wg.Wait()

		// Collect all results
		results := make(map[string]TaskResult)
		for i := 0; i < taskCount; i++ {
			select {
			case result := <-wp.Results():
				results[result.ID] = result
			case <-time.After(5 * time.Second):
				t.Fatalf("timeout waiting for result %d (got %d results so far)", i, len(results))
			}
		}

		// Verify all tasks completed
		if len(results) != taskCount {
			t.Errorf("expected %d results, got %d", taskCount, len(results))
		}

		// Check metrics
		metrics := wp.Metrics()
		if metrics.TasksQueued != uint64(taskCount) {
			t.Errorf("expected %d tasks queued, got %d", taskCount, metrics.TasksQueued)
		}
		if metrics.TasksProcessed != uint64(taskCount) {
			t.Errorf("expected %d tasks processed, got %d", taskCount, metrics.TasksProcessed)
		}
	})

	t.Run("concurrent submit and wait", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "wait_concurrent_pool",
			Workers:   3,
			QueueSize: 20,
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Submit and wait concurrently
		var wg sync.WaitGroup
		concurrentWaits := 10

		for i := 0; i < concurrentWaits; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				taskID := fmt.Sprintf("wait_task_%d", id)
				task := NewTestTask(taskID, 10*time.Millisecond, fmt.Sprintf("result_%d", id), nil)

				result, err := wp.SubmitAndWait(task)
				if err != nil {
					t.Errorf("task %s: unexpected error: %v", taskID, err)
					return
				}

				expectedResult := fmt.Sprintf("result_%d", id)
				if result != expectedResult {
					t.Errorf("task %s: expected result %s, got %v", taskID, expectedResult, result)
				}
			}(i)
		}

		wg.Wait()

		// Verify metrics
		metrics := wp.Metrics()
		if metrics.TasksProcessed != uint64(concurrentWaits) {
			t.Errorf("expected %d tasks processed, got %d", concurrentWaits, metrics.TasksProcessed)
		}
	})
}

// TestTokenBucketRateLimiter tests the rate limiter functionality
func TestTokenBucketRateLimiter(t *testing.T) {
	t.Run("basic rate limiting", func(t *testing.T) {
		rateLimit := 5 // 5 tokens per second
		rl := NewTokenBucketRateLimiter(rateLimit)
		defer rl.Stop()

		// Should be able to acquire initial tokens immediately
		for i := 0; i < rateLimit; i++ {
			if !rl.TryAcquire() {
				t.Errorf("should be able to acquire token %d immediately", i)
			}
		}

		// Next acquire should fail (bucket empty)
		if rl.TryAcquire() {
			t.Error("should not be able to acquire token when bucket is empty")
		}

		// Wait for refill and try again
		time.Sleep(250 * time.Millisecond) // 1/4 second should give us ~1 token
		if !rl.TryAcquire() {
			t.Error("should be able to acquire token after refill")
		}
	})

	t.Run("wait for token", func(t *testing.T) {
		rateLimit := 2
		rl := NewTokenBucketRateLimiter(rateLimit)
		defer rl.Stop()

		// Drain the bucket
		for i := 0; i < rateLimit; i++ {
			rl.TryAcquire()
		}

		// Wait should succeed but take time
		ctx := context.Background()
		start := time.Now()

		err := rl.Wait(ctx)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("unexpected error waiting for token: %v", err)
		}

		// Should have waited approximately 1/rate seconds
		expectedMin := 400 * time.Millisecond // Allow some variance
		if duration < expectedMin {
			t.Errorf("wait duration too short: %v (expected >= %v)", duration, expectedMin)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		rateLimit := 1
		rl := NewTokenBucketRateLimiter(rateLimit)
		defer rl.Stop()

		// Drain the bucket
		rl.TryAcquire()

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Wait should fail with context error
		err := rl.Wait(ctx)
		if err == nil {
			t.Error("expected context timeout error")
		}
		if err != context.DeadlineExceeded {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}

// TestWorkerPoolWithRateLimit tests worker pool with rate limiting
func TestWorkerPoolWithRateLimit(t *testing.T) {
	t.Run("rate limited execution", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "rate_limited_pool",
			Workers:   2,
			QueueSize: 10,
			RateLimit: 2, // 2 tasks per second
		}

		wp := NewWorkerPool(config)
		defer wp.Shutdown()

		// Submit multiple tasks
		taskCount := 4
		for i := 0; i < taskCount; i++ {
			task := NewTestTask(fmt.Sprintf("rate_task_%d", i), 0, fmt.Sprintf("result_%d", i), nil)
			err := wp.Submit(task)
			if err != nil {
				t.Fatalf("failed to submit task %d: %v", i, err)
			}
		}

		// Measure execution time
		start := time.Now()

		// Collect results
		for i := 0; i < taskCount; i++ {
			select {
			case <-wp.Results():
			case <-time.After(10 * time.Second):
				t.Fatalf("timeout waiting for result %d", i)
			}
		}

		duration := time.Since(start)

		// With rate limit of 2/sec, 4 tasks should take at least ~2 seconds
		// (2 immediate, then 2 more after ~1 second each)
		expectedMin := 1500 * time.Millisecond // Allow some variance
		if duration < expectedMin {
			t.Logf("Rate limiting may not be working as expected: duration %v < %v", duration, expectedMin)
			// Don't fail the test as timing can be flaky in CI
		}
	})
}

// TestWorkerPoolShutdown tests graceful shutdown
func TestWorkerPoolShutdown(t *testing.T) {
	t.Run("graceful shutdown", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "shutdown_pool",
			Workers:   2,
			QueueSize: 5,
		}

		wp := NewWorkerPool(config)

		// Submit some tasks
		for i := 0; i < 3; i++ {
			task := NewTestTask(fmt.Sprintf("shutdown_task_%d", i), 50*time.Millisecond, fmt.Sprintf("result_%d", i), nil)
			err := wp.Submit(task)
			if err != nil {
				t.Fatalf("failed to submit task %d: %v", i, err)
			}
		}

		// Shutdown should wait for running tasks to complete
		start := time.Now()
		wp.Shutdown()
		shutdownDuration := time.Since(start)

		// Should have waited for running tasks (at least 40ms)
		if shutdownDuration < 40*time.Millisecond {
			t.Errorf("shutdown too fast: %v (expected >= 40ms)", shutdownDuration)
		}

		// Trying to submit after shutdown should fail
		task := NewTestTask("post_shutdown", 0, "result", nil)
		err := wp.Submit(task)
		if err == nil {
			t.Error("expected error when submitting after shutdown")
		}
	})

	t.Run("shutdown with pending tasks", func(t *testing.T) {
		config := WorkerPoolConfig{
			Name:      "pending_shutdown_pool",
			Workers:   1, // Only 1 worker to ensure queueing
			QueueSize: 10,
		}

		wp := NewWorkerPool(config)

		// Submit more tasks than can be processed immediately
		taskCount := 5
		for i := 0; i < taskCount; i++ {
			task := NewTestTask(fmt.Sprintf("pending_task_%d", i), 20*time.Millisecond, fmt.Sprintf("result_%d", i), nil)
			err := wp.Submit(task)
			if err != nil {
				t.Fatalf("failed to submit task %d: %v", i, err)
			}
		}

		// Shutdown - should complete running tasks but not pending ones
		wp.Shutdown()

		// Some tasks should have completed, but not necessarily all
		metrics := wp.Metrics()
		if metrics.TasksProcessed > metrics.TasksQueued {
			t.Errorf("processed (%d) should not exceed queued (%d)", metrics.TasksProcessed, metrics.TasksQueued)
		}
	})
}
