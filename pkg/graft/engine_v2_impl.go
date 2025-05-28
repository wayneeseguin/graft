package graft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/tree"
)

// engineV2Impl implements the EngineV2 interface
type engineV2Impl struct {
	engine EngineContext
	options EngineOptions
}

// createEngineV2FromOptions creates a new EngineV2 instance from options
func createEngineV2FromOptions(opts *EngineOptions) (EngineV2, error) {
	// Validate options
	if opts.MaxConcurrency < 1 {
		return nil, NewConfigurationError("MaxConcurrency must be at least 1")
	}

	// Ensure operators are registered
	if err := ensureOperatorsRegistered(); err != nil {
		return nil, NewConfigurationError("failed to register operators: " + err.Error())
	}

	// Create underlying engine 
	engine := NewEngine()
	
	// The engine should implement EngineContext
	engineContext, ok := engine.(EngineContext)
	if !ok {
		return nil, NewConfigurationError("engine does not implement EngineContext interface")
	}

	return &engineV2Impl{
		engine:  engineContext,
		options: *opts,
	}, nil
}

// ParseYAML parses YAML data into a DocumentV2
func (e *engineV2Impl) ParseYAML(data []byte) (DocumentV2, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// Use simpleyaml for compatibility with existing codebase
	yamlDoc, err := simpleyaml.NewYaml(data)
	if err != nil {
		return nil, NewParseError("failed to parse YAML", err)
	}

	// Try to get as map first (most common case)
	docData, err := yamlDoc.Map()
	if err != nil {
		// If not a map, try as array
		arrayData, arrErr := yamlDoc.Array()
		if arrErr != nil {
			// If not array either, try as string/scalar
			strData, strErr := yamlDoc.String()
			if strErr != nil {
				return nil, NewParseError("YAML content is not a map, array, or string", err)
			}
			// Wrap scalar in a document
			wrapper := map[interface{}]interface{}{
				"value": strData,
			}
			return NewDocumentV2(wrapper), nil
		}
		
		// Wrap array in a document
		wrapper := map[interface{}]interface{}{
			"value": arrayData,
		}
		return NewDocumentV2(wrapper), nil
	}

	return NewDocumentV2(docData), nil
}

// ParseJSON parses JSON data into a DocumentV2
func (e *engineV2Impl) ParseJSON(data []byte) (DocumentV2, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var jsonData interface{}
	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		return nil, NewParseError("failed to parse JSON", err)
	}

	// Convert to map[interface{}]interface{} format for compatibility
	converted, err := convertToInterfaceMap(jsonData)
	if err != nil {
		return nil, NewParseError("failed to convert JSON data", err)
	}

	if mapData, ok := converted.(map[interface{}]interface{}); ok {
		return NewDocumentV2(mapData), nil
	}

	// Wrap non-map JSON in a document
	wrapper := map[interface{}]interface{}{
		"value": converted,
	}
	return NewDocumentV2(wrapper), nil
}

// ParseFile parses a file into a DocumentV2
func (e *engineV2Impl) ParseFile(path string) (DocumentV2, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, NewExternalError("file", "failed to read file: "+path, err)
	}

	// Determine format by extension
	if isJSONFile(path) {
		return e.ParseJSON(data)
	}
	return e.ParseYAML(data)
}

// ParseReader parses data from a reader into a DocumentV2
func (e *engineV2Impl) ParseReader(reader io.Reader) (DocumentV2, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, NewExternalError("reader", "failed to read data", err)
	}

	// Default to YAML parsing
	return e.ParseYAML(data)
}

// Merge creates a new merge builder for combining documents
func (e *engineV2Impl) Merge(ctx context.Context, docs ...DocumentV2) MergeBuilder {
	return &mergeBuilderImpl{
		engine: e.engine,
		ctx:    ctx,
		docs:   docs,
		pruneKeys: []string{},
		cherryPickKeys: []string{},
		skipEvaluation: false,
		goPatch: false,
		fallbackAppend: false,
	}
}

