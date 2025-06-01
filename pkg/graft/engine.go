package graft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/starkandwayne/goutils/tree"
	vaultkv "github.com/cloudfoundry-community/vaultkv"
	"gopkg.in/yaml.v2"
)

// DefaultEngine is the default implementation of the Engine interface
// It provides all the core functionality needed by graft
type DefaultEngine struct {
	// Configuration
	config EngineConfig
	
	// Operator registry
	operators map[string]Operator
	opMutex   sync.RWMutex
	
	// Vault state
	vaultKV          *vaultkv.KV
	vaultSecretCache map[string]map[string]interface{}
	vaultRefs        map[string][]string
	vaultMutex       sync.RWMutex
	skipVault        bool
	
	// AWS state
	awsSession           *session.Session
	secretsManagerClient secretsmanageriface.SecretsManagerAPI
	parameterstoreClient ssmiface.SSMAPI
	awsSecretsCache      map[string]string
	awsParamsCache       map[string]string
	awsMutex            sync.RWMutex
	skipAws             bool
	
	// Static IPs state
	usedIPs   map[string]string
	ipMutex   sync.RWMutex
	
	// Prune state
	keysToPrune []string
	pruneMutex  sync.RWMutex
	
	// Sort state
	pathsToSort map[string]string
	sortMutex   sync.RWMutex
	
	// Parser configuration
	useEnhancedParser bool
	
	// Metrics and monitoring
	metrics *EngineMetrics
}

// EngineConfig holds configuration for the engine
type EngineConfig struct {
	// Vault configuration
	VaultAddr      string
	VaultToken     string
	VaultSkipTLS   bool
	SkipVault      bool
	
	// AWS configuration
	AWSRegion    string
	AWSProfile   string
	SkipAWS      bool
	
	// Parser configuration
	UseEnhancedParser bool
	
	// Performance configuration
	EnableCaching    bool
	CacheSize        int
	EnableParallel   bool
	MaxWorkers       int
	
	// Dataflow configuration
	DataflowOrder    string // "alphabetical" (default) or "insertion"
}

// EngineMetrics tracks engine performance metrics
type EngineMetrics struct {
	OperatorCalls   map[string]int64
	CacheHits       int64
	CacheMisses     int64
	VaultCalls      int64
	AWSCalls        int64
	mu              sync.RWMutex
}

// NewDefaultEngine creates a new default engine with default configuration
func NewDefaultEngine() *DefaultEngine {
	return NewDefaultEngineWithConfig(DefaultEngineConfig())
}

// NewDefaultEngineWithConfig creates a new default engine with custom configuration
func NewDefaultEngineWithConfig(config EngineConfig) *DefaultEngine {
	e := &DefaultEngine{
		config:           config,
		operators:        make(map[string]Operator),
		vaultSecretCache: make(map[string]map[string]interface{}),
		vaultRefs:        make(map[string][]string),
		awsSecretsCache:  make(map[string]string),
		awsParamsCache:   make(map[string]string),
		usedIPs:         make(map[string]string),
		pathsToSort:     make(map[string]string),
		skipVault:       config.SkipVault,
		skipAws:         config.SkipAWS,
		useEnhancedParser: config.UseEnhancedParser,
		metrics: &EngineMetrics{
			OperatorCalls: make(map[string]int64),
		},
	}
	
	// Register default operators
	e.registerDefaultOperators()
	
	// Initialize vault if configured
	if !config.SkipVault && config.VaultAddr != "" {
		e.initializeVault()
	}
	
	// Initialize AWS if configured
	if !config.SkipAWS && config.AWSRegion != "" {
		e.initializeAWS()
	}
	
	return e
}

// DefaultEngineConfig returns default engine configuration
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		UseEnhancedParser: true,
		EnableCaching:     true,
		CacheSize:         10000,
		EnableParallel:    false,
		MaxWorkers:        4,
	}
}

// RegisterOperator registers a custom operator
func (e *DefaultEngine) RegisterOperator(name string, op Operator) error {
	e.opMutex.Lock()
	defer e.opMutex.Unlock()
	
	if _, exists := e.operators[name]; exists {
		return fmt.Errorf("operator %s already registered", name)
	}
	
	e.operators[name] = op
	return nil
}

