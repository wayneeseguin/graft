package graft

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// PerformanceConfig represents the complete performance configuration
type PerformanceConfig struct {
	Performance struct {
		Cache       CacheConfiguration       `yaml:"cache"`
		Concurrency ConcurrencyConfiguration `yaml:"concurrency"`
		Memory      MemoryConfiguration      `yaml:"memory"`
		Parsing     ParsingConfiguration     `yaml:"parsing"`
		IO          IOConfiguration          `yaml:"io"`
		Monitoring  MonitoringConfiguration  `yaml:"monitoring"`
		AutoTuning  AutoTuningConfiguration  `yaml:"auto_tuning"`
	} `yaml:"performance"`
}

// CacheConfiguration holds all cache-related settings
type CacheConfiguration struct {
	ExpressionCacheSize int                       `yaml:"expression_cache_size" env:"GRAFT_EXPRESSION_CACHE_SIZE" default:"10000"`
	OperatorCacheSize   int                       `yaml:"operator_cache_size" env:"GRAFT_OPERATOR_CACHE_SIZE" default:"50000"`
	TokenCacheSize      int                       `yaml:"token_cache_size" env:"GRAFT_TOKEN_CACHE_SIZE" default:"20000"`
	TTLSeconds          int                       `yaml:"ttl_seconds" env:"GRAFT_CACHE_TTL" default:"3600"`
	Hierarchical        HierarchicalCacheSettings   `yaml:"hierarchical"`
	Warming             CacheWarmingConfiguration `yaml:"warming"`
}

// HierarchicalCacheConfig for L1/L2 cache settings
type HierarchicalCacheSettings struct {
	L1Size              int  `yaml:"l1_size" env:"GRAFT_L1_CACHE_SIZE" default:"1000"`
	L2Size              int  `yaml:"l2_size" env:"GRAFT_L2_CACHE_SIZE" default:"10000"`
	L2Enabled           bool `yaml:"l2_enabled" env:"GRAFT_L2_ENABLED" default:"true"`
	SyncIntervalSeconds int  `yaml:"sync_interval_seconds" env:"GRAFT_L2_SYNC_INTERVAL" default:"300"`
}

// CacheWarmingConfiguration for cache warming settings
type CacheWarmingConfiguration struct {
	Enabled               bool   `yaml:"enabled" env:"GRAFT_CACHE_WARMING_ENABLED" default:"true"`
	Strategy              string `yaml:"strategy" env:"GRAFT_CACHE_WARMING_STRATEGY" default:"hybrid"`
	StartupTimeoutSeconds int    `yaml:"startup_timeout_seconds" env:"GRAFT_STARTUP_TIMEOUT" default:"30"`
	TopExpressions        int    `yaml:"top_expressions" env:"GRAFT_TOP_EXPRESSIONS" default:"50"`
}

// ConcurrencyConfiguration holds concurrency-related settings
type ConcurrencyConfiguration struct {
	MaxWorkers            int                     `yaml:"max_workers" env:"GRAFT_MAX_WORKERS" default:"100"`
	QueueSize             int                     `yaml:"queue_size" env:"GRAFT_QUEUE_SIZE" default:"1000"`
	WorkerIdleTimeoutSecs int                     `yaml:"worker_idle_timeout_seconds" env:"GRAFT_WORKER_IDLE_TIMEOUT" default:"60"`
	RateLimit             RateLimitConfiguration  `yaml:"rate_limit"`
}

// RateLimitConfiguration for rate limiting settings
type RateLimitConfiguration struct {
	Enabled            bool `yaml:"enabled" env:"GRAFT_RATE_LIMIT_ENABLED" default:"true"`
	RequestsPerSecond  int  `yaml:"requests_per_second" env:"GRAFT_RPS" default:"1000"`
	BurstSize          int  `yaml:"burst_size" env:"GRAFT_BURST_SIZE" default:"100"`
}

// MemoryConfiguration holds memory-related settings
type MemoryConfiguration struct {
	MaxHeapMB       int                      `yaml:"max_heap_mb" env:"GRAFT_MAX_HEAP_MB" default:"4096"`
	GCPercent       int                      `yaml:"gc_percent" env:"GOGC" default:"100"`
	PoolSizes       PoolConfiguration        `yaml:"pool_sizes"`
	StringInterning StringInterningConfig    `yaml:"string_interning"`
}

// PoolConfiguration for object pool settings
type PoolConfiguration struct {
	BufferPool     int `yaml:"buffer_pool" env:"GRAFT_BUFFER_POOL_SIZE" default:"1000"`
	StringSlicePool int `yaml:"string_slice_pool" env:"GRAFT_STRING_SLICE_POOL_SIZE" default:"5000"`
	TokenPool      int `yaml:"token_pool" env:"GRAFT_TOKEN_POOL_SIZE" default:"10000"`
}