// MergeFiles creates a merge builder for files
func (e *engineV2Impl) MergeFiles(ctx context.Context, paths ...string) MergeBuilder {
	docs := make([]DocumentV2, 0, len(paths))
	
	for _, path := range paths {
		doc, err := e.ParseFile(path)
		if err != nil {
			// Return a builder that will error on Execute()
			return &mergeBuilderImpl{
				engine: e.engine,
				ctx:    ctx,
				error:  err,
			}
		}
		if doc != nil {
			docs = append(docs, doc)
		}
	}

	return e.Merge(ctx, docs...)
}

// MergeReaders creates a merge builder for readers
func (e *engineV2Impl) MergeReaders(ctx context.Context, readers ...io.Reader) MergeBuilder {
	docs := make([]DocumentV2, 0, len(readers))
	
	for _, reader := range readers {
		doc, err := e.ParseReader(reader)
		if err != nil {
			// Return a builder that will error on Execute()
			return &mergeBuilderImpl{
				engine: e.engine,
				ctx:    ctx,
				error:  err,
			}
		}
		if doc != nil {
			docs = append(docs, doc)
		}
	}

	return e.Merge(ctx, docs...)
}

// Evaluate processes operators in a document
func (e *engineV2Impl) Evaluate(ctx context.Context, doc DocumentV2) (DocumentV2, error) {
	if doc == nil {
		return nil, nil
	}

	// Get the raw data from the document
	rawData := doc.RawData()
	
	// Create evaluator with engine context
	evaluator := &Evaluator{
		Tree:   rawData.(map[interface{}]interface{}),
		engine: e.engine, // Connect the engine context
	}

	// Run evaluation
	err := evaluator.Run(nil, nil)
	if err != nil {
		return nil, NewEvaluationError("", "failed to evaluate document", err)
	}

	// Return new document with evaluated data
	return NewDocumentV2(evaluator.Tree), nil
}

// ToYAML converts a document to YAML bytes
func (e *engineV2Impl) ToYAML(doc DocumentV2) ([]byte, error) {
	if doc == nil {
		return []byte{}, nil
	}
	return doc.ToYAML()
}

// ToJSON converts a document to JSON bytes
func (e *engineV2Impl) ToJSON(doc DocumentV2) ([]byte, error) {
	if doc == nil {
		return []byte{}, nil
	}
	return doc.ToJSON()
}

// ToJSONIndent converts a document to indented JSON bytes
func (e *engineV2Impl) ToJSONIndent(doc DocumentV2, indent string) ([]byte, error) {
	if doc == nil {
		return []byte{}, nil
	}
	
	rawData := doc.RawData()
	return json.MarshalIndent(rawData, "", indent)
}

// RegisterOperator registers a custom operator
func (e *engineV2Impl) RegisterOperator(name string, op Operator) error {
	// For now, register in the global operator registry for compatibility
	// TODO: Move to engine-specific registry when fully implemented
	if OpRegistry == nil {
		OpRegistry = make(map[string]Operator)
	}
	OpRegistry[name] = op
	return nil
}

// UnregisterOperator removes a custom operator
func (e *engineV2Impl) UnregisterOperator(name string) error {
	if OpRegistry != nil {
		delete(OpRegistry, name)
	}
	return nil
}

// ListOperators returns all available operators
func (e *engineV2Impl) ListOperators() []string {
	var names []string
	if OpRegistry != nil {
		for name := range OpRegistry {
			names = append(names, name)
		}
	}
	// Add core operators that might not be in registry yet
	coreOps := []string{"grab", "concat", "calc", "vault", "static_ips", "merge", "prune", "defer", "join", "sort"}
	for _, op := range coreOps {
		found := false
		for _, existing := range names {
			if existing == op {
				found = true
				break
			}
		}
		if !found {
			names = append(names, op)
		}
	}
	return names
}

