package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Timer tracks operation timing with hierarchical support
type Timer struct {
	name      string
	start     time.Time
	end       time.Time
	duration  time.Duration
	parent    *Timer
	children  []*Timer
	metadata  map[string]interface{}
	mu        sync.RWMutex
	completed int32
}

// NewTimer creates a new timer
func NewTimer(name string) *Timer {
	return &Timer{
		name:     name,
		start:    time.Now(),
		metadata: make(map[string]interface{}),
		children: make([]*Timer, 0),
	}
}

// Stop stops the timer and records the duration
func (t *Timer) Stop() time.Duration {
	if atomic.CompareAndSwapInt32(&t.completed, 0, 1) {
		t.end = time.Now()
		t.duration = t.end.Sub(t.start)

		// Record metric if enabled
		if MetricsEnabled() {
			t.recordMetric()
		}

		// Check for slow operation
		if slowOpDetector != nil {
			slowOpDetector.Check(t)
		}
	}
	return t.duration
}

// StopWithError stops the timer and records any error
func (t *Timer) StopWithError(err error) (time.Duration, error) {
	duration := t.Stop()
	if err != nil {
		t.SetMetadata("error", err.Error())
	}
	return duration, err
}

// Child creates a child timer
func (t *Timer) Child(name string) *Timer {
	child := NewTimer(name)
	child.parent = t

	t.mu.Lock()
	t.children = append(t.children, child)
	t.mu.Unlock()

	return child
}

// SetMetadata sets metadata for the timer
func (t *Timer) SetMetadata(key string, value interface{}) {
	t.mu.Lock()
	t.metadata[key] = value
	t.mu.Unlock()
}

// GetMetadata gets metadata value
func (t *Timer) GetMetadata(key string) (interface{}, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	val, ok := t.metadata[key]
	return val, ok
}

// Duration returns the timer duration
func (t *Timer) Duration() time.Duration {
	if atomic.LoadInt32(&t.completed) == 1 {
		return t.duration
	}
	// Return elapsed time for running timer
	return time.Since(t.start)
}

// Name returns the timer name
func (t *Timer) Name() string {
	return t.name
}

// Parent returns the parent timer
func (t *Timer) Parent() *Timer {
	return t.parent
}

// Children returns child timers
func (t *Timer) Children() []*Timer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]*Timer{}, t.children...)
}

// GetTree returns the timing tree
func (t *Timer) GetTree() *TimingTree {
	return &TimingTree{
		Name:     t.name,
		Duration: t.Duration(),
		Start:    t.start,
		End:      t.end,
		Metadata: t.copyMetadata(),
		Children: t.getChildTrees(),
	}
}

// copyMetadata creates a copy of metadata
func (t *Timer) copyMetadata() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	copy := make(map[string]interface{})
	for k, v := range t.metadata {
		copy[k] = v
	}
	return copy
}

// getChildTrees recursively builds child trees
func (t *Timer) getChildTrees() []*TimingTree {
	t.mu.RLock()
	defer t.mu.RUnlock()

	trees := make([]*TimingTree, len(t.children))
	for i, child := range t.children {
		trees[i] = child.GetTree()
	}
	return trees
}

// recordMetric records timing metric
func (t *Timer) recordMetric() {
	mc := GetMetricsCollector()

	// Determine metric type from name
	labels := make(map[string]string)

	switch {
	case t.name == "parse":
		mc.ParseDuration.GetOrCreate(labels).(*Histogram).ObserveDuration(t.start)
	case t.name == "eval":
		mc.EvalDuration.GetOrCreate(labels).(*Histogram).ObserveDuration(t.start)
	case t.parent != nil && t.parent.name == "operator":
		// This is an operator timing
		if opName, ok := t.metadata["operator"].(string); ok {
			labels["operator"] = opName
			mc.OperatorDuration.GetOrCreate(labels).(*Histogram).ObserveDuration(t.start)
		}
	default:
		// Custom timing - register if not exists
		metricName := fmt.Sprintf("graft_%s_duration_seconds", t.name)
		if _, exists := mc.CustomMetrics[metricName]; !exists {
			mc.RegisterCustomMetric(metricName, fmt.Sprintf("Duration of %s operations", t.name), MetricTypeHistogram)
		}
		if family, ok := mc.CustomMetrics[metricName]; ok {
			family.GetOrCreate(labels).(*Histogram).ObserveDuration(t.start)
		}
	}
}

