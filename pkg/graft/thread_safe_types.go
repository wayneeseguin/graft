package graft

import (
	"sync"
)

// ThreadSafeTree provides thread-safe access to a tree structure
type ThreadSafeTree interface {
	Get(path string) (interface{}, error)
	Set(path string, value interface{}) error
	Delete(path ...string) error
	Clone() map[interface{}]interface{}
	Copy() ThreadSafeTree
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

// TreeTransaction represents a transaction on a tree
type TreeTransaction interface {
	Get(path ...string) (interface{}, error)
	Set(value interface{}, path ...string) error
	Delete(path ...string) error
	Commit() error
	Rollback()
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers int
	tasks   chan func()
	wg      sync.WaitGroup
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	return &WorkerPool{
		workers: workers,
		tasks:   make(chan func(), workers*2),
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for task := range p.tasks {
				task()
			}
		}()
	}
}

// Submit submits a task to the pool
func (p *WorkerPool) Submit(task func()) {
	p.tasks <- task
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	close(p.tasks)
	p.wg.Wait()
}
