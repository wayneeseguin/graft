package internal

import (
	"github.com/wayneeseguin/graft/pkg/graft"
)
import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

// FeatureFlags controls experimental features
type FeatureFlags struct {
	mu sync.RWMutex
	
	// Parallel execution features
	ParallelExecution       bool
	ParallelMaxWorkers      int
	ParallelMinOps          int
	ParallelStrategy        string
	ParallelAutoTune        bool
	
	// Monitoring features
	EnableMetrics           bool
	EnableTracing           bool
	MetricsPort             int
	
	// Safety features
	StrictMode              bool
	ValidateOperators       bool
	EnableRaceDetection     bool
	
	// Performance features
	EnableCaching           bool
	EnableBatching          bool
	CacheSize               int
}

var (
	globalFeatures     *FeatureFlags
	globalFeaturesOnce sync.Once
)

// GetFeatures returns the global feature flags instance
func GetFeatures() *FeatureFlags {
	globalFeaturesOnce.Do(func() {
		globalFeatures = loadFeatureFlags()
	})
	return globalFeatures
}

// loadFeatureFlags loads feature flags from environment
func loadFeatureFlags() *FeatureFlags {
	flags := &FeatureFlags{
		// Parallel execution defaults
		ParallelExecution:  getEnvBool("GRAFT_PARALLEL", false),
		ParallelMaxWorkers: getEnvInt("GRAFT_PARALLEL_WORKERS", 0), // 0 = auto
		ParallelMinOps:     getEnvInt("GRAFT_PARALLEL_MIN_OPS", 10),
		ParallelStrategy:   getEnvString("GRAFT_PARALLEL_STRATEGY", "conservative"),
		ParallelAutoTune:   getEnvBool("GRAFT_PARALLEL_AUTO_TUNE", false),
		
		// Monitoring defaults
		EnableMetrics:      getEnvBool("GRAFT_METRICS", false),
		EnableTracing:      getEnvBool("GRAFT_TRACING", false),
		MetricsPort:        getEnvInt("GRAFT_METRICS_PORT", 9090),
		
		// Safety defaults
		StrictMode:         getEnvBool("GRAFT_STRICT", false),
		ValidateOperators:  getEnvBool("GRAFT_VALIDATE_OPS", true),
		EnableRaceDetection: getEnvBool("GRAFT_RACE_DETECTION", false),
		
		// Performance defaults
		EnableCaching:      getEnvBool("GRAFT_CACHE", true),
		EnableBatching:     getEnvBool("GRAFT_BATCH", true),
		CacheSize:          getEnvInt("GRAFT_CACHE_SIZE", 1000),
	}
	
	// Validate settings
	flags.validate()
	
	return flags
}

// validate ensures feature flag combinations are valid
func (f *FeatureFlags) validate() {
	// Ensure strategy is valid
	switch f.ParallelStrategy {
	case "conservative", "aggressive", "adaptive":
		// Valid
	default:
		f.ParallelStrategy = "conservative"
	}
	
	// Auto-tune requires metrics
	if f.ParallelAutoTune && !f.EnableMetrics {
		f.EnableMetrics = true
	}
	
	// Race detection implies strict mode
	if f.EnableRaceDetection {
		f.StrictMode = true
	}
}

// IsParallelEnabled returns true if parallel execution is enabled
func (f *FeatureFlags) IsParallelEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.ParallelExecution
}

// SetParallelEnabled enables or disables parallel execution
func (f *FeatureFlags) SetParallelEnabled(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ParallelExecution = enabled
}

// GetParallelConfig returns parallel execution configuration
func (f *FeatureFlags) GetParallelConfig() *ParallelEvaluatorConfig {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return &ParallelEvaluatorConfig{
		Enabled:           f.ParallelExecution,
		MaxWorkers:        f.ParallelMaxWorkers,
		MinOpsForParallel: f.ParallelMinOps,
		Strategy:          f.ParallelStrategy,
		SafeOperators:     DefaultParallelConfig().SafeOperators, // Use defaults for now
	}
}

// Update updates feature flags at runtime
func (f *FeatureFlags) Update(updates map[string]interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	for key, value := range updates {
		switch key {
		case "parallel_execution":
			if v, ok := value.(bool); ok {
				f.ParallelExecution = v
			}
		case "parallel_max_workers":
			if v, ok := value.(int); ok {
				f.ParallelMaxWorkers = v
			}
		case "parallel_strategy":
			if v, ok := value.(string); ok {
				f.ParallelStrategy = v
			}
		case "enable_metrics":
			if v, ok := value.(bool); ok {
				f.EnableMetrics = v
			}
		case "strict_mode":
			if v, ok := value.(bool); ok {
				f.StrictMode = v
			}
		}
	}
	
	f.validate()
}

// String returns a string representation of feature flags
func (f *FeatureFlags) String() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return fmt.Sprintf(
		"FeatureFlags{Parallel:%v Workers:%d MinOps:%d Strategy:%s Metrics:%v Strict:%v}",
		f.ParallelExecution,
		f.ParallelMaxWorkers,
		f.ParallelMinOps,
		f.ParallelStrategy,
		f.EnableMetrics,
		f.StrictMode,
	)
}

// Helper functions
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}