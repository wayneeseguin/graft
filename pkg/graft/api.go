package graft

import (
	"context"
	"io"
	
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	vaultkv "github.com/cloudfoundry-community/vaultkv"
)

// Document represents a YAML/JSON document in a more user-friendly format
// This abstraction hides the internal map[interface{}]interface{} representation
type Document interface {
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
	GetMap(path string) (map[string]interface{}, error)
	
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
	Clone() Document
	
	// Prune removes a key from the document
	Prune(key string) Document
	
	// CherryPick creates a new document with only the specified keys
	CherryPick(keys ...string) Document
	
	// GetData returns the underlying data (for backward compatibility)
	GetData() interface{}
	
	// Additional type-safe getters
	GetInt64(path string) (int64, error)
	GetFloat64(path string) (float64, error)
	GetStringSlice(path string) ([]string, error)
	GetMapStringString(path string) (map[string]string, error)
}

// Engine is the enhanced interface for using graft as a library
type Engine interface {
	// Document operations
	ParseYAML(data []byte) (Document, error)
	ParseJSON(data []byte) (Document, error)
	ParseFile(path string) (Document, error)
	ParseReader(reader io.Reader) (Document, error)
	
	// Merge operations with builder pattern options
	Merge(ctx context.Context, docs ...Document) MergeBuilder
	MergeFiles(ctx context.Context, paths ...string) MergeBuilder
	MergeReaders(ctx context.Context, readers ...io.Reader) MergeBuilder
	
	// Evaluate processes operators in a document
	Evaluate(ctx context.Context, doc Document) (Document, error)
	
	// Output operations
	ToYAML(doc Document) ([]byte, error)
	ToJSON(doc Document) ([]byte, error)
	ToJSONIndent(doc Document, indent string) ([]byte, error)
	
	// Operator management
	RegisterOperator(name string, op Operator) error
	UnregisterOperator(name string) error
	ListOperators() []string
	GetOperator(name string) (Operator, bool)
	
	// Configuration
	WithLogger(logger Logger) Engine
	WithVaultClient(client VaultClient) Engine
	WithAWSConfig(config AWSConfig) Engine
	
	// State access for operators
	GetOperatorState() OperatorState
}

// OperatorState provides state access for operators during evaluation
type OperatorState interface {
	// Vault operations
	GetVaultClient() *vaultkv.KV
	GetVaultCache() map[string]map[string]interface{}
	SetVaultCache(path string, data map[string]interface{})
	AddVaultRef(path string, keys []string)
	IsVaultSkipped() bool
	
	// AWS operations
	GetAWSSession() *session.Session
	GetSecretsManagerClient() secretsmanageriface.SecretsManagerAPI
	GetParameterStoreClient() ssmiface.SSMAPI
	GetAWSSecretsCache() map[string]string
	SetAWSSecretCache(key, value string)
	GetAWSParamsCache() map[string]string
	SetAWSParamCache(key, value string)
	IsAWSSkipped() bool
	
	// Static IPs
	GetUsedIPs() map[string]string
	SetUsedIP(key, ip string)
	
	// Prune operations
	AddKeyToPrune(key string)
	GetKeysToPrune() []string
	
	// Sort operations
	AddPathToSort(path, order string)
	GetPathsToSort() map[string]string
}

// ArrayMergeStrategy defines how arrays are merged
type ArrayMergeStrategy int

const (
	// InlineArrays is the default - arrays are merged inline by index
	InlineArrays ArrayMergeStrategy = iota
	// AppendArrays appends arrays instead of merging inline
	AppendArrays
	// ReplaceArrays replaces the entire array
	ReplaceArrays
	// PrependArrays prepends new array elements
	PrependArrays
)

// MergeBuilder provides a fluent interface for merge operations
type MergeBuilder interface {
	// WithPrune specifies keys to remove from the final output
	WithPrune(keys ...string) MergeBuilder
	
	// WithCherryPick specifies keys to keep in the final output (all others removed)
	WithCherryPick(keys ...string) MergeBuilder
	
	// WithArrayMergeStrategy sets how arrays are merged
	WithArrayMergeStrategy(strategy ArrayMergeStrategy) MergeBuilder
	
	// SkipEvaluation skips operator evaluation after merging
	SkipEvaluation() MergeBuilder
	
	// EnableGoPatch enables go-patch format parsing
	EnableGoPatch() MergeBuilder
	
	// FallbackAppend uses append instead of inline for arrays by default
	FallbackAppend() MergeBuilder
	
	// Execute performs the merge operation
	Execute() (Document, error)
}

// EngineOptions configures a new engine instance using functional options
type EngineOptions struct {
	Logger             Logger
	VaultClient        VaultClient
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

// WithLogger sets the logger for the engine
func WithLogger(logger Logger) EngineOption {
	return func(opts *EngineOptions) {
		opts.Logger = logger
	}
}

// WithVaultClient sets the vault client for the engine
func WithVaultClient(client VaultClient) EngineOption {
	return func(opts *EngineOptions) {
		opts.VaultClient = client
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

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// VaultClient interface for vault operations
type VaultClient interface {
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

// NewEngine creates a new engine instance with the given options
func NewEngine(options ...EngineOption) (Engine, error) {
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
	
	return createEngineFromOptions(opts)
}

// CreateDefaultEngine creates an engine with sensible defaults
func CreateDefaultEngine() (Engine, error) {
	return NewEngine(
		WithCache(true, 1000),
		WithConcurrency(10),
		WithEnhancedParser(true),
	)
}

// TODO: Implement convenience functions after Engine implementation is complete

// QuickMerge is a convenience function for simple merge operations
// func QuickMerge(yamlSources ...string) ([]byte, error) {
// 	engine, err := DefaultEngine()
// 	if err != nil {
// 		return nil, err
// 	}
// 	
// 	var docs []Document
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
// 	engine, err := DefaultEngine()
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
// Global variables for backward compatibility with CLI
var (
	// VaultRefs tracks vault references found during evaluation
	VaultRefs = map[string][]string{}
	
	// SkipVault disables vault operations when true
	SkipVault bool
	
	// SkipAws disables AWS operations when true
	SkipAws bool
	
	// OpRegistry stores all registered operators (temporary until Phase 2)
	OpRegistry = make(map[string]Operator)
	
	// keysToPrune tracks keys to prune (temporary until Phase 2)
	keysToPrune []string
	
	// pathsToSort tracks paths to sort (temporary until Phase 2)
	pathsToSort = make(map[string]string)
)

// addToPruneListIfNecessary adds a path to the prune list if needed (temporary until Phase 2)
func addToPruneListIfNecessary(paths ...string) {
	for _, path := range paths {
		keysToPrune = append(keysToPrune, path)
	}
}