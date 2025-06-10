package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error: field '%s' with value '%v': %s", e.Field, e.Value, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// Validate validates the entire configuration
func Validate(cfg *Config) error {
	var errors ValidationErrors

	// Validate engine configuration
	if errs := validateEngine(&cfg.Engine); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate performance configuration
	if errs := validatePerformance(&cfg.Performance); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate logging configuration
	if errs := validateLogging(&cfg.Logging); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate version
	if cfg.Version == "" {
		errors = append(errors, ValidationError{
			Field:   "version",
			Value:   cfg.Version,
			Message: "version cannot be empty",
		})
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// validateEngine validates engine configuration
func validateEngine(cfg *EngineConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate Vault config
	if errs := validateVault(&cfg.Vault); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate AWS config
	if errs := validateAWS(&cfg.AWS); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate parser config
	if errs := validateParser(&cfg.Parser); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate dataflow order
	validOrders := []string{"breadth-first", "depth-first", "legacy"}
	if !contains(validOrders, cfg.DataflowOrder) {
		errors = append(errors, ValidationError{
			Field:   "engine.dataflow_order",
			Value:   cfg.DataflowOrder,
			Message: fmt.Sprintf("must be one of: %v", validOrders),
		})
	}

	// Validate output format
	validFormats := []string{"yaml", "json", "go-patch"}
	if !contains(validFormats, cfg.OutputFormat) {
		errors = append(errors, ValidationError{
			Field:   "engine.output_format",
			Value:   cfg.OutputFormat,
			Message: fmt.Sprintf("must be one of: %v", validFormats),
		})
	}

	return errors
}

// validateVault validates Vault configuration
func validateVault(cfg *VaultConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate address if provided
	if cfg.Address != "" {
		if u, err := url.Parse(cfg.Address); err != nil {
			errors = append(errors, ValidationError{
				Field:   "engine.vault.address",
				Value:   cfg.Address,
				Message: fmt.Sprintf("invalid URL: %v", err),
			})
		} else if u.Scheme == "" || u.Host == "" {
			errors = append(errors, ValidationError{
				Field:   "engine.vault.address",
				Value:   cfg.Address,
				Message: "invalid URL: must have scheme and host",
			})
		}
	}

	// Validate timeout
	if cfg.Timeout != "" {
		if _, err := time.ParseDuration(cfg.Timeout); err != nil {
			errors = append(errors, ValidationError{
				Field:   "engine.vault.timeout",
				Value:   cfg.Timeout,
				Message: fmt.Sprintf("invalid duration: %v", err),
			})
		}
	}

	return errors
}

// validateAWS validates AWS configuration
func validateAWS(cfg *AWSConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate region format if provided
	if cfg.Region != "" && !isValidAWSRegion(cfg.Region) {
		errors = append(errors, ValidationError{
			Field:   "engine.aws.region",
			Value:   cfg.Region,
			Message: "invalid AWS region format",
		})
	}

	// Validate endpoint if provided
	if cfg.Endpoint != "" {
		if u, err := url.Parse(cfg.Endpoint); err != nil {
			errors = append(errors, ValidationError{
				Field:   "engine.aws.endpoint",
				Value:   cfg.Endpoint,
				Message: fmt.Sprintf("invalid URL: %v", err),
			})
		} else if u.Scheme == "" || u.Host == "" {
			errors = append(errors, ValidationError{
				Field:   "engine.aws.endpoint",
				Value:   cfg.Endpoint,
				Message: "invalid URL: must have scheme and host",
			})
		}
	}

	return errors
}

// validateParser validates parser configuration
func validateParser(cfg *ParserConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate max document size
	if cfg.MaxDocumentSize <= 0 {
		errors = append(errors, ValidationError{
			Field:   "engine.parser.max_document_size",
			Value:   cfg.MaxDocumentSize,
			Message: "must be greater than 0",
		})
	}

	// Warn if max document size is very large
	if cfg.MaxDocumentSize > 100*1024*1024 { // 100MB
		errors = append(errors, ValidationError{
			Field:   "engine.parser.max_document_size",
			Value:   cfg.MaxDocumentSize,
			Message: "warning: very large document size may cause memory issues",
		})
	}

	return errors
}

// validatePerformance validates performance configuration
func validatePerformance(cfg *PerformanceConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate cache config
	if errs := validateCache(&cfg.Cache); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate concurrency config
	if errs := validateConcurrency(&cfg.Concurrency); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate memory config
	if errs := validateMemory(&cfg.Memory); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Validate I/O config
	if errs := validateIO(&cfg.IO); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	return errors
}

// validateCache validates cache configuration
func validateCache(cfg *CacheConfig) ValidationErrors {
	var errors ValidationErrors

	if cfg.ExpressionCacheSize < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.cache.expression_cache_size",
			Value:   cfg.ExpressionCacheSize,
			Message: "cannot be negative",
		})
	}

	if cfg.OperatorCacheSize < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.cache.operator_cache_size",
			Value:   cfg.OperatorCacheSize,
			Message: "cannot be negative",
		})
	}

	if cfg.FileCacheSize < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.cache.file_cache_size",
			Value:   cfg.FileCacheSize,
			Message: "cannot be negative",
		})
	}

	if cfg.TTL < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.cache.ttl",
			Value:   cfg.TTL,
			Message: "cannot be negative",
		})
	}

	return errors
}

