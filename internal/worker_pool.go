package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Task represents a unit of work to be executed by the worker pool
type Task interface {
	Execute(ctx context.Context) (interface{}, error)
	ID() string
}

// TaskResult represents the result of executing a task
type TaskResult struct {
	ID       string
	Value    interface{}
	Err      error
	Duration time.Duration
}

// WorkerPool manages a pool of workers for executing tasks concurrently
type WorkerPool struct {
	name      string
	workers   int
	taskQueue chan Task
	results   chan TaskResult
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Metrics
	tasksProcessed atomic.Uint64
	tasksQueued    atomic.Uint64
	errors         atomic.Uint64

	// Rate limiting
	rateLimiter RateLimiter
}

// WorkerPoolConfig holds configuration for creating a worker pool
type WorkerPoolConfig struct {
	Name      string
	Workers   int
	QueueSize int
	RateLimit int // requests per second, 0 for unlimited
}

// NewWorkerPool creates a new worker pool with the given configuration
func NewWorkerPool(config WorkerPoolConfig) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		name:      config.Name,
		workers:   config.Workers,
		taskQueue: make(chan Task, config.QueueSize),
		results:   make(chan TaskResult, config.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	if config.RateLimit > 0 {
		wp.rateLimiter = NewTokenBucketRateLimiter(config.RateLimit)
	}

	wp.start()
	return wp
}

// start initializes and starts all workers
func (wp *WorkerPool) start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// worker is the main worker loop
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return // Channel closed, worker should exit
			}

			// Apply rate limiting if configured
			if wp.rateLimiter != nil {
				_ = wp.rateLimiter.Wait(wp.ctx)
			}

			// Execute the task
			startTime := time.Now()
			value, err := task.Execute(wp.ctx)
			duration := time.Since(startTime)

			// Update metrics
			wp.tasksProcessed.Add(1)
			if err != nil {
				wp.errors.Add(1)
			}

			// Send result
			select {
			case wp.results <- TaskResult{
				ID:       task.ID(),
				Value:    value,
				Err:      err,
				Duration: duration,
			}:
			case <-wp.ctx.Done():
				return
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// Submit adds a task to the worker pool queue
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case wp.taskQueue <- task:
		wp.tasksQueued.Add(1)
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitAndWait submits a task and waits for its result
func (wp *WorkerPool) SubmitAndWait(task Task) (interface{}, error) {
	resultChan := make(chan TaskResult, 1)

	// Create a wrapper task that sends result to our channel
	wrappedTask := &channelTask{
		task:       task,
		resultChan: resultChan,
	}

	if err := wp.Submit(wrappedTask); err != nil {
		return nil, err
	}

	select {
	case result := <-resultChan:
		return result.Value, result.Err
	case <-wp.ctx.Done():
		return nil, fmt.Errorf("worker pool shut down while waiting for result")
	}
}

// Results returns the results channel for reading completed tasks
func (wp *WorkerPool) Results() <-chan TaskResult {
	return wp.results
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown() {
	wp.cancel()
	close(wp.taskQueue)
	wp.wg.Wait()
	close(wp.results)
}

// Metrics returns current metrics for the worker pool
func (wp *WorkerPool) Metrics() WorkerPoolMetrics {
	return WorkerPoolMetrics{
		Name:           wp.name,
		Workers:        wp.workers,
		TasksQueued:    wp.tasksQueued.Load(),
		TasksProcessed: wp.tasksProcessed.Load(),
		Errors:         wp.errors.Load(),
		QueueLength:    len(wp.taskQueue),
		QueueCapacity:  cap(wp.taskQueue),
	}
}

// WorkerPoolMetrics holds runtime metrics for a worker pool
type WorkerPoolMetrics struct {
	Name           string
	Workers        int
	TasksQueued    uint64
	TasksProcessed uint64
	Errors         uint64
	QueueLength    int
	QueueCapacity  int
}

// channelTask wraps a task to send its result to a specific channel
type channelTask struct {
	task       Task
	resultChan chan<- TaskResult
}

func (ct *channelTask) Execute(ctx context.Context) (interface{}, error) {
	value, err := ct.task.Execute(ctx)
	result := TaskResult{
		ID:    ct.task.ID(),
		Value: value,
		Err:   err,
	}

	select {
	case ct.resultChan <- result:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return value, err
}

func (ct *channelTask) ID() string {
	return ct.task.ID()
}

// RateLimiter interface for rate limiting strategies
type RateLimiter interface {
	Wait(ctx context.Context) error
	TryAcquire() bool
}

// TokenBucketRateLimiter implements token bucket rate limiting
type TokenBucketRateLimiter struct {
	rate       int
	bucket     chan struct{}
	refillStop chan struct{}
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(ratePerSecond int) *TokenBucketRateLimiter {
	rl := &TokenBucketRateLimiter{
		rate:       ratePerSecond,
		bucket:     make(chan struct{}, ratePerSecond),
		refillStop: make(chan struct{}),
	}

	// Fill bucket initially
	for i := 0; i < ratePerSecond; i++ {
		rl.bucket <- struct{}{}
	}

	// Start refill goroutine
	go rl.refill()

	return rl
}

// refill adds tokens to the bucket at the configured rate
func (rl *TokenBucketRateLimiter) refill() {
	ticker := time.NewTicker(time.Second / time.Duration(rl.rate))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case rl.bucket <- struct{}{}:
			default:
				// Bucket is full
			}
		case <-rl.refillStop:
			return
		}
	}
}

// Wait blocks until a token is available
func (rl *TokenBucketRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.bucket:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire a token without blocking
func (rl *TokenBucketRateLimiter) TryAcquire() bool {
	select {
	case <-rl.bucket:
		return true
	default:
		return false
	}
}

// Stop stops the rate limiter
func (rl *TokenBucketRateLimiter) Stop() {
	close(rl.refillStop)
}
