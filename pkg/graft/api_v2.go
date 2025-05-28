package graft

import (
	"context"
	"io"
)

// DocumentV2 represents a YAML/JSON document in a more user-friendly format
// This abstraction hides the internal map[interface{}]interface{} representation
type DocumentV2 interface {
	// Get retrieves a value at the given path (e.g., "meta.instance_groups.0.name")
	Get(path string) (interface{}, error)
	
	// GetString retrieves a string value at the given path
	GetString(path string) (string, error)
	
	// GetInt retrieves an integer value at the given path
	GetInt(path string) (int, error)
	
	// GetBool retrieves a boolean value at the given path
	GetBool(path string) (bool, error)
	
	// GetSlice retrieves a slice value at the given path
	GetSlice(path string) ([]interface{}, error)
	
	// GetMap retrieves a map value at the given path
	GetMap(path string) (map[interface{}]interface{}, error)
	
	// Set sets a value at the given path
	Set(path string, value interface{}) error
	
	// Delete removes a value at the given path
	Delete(path string) error
	
	// Keys returns all top-level keys
	Keys() []string
	
	// ToYAML converts the document to YAML bytes
	ToYAML() ([]byte, error)
	
	// ToJSON converts the document to JSON bytes
	ToJSON() ([]byte, error)
	
	// RawData returns the underlying data structure
	RawData() interface{}
	
	// Clone creates a deep copy of the document
	Clone() DocumentV2
}

// EngineV2 is the enhanced interface for using graft as a library
type EngineV2 interface {
	// Document operations
	ParseYAML(data []byte) (DocumentV2, error)
	ParseJSON(data []byte) (DocumentV2, error)
	ParseFile(path string) (DocumentV2, error)
	ParseReader(reader io.Reader) (DocumentV2, error)
	
	// Merge operations with builder pattern options
	Merge(ctx context.Context, docs ...DocumentV2) MergeBuilder
	MergeFiles(ctx context.Context, paths ...string) MergeBuilder
	MergeReaders(ctx context.Context, readers ...io.Reader) MergeBuilder
	
	// Evaluate processes operators in a document
	Evaluate(ctx context.Context, doc DocumentV2) (DocumentV2, error)
	
	// Output operations
	ToYAML(doc DocumentV2) ([]byte, error)
	ToJSON(doc DocumentV2) ([]byte, error)
	ToJSONIndent(doc DocumentV2, indent string) ([]byte, error)
	
	// Operator management
	RegisterOperator(name string, op Operator) error
	UnregisterOperator(name string) error
	ListOperators() []string
	
	// Configuration
	WithLoggerV2(logger LoggerV2) EngineV2
	WithVaultClientV2(client VaultClientV2) EngineV2
	WithAWSConfig(config AWSConfig) EngineV2
}

// MergeBuilder provides a fluent interface for merge operations
type MergeBuilder interface {
	// WithPrune specifies keys to remove from the final output
	WithPrune(keys ...string) MergeBuilder
	
	// WithCherryPick specifies keys to keep in the final output (all others removed)
	WithCherryPick(keys ...string) MergeBuilder
	
	// SkipEvaluation skips operator evaluation after merging
	SkipEvaluation() MergeBuilder
	
	// EnableGoPatch enables go-patch format parsing
	EnableGoPatch() MergeBuilder
	
	// FallbackAppend uses append instead of inline for arrays by default
	FallbackAppend() MergeBuilder
	
	// Execute performs the merge operation
	Execute() (DocumentV2, error)
}

// EngineOptions configures a new engine instance using functional options
type EngineOptions struct {
	LoggerV2             LoggerV2
	VaultClientV2        VaultClientV2
	AWSConfig          *AWSConfig
	EnableCache        bool
	CacheSize          int
	MaxConcurrency     int
	UseEnhancedParser  bool // Deprecated: enhanced parser is now always enabled
	EnableMetrics      bool
	CustomOperators    map[string]Operator
	VaultAddress       string
	VaultToken         string
	DebugLogging       bool
	AWSRegion          string
}

// EngineOption is a functional option for configuring an engine
type EngineOption func(*EngineOptions)

// WithLoggerV2 sets the logger for the engine
func WithLoggerV2(logger LoggerV2) EngineOption {
	return func(opts *EngineOptions) {
		opts.LoggerV2 = logger
	}
}

// WithVaultClientV2 sets the vault client for the engine
func WithVaultClientV2(client VaultClientV2) EngineOption {
	return func(opts *EngineOptions) {
		opts.VaultClientV2 = client
	}
}