// validateConcurrency validates concurrency configuration
func validateConcurrency(cfg *ConcurrencyConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate max workers
	if cfg.MaxWorkers < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.concurrency.max_workers",
			Value:   cfg.MaxWorkers,
			Message: "cannot be negative",
		})
	}

	// Auto-detect CPU count if max workers is 0
	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = runtime.NumCPU()
	}

	// Warn if max workers is very high
	if cfg.MaxWorkers > runtime.NumCPU()*4 {
		errors = append(errors, ValidationError{
			Field:   "performance.concurrency.max_workers",
			Value:   cfg.MaxWorkers,
			Message: fmt.Sprintf("warning: very high worker count (%d) for %d CPUs", cfg.MaxWorkers, runtime.NumCPU()),
		})
	}

	// Validate queue size
	if cfg.QueueSize <= 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.concurrency.queue_size",
			Value:   cfg.QueueSize,
			Message: "must be greater than 0",
		})
	}

	// Validate batch size
	if cfg.BatchSize <= 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.concurrency.batch_size",
			Value:   cfg.BatchSize,
			Message: "must be greater than 0",
		})
	}

	return errors
}

// validateMemory validates memory configuration
func validateMemory(cfg *MemoryConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate max heap size
	if cfg.MaxHeapSize < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.memory.max_heap_size",
			Value:   cfg.MaxHeapSize,
			Message: "cannot be negative",
		})
	}

	// Validate GC percent
	if cfg.GCPercent < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.memory.gc_percent",
			Value:   cfg.GCPercent,
			Message: "cannot be negative",
		})
	}

	return errors
}

// validateIO validates I/O configuration
func validateIO(cfg *IOConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate timeouts
	if cfg.ConnectionTimeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.io.connection_timeout",
			Value:   cfg.ConnectionTimeout,
			Message: "must be greater than 0",
		})
	}

	if cfg.RequestTimeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.io.request_timeout",
			Value:   cfg.RequestTimeout,
			Message: "must be greater than 0",
		})
	}

	// Validate retries
	if cfg.MaxRetries < 0 {
		errors = append(errors, ValidationError{
			Field:   "performance.io.max_retries",
			Value:   cfg.MaxRetries,
			Message: "cannot be negative",
		})
	}

	return errors
}

// validateLogging validates logging configuration
func validateLogging(cfg *LoggingConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate log level
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal"}
	if !contains(validLevels, strings.ToLower(cfg.Level)) {
		errors = append(errors, ValidationError{
			Field:   "logging.level",
			Value:   cfg.Level,
			Message: fmt.Sprintf("must be one of: %v", validLevels),
		})
	}

	// Validate log format
	validFormats := []string{"text", "json", "logfmt"}
	if !contains(validFormats, cfg.Format) {
		errors = append(errors, ValidationError{
			Field:   "logging.format",
			Value:   cfg.Format,
			Message: fmt.Sprintf("must be one of: %v", validFormats),
		})
	}

	// Validate output
	if cfg.Output != "stdout" && cfg.Output != "stderr" {
		// Check if it's a valid file path
		dir := filepath.Dir(cfg.Output)
		if _, err := os.Stat(dir); err != nil {
			errors = append(errors, ValidationError{
				Field:   "logging.output",
				Value:   cfg.Output,
				Message: fmt.Sprintf("directory does not exist: %s", dir),
			})
		}
	}

	return errors
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isValidAWSRegion(region string) bool {
	// Basic validation for AWS region format
	// Format: xx-xxxx-n (e.g., us-east-1, eu-west-2)
	parts := strings.Split(region, "-")
	return len(parts) >= 3 && len(parts[0]) >= 2 && len(parts[1]) >= 3
}
