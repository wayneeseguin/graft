package graft

import (
	"context"
	"fmt"
	"strings"

	"github.com/wayneeseguin/graft/pkg/graft/merger"
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
		// For single documents, we need to validate arrays even without merging
		// to match legacy behavior for Issue #172
		data := m.docs[0].RawData().(map[interface{}]interface{})
		
		// Always process through merger for consistency with legacy behavior
		mergerInstance := &merger.Merger{
			AppendByDefault: m.fallbackAppend,
		}
		
		// Create an empty base and merge our document into it
		// This triggers the array validation logic
		base := make(map[interface{}]interface{})
		err := mergerInstance.Merge(base, data)
		if err != nil {
			// Convert merger.MultiError to graft.MultiError for consistent error formatting
			if mergerMultiErr, ok := err.(merger.MultiError); ok {
				graftMultiErr := &MultiError{}
				for _, e := range mergerMultiErr.Errors {
					graftMultiErr.Append(e)
				}
				return nil, graftMultiErr
			}
			return nil, err
		}
		
		return m.applyPostProcessing(NewDocument(base))
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
			// Check if this is a detailed merger error that should be preserved
			if isMergerError(err) {
				return nil, err
			}
			return nil, NewMergeError("failed to merge documents", err)
		}
	}

	return NewDocument(result), nil
}

// mergeInto merges overlay data into base data using legacy merger when needed
func (m *mergeBuilderImpl) mergeInto(base, overlay map[interface{}]interface{}) error {
	// For complex merging with array operators, use the legacy merger
	// Array operators are processed even when evaluation is skipped
	if m.hasArrayOperators(overlay) || m.hasArraysWithMaps(overlay) {
		mergerInstance := &merger.Merger{
			AppendByDefault: m.fallbackAppend,
		}
		
		// Create a copy of base to merge into
		baseCopy := deepCopyMap(base)
		
		// Perform the merge
		err := mergerInstance.Merge(baseCopy, overlay)
		if err != nil {
			// Convert merger.MultiError to graft.MultiError for consistent error formatting
			if mergerMultiErr, ok := err.(merger.MultiError); ok {
				graftMultiErr := &MultiError{}
				for _, e := range mergerMultiErr.Errors {
					graftMultiErr.Append(e)
				}
				return graftMultiErr
			}
			return err
		}
		
		// Copy result back to base
		for key, value := range baseCopy {
			base[key] = value
		}
		return nil
	}
	
	// Use simple merging for cases without array operators
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

// mergeArraysWithOperators merges arrays using the legacy merger logic that supports operators
func (m *mergeBuilderImpl) mergeArraysWithOperators(base, overlay []interface{}) (interface{}, error) {
	// Use the existing merger package to handle array operators
	baseMap := map[interface{}]interface{}{"array": base}
	overlayMap := map[interface{}]interface{}{"array": overlay}
	
	result, err := merger.Merge(baseMap, overlayMap)
	if err != nil {
		// Preserve the original error message from the merger, but need to adjust path context
		// The merger uses "$.array.X" but we need to use the actual field name
		// For now, preserve as-is since the test might need updating for the new API
		return nil, err
	}
	
	// Extract the merged array
	mergedArray, exists := result["array"]
	if !exists {
		// Fallback: if merger didn't work as expected, use simple logic
		if m.fallbackAppend {
			// Append arrays
			finalResult := make([]interface{}, len(base)+len(overlay))
			copy(finalResult, base)
			copy(finalResult[len(base):], overlay)
			return finalResult, nil
		} else {
			// Replace arrays (default behavior)
			return deepCopyValue(overlay), nil
		}
	}
	
	return mergedArray, nil
}

// hasArrayOperators checks if a map contains arrays with merge operators
func (m *mergeBuilderImpl) hasArrayOperators(data map[interface{}]interface{}) bool {
	for _, value := range data {
		if array, ok := value.([]interface{}); ok {
			if m.arrayHasOperators(array) {
				return true
			}
		}
	}
	return false
}

// arrayHasOperators checks if an array contains merge operators
func (m *mergeBuilderImpl) arrayHasOperators(array []interface{}) bool {
	for _, item := range array {
		if str, ok := item.(string); ok {
			if strings.Contains(str, "(( append ))") ||
				strings.Contains(str, "(( prepend ))") ||
				strings.Contains(str, "(( replace ))") ||
				strings.Contains(str, "(( inline ))") ||
				strings.Contains(str, "(( merge ))") {
				return true
			}
		}
	}
	return false
}

// hasArraysWithMaps checks if a map contains arrays with map elements (for merge-by-key detection)
func (m *mergeBuilderImpl) hasArraysWithMaps(data map[interface{}]interface{}) bool {
	for _, value := range data {
		if array, ok := value.([]interface{}); ok {
			for _, item := range array {
				if _, isMap := item.(map[interface{}]interface{}); isMap {
					return true
				}
			}
		}
	}
	return false
}

// isMergerError checks if an error came from the merger package and contains detailed info
func isMergerError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// Check for patterns that indicate this is a detailed merger error
	return strings.Contains(errMsg, "cannot merge by key") ||
		strings.Contains(errMsg, "inappropriate use of") ||
		strings.Contains(errMsg, "unable to find specified modification point") ||
		strings.Contains(errMsg, "item in array directly after")
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