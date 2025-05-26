package spruce

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	rpprof "runtime/pprof"
	"runtime/trace"
	"sync"
	"time"
)

// Profiler manages performance profiling
type Profiler struct {
	mu              sync.RWMutex
	cpuProfile      *os.File
	heapProfile     *os.File
	traceFile       *os.File
	profileDir      string
	enabled         bool
	pprofServer     *http.Server
	customProfiles  map[string]*rpprof.Profile
}

// ProfilerConfig configures the profiler
type ProfilerConfig struct {
	Enabled        bool
	ProfileDir     string
	EnableCPU      bool
	EnableHeap     bool
	EnableTrace    bool
	PProfAddr      string
	SampleRate     int
}

// DefaultProfilerConfig returns default profiler configuration
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		Enabled:     false,
		ProfileDir:  "profiles",
		EnableCPU:   true,
		EnableHeap:  true,
		EnableTrace: false,
		PProfAddr:   "",
		SampleRate:  100,
	}
}

// NewProfiler creates a new profiler
func NewProfiler(config *ProfilerConfig) *Profiler {
	if config == nil {
		config = DefaultProfilerConfig()
	}
	
	p := &Profiler{
		profileDir:     config.ProfileDir,
		enabled:        config.Enabled,
		customProfiles: make(map[string]*rpprof.Profile),
	}
	
	if config.Enabled {
		// Create profile directory
		os.MkdirAll(config.ProfileDir, 0755)
		
		// Set sample rate
		runtime.SetCPUProfileRate(config.SampleRate)
		
		// Start pprof server if configured
		if config.PProfAddr != "" {
			p.startPProfServer(config.PProfAddr)
		}
	}
	
	return p
}

// startPProfServer starts the pprof HTTP server
func (p *Profiler) startPProfServer(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.HandleFunc("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.HandleFunc("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.HandleFunc("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.HandleFunc("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	
	// Add custom profiles
	mux.HandleFunc("/debug/pprof/custom", p.handleCustomProfiles)
	
	p.pprofServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	go func() {
		if err := p.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("pprof server error: %v\n", err)
		}
	}()
}

// handleCustomProfiles handles custom profile requests
func (p *Profiler) handleCustomProfiles(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><title>Custom Profiles</title></head><body>\n")
	fmt.Fprintf(w, "<h1>Custom Profiles</h1>\n")
	fmt.Fprintf(w, "<ul>\n")
	
	for name := range p.customProfiles {
		fmt.Fprintf(w, `<li><a href="/debug/pprof/%s">%s</a></li>`+"\n", name, name)
	}
	
	fmt.Fprintf(w, "</ul>\n</body></html>")
}

// StartCPUProfile starts CPU profiling
func (p *Profiler) StartCPUProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.enabled {
		return fmt.Errorf("profiler is not enabled")
	}
	
	if p.cpuProfile != nil {
		return fmt.Errorf("CPU profiling already started")
	}
	
	filename := filepath.Join(p.profileDir, fmt.Sprintf("cpu_%s.prof", timestamp()))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile: %v", err)
	}
	
	if err := rpprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start CPU profile: %v", err)
	}
	
	p.cpuProfile = f
	return nil
}

// StopCPUProfile stops CPU profiling
func (p *Profiler) StopCPUProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.cpuProfile == nil {
		return fmt.Errorf("CPU profiling not started")
	}
	
	rpprof.StopCPUProfile()
	err := p.cpuProfile.Close()
	p.cpuProfile = nil
	
	return err
}

// WriteHeapProfile writes a heap profile
func (p *Profiler) WriteHeapProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.enabled {
		return fmt.Errorf("profiler is not enabled")
	}
	
	filename := filepath.Join(p.profileDir, fmt.Sprintf("heap_%s.prof", timestamp()))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create heap profile: %v", err)
	}
	defer f.Close()
	
	// Force GC before heap profile for accuracy
	runtime.GC()
	
	if err := rpprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile: %v", err)
	}
	
	return nil
}

// WriteAllProfiles writes all available profiles
func (p *Profiler) WriteAllProfiles() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.enabled {
		return fmt.Errorf("profiler is not enabled")
	}
	
	timestamp := timestamp()
	
	// Standard profiles
	profiles := []struct {
		name string
		prof *rpprof.Profile
	}{
		{"heap", rpprof.Lookup("heap")},
		{"goroutine", rpprof.Lookup("goroutine")},
		{"threadcreate", rpprof.Lookup("threadcreate")},
		{"block", rpprof.Lookup("block")},
		{"mutex", rpprof.Lookup("mutex")},
	}
	
	for _, profile := range profiles {
		if profile.prof == nil {
			continue
		}
		
		filename := filepath.Join(p.profileDir, fmt.Sprintf("%s_%s.prof", profile.name, timestamp))
		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create %s profile: %v", profile.name, err)
		}
		
		err = profile.prof.WriteTo(f, 0)
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to write %s profile: %v", profile.name, err)
		}
	}
	
	// Custom profiles
	for name, profile := range p.customProfiles {
		filename := filepath.Join(p.profileDir, fmt.Sprintf("%s_%s.prof", name, timestamp))
		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create %s profile: %v", name, err)
		}
		
		err = profile.WriteTo(f, 0)
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to write %s profile: %v", name, err)
		}
	}
	
	return nil
}