// WithLoggerV2 sets a new logger (returns new engine instance)
func (e *engineV2Impl) WithLoggerV2(logger LoggerV2) EngineV2 {
	newOptions := e.options
	newOptions.LoggerV2 = logger
	
	return &engineV2Impl{
		engine:  e.engine, // Share the engine context
		options: newOptions,
	}
}

// WithVaultClientV2 sets a new vault client (returns new engine instance)
func (e *engineV2Impl) WithVaultClientV2(client VaultClientV2) EngineV2 {
	newOptions := e.options
	newOptions.VaultClientV2 = client
	
	return &engineV2Impl{
		engine:  e.engine, // Share the engine context
		options: newOptions,
	}
}

// WithAWSConfig sets AWS configuration (returns new engine instance)
func (e *engineV2Impl) WithAWSConfig(config AWSConfig) EngineV2 {
	newOptions := e.options
	newOptions.AWSConfig = &config
	
	return &engineV2Impl{
		engine:  e.engine, // Share the engine context
		options: newOptions,
	}
}

// Helper functions

// ensureOperatorsRegistered ensures all necessary operators are registered
func ensureOperatorsRegistered() error {
	// Initialize operator registry if not already done
	if OpRegistry == nil {
		OpRegistry = make(map[string]Operator)
	}

	// Register core operators if not already registered
	if _, exists := OpRegistry["grab"]; !exists {
		RegisterOp("grab", &basicGrabOperator{})
	}
	
	if _, exists := OpRegistry["concat"]; !exists {
		RegisterOp("concat", &basicConcatOperator{})
	}

	return nil
}

// basicGrabOperator is a simple grab implementation for testing
type basicGrabOperator struct{}

func (basicGrabOperator) Setup() error { return nil }
func (basicGrabOperator) Phase() OperatorPhase { return EvalPhase }
func (basicGrabOperator) Dependencies(ev *Evaluator, args []*Expr, cursors []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	for _, arg := range args {
		if arg.Type == Reference {
			auto = append(auto, arg.Reference)
		}
	}
	return auto
}

func (basicGrabOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("grab operator requires exactly one argument")
	}

	arg := args[0]
	if arg.Type != Reference {
		return nil, fmt.Errorf("grab operator requires a reference argument")
	}

	value, err := arg.Reference.Resolve(ev.Tree)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve reference %s: %v", arg.Reference, err)
	}

	return &Response{
		Type:  Replace,
		Value: value,
	}, nil
}

// basicConcatOperator is a simple concat implementation
type basicConcatOperator struct{}

func (basicConcatOperator) Setup() error { return nil }
func (basicConcatOperator) Phase() OperatorPhase { return EvalPhase }
func (basicConcatOperator) Dependencies(ev *Evaluator, args []*Expr, cursors []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	for _, arg := range args {
		if arg.Type == Reference {
			auto = append(auto, arg.Reference)
		}
	}
	return auto
}

func (basicConcatOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	var result string
	
	for i, arg := range args {
		var value interface{}
		var err error
		
		switch arg.Type {
		case Reference:
			value, err = arg.Reference.Resolve(ev.Tree)
			if err != nil {
				return nil, fmt.Errorf("unable to resolve reference %s: %v", arg.Reference, err)
			}
		case Literal:
			value = arg.Literal
		default:
			return nil, fmt.Errorf("unsupported argument type at position %d", i)
		}
		
		result += fmt.Sprintf("%v", value)
	}

	return &Response{
		Type:  Replace,
		Value: result,
	}, nil
}

func convertToInterfaceMap(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			converted, err := convertToInterfaceMap(value)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			converted, err := convertToInterfaceMap(item)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	default:
		return v, nil
	}
}

func isJSONFile(path string) bool {
	// Simple extension-based detection
	return len(path) > 5 && path[len(path)-5:] == ".json"
}