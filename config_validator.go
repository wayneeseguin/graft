package spruce

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return "configuration validation failed:\n" + strings.Join(messages, "\n")
}

// ConfigValidator validates performance configurations
type ConfigValidator struct {
	rules []ValidationRule
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Name      string
	Validator func(*PerformanceConfig) []ValidationError
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	v := &ConfigValidator{
		rules: make([]ValidationRule, 0),
	}
	v.registerDefaultRules()
	return v
}

// Validate validates a configuration against all rules
func (v *ConfigValidator) Validate(config *PerformanceConfig) error {
	var allErrors ValidationErrors

	for _, rule := range v.rules {
		if errors := rule.Validator(config); len(errors) > 0 {
			allErrors = append(allErrors, errors...)
		}
	}

	if len(allErrors) > 0 {
		return allErrors
	}
	return nil
}

// AddRule adds a custom validation rule
func (v *ConfigValidator) AddRule(name string, validator func(*PerformanceConfig) []ValidationError) {
	v.rules = append(v.rules, ValidationRule{
		Name:      name,
		Validator: validator,
	})
}

// registerDefaultRules registers built-in validation rules
func (v *ConfigValidator) registerDefaultRules() {
	// Cache size validation
	v.AddRule("cache_sizes", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError
		cache := config.Performance.Cache

		if cache.ExpressionCacheSize < 100 {
			errors = append(errors, ValidationError{
				Field:   "cache.expression_cache_size",
				Message: "must be at least 100",
			})
		}

		if cache.OperatorCacheSize < 100 {
			errors = append(errors, ValidationError{
				Field:   "cache.operator_cache_size",
				Message: "must be at least 100",
			})
		}

		if cache.TokenCacheSize < 100 {
			errors = append(errors, ValidationError{
				Field:   "cache.token_cache_size",
				Message: "must be at least 100",
			})
		}

		if cache.TTLSeconds < 0 {
			errors = append(errors, ValidationError{
				Field:   "cache.ttl_seconds",
				Message: "must be non-negative",
			})
		}

		// Hierarchical cache validation
		if cache.Hierarchical.L1Size < 10 {
			errors = append(errors, ValidationError{
				Field:   "cache.hierarchical.l1_size",
				Message: "must be at least 10",
			})
		}

		if cache.Hierarchical.L2Size < cache.Hierarchical.L1Size {
			errors = append(errors, ValidationError{
				Field:   "cache.hierarchical.l2_size",
				Message: "must be greater than or equal to L1 size",
			})
		}

		if cache.Hierarchical.SyncIntervalSeconds < 0 {
			errors = append(errors, ValidationError{
				Field:   "cache.hierarchical.sync_interval_seconds",
				Message: "must be non-negative",
			})
		}

		// Cache warming validation
		validStrategies := map[string]bool{
			"frequency": true,
			"pattern":   true,
			"hybrid":    true,
		}
		if !validStrategies[cache.Warming.Strategy] {
			errors = append(errors, ValidationError{
				Field:   "cache.warming.strategy",
				Message: fmt.Sprintf("must be one of: frequency, pattern, hybrid (got %s)", cache.Warming.Strategy),
			})
		}

		if cache.Warming.TopExpressions < 0 {
			errors = append(errors, ValidationError{
				Field:   "cache.warming.top_expressions",
				Message: "must be non-negative",
			})
		}

		return errors
	})

	// Concurrency validation
	v.AddRule("concurrency", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError
		conc := config.Performance.Concurrency

		if conc.MaxWorkers < 1 {
			errors = append(errors, ValidationError{
				Field:   "concurrency.max_workers",
				Message: "must be at least 1",
			})
		}

		if conc.QueueSize < 1 {
			errors = append(errors, ValidationError{
				Field:   "concurrency.queue_size",
				Message: "must be at least 1",
			})
		}

		if conc.WorkerIdleTimeoutSecs < 0 {
			errors = append(errors, ValidationError{
				Field:   "concurrency.worker_idle_timeout_seconds",
				Message: "must be non-negative",
			})
		}

		// Rate limit validation
		if conc.RateLimit.RequestsPerSecond < 0 {
			errors = append(errors, ValidationError{
				Field:   "concurrency.rate_limit.requests_per_second",
				Message: "must be non-negative",
			})
		}

		if conc.RateLimit.BurstSize < 0 {
			errors = append(errors, ValidationError{
				Field:   "concurrency.rate_limit.burst_size",
				Message: "must be non-negative",
			})
		}

		return errors
	})

	// Memory validation
	v.AddRule("memory", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError
		mem := config.Performance.Memory

		if mem.MaxHeapMB < 32 {
			errors = append(errors, ValidationError{
				Field:   "memory.max_heap_mb",
				Message: "must be at least 32MB",
			})
		}

		if mem.GCPercent < 0 {
			errors = append(errors, ValidationError{
				Field:   "memory.gc_percent",
				Message: "must be non-negative",
			})
		}

		// Pool sizes validation
		if mem.PoolSizes.BufferPool < 0 {
			errors = append(errors, ValidationError{
				Field:   "memory.pool_sizes.buffer_pool",
				Message: "must be non-negative",
			})
		}

		if mem.PoolSizes.StringSlicePool < 0 {
			errors = append(errors, ValidationError{
				Field:   "memory.pool_sizes.string_slice_pool",
				Message: "must be non-negative",
			})
		}

		if mem.PoolSizes.TokenPool < 0 {
			errors = append(errors, ValidationError{
				Field:   "memory.pool_sizes.token_pool",
				Message: "must be non-negative",
			})
		}

		// String interning validation
		if mem.StringInterning.MaxEntries < 0 {
			errors = append(errors, ValidationError{
				Field:   "memory.string_interning.max_entries",
				Message: "must be non-negative",
			})
		}

		return errors
	})

	// Parsing validation
	v.AddRule("parsing", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError
		parse := config.Performance.Parsing

		// Memoization validation
		if parse.Memoization.CacheSize < 0 {
			errors = append(errors, ValidationError{
				Field:   "parsing.memoization.cache_size",
				Message: "must be non-negative",
			})
		}

		if parse.Memoization.TTLSeconds < 0 {
			errors = append(errors, ValidationError{
				Field:   "parsing.memoization.ttl_seconds",
				Message: "must be non-negative",
			})
		}

		// Validate expensive operators list
		validOperators := map[string]bool{
			"vault": true, "file": true, "awsparam": true, "awssecret": true,
			"grab": true, "concat": true, "join": true, "static_ips": true,
			"calc": true, "defer": true, "load": true,
		}

		for _, op := range parse.LazyEvaluation.ExpensiveOperators {
			if !validOperators[op] {
				errors = append(errors, ValidationError{
					Field:   "parsing.lazy_evaluation.expensive_operators",
					Message: fmt.Sprintf("unknown operator: %s", op),
				})
			}
		}

		return errors
	})

	// I/O validation
	v.AddRule("io", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError
		io := config.Performance.IO

		if io.ConnectionPoolSize < 1 {
			errors = append(errors, ValidationError{
				Field:   "io.connection_pool_size",
				Message: "must be at least 1",
			})
		}

		if io.RequestTimeoutSeconds < 1 {
			errors = append(errors, ValidationError{
				Field:   "io.request_timeout_seconds",
				Message: "must be at least 1 second",
			})
		}

		if io.RetryAttempts < 0 {
			errors = append(errors, ValidationError{
				Field:   "io.retry_attempts",
				Message: "must be non-negative",
			})
		}

		if io.RetryBackoffSeconds < 0 {
			errors = append(errors, ValidationError{
				Field:   "io.retry_backoff_seconds",
				Message: "must be non-negative",
			})
		}

		// Deduplication validation
		if io.Deduplication.WindowSeconds < 0 {
			errors = append(errors, ValidationError{
				Field:   "io.deduplication.window_seconds",
				Message: "must be non-negative",
			})
		}

		if io.Deduplication.MaxPending < 0 {
			errors = append(errors, ValidationError{
				Field:   "io.deduplication.max_pending",
				Message: "must be non-negative",
			})
		}

		return errors
	})

	// Monitoring validation
	v.AddRule("monitoring", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError
		mon := config.Performance.Monitoring

		if mon.MetricsIntervalSeconds < 1 {
			errors = append(errors, ValidationError{
				Field:   "monitoring.metrics_interval_seconds",
				Message: "must be at least 1 second",
			})
		}

		if mon.SlowOperationThresholdMs < 0 {
			errors = append(errors, ValidationError{
				Field:   "monitoring.slow_operation_threshold_ms",
				Message: "must be non-negative",
			})
		}

		return errors
	})

	// Auto-tuning validation
	v.AddRule("auto_tuning", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError
		auto := config.Performance.AutoTuning

		if auto.AnalysisIntervalSeconds < 60 {
			errors = append(errors, ValidationError{
				Field:   "auto_tuning.analysis_interval_seconds",
				Message: "must be at least 60 seconds",
			})
		}

		if auto.AdjustmentThreshold < 0 || auto.AdjustmentThreshold > 1 {
			errors = append(errors, ValidationError{
				Field:   "auto_tuning.adjustment_threshold",
				Message: "must be between 0 and 1",
			})
		}

		if auto.MaxAdjustmentsPerHour < 0 {
			errors = append(errors, ValidationError{
				Field:   "auto_tuning.max_adjustments_per_hour",
				Message: "must be non-negative",
			})
		}

		return errors
	})

	// Cross-field validation
	v.AddRule("cross_field", func(config *PerformanceConfig) []ValidationError {
		var errors []ValidationError

		// Ensure worker count doesn't exceed queue size
		if config.Performance.Concurrency.MaxWorkers > config.Performance.Concurrency.QueueSize {
			errors = append(errors, ValidationError{
				Field:   "concurrency",
				Message: "max_workers should not exceed queue_size",
			})
		}

		// Ensure rate limit burst size is reasonable
		if config.Performance.Concurrency.RateLimit.Enabled &&
			config.Performance.Concurrency.RateLimit.BurstSize > config.Performance.Concurrency.RateLimit.RequestsPerSecond*10 {
			errors = append(errors, ValidationError{
				Field:   "concurrency.rate_limit",
				Message: "burst_size should not exceed 10x requests_per_second",
			})
		}

		// Ensure cache TTL is reasonable compared to analysis interval
		if config.Performance.AutoTuning.Enabled &&
			config.Performance.Cache.TTLSeconds > 0 &&
			config.Performance.Cache.TTLSeconds < config.Performance.AutoTuning.AnalysisIntervalSeconds {
			errors = append(errors, ValidationError{
				Field:   "cache.ttl_seconds",
				Message: "should be greater than auto_tuning.analysis_interval_seconds for effective tuning",
			})
		}

		return errors
	})
}

