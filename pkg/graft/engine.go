package graft

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/starkandwayne/goutils/tree"
	vaultkv "github.com/cloudfoundry-community/vaultkv"
)

// engine implements the Engine interface
type engine struct {
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

// NewEngine creates a new engine with default configuration
func NewEngine() Engine {
	return NewEngineWithConfig(DefaultEngineConfig())
}

// NewEngineWithConfig creates a new engine with custom configuration
func NewEngineWithConfig(config EngineConfig) Engine {
	e := &engine{
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
func (e *engine) RegisterOperator(name string, op Operator) error {
	e.opMutex.Lock()
	defer e.opMutex.Unlock()
	
	if _, exists := e.operators[name]; exists {
		return fmt.Errorf("operator %s already registered", name)
	}
	
	e.operators[name] = op
	return nil
}

// GetOperator retrieves an operator by name
func (e *engine) GetOperator(name string) (Operator, bool) {
	e.opMutex.RLock()
	defer e.opMutex.RUnlock()
	
	op, exists := e.operators[name]
	return op, exists
}

// Merge merges multiple documents according to graft semantics
func (e *engine) Merge(ctx context.Context, docs []Document, opts MergeOptions) (Document, error) {
	if len(docs) == 0 {
		return nil, fmt.Errorf("no documents to merge")
	}
	
	// Convert documents to the internal format
	trees := make([]map[interface{}]interface{}, len(docs))
	for i, d := range docs {
		// Document is already map[interface{}]interface{}
		trees[i] = d
	}
	
	// Perform the merge
	result := trees[0]
	for i := 1; i < len(trees); i++ {
		var err error
		result, err = e.mergeTwo(result, trees[i], opts)
		if err != nil {
			return nil, err
		}
	}
	
	// Apply post-merge operations
	if len(opts.Prune) > 0 {
		result = e.pruneKeys(result, opts.Prune)
	}
	
	if len(opts.CherryPick) > 0 {
		result = e.cherryPick(result, opts.CherryPick)
	}
	
	// Evaluate if requested
	if !opts.SkipEval {
		evaluator := e.createEvaluator(result)
		if err := e.evaluate(ctx, evaluator); err != nil {
			return nil, err
		}
		result = evaluator.Tree
	}
	
	return result, nil
}

// MergeReaders merges documents from readers
func (e *engine) MergeReaders(ctx context.Context, readers []io.Reader, opts MergeOptions) (Document, error) {
	docs := make([]Document, 0, len(readers))
	
	for _, reader := range readers {
		doc, err := parseDocument(reader)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	
	return e.Merge(ctx, docs, opts)
}

// MergeFiles merges documents from files
func (e *engine) MergeFiles(ctx context.Context, paths []string, opts MergeOptions) (Document, error) {
	// Implementation will use MergeReaders after opening files
	// Placeholder for now
	return nil, fmt.Errorf("not implemented")
}

// Evaluate evaluates graft operators in a document
func (e *engine) Evaluate(ctx context.Context, doc Document) (Document, error) {
	// Document is already map[interface{}]interface{}
	evaluator := e.createEvaluator(doc)
	if err := e.evaluate(ctx, evaluator); err != nil {
		return nil, err
	}
	
	return evaluator.Tree, nil
}

// ToJSON converts a document to JSON
func (e *engine) ToJSON(doc Document, strict bool) ([]byte, error) {
	// Implementation will handle conversion
	// Placeholder for now
	return nil, fmt.Errorf("not implemented")
}

// ToYAML converts a document to YAML
func (e *engine) ToYAML(doc Document) ([]byte, error) {
	// Implementation will handle conversion
	// Placeholder for now
	return nil, fmt.Errorf("not implemented")
}

// Internal methods

func (e *engine) registerDefaultOperators() {
	// Basic operators are registered through the factory
	// This is kept minimal to avoid circular dependencies
	// Use NewDefaultEngine() from engine_factory.go for full operator set
}

func (e *engine) initializeVault() {
	// Initialize vault connection
	// Implementation will set up e.vaultKV
}

func (e *engine) initializeAWS() {
	// Initialize AWS clients
	// Implementation will set up AWS session and clients
}

func (e *engine) createEvaluator(t map[interface{}]interface{}) *Evaluator {
	here, _ := tree.ParseCursor("$")
	return &Evaluator{
		Tree: t,
		Deps: map[string][]tree.Cursor{},
		Here: here,
		engine: e,
	}
}

func (e *engine) evaluate(ctx context.Context, ev *Evaluator) error {
	// Set the engine context on the evaluator
	ev.engine = e
	
	// Run evaluation phases
	for _, phase := range []OperatorPhase{MergePhase, EvalPhase} {
		if err := ev.RunPhase(phase); err != nil {
			return err
		}
	}
	
	return nil
}

func (e *engine) mergeTwo(a, b map[interface{}]interface{}, opts MergeOptions) (map[interface{}]interface{}, error) {
	// This will use the merger package
	// Placeholder for now
	result := make(map[interface{}]interface{})
	
	// Copy from a
	for k, v := range a {
		result[k] = v
	}
	
	// Merge from b
	for k, v := range b {
		result[k] = v
	}
	
	return result, nil
}

func (e *engine) pruneKeys(doc map[interface{}]interface{}, keys []string) map[interface{}]interface{} {
	// Implementation for pruning
	return doc
}

func (e *engine) cherryPick(doc map[interface{}]interface{}, keys []string) map[interface{}]interface{} {
	// Implementation for cherry-picking
	return doc
}

func convertToTree(input interface{}) map[interface{}]interface{} {
	// Convert various document formats to the internal tree format
	switch v := input.(type) {
	case map[string]interface{}:
		result := make(map[interface{}]interface{})
		for k, val := range v {
			result[k] = val
		}
		return result
	default:
		// Return as-is wrapped in a map
		return map[interface{}]interface{}{"value": input}
	}
}

func parseDocument(reader io.Reader) (Document, error) {
	// Parse YAML/JSON from reader
	// Placeholder for now
	return nil, fmt.Errorf("not implemented")
}

// GetVaultClient returns the vault client (for operators that need it)
func (e *engine) GetVaultClient() *vaultkv.KV {
	e.vaultMutex.RLock()
	defer e.vaultMutex.RUnlock()
	return e.vaultKV
}

// GetVaultCache returns the vault cache
func (e *engine) GetVaultCache() map[string]map[string]interface{} {
	// Return a copy to avoid concurrent modification
	e.vaultMutex.RLock()
	defer e.vaultMutex.RUnlock()
	
	cache := make(map[string]map[string]interface{})
	for k, v := range e.vaultSecretCache {
		cache[k] = v
	}
	return cache
}

// SetVaultCache updates the vault cache
func (e *engine) SetVaultCache(path string, data map[string]interface{}) {
	e.vaultMutex.Lock()
	defer e.vaultMutex.Unlock()
	e.vaultSecretCache[path] = data
}

// AddVaultRef adds a vault reference
func (e *engine) AddVaultRef(path string, keys []string) {
	e.vaultMutex.Lock()
	defer e.vaultMutex.Unlock()
	e.vaultRefs[path] = keys
}

// GetUsedIPs returns the used IPs map
func (e *engine) GetUsedIPs() map[string]string {
	e.ipMutex.RLock()
	defer e.ipMutex.RUnlock()
	
	ips := make(map[string]string)
	for k, v := range e.usedIPs {
		ips[k] = v
	}
	return ips
}

// SetUsedIP sets a used IP
func (e *engine) SetUsedIP(key, ip string) {
	e.ipMutex.Lock()
	defer e.ipMutex.Unlock()
	e.usedIPs[key] = ip
}

// AddKeyToPrune adds a key to be pruned
func (e *engine) AddKeyToPrune(key string) {
	e.pruneMutex.Lock()
	defer e.pruneMutex.Unlock()
	e.keysToPrune = append(e.keysToPrune, key)
}

// GetKeysToPrune returns keys to prune
func (e *engine) GetKeysToPrune() []string {
	e.pruneMutex.RLock()
	defer e.pruneMutex.RUnlock()
	
	keys := make([]string, len(e.keysToPrune))
	copy(keys, e.keysToPrune)
	return keys
}

// IsVaultSkipped returns whether vault operations are skipped
func (e *engine) IsVaultSkipped() bool {
	return e.skipVault
}

// IsAWSSkipped returns whether AWS operations are skipped
func (e *engine) IsAWSSkipped() bool {
	return e.skipAws
}

// GetAWSSession returns the AWS session
func (e *engine) GetAWSSession() *session.Session {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	return e.awsSession
}

// GetSecretsManagerClient returns the AWS Secrets Manager client
func (e *engine) GetSecretsManagerClient() secretsmanageriface.SecretsManagerAPI {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	return e.secretsManagerClient
}

// GetParameterStoreClient returns the AWS Parameter Store client
func (e *engine) GetParameterStoreClient() ssmiface.SSMAPI {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	return e.parameterstoreClient
}

// GetAWSSecretsCache returns the AWS secrets cache
func (e *engine) GetAWSSecretsCache() map[string]string {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	
	cache := make(map[string]string)
	for k, v := range e.awsSecretsCache {
		cache[k] = v
	}
	return cache
}

// SetAWSSecretCache sets a value in the AWS secrets cache
func (e *engine) SetAWSSecretCache(key, value string) {
	e.awsMutex.Lock()
	defer e.awsMutex.Unlock()
	e.awsSecretsCache[key] = value
}

// GetAWSParamsCache returns the AWS parameters cache
func (e *engine) GetAWSParamsCache() map[string]string {
	e.awsMutex.RLock()
	defer e.awsMutex.RUnlock()
	
	cache := make(map[string]string)
	for k, v := range e.awsParamsCache {
		cache[k] = v
	}
	return cache
}

// SetAWSParamCache sets a value in the AWS parameters cache
func (e *engine) SetAWSParamCache(key, value string) {
	e.awsMutex.Lock()
	defer e.awsMutex.Unlock()
	e.awsParamsCache[key] = value
}

// AddPathToSort adds a path to be sorted
func (e *engine) AddPathToSort(path, order string) {
	e.sortMutex.Lock()
	defer e.sortMutex.Unlock()
	e.pathsToSort[path] = order
}

// GetPathsToSort returns paths to sort
func (e *engine) GetPathsToSort() map[string]string {
	e.sortMutex.RLock()
	defer e.sortMutex.RUnlock()
	
	paths := make(map[string]string)
	for k, v := range e.pathsToSort {
		paths[k] = v
	}
	return paths
}