// StringInterningConfig for string interning settings
type StringInterningConfig struct {
	Enabled    bool `yaml:"enabled" env:"GRAFT_STRING_INTERNING_ENABLED" default:"true"`
	MaxEntries int  `yaml:"max_entries" env:"GRAFT_STRING_INTERNING_MAX" default:"10000"`
}

// ParsingConfiguration holds parsing-related settings
type ParsingConfiguration struct {
	Memoization     MemoizationConfig     `yaml:"memoization"`
	LazyEvaluation  LazyEvaluationConfig  `yaml:"lazy_evaluation"`
}

// MemoizationConfig for parser memoization settings
type MemoizationConfig struct {
	Enabled    bool `yaml:"enabled" env:"GRAFT_MEMOIZATION_ENABLED" default:"true"`
	CacheSize  int  `yaml:"cache_size" env:"GRAFT_MEMOIZATION_CACHE_SIZE" default:"5000"`
	TTLSeconds int  `yaml:"ttl_seconds" env:"GRAFT_MEMOIZATION_TTL" default:"1800"`
}

// LazyEvaluationConfig for lazy evaluation settings
type LazyEvaluationConfig struct {
	Enabled            bool     `yaml:"enabled" env:"GRAFT_LAZY_EVAL_ENABLED" default:"true"`
	ExpensiveOperators []string `yaml:"expensive_operators"`
}

// IOConfiguration holds I/O-related settings
type IOConfiguration struct {
	ConnectionPoolSize    int                `yaml:"connection_pool_size" env:"GRAFT_CONN_POOL_SIZE" default:"50"`
	RequestTimeoutSeconds int                `yaml:"request_timeout_seconds" env:"GRAFT_REQUEST_TIMEOUT" default:"30"`
	RetryAttempts         int                `yaml:"retry_attempts" env:"GRAFT_RETRY_ATTEMPTS" default:"3"`
	RetryBackoffSeconds   int                `yaml:"retry_backoff_seconds" env:"GRAFT_RETRY_BACKOFF" default:"1"`
	Deduplication         DeduplicationConfig `yaml:"deduplication"`
}

// DeduplicationConfig for I/O deduplication settings
type DeduplicationConfig struct {
	Enabled       bool `yaml:"enabled" env:"GRAFT_DEDUP_ENABLED" default:"true"`
	WindowSeconds int  `yaml:"window_seconds" env:"GRAFT_DEDUP_WINDOW" default:"5"`
	MaxPending    int  `yaml:"max_pending" env:"GRAFT_DEDUP_MAX_PENDING" default:"100"`
}

// MonitoringConfiguration holds monitoring-related settings
type MonitoringConfiguration struct {
	MetricsEnabled             bool `yaml:"metrics_enabled" env:"GRAFT_METRICS_ENABLED" default:"true"`
	MetricsIntervalSeconds     int  `yaml:"metrics_interval_seconds" env:"GRAFT_METRICS_INTERVAL" default:"60"`
	PerformanceTracking        bool `yaml:"performance_tracking" env:"GRAFT_PERF_TRACKING" default:"true"`
	SlowOperationThresholdMs   int  `yaml:"slow_operation_threshold_ms" env:"GRAFT_SLOW_OP_THRESHOLD" default:"100"`
}

// AutoTuningConfiguration holds auto-tuning settings
type AutoTuningConfiguration struct {
	Enabled                  bool    `yaml:"enabled" env:"GRAFT_AUTO_TUNING_ENABLED" default:"false"`
	AnalysisIntervalSeconds  int     `yaml:"analysis_interval_seconds" env:"GRAFT_ANALYSIS_INTERVAL" default:"300"`
	AdjustmentThreshold      float64 `yaml:"adjustment_threshold" env:"GRAFT_ADJUSTMENT_THRESHOLD" default:"0.1"`
	MaxAdjustmentsPerHour    int     `yaml:"max_adjustments_per_hour" env:"GRAFT_MAX_ADJUSTMENTS" default:"6"`
}

// ConfigLoader manages configuration loading and environment variable overrides
type ConfigLoader struct {
	mu            sync.RWMutex
	config        *PerformanceConfig
	configPath    string
	changeHandlers []func(*PerformanceConfig)
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath:     configPath,
		changeHandlers: make([]func(*PerformanceConfig), 0),
	}
}

// Load reads configuration from file and applies environment overrides
func (c *ConfigLoader) Load() (*PerformanceConfig, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Start with defaults
	config := &PerformanceConfig{}
	c.applyDefaults(config)

	// Load from file if it exists
	if c.configPath != "" {
		if _, err := os.Stat(c.configPath); err == nil {
			data, err := ioutil.ReadFile(c.configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %v", err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %v", err)
			}
		}
	}

	// Apply environment variable overrides
	c.applyEnvironmentOverrides(config)

	c.config = config
	return config, nil
}

