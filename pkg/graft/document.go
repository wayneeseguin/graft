package graft

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	
	"github.com/starkandwayne/goutils/tree"
	"gopkg.in/yaml.v2"
)

// document implements the DocumentV2 interface
type document struct {
	data map[interface{}]interface{}
}

// NewDocumentV2 creates a new document from a map
func NewDocumentV2(data map[interface{}]interface{}) DocumentV2 {
	if data == nil {
		data = make(map[interface{}]interface{})
	}
	return &document{data: data}
}

// NewDocumentFromInterface creates a document from any interface{}
func NewDocumentFromInterface(data interface{}) (DocumentV2, error) {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		return NewDocumentV2(v), nil
	case map[string]interface{}:
		// Convert map[string]interface{} to map[interface{}]interface{}
		converted := make(map[interface{}]interface{})
		for k, val := range v {
			converted[k] = val
		}
		return NewDocumentV2(converted), nil
	case nil:
		return NewDocumentV2(nil), nil
	default:
		return nil, NewValidationError(fmt.Sprintf("cannot create document from type %T", data))
	}
}

// Get retrieves a value at the given path
func (d *document) Get(path string) (interface{}, error) {
	if path == "" || path == "$" {
		return d.data, nil
	}
	
	cursor, err := tree.ParseCursor(path)
	if err != nil {
		return nil, NewValidationError(fmt.Sprintf("invalid path '%s': %v", path, err))
	}
	
	value, err := cursor.Resolve(d.data)
	if err != nil {
		return nil, NewEvaluationError(path, fmt.Sprintf("path not found: %v", err), err)
	}
	
	return value, nil
}

// Set sets a value at the given path
func (d *document) Set(path string, value interface{}) error {
	if path == "" || path == "$" {
		if mapValue, ok := value.(map[interface{}]interface{}); ok {
			d.data = mapValue
			return nil
		}
		return NewValidationError("cannot set root to non-map value")
	}
	
	cursor, err := tree.ParseCursor(path)
	if err != nil {
		return NewValidationError(fmt.Sprintf("invalid path '%s': %v", path, err))
	}
	
	err = d.ensurePathExists(cursor)
	if err != nil {
		return err
	}
	
	// TODO: Implement cursor.Set method or alternative approach
	return NewValidationError("Set operation not yet implemented")
}

// Delete removes a value at the given path
func (d *document) Delete(path string) error {
	if path == "" || path == "$" {
		return NewValidationError("cannot delete root")
	}
	
	_, err := tree.ParseCursor(path)
	if err != nil {
		return NewValidationError(fmt.Sprintf("invalid path '%s': %v", path, err))
	}
	
	// TODO: Implement cursor.Delete method or alternative approach  
	return NewValidationError("Delete operation not yet implemented")
}

// GetString retrieves a string value at the given path
func (d *document) GetString(path string) (string, error) {
	val, err := d.Get(path)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", NewValidationError(fmt.Sprintf("value at path '%s' is not a string (got %T)", path, val))
}

// GetInt retrieves an integer value at the given path
func (d *document) GetInt(path string) (int, error) {
	val, err := d.Get(path)
	if err != nil {
		return 0, err
	}
	
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		// JSON numbers are parsed as float64
		if v == float64(int(v)) {
			return int(v), nil
		}
		return 0, NewValidationError(fmt.Sprintf("value at path '%s' is not a whole number (got %f)", path, v))
	case float32:
		if v == float32(int(v)) {
			return int(v), nil
		}
		return 0, NewValidationError(fmt.Sprintf("value at path '%s' is not a whole number (got %f)", path, v))
	default:
		return 0, NewValidationError(fmt.Sprintf("value at path '%s' is not a number (got %T)", path, val))
	}
}

// GetBool retrieves a boolean value at the given path
func (d *document) GetBool(path string) (bool, error) {
	val, err := d.Get(path)
	if err != nil {
		return false, err
	}
	if b, ok := val.(bool); ok {
		return b, nil
	}
	return false, NewValidationError(fmt.Sprintf("value at path '%s' is not a boolean (got %T)", path, val))
}