// GetOperator retrieves an operator by name
func (e *DefaultEngine) GetOperator(name string) (Operator, bool) {
	e.opMutex.RLock()
	defer e.opMutex.RUnlock()
	
	// First check engine's registry
	if op, exists := e.operators[name]; exists {
		return op, true
	}
	
	// Fall back to global registry for backward compatibility
	op, exists := OpRegistry[name]
	return op, exists
}

// EngineContext interface methods for internal operator access

func (e *DefaultEngine) GetVaultClient() *vaultkv.KV {
	e.vaultMutex.RLock()
	defer e.vaultMutex.RUnlock()
	return e.vaultKV
}

func (e *DefaultEngine) GetVaultCache() map[string]map[string]interface{} {
	// Return a copy to avoid concurrent modification
	e.vaultMutex.RLock()
	defer e.vaultMutex.RUnlock()
	
	cache := make(map[string]map[string]interface{})
	for k, v := range e.vaultSecretCache {
		cache[k] = v
	}
	return cache
}

func (e *DefaultEngine) SetVaultCache(path string, data map[string]interface{}) {
	e.vaultMutex.Lock()
	defer e.vaultMutex.Unlock()
	e.vaultSecretCache[path] = data
}

func (e *DefaultEngine) AddVaultRef(path string, keys []string) {
	e.vaultMutex.Lock()
	defer e.vaultMutex.Unlock()
	
	// Update internal vault refs
	if e.vaultRefs[path] == nil {
		e.vaultRefs[path] = []string{}
	}
	e.vaultRefs[path] = append(e.vaultRefs[path], keys...)
	
	// Also update global VaultRefs for backward compatibility with vaultinfo command
	if SkipVault || e.skipVault {
		if VaultRefs[path] == nil {
			VaultRefs[path] = []string{}
		}
		VaultRefs[path] = append(VaultRefs[path], keys...)
	}
}

func (e *DefaultEngine) IsVaultSkipped() bool {
	// Check both the engine's skipVault and the global SkipVault for backward compatibility
	return e.skipVault || SkipVault
}

func (e *DefaultEngine) GetAWSSession() *session.Session {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	return e.awsSession
}

func (e *DefaultEngine) GetSecretsManagerClient() secretsmanageriface.SecretsManagerAPI {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	return e.secretsManagerClient
}

func (e *DefaultEngine) GetParameterStoreClient() ssmiface.SSMAPI {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	return e.parameterstoreClient
}

func (e *DefaultEngine) GetAWSSecretsCache() map[string]string {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	
	cache := make(map[string]string)
	for k, v := range e.awsSecretsCache {
		cache[k] = v
	}
	return cache
}

func (e *DefaultEngine) SetAWSSecretCache(key, value string) {
	e.awsMutex.Lock()
	defer e.awsMutex.Unlock()
	e.awsSecretsCache[key] = value
}

func (e *DefaultEngine) GetAWSParamsCache() map[string]string {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	
	cache := make(map[string]string)
	for k, v := range e.awsParamsCache {
		cache[k] = v
	}
	return cache
}

func (e *DefaultEngine) SetAWSParamCache(key, value string) {
	e.awsMutex.Lock()
	defer e.awsMutex.Unlock()
	e.awsParamsCache[key] = value
}

func (e *DefaultEngine) IsAWSSkipped() bool {
	return e.skipAws
}

func (e *DefaultEngine) GetUsedIPs() map[string]string {
	e.ipMutex.RLock()
	defer e.ipMutex.RUnlock()
	
	ips := make(map[string]string)
	for k, v := range e.usedIPs {
		ips[k] = v
	}
	return ips
}

func (e *DefaultEngine) SetUsedIP(key, ip string) {
	e.ipMutex.Lock()
	defer e.ipMutex.Unlock()
	e.usedIPs[key] = ip
}

func (e *DefaultEngine) AddKeyToPrune(key string) {
	e.pruneMutex.Lock()
	defer e.pruneMutex.Unlock()
	e.keysToPrune = append(e.keysToPrune, key)
}

func (e *DefaultEngine) GetKeysToPrune() []string {
	e.pruneMutex.RLock()
	defer e.pruneMutex.RUnlock()
	
	keys := make([]string, len(e.keysToPrune))
	copy(keys, e.keysToPrune)
	return keys
}

func (e *DefaultEngine) AddPathToSort(path, order string) {
	e.sortMutex.Lock()
	defer e.sortMutex.Unlock()
	e.pathsToSort[path] = order
}

