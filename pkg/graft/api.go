package graft

import (
	"context"
	"fmt"
	"io"
	"github.com/starkandwayne/goutils/tree"
)

// Document represents a YAML/JSON document
type Document map[interface{}]interface{}

// MergeOptions configures how documents are merged
type MergeOptions struct {
	SkipEval       bool     // Skip operator evaluation after merging
	Prune          []string // Keys to prune from final output
	CherryPick     []string // Keys to cherry-pick from final output
	FallbackAppend bool     // Default to append instead of inline for arrays
	EnableGoPatch  bool     // Enable go-patch format parsing
}

// Engine is the main interface for using graft as a library
type Engine interface {
	// Merge combines multiple documents into one
	Merge(ctx context.Context, docs []Document, opts MergeOptions) (Document, error)
	
	// MergeReaders merges documents from io.Readers
	MergeReaders(ctx context.Context, readers []io.Reader, opts MergeOptions) (Document, error)
	
	// MergeFiles merges documents from file paths
	MergeFiles(ctx context.Context, paths []string, opts MergeOptions) (Document, error)
	
	// Evaluate processes operators in a document
	Evaluate(ctx context.Context, doc Document) (Document, error)
	
	// ToJSON converts a document to JSON
	ToJSON(doc Document, strict bool) ([]byte, error)
	
	// ToYAML converts a document to YAML
	ToYAML(doc Document) ([]byte, error)
}

// Config holds configuration for the Engine
type Config struct {
	// Logger for debug/trace output
	Logger Logger
	
	// VaultClient for vault operations
	VaultClient VaultClient
	
	// Cache configuration
	EnableCache bool
	CacheSize   int
	
	// Performance tuning
	MaxConcurrency int
	UseEnhancedParser bool
	
	// Feature flags
	EnableMetrics bool
}

// Logger interface for logging
type Logger interface {
	Debug(format string, args ...interface{})
	Trace(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// VaultClient interface for vault operations
type VaultClient interface {
	Get(path string) (map[string]interface{}, error)
	List(path string) ([]string, error)
}

// NewEngine creates a new graft engine with the given configuration
func NewEngine(cfg *Config) (Engine, error) {
	// TODO: Implement engine creation
	return nil, nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		EnableCache:       true,
		CacheSize:         1000,
		MaxConcurrency:    10,
		UseEnhancedParser: true,
		EnableMetrics:     false,
	}
}

// Global variables for vault operations (will be encapsulated in Phase 2)
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

// SetupOperators initializes all operators for a given phase
func SetupOperators(phase OperatorPhase) error {
	errors := MultiError{Errors: []error{}}
	for _, op := range OpRegistry {
		if op.Phase() == phase {
			if err := op.Setup(); err != nil {
				errors.Append(err)
			}
		}
	}
	if len(errors.Errors) > 0 {
		return errors
	}
	return nil
}

// OperatorFor returns the operator for the given name
func OperatorFor(name string) Operator {
	if op, ok := OpRegistry[name]; ok {
		return op
	}
	return &NullOperator{}
}

// NullOperator is returned when an operator is not found
type NullOperator struct{}

func (n *NullOperator) Setup() error { return nil }
func (n *NullOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	return nil, fmt.Errorf("operator not found")
}
func (n *NullOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return nil
}
func (n *NullOperator) Phase() OperatorPhase { return EvalPhase }

// addToPruneListIfNecessary adds a path to the prune list if needed (temporary until Phase 2)
func addToPruneListIfNecessary(paths ...string) {
	for _, path := range paths {
		keysToPrune = append(keysToPrune, path)
	}
}