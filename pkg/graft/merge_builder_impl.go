package graft

import (
	"context"
	"fmt"
	"strings"
)

// mergeBuilderImpl implements the MergeBuilder interface
type mergeBuilderImpl struct {
	engine           Engine
	ctx              context.Context
	docs             []Document
	pruneKeys        []string
	cherryPickKeys   []string
	skipEvaluation   bool
	goPatch          bool
	fallbackAppend   bool
	arrayStrategy    ArrayMergeStrategy
	error            error // Stores any error from construction
}

// WithPrune adds keys to remove from the final output
func (m *mergeBuilderImpl) WithPrune(keys ...string) MergeBuilder {
	if m.error != nil {
		return m // Propagate error
	}
	
	newBuilder := *m // Copy the builder
	newBuilder.pruneKeys = append(m.pruneKeys, keys...)
	return &newBuilder
}

// WithCherryPick specifies keys to keep in the final output
func (m *mergeBuilderImpl) WithCherryPick(keys ...string) MergeBuilder {
	if m.error != nil {
		return m // Propagate error
	}
	
	newBuilder := *m // Copy the builder
	newBuilder.cherryPickKeys = append(m.cherryPickKeys, keys...)
	return &newBuilder
}

// SkipEvaluation disables operator evaluation after merging
func (m *mergeBuilderImpl) SkipEvaluation() MergeBuilder {
	if m.error != nil {
		return m // Propagate error
	}
	
	newBuilder := *m // Copy the builder
	newBuilder.skipEvaluation = true
	return &newBuilder
}

// EnableGoPatch enables go-patch format parsing
func (m *mergeBuilderImpl) EnableGoPatch() MergeBuilder {
	if m.error != nil {
		return m // Propagate error
	}
	
	newBuilder := *m // Copy the builder
	newBuilder.goPatch = true
	return &newBuilder
}

// FallbackAppend uses append instead of inline for arrays by default
func (m *mergeBuilderImpl) FallbackAppend() MergeBuilder {
	if m.error != nil {
		return m // Propagate error
	}
	
	newBuilder := *m // Copy the builder
	newBuilder.fallbackAppend = true
	newBuilder.arrayStrategy = AppendArrays
	return &newBuilder
}

// WithArrayMergeStrategy sets how arrays are merged
func (m *mergeBuilderImpl) WithArrayMergeStrategy(strategy ArrayMergeStrategy) MergeBuilder {
	if m.error != nil {
		return m // Propagate error
	}
	
	newBuilder := *m // Copy the builder
	newBuilder.arrayStrategy = strategy
	// Update fallbackAppend based on strategy
	if strategy == AppendArrays {
		newBuilder.fallbackAppend = true
	}
	return &newBuilder
}

// Execute performs the merge operation
func (m *mergeBuilderImpl) Execute() (Document, error) {
	// Check for construction errors first
	if m.error != nil {
		return nil, m.error
	}

	// Check context cancellation
	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	default:
	}

	// Handle empty document list
	if len(m.docs) == 0 {
		return NewDocument(make(map[interface{}]interface{})), nil
	}

	// Handle single document case
	if len(m.docs) == 1 {
		result := m.docs[0].Clone()
		return m.applyPostProcessing(result)
	}

	// Merge multiple documents
	result, err := m.mergeDocuments()
	if err != nil {
		return nil, err
	}

	return m.applyPostProcessing(result)
}

// mergeDocuments performs the actual document merging
func (m *mergeBuilderImpl) mergeDocuments() (Document, error) {
	// Start with the first document as base
	baseData := m.docs[0].RawData().(map[interface{}]interface{})
	result := deepCopyMap(baseData)

	// Merge subsequent documents
	for i := 1; i < len(m.docs); i++ {
		// Check context cancellation during merge
		select {
		case <-m.ctx.Done():
			return nil, m.ctx.Err()
		default:
		}

		overlayData := m.docs[i].RawData().(map[interface{}]interface{})
		err := m.mergeInto(result, overlayData)
		if err != nil {
			return nil, NewMergeError("failed to merge documents", err)
		}
	}

	return NewDocument(result), nil
}

// mergeInto merges overlay data into base data
func (m *mergeBuilderImpl) mergeInto(base, overlay map[interface{}]interface{}) error {
	for key, overlayValue := range overlay {
		baseValue, exists := base[key]
		
		if !exists {
			// Key doesn't exist in base, add it
			base[key] = deepCopyValue(overlayValue)
			continue
		}

		// Both values exist, need to merge them
		merged, err := m.mergeValues(baseValue, overlayValue)
		if err != nil {
			return err
		}
		base[key] = merged
	}
	
	return nil
}