// GetSlice retrieves a slice value at the given path
func (d *document) GetSlice(path string) ([]interface{}, error) {
	val, err := d.Get(path)
	if err != nil {
		return nil, err
	}
	if slice, ok := val.([]interface{}); ok {
		return slice, nil
	}
	return nil, NewValidationError(fmt.Sprintf("value at path '%s' is not a slice (got %T)", path, val))
}

// GetMap retrieves a map value at the given path
func (d *document) GetMap(path string) (map[interface{}]interface{}, error) {
	val, err := d.Get(path)
	if err != nil {
		return nil, err
	}
	if m, ok := val.(map[interface{}]interface{}); ok {
		return m, nil
	}
	return nil, NewValidationError(fmt.Sprintf("value at path '%s' is not a map (got %T)", path, val))
}

// Keys returns all top-level keys
func (d *document) Keys() []string {
	var keys []string
	for k := range d.data {
		if s, ok := k.(string); ok {
			keys = append(keys, s)
		} else {
			keys = append(keys, fmt.Sprintf("%v", k))
		}
	}
	return keys
}

// ToMap returns the underlying map representation
func (d *document) ToMap() map[interface{}]interface{} {
	return d.data
}

// ToYAML converts the document to YAML bytes
func (d *document) ToYAML() ([]byte, error) {
	return yaml.Marshal(d.data)
}

// ToJSON converts the document to JSON bytes
func (d *document) ToJSON() ([]byte, error) {
	return json.Marshal(d.data)
}

// RawData returns the underlying data structure
func (d *document) RawData() interface{} {
	return d.data
}

// Clone creates a deep copy of the document
func (d *document) Clone() DocumentV2 {
	cloned := deepCopy(d.data)
	if clonedMap, ok := cloned.(map[interface{}]interface{}); ok {
		return NewDocumentV2(clonedMap)
	}
	// Fallback - this shouldn't happen
	return NewDocumentV2(make(map[interface{}]interface{}))
}

// ensurePathExists creates intermediate maps/slices as needed for the given path
func (d *document) ensurePathExists(cursor *tree.Cursor) error {
	// This is a simplified implementation
	// A full implementation would need to handle array indices and create intermediate structures
	return nil
}

// deepCopy performs a deep copy of the data structure
func deepCopy(src interface{}) interface{} {
	switch v := src.(type) {
	case map[interface{}]interface{}:
		dst := make(map[interface{}]interface{})
		for key, value := range v {
			dst[key] = deepCopy(value)
		}
		return dst
		
	case map[string]interface{}:
		dst := make(map[string]interface{})
		for key, value := range v {
			dst[key] = deepCopy(value)
		}
		return dst
		
	case []interface{}:
		dst := make([]interface{}, len(v))
		for i, value := range v {
			dst[i] = deepCopy(value)
		}
		return dst
		
	default:
		// For primitive types and other types, return as-is
		// This handles strings, numbers, booleans, etc.
		return v
	}
}

// pathParts splits a path into its components
func pathParts(path string) []string {
	if path == "" || path == "$" {
		return nil
	}
	
	// Remove leading $ if present
	if strings.HasPrefix(path, "$.") {
		path = path[2:]
	} else if path == "$" {
		return nil
	}
	
	return strings.Split(path, ".")
}

// parseIndex extracts array index from a path component like "items[0]"
func parseIndex(component string) (string, int, bool) {
	if !strings.Contains(component, "[") {
		return component, 0, false
	}
	
	parts := strings.SplitN(component, "[", 2)
	if len(parts) != 2 {
		return component, 0, false
	}
	
	indexStr := strings.TrimSuffix(parts[1], "]")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return component, 0, false
	}
	
	return parts[0], index, true
}

// CreateEmptyDocument creates a new empty document
func CreateEmptyDocument() DocumentV2 {
	return NewDocumentV2(make(map[interface{}]interface{}))
}