// TimingTree represents a hierarchical timing structure
type TimingTree struct {
	Name     string                 `json:"name"`
	Duration time.Duration          `json:"duration"`
	Start    time.Time              `json:"start"`
	End      time.Time              `json:"end"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Children []*TimingTree          `json:"children,omitempty"`
}

// GetSelfDuration returns duration excluding children
func (tt *TimingTree) GetSelfDuration() time.Duration {
	childTotal := time.Duration(0)
	for _, child := range tt.Children {
		childTotal += child.Duration
	}
	return tt.Duration - childTotal
}

// TimingContext manages timing within a context
type TimingContext struct {
	mu      sync.RWMutex
	timers  map[string]*Timer
	current *Timer
}

// NewTimingContext creates a new timing context
func NewTimingContext() *TimingContext {
	return &TimingContext{
		timers: make(map[string]*Timer),
	}
}

// Start starts a new timer in the context
func (tc *TimingContext) Start(name string) *Timer {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	timer := NewTimer(name)
	if tc.current != nil {
		timer.parent = tc.current
		tc.current.mu.Lock()
		tc.current.children = append(tc.current.children, timer)
		tc.current.mu.Unlock()
	}

	tc.timers[name] = timer
	tc.current = timer
	return timer
}

// Stop stops the current timer
func (tc *TimingContext) Stop() time.Duration {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.current == nil {
		return 0
	}

	duration := tc.current.Stop()
	tc.current = tc.current.parent
	return duration
}

// GetTimer gets a timer by name
func (tc *TimingContext) GetTimer(name string) *Timer {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.timers[name]
}

// GetRoot returns the root timer
func (tc *TimingContext) GetRoot() *Timer {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// Find timer with no parent
	for _, timer := range tc.timers {
		if timer.parent == nil {
			return timer
		}
	}
	return nil
}

// Context key for timing
type timingContextKey struct{}

// WithTiming adds timing context to a context
func WithTiming(ctx context.Context) context.Context {
	return context.WithValue(ctx, timingContextKey{}, NewTimingContext())
}

// GetTimingContext gets timing context from context
func GetTimingContext(ctx context.Context) *TimingContext {
	if tc, ok := ctx.Value(timingContextKey{}).(*TimingContext); ok {
		return tc
	}
	return nil
}

// StartTimer starts a timer in the context
func StartTimer(ctx context.Context, name string) (*Timer, context.Context) {
	tc := GetTimingContext(ctx)
	if tc == nil {
		tc = NewTimingContext()
		ctx = context.WithValue(ctx, timingContextKey{}, tc)
	}
	return tc.Start(name), ctx
}

// TimeFunc times a function execution
func TimeFunc(name string, fn func() error) error {
	timer := NewTimer(name)
	defer timer.Stop()
	return fn()
}

// TimeFuncWithMetadata times a function with metadata
func TimeFuncWithMetadata(name string, metadata map[string]interface{}, fn func() error) error {
	timer := NewTimer(name)
	for k, v := range metadata {
		timer.SetMetadata(k, v)
	}
	defer timer.Stop()
	return fn()
}

// AutoTimer provides automatic timing with defer
type AutoTimer struct {
	timer *Timer
}

// NewAutoTimer creates an auto timer that stops on defer
func NewAutoTimer(name string) *AutoTimer {
	return &AutoTimer{
		timer: NewTimer(name),
	}
}

// Stop stops the auto timer (called via defer)
func (at *AutoTimer) Stop() {
	at.timer.Stop()
}

// Timer returns the underlying timer
func (at *AutoTimer) Timer() *Timer {
	return at.timer
}

// Example usage:
// func someOperation() {
//     defer NewAutoTimer("operation").Stop()
//     // ... operation code ...
// }