func (e *DefaultEngine) GetPathsToSort() map[string]string {
	e.sortMutex.RLock()
	defer e.sortMutex.RUnlock()
	
	paths := make(map[string]string)
	for k, v := range e.pathsToSort {
		paths[k] = v
	}
	return paths
}

// Internal methods

func (e *DefaultEngine) registerDefaultOperators() {
	// Basic operators are registered through the factory
	// This is kept minimal to avoid circular dependencies
	// Use factory.NewDefaultEngine() for full operator set
}

func (e *DefaultEngine) initializeVault() {
	// Initialize vault connection
	// Implementation will set up e.vaultKV
}

func (e *DefaultEngine) initializeAWS() {
	// Initialize AWS clients
	// Implementation will set up AWS session and clients
}

func (e *DefaultEngine) createEvaluator(t map[interface{}]interface{}) *Evaluator {
	here, _ := tree.ParseCursor("$")
	return &Evaluator{
		Tree: t,
		Deps: map[string][]tree.Cursor{},
		Here: here,
		engine: e,
		DataflowOrder: e.config.DataflowOrder,
	}
}

func (e *DefaultEngine) evaluate(ctx context.Context, ev *Evaluator) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Set the engine on the evaluator
	ev.engine = Engine(e)
	
	// Run evaluation phases
	for _, phase := range []OperatorPhase{MergePhase, ParamPhase, EvalPhase} {
		// Check context cancellation before each phase
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		if err := ev.RunPhase(phase); err != nil {
			return err
		}
	}
	
	// Post-processing: apply operator-level pruning
	prunePaths := e.GetKeysToPrune()
	if len(prunePaths) > 0 {
		// Convert tree paths to Document paths and remove them
		doc := NewDocument(ev.Tree)
		for _, path := range prunePaths {
			// Remove the "$." prefix if present
			cleanPath := strings.TrimPrefix(path, "$.")
			doc = doc.Prune(cleanPath)
		}
		// Update the evaluator tree with the pruned document
		ev.Tree = doc.RawData().(map[interface{}]interface{})
	}
	
	return nil
}

// Implement Engine interface methods

// ParseYAML parses YAML data into a Document
func (e *DefaultEngine) ParseYAML(data []byte) (Document, error) {
	// Handle empty data
	if len(data) == 0 {
		return nil, nil
	}
	
	// First parse as generic interface to check document type
	var genericResult interface{}
	err := yaml.Unmarshal(data, &genericResult)
	if err != nil {
		return nil, NewParseError("failed to parse YAML", err)
	}
	
	if genericResult == nil {
		return nil, nil
	}
	
	// Check that root is a map/hash
	result, ok := genericResult.(map[interface{}]interface{})
	if !ok {
		// Return plain error for compatibility with tests
		return nil, fmt.Errorf("Root of YAML document is not a hash/map:")
	}
	
	return NewDocument(result), nil
}

// ParseJSON parses JSON data into a Document
func (e *DefaultEngine) ParseJSON(data []byte) (Document, error) {
	// Handle empty data
	if len(data) == 0 {
		return nil, nil
	}
	
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, NewParseError("failed to parse JSON", err)
	}
	
	if result == nil {
		return nil, nil
	}
	
	// Convert to map[interface{}]interface{}
	converted := make(map[interface{}]interface{})
	for k, v := range result {
		converted[k] = v
	}
	
	return NewDocument(converted), nil
}

// ParseFile parses a file into a Document
func (e *DefaultEngine) ParseFile(path string) (Document, error) {
	// Implementation will be added
	return nil, fmt.Errorf("not implemented")
}

// ParseReader parses data from a reader into a Document
func (e *DefaultEngine) ParseReader(reader io.Reader) (Document, error) {
	// Implementation will be added
	return nil, fmt.Errorf("not implemented")
}

// Merge creates a new merge builder for combining documents
func (e *DefaultEngine) Merge(ctx context.Context, docs ...Document) MergeBuilder {
	if ctx == nil {
		ctx = context.Background()
	}
	
	return &mergeBuilderImpl{
		engine: e,
		ctx:    ctx,
		docs:   docs,
	}
}

// MergeFiles creates a merge builder for files
func (e *DefaultEngine) MergeFiles(ctx context.Context, paths ...string) MergeBuilder {
	// Implementation will be added
	return nil
}