// WithAWSConfig sets the AWS configuration
func WithAWSConfig(config *AWSConfig) EngineOption {
	return func(opts *EngineOptions) {
		opts.AWSConfig = config
	}
}

// WithCache enables caching with the specified size
func WithCache(enabled bool, size int) EngineOption {
	return func(opts *EngineOptions) {
		opts.EnableCache = enabled
		opts.CacheSize = size
	}
}

// WithConcurrency sets the maximum number of concurrent operations
func WithConcurrency(max int) EngineOption {
	return func(opts *EngineOptions) {
		opts.MaxConcurrency = max
	}
}

// WithEnhancedParser is deprecated - the enhanced parser is now the default
// This option is kept for backward compatibility but has no effect
func WithEnhancedParser(enabled bool) EngineOption {
	return func(opts *EngineOptions) {
		// No-op - enhanced parser is now always enabled
	}
}

// WithMetrics enables metrics collection
func WithMetrics(enabled bool) EngineOption {
	return func(opts *EngineOptions) {
		opts.EnableMetrics = enabled
	}
}

// WithCustomOperator registers a custom operator
func WithCustomOperator(name string, op Operator) EngineOption {
	return func(opts *EngineOptions) {
		if opts.CustomOperators == nil {
			opts.CustomOperators = make(map[string]Operator)
		}
		opts.CustomOperators[name] = op
	}
}

// WithVaultConfig configures vault settings
func WithVaultConfig(address, token string) EngineOption {
	return func(opts *EngineOptions) {
		opts.VaultAddress = address
		opts.VaultToken = token
	}
}

// WithDebugLogging enables debug logging
func WithDebugLogging(enabled bool) EngineOption {
	return func(opts *EngineOptions) {
		opts.DebugLogging = enabled
	}
}

// WithAWSRegion sets the AWS region
func WithAWSRegion(region string) EngineOption {
	return func(opts *EngineOptions) {
		opts.AWSRegion = region
	}
}

// LoggerV2 interface for structured logging
type LoggerV2 interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// VaultClientV2 interface for vault operations
type VaultClientV2 interface {
	Get(path string) (map[string]interface{}, error)
	List(path string) ([]string, error)
	Put(path string, data map[string]interface{}) error
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	Region    string
	Profile   string
	Role      string
	SkipAuth  bool
	Endpoint  string // For testing with localstack
}

// NewEngineV2 creates a new engine instance with the given options
func NewEngineV2(options ...EngineOption) (EngineV2, error) {
	opts := &EngineOptions{
		EnableCache:        true,
		CacheSize:          1000,
		MaxConcurrency:     10,
		UseEnhancedParser:  true,
		EnableMetrics:      false,
	}
	
	for _, option := range options {
		option(opts)
	}
	
	return newEngineV2(opts)
}

// newEngineV2 is the internal constructor (implemented in engine_v2_impl.go)
func newEngineV2(opts *EngineOptions) (EngineV2, error) {
	// Implementation moved to engine_v2_impl.go to avoid circular dependencies
	return createEngineV2FromOptions(opts)
}

// DefaultEngineV2 creates an engine with sensible defaults
func DefaultEngineV2() (EngineV2, error) {
	return NewEngineV2(
		WithCache(true, 1000),
		WithConcurrency(10),
		WithEnhancedParser(true),
	)
}

// TODO: Implement convenience functions after EngineV2 implementation is complete

// QuickMerge is a convenience function for simple merge operations
// func QuickMerge(yamlSources ...string) ([]byte, error) {
// 	engine, err := DefaultEngineV2()
// 	if err != nil {
// 		return nil, err
// 	}
// 	
// 	var docs []DocumentV2
// 	for _, source := range yamlSources {
// 		doc, err := engine.ParseYAML([]byte(source))
// 		if err != nil {
// 			return nil, NewParseError("failed to parse YAML", err)
// 		}
// 		docs = append(docs, doc)
// 	}
// 	
// 	result, err := engine.Merge(context.Background(), docs...).Execute()
// 	if err != nil {
// 		return nil, err
// 	}
// 	
// 	return engine.ToYAML(result)
// }

// QuickMergeFiles is a convenience function for merging files
// func QuickMergeFiles(paths ...string) ([]byte, error) {
// 	engine, err := DefaultEngineV2()
// 	if err != nil {
// 		return nil, err
// 	}
// 	
// 	result, err := engine.MergeFiles(context.Background(), paths...).Execute()
// 	if err != nil {
// 		return nil, err
// 	}
// 	
// 	return engine.ToYAML(result)
// }