// mergeValues merges two values based on their types
func (m *mergeBuilderImpl) mergeValues(base, overlay interface{}) (interface{}, error) {
	// If overlay is nil, keep base
	if overlay == nil {
		return base, nil
	}

	// If base is nil, use overlay
	if base == nil {
		return deepCopyValue(overlay), nil
	}

	// Handle map merging
	baseMap, baseIsMap := base.(map[interface{}]interface{})
	overlayMap, overlayIsMap := overlay.(map[interface{}]interface{})
	
	if baseIsMap && overlayIsMap {
		result := deepCopyMap(baseMap)
		err := m.mergeInto(result, overlayMap)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// Handle array merging
	baseArray, baseIsArray := base.([]interface{})
	overlayArray, overlayIsArray := overlay.([]interface{})
	
	if baseIsArray && overlayIsArray {
		if m.fallbackAppend {
			// Append arrays
			result := make([]interface{}, len(baseArray)+len(overlayArray))
			copy(result, baseArray)
			copy(result[len(baseArray):], overlayArray)
			return result, nil
		} else {
			// Replace arrays (default behavior)
			return deepCopyValue(overlayArray), nil
		}
	}

	// For different types or scalars, overlay replaces base
	return deepCopyValue(overlay), nil
}

// applyPostProcessing applies pruning, cherry-picking, and evaluation
func (m *mergeBuilderImpl) applyPostProcessing(doc Document) (Document, error) {
	result := doc

	// Apply pruning
	if len(m.pruneKeys) > 0 {
		pruned, err := m.applyPruning(result)
		if err != nil {
			return nil, err
		}
		result = pruned
	}

	// Apply cherry-picking
	if len(m.cherryPickKeys) > 0 {
		cherryPicked, err := m.applyCherryPicking(result)
		if err != nil {
			return nil, err
		}
		result = cherryPicked
	}

	// Apply evaluation if not skipped
	if !m.skipEvaluation {
		evaluated, err := m.applyEvaluation(result)
		if err != nil {
			return nil, err
		}
		result = evaluated
	}

	return result, nil
}

// applyPruning removes specified keys from the document
func (m *mergeBuilderImpl) applyPruning(doc Document) (Document, error) {
	data := doc.RawData().(map[interface{}]interface{})
	result := deepCopyMap(data)

	for _, key := range m.pruneKeys {
		err := m.removeKey(result, key)
		if err != nil {
			// Log warning but don't fail - key might not exist
			continue
		}
	}

	return NewDocument(result), nil
}

// applyCherryPicking keeps only specified keys in the document
func (m *mergeBuilderImpl) applyCherryPicking(doc Document) (Document, error) {
	data := doc.RawData().(map[interface{}]interface{})
	result := make(map[interface{}]interface{})

	for _, key := range m.cherryPickKeys {
		value, err := m.extractKey(data, key)
		if err != nil {
			// Log warning but continue - key might not exist
			continue
		}
		err = m.setKey(result, key, value)
		if err != nil {
			return nil, err
		}
	}

	return NewDocument(result), nil
}

// applyEvaluation runs operator evaluation on the document
func (m *mergeBuilderImpl) applyEvaluation(doc Document) (Document, error) {
	// Use the engine's evaluate method if available
	if m.engine != nil {
		return m.engine.Evaluate(m.ctx, doc)
	}

	// Fallback: create basic evaluator (this should not happen in practice)
	data := doc.RawData().(map[interface{}]interface{})
	
	// Create evaluator
	evaluator := &Evaluator{
		Tree: data,
	}

	// Run evaluation
	err := evaluator.Run(nil, nil)
	if err != nil {
		return nil, NewEvaluationError("", "failed to evaluate merged document", err)
	}

	return NewDocument(evaluator.Tree), nil
}

// Helper functions for key manipulation

func (m *mergeBuilderImpl) removeKey(data map[interface{}]interface{}, keyPath string) error {
	// Handle nested paths like "config.enabled"
	if keyPath == "" {
		return nil
	}
	
	// Split path by dots
	parts := strings.Split(keyPath, ".")
	if len(parts) == 1 {
		// Simple key
		delete(data, keyPath)
		return nil
	}
	
	// Navigate to parent and delete the final key
	current := data
	for i, part := range parts[:len(parts)-1] {
		value, exists := current[part]
		if !exists {
			// Path doesn't exist, nothing to remove
			return nil
		}
		
		nextMap, ok := value.(map[interface{}]interface{})
		if !ok {
			// Can't navigate further, path is invalid
			return NewValidationError(fmt.Sprintf("cannot navigate path '%s' at segment %d: '%s' is not a map", keyPath, i, part))
		}
		current = nextMap
	}
	
	// Remove the final key
	finalKey := parts[len(parts)-1]
	delete(current, finalKey)
	return nil
}

func (m *mergeBuilderImpl) extractKey(data map[interface{}]interface{}, keyPath string) (interface{}, error) {
	// Simple implementation - for full implementation would need path parsing
	if simpleKey := keyPath; simpleKey != "" {
		if value, exists := data[simpleKey]; exists {
			return deepCopyValue(value), nil
		}
	}
	return nil, NewValidationError("key not found: " + keyPath)
}

func (m *mergeBuilderImpl) setKey(data map[interface{}]interface{}, keyPath string, value interface{}) error {
	// Simple implementation - for full implementation would need path parsing
	if simpleKey := keyPath; simpleKey != "" {
		data[simpleKey] = value
	}
	return nil
}

// Deep copy helpers

func deepCopyMap(src map[interface{}]interface{}) map[interface{}]interface{} {
	dst := make(map[interface{}]interface{})
	for key, value := range src {
		dst[key] = deepCopyValue(value)
	}
	return dst
}

func deepCopyValue(src interface{}) interface{} {
	switch v := src.(type) {
	case map[interface{}]interface{}:
		return deepCopyMap(v)
	case []interface{}:
		dst := make([]interface{}, len(v))
		for i, item := range v {
			dst[i] = deepCopyValue(item)
		}
		return dst
	case map[string]interface{}:
		dst := make(map[string]interface{})
		for key, value := range v {
			dst[key] = deepCopyValue(value)
		}
		return dst
	default:
		return v // Primitives are copied by value
	}
}