// StartTrace starts execution tracing
func (p *Profiler) StartTrace() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.enabled {
		return fmt.Errorf("profiler is not enabled")
	}
	
	if p.traceFile != nil {
		return fmt.Errorf("tracing already started")
	}
	
	filename := filepath.Join(p.profileDir, fmt.Sprintf("trace_%s.out", timestamp()))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %v", err)
	}
	
	if err := trace.Start(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start trace: %v", err)
	}
	
	p.traceFile = f
	return nil
}

// StopTrace stops execution tracing
func (p *Profiler) StopTrace() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.traceFile == nil {
		return fmt.Errorf("tracing not started")
	}
	
	trace.Stop()
	err := p.traceFile.Close()
	p.traceFile = nil
	
	return err
}

// RegisterCustomProfile registers a custom profile
func (p *Profiler) RegisterCustomProfile(name string, profile *rpprof.Profile) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.customProfiles[name] = profile
	
	// Register with default mux if pprof server is running
	if p.pprofServer != nil {
		http.HandleFunc("/debug/pprof/"+name, func(w http.ResponseWriter, r *http.Request) {
			profile.WriteTo(w, 0)
		})
	}
}

// EnableBlockProfiling enables block profiling
func (p *Profiler) EnableBlockProfiling(rate int) {
	runtime.SetBlockProfileRate(rate)
}

// EnableMutexProfiling enables mutex profiling
func (p *Profiler) EnableMutexProfiling(fraction int) {
	runtime.SetMutexProfileFraction(fraction)
}

// GetProfileDir returns the profile directory
func (p *Profiler) GetProfileDir() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.profileDir
}

// IsEnabled returns whether profiling is enabled
func (p *Profiler) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// SetEnabled enables or disables profiling
func (p *Profiler) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// Stop stops all profiling and shuts down the pprof server
func (p *Profiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Stop CPU profiling if active
	if p.cpuProfile != nil {
		rpprof.StopCPUProfile()
		p.cpuProfile.Close()
		p.cpuProfile = nil
	}
	
	// Stop tracing if active
	if p.traceFile != nil {
		trace.Stop()
		p.traceFile.Close()
		p.traceFile = nil
	}
	
	// Stop pprof server
	if p.pprofServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.pprofServer.Shutdown(ctx)
		p.pprofServer = nil
	}
	
	return nil
}

// ProfileScope provides scoped profiling
type ProfileScope struct {
	profiler *Profiler
	name     string
	start    time.Time
}

// NewProfileScope creates a new profile scope
func (p *Profiler) NewProfileScope(name string) *ProfileScope {
	return &ProfileScope{
		profiler: p,
		name:     name,
		start:    time.Now(),
	}
}

// End ends the profile scope
func (ps *ProfileScope) End() {
	// Record custom metric if profiling is enabled
	if ps.profiler.IsEnabled() {
		duration := time.Since(ps.start)
		// This could be extended to record to custom profiles
		_ = duration
	}
}

// Helper functions

func timestamp() string {
	return time.Now().Format("20060102_150405")
}

// Global profiler instance
var globalProfiler *Profiler
var profilerOnce sync.Once

// InitializeProfiler initializes the global profiler
func InitializeProfiler(config *ProfilerConfig) {
	profilerOnce.Do(func() {
		globalProfiler = NewProfiler(config)
	})
}

// GetProfiler returns the global profiler
func GetProfiler() *Profiler {
	if globalProfiler == nil {
		InitializeProfiler(nil)
	}
	return globalProfiler
}

// ProfileCPU runs a function with CPU profiling
func ProfileCPU(name string, fn func() error) error {
	profiler := GetProfiler()
	if !profiler.IsEnabled() {
		return fn()
	}
	
	if err := profiler.StartCPUProfile(); err != nil {
		return err
	}
	defer profiler.StopCPUProfile()
	
	return fn()
}

// ProfileHeap runs a function and captures heap profile after
func ProfileHeap(name string, fn func() error) error {
	profiler := GetProfiler()
	
	err := fn()
	
	if profiler.IsEnabled() {
		profiler.WriteHeapProfile()
	}
	
	return err
}