// ValidateConfigFile validates a configuration file
func ValidateConfigFile(path string) error {
	loader := NewConfigLoader(path)
	config, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	validator := NewConfigValidator()
	return validator.Validate(config)
}

// GetFieldValue gets a field value from the configuration using dot notation
func GetFieldValue(config *PerformanceConfig, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 || parts[0] != "performance" {
		return nil, fmt.Errorf("path must start with 'performance'")
	}

	v := reflect.ValueOf(config.Performance)
	for i := 1; i < len(parts); i++ {
		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("cannot traverse non-struct field at %s", strings.Join(parts[:i], "."))
		}

		// Convert snake_case to CamelCase for struct field lookup
		fieldName := toCamelCase(parts[i])
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found", parts[i])
		}
		v = field
	}

	return v.Interface(), nil
}

// SetFieldValue sets a field value in the configuration using dot notation
func SetFieldValue(config *PerformanceConfig, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 || parts[0] != "performance" {
		return fmt.Errorf("path must start with 'performance'")
	}

	v := reflect.ValueOf(&config.Performance).Elem()
	for i := 1; i < len(parts)-1; i++ {
		if v.Kind() != reflect.Struct {
			return fmt.Errorf("cannot traverse non-struct field at %s", strings.Join(parts[:i], "."))
		}

		fieldName := toCamelCase(parts[i])
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			return fmt.Errorf("field %s not found", parts[i])
		}
		v = field
	}

	// Set the final field
	fieldName := toCamelCase(parts[len(parts)-1])
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found", parts[len(parts)-1])
	}

	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set", parts[len(parts)-1])
	}

	// Convert value to appropriate type
	val := reflect.ValueOf(value)
	if field.Type() != val.Type() {
		// Try to convert
		if val.Type().ConvertibleTo(field.Type()) {
			val = val.Convert(field.Type())
		} else {
			return fmt.Errorf("cannot convert %v to %v", val.Type(), field.Type())
		}
	}

	field.Set(val)
	return nil
}

// toCamelCase converts snake_case to CamelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}