// MergeReaders creates a merge builder for readers
func (e *DefaultEngine) MergeReaders(ctx context.Context, readers ...io.Reader) MergeBuilder {
	// Implementation will be added
	return nil
}

// Evaluate processes operators in a document
func (e *DefaultEngine) Evaluate(ctx context.Context, doc Document) (Document, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	
	// Get the raw data
	data, ok := doc.RawData().(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("document data is not a map")
	}
	
	// Create evaluator
	ev := e.createEvaluator(data)
	
	// Run evaluation
	err := e.evaluate(ctx, ev)
	if err != nil {
		return nil, err
	}
	
	// Return evaluated document
	return NewDocument(ev.Tree), nil
}

// ToYAML converts a document to YAML bytes
func (e *DefaultEngine) ToYAML(doc Document) ([]byte, error) {
	// Implementation will be added
	return nil, fmt.Errorf("not implemented")
}

// ToJSON converts a document to JSON bytes
func (e *DefaultEngine) ToJSON(doc Document) ([]byte, error) {
	// Implementation will be added
	return nil, fmt.Errorf("not implemented")
}

// ToJSONIndent converts a document to indented JSON bytes
func (e *DefaultEngine) ToJSONIndent(doc Document, indent string) ([]byte, error) {
	// Implementation will be added
	return nil, fmt.Errorf("not implemented")
}

// UnregisterOperator removes a custom operator
func (e *DefaultEngine) UnregisterOperator(name string) error {
	e.opMutex.Lock()
	defer e.opMutex.Unlock()
	
	delete(e.operators, name)
	return nil
}

// ListOperators returns all available operators
func (e *DefaultEngine) ListOperators() []string {
	e.opMutex.RLock()
	defer e.opMutex.RUnlock()
	
	names := make([]string, 0, len(e.operators)+len(OpRegistry))
	
	// Add engine-specific operators
	for name := range e.operators {
		names = append(names, name)
	}
	
	// Add global operators
	for name := range OpRegistry {
		// Check if not already in the list
		found := false
		for _, existing := range names {
			if existing == name {
				found = true
				break
			}
		}
		if !found {
			names = append(names, name)
		}
	}
	
	return names
}

// WithLogger sets a new logger (returns new engine instance)
func (e *DefaultEngine) WithLogger(logger Logger) Engine {
	// For now, return self as logging is not implemented yet
	return e
}

// WithVaultClient sets a new vault client (returns new engine instance)
func (e *DefaultEngine) WithVaultClient(client VaultClient) Engine {
	// For now, return self as custom vault client is not implemented yet
	return e
}

// WithAWSConfig sets AWS configuration (returns new engine instance)
func (e *DefaultEngine) WithAWSConfig(config AWSConfig) Engine {
	// For now, return self as AWS config is not fully implemented yet
	return e
}

// UpdateConfig updates the engine configuration
func (e *DefaultEngine) UpdateConfig(config EngineConfig) {
	e.config = config
	e.skipVault = config.SkipVault
	e.skipAws = config.SkipAWS
	e.useEnhancedParser = config.UseEnhancedParser
}

// GetOperatorState returns the operator state interface
func (e *DefaultEngine) GetOperatorState() OperatorState {
	// The engine itself implements OperatorState
	return e
}

// createEngineFromOptions creates an engine from EngineOptions (used by api.go)
func createEngineFromOptions(opts *EngineOptions) (Engine, error) {
	// Validate options
	if opts.MaxConcurrency < 0 {
		return nil, NewConfigurationError("concurrency must be non-negative")
	}
	
	// Create engine config from options
	config := EngineConfig{
		VaultAddr:         opts.VaultAddress,
		VaultToken:        opts.VaultToken,
		AWSRegion:         opts.AWSRegion,
		UseEnhancedParser: opts.UseEnhancedParser,
		EnableCaching:     opts.EnableCache,
		CacheSize:         opts.CacheSize,
		EnableParallel:    opts.MaxConcurrency > 1,
		MaxWorkers:        opts.MaxConcurrency,
		DataflowOrder:     opts.DataflowOrder,
	}
	
	// Create the engine
	engine := NewDefaultEngineWithConfig(config)
	
	// Register custom operators if any
	if opts.CustomOperators != nil {
		for name, op := range opts.CustomOperators {
			if err := engine.RegisterOperator(name, op); err != nil {
				return nil, err
			}
		}
	}
	
	return engine, nil
}