// GetConfig returns the current configuration
func (c *ConfigLoader) GetConfig() *PerformanceConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// Reload reloads the configuration from file
func (c *ConfigLoader) Reload() error {
	newConfig, err := c.Load()
	if err != nil {
		return err
	}

	// Notify change handlers
	for _, handler := range c.changeHandlers {
		handler(newConfig)
	}

	return nil
}

// OnChange registers a callback for configuration changes
func (c *ConfigLoader) OnChange(handler func(*PerformanceConfig)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changeHandlers = append(c.changeHandlers, handler)
}

// applyDefaults sets default values using struct tags
func (c *ConfigLoader) applyDefaults(config *PerformanceConfig) {
	c.applyDefaultsToStruct(reflect.ValueOf(&config.Performance).Elem())
}

// applyDefaultsToStruct recursively applies defaults to a struct
func (c *ConfigLoader) applyDefaultsToStruct(v reflect.Value) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		typeField := t.Field(i)

		if field.Kind() == reflect.Struct {
			c.applyDefaultsToStruct(field)
			continue
		}

		defaultTag := typeField.Tag.Get("default")
		if defaultTag == "" {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			if field.String() == "" {
				field.SetString(defaultTag)
			}
		case reflect.Int:
			if field.Int() == 0 {
				if val, err := strconv.Atoi(defaultTag); err == nil {
					field.SetInt(int64(val))
				}
			}
		case reflect.Bool:
			if !field.Bool() {
				if val, err := strconv.ParseBool(defaultTag); err == nil {
					field.SetBool(val)
				}
			}
		case reflect.Float64:
			if field.Float() == 0 {
				if val, err := strconv.ParseFloat(defaultTag, 64); err == nil {
					field.SetFloat(val)
				}
			}
		case reflect.Slice:
			if field.Len() == 0 && field.Type().Elem().Kind() == reflect.String {
				// Special handling for string slices with defaults
				if defaultTag != "" {
					defaults := strings.Split(defaultTag, ",")
					field.Set(reflect.ValueOf(defaults))
				}
			}
		}
	}
}

// applyEnvironmentOverrides applies environment variable overrides
func (c *ConfigLoader) applyEnvironmentOverrides(config *PerformanceConfig) {
	c.applyEnvToStruct(reflect.ValueOf(&config.Performance).Elem())
}

// applyEnvToStruct recursively applies environment overrides to a struct
func (c *ConfigLoader) applyEnvToStruct(v reflect.Value) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		typeField := t.Field(i)

		if field.Kind() == reflect.Struct {
			c.applyEnvToStruct(field)
			continue
		}

		envTag := typeField.Tag.Get("env")
		if envTag == "" {
			continue
		}

		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(envValue)
		case reflect.Int:
			if val, err := strconv.Atoi(envValue); err == nil {
				field.SetInt(int64(val))
			}
		case reflect.Bool:
			if val, err := strconv.ParseBool(envValue); err == nil {
				field.SetBool(val)
			}
		case reflect.Float64:
			if val, err := strconv.ParseFloat(envValue, 64); err == nil {
				field.SetFloat(val)
			}
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				values := strings.Split(envValue, ",")
				field.Set(reflect.ValueOf(values))
			}
		}
	}
}

// Global configuration instance
var (
	globalConfig     *PerformanceConfig
	globalConfigOnce sync.Once
	configLoader     *ConfigLoader
)

// InitializeConfig initializes the global configuration
func InitializeConfig(configPath string) error {
	var err error
	globalConfigOnce.Do(func() {
		configLoader = NewConfigLoader(configPath)
		globalConfig, err = configLoader.Load()
	})
	return err
}

// GetGlobalConfig returns the global configuration instance
func GetGlobalConfig() *PerformanceConfig {
	if globalConfig == nil {
		// Initialize with defaults if not yet initialized
		InitializeConfig("")
	}
	return globalConfig
}

// ReloadGlobalConfig reloads the global configuration
func ReloadGlobalConfig() error {
	if configLoader == nil {
		return fmt.Errorf("configuration not initialized")
	}
	return configLoader.Reload()
}

// RegisterConfigChangeHandler registers a handler for configuration changes
func RegisterConfigChangeHandler(handler func(*PerformanceConfig)) {
	if configLoader != nil {
		configLoader.OnChange(handler)
	}
}

// ConfigToYAML converts a configuration to YAML string
func ConfigToYAML(config *PerformanceConfig) (string, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ConfigFromYAML creates a configuration from YAML string
func ConfigFromYAML(yamlStr string) (*PerformanceConfig, error) {
	config := &PerformanceConfig{}
	loader := NewConfigLoader("")
	loader.applyDefaults(config)
	
	if err := yaml.Unmarshal([]byte(yamlStr), config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// ConfigDuration converts seconds to time.Duration
func ConfigDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}