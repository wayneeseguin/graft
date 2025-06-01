package graft

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	
	"github.com/starkandwayne/goutils/tree"
	"gopkg.in/yaml.v2"
)

// document implements the Document interface
type document struct {
	data map[interface{}]interface{}
}

// NewDocument creates a new document from a map
func NewDocument(data map[interface{}]interface{}) Document {
	if data == nil {
		data = make(map[interface{}]interface{})
	}
	return &document{data: data}
}

// NewDocumentFromInterface creates a document from any interface{}
func NewDocumentFromInterface(data interface{}) (Document, error) {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		return NewDocument(v), nil
	case map[string]interface{}:
		// Convert map[string]interface{} to map[interface{}]interface{}
		converted := make(map[interface{}]interface{})
		for k, val := range v {
			converted[k] = val
		}
		return NewDocument(converted), nil
	case nil:
		return NewDocument(nil), nil
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
func (d *document) GetMap(path string) (map[string]interface{}, error) {
	val, err := d.Get(path)
	if err != nil {
		return nil, err
	}
	
	switch v := val.(type) {
	case map[string]interface{}:
		return v, nil
	case map[interface{}]interface{}:
		// Convert to string-keyed map
		result := make(map[string]interface{})
		for k, v := range v {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("map at path %s contains non-string key: %v", path, k)
			}
			result[key] = v
		}
		return result, nil
	default:
		return nil, NewValidationError(fmt.Sprintf("value at path '%s' is not a map (got %T)", path, val))
	}
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

// convertToJSONCompatible converts map[interface{}]interface{} to map[string]interface{}
func convertToJSONCompatible(v interface{}) interface{} {
	switch v := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, val := range v {
			key, ok := k.(string)
			if !ok {
				key = fmt.Sprintf("%v", k)
			}
			m[key] = convertToJSONCompatible(val)
		}
		return m
	case []interface{}:
		arr := make([]interface{}, len(v))
		for i, val := range v {
			arr[i] = convertToJSONCompatible(val)
		}
		return arr
	default:
		return v
	}
}

// ToYAML converts the document to YAML bytes
func (d *document) ToYAML() ([]byte, error) {
	return yaml.Marshal(d.data)
}

// ToJSON converts the document to JSON bytes
func (d *document) ToJSON() ([]byte, error) {
	// Convert to JSON-compatible format first
	jsonData := convertToJSONCompatible(d.data)
	return json.Marshal(jsonData)
}

// RawData returns the underlying data structure
func (d *document) RawData() interface{} {
	return d.data
}

// Clone creates a deep copy of the document
func (d *document) Clone() Document {
	cloned := deepCopy(d.data)
	if clonedMap, ok := cloned.(map[interface{}]interface{}); ok {
		return NewDocument(clonedMap)
	}
	// Fallback - this shouldn't happen
	return NewDocument(make(map[interface{}]interface{}))
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
func CreateEmptyDocument() Document {
	return NewDocument(make(map[interface{}]interface{}))
}

// Prune removes a key from the document
func (d *document) Prune(key string) Document {
	// Clone the document to avoid modifying the original
	cloned := d.Clone().(*document)
	
	// Check if this is a path or a simple key
	if strings.Contains(key, ".") {
		// Handle nested path
		segments := strings.Split(key, ".")
		current := cloned.data
		
		// Navigate to the parent of the key to remove
		var lastList []interface{}
		var lastListKey string
		for i := 0; i < len(segments)-1; i++ {
			segment := segments[i]
			
			switch v := current[segment].(type) {
			case map[interface{}]interface{}:
				current = v
			case map[string]interface{}:
				// Convert to map[interface{}]interface{} for consistency
				newMap := make(map[interface{}]interface{})
				for k, val := range v {
					newMap[k] = val
				}
				current[segment] = newMap
				current = newMap
			case []interface{}:
				// Handle list navigation
				if i == len(segments)-2 {
					// Next segment should be the index to remove
					lastList = v
					lastListKey = segment
				} else {
					// Can't navigate through lists except at the end
					return cloned
				}
			default:
				// Path doesn't exist or leads through a non-map/list
				return cloned
			}
		}
		
		// Remove the final key
		finalSegment := segments[len(segments)-1]
		if lastList != nil {
			// Remove from list by index
			if index, err := strconv.Atoi(finalSegment); err == nil {
				if index >= 0 && index < len(lastList) {
					// Create new list without the element at index
					newList := make([]interface{}, 0, len(lastList)-1)
					newList = append(newList, lastList[:index]...)
					newList = append(newList, lastList[index+1:]...)
					current[lastListKey] = newList
				}
			}
		} else {
			// Remove from map
			delete(current, finalSegment)
		}
	} else {
		// Simple key - remove from top level
		delete(cloned.data, key)
	}
	
	return cloned
}

// CherryPick creates a new document with only the specified keys
func (d *document) CherryPick(keys ...string) Document {
	picked := make(map[interface{}]interface{})
	errors := []error{}
	
	// Track list items separately to preserve order
	listItems := make(map[string][]interface{})
	
	for _, keyPath := range keys {
		// Check if this is a simple key or a path
		if !strings.Contains(keyPath, ".") {
			// Simple key - use the old behavior
			if val, exists := d.data[keyPath]; exists {
				picked[keyPath] = deepCopy(val)
			} else {
				// Path not found - we need to handle this error
				// For now, we'll track it but can't return it since Document interface doesn't support errors
				errors = append(errors, fmt.Errorf("`$.%s` could not be found in the datastructure", keyPath))
			}
		} else {
			// Path-based cherry-pick
			segments := strings.Split(keyPath, ".")
			
			// Special handling for list entries (e.g., "list.two" or "list.1")
			if len(segments) == 2 {
				listKey := segments[0]
				listItemKey := segments[1]
				
				// Get the list
				if listVal, exists := d.data[listKey]; exists {
					if list, ok := listVal.([]interface{}); ok {
						var foundItem interface{}
						
						// First, try to parse as numeric index
						if index, err := strconv.Atoi(listItemKey); err == nil {
							// It's a numeric index
							if index >= 0 && index < len(list) {
								foundItem = list[index]
							}
						} else {
							// Not a number, look for named item
							for _, item := range list {
								if itemMap, ok := item.(map[interface{}]interface{}); ok {
									// Check common identifier fields
									for _, idField := range []string{"key", "id", "name"} {
										if idVal, hasId := itemMap[idField]; hasId {
											if idStr, ok := idVal.(string); ok && idStr == listItemKey {
												foundItem = item
												break
											}
										}
									}
									if foundItem != nil {
										break
									}
								}
							}
						}
						
						// If we found the item, track it in order
						if foundItem != nil {
							listItems[listKey] = append(listItems[listKey], deepCopy(foundItem))
						}
					}
				}
			} else {
				// For other paths, try to extract the value and reconstruct the path
				val, err := d.Get(keyPath)
				if err == nil && val != nil {
					// Reconstruct the nested structure
					current := picked
					for i := 0; i < len(segments)-1; i++ {
						if _, exists := current[segments[i]]; !exists {
							current[segments[i]] = make(map[interface{}]interface{})
						}
						if m, ok := current[segments[i]].(map[interface{}]interface{}); ok {
							current = m
						} else {
							// Can't continue if intermediate value is not a map
							break
						}
					}
					// Set the final value
					if len(segments) > 0 {
						current[segments[len(segments)-1]] = deepCopy(val)
					}
				}
			}
		}
	}
	
	// Add all collected list items to the picked document
	for listKey, items := range listItems {
		if len(items) > 0 {
			picked[listKey] = items
		}
	}
	
	return NewDocument(picked)
}

// GetData returns the underlying data (for backward compatibility)
func (d *document) GetData() interface{} {
	return d.data
}

// GetInt64 retrieves an int64 value at the given path
func (d *document) GetInt64(path string) (int64, error) {
	val, err := d.Get(path)
	if err != nil {
		return 0, err
	}
	
	switch v := val.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		if v == float64(int64(v)) {
			return int64(v), nil
		}
		return 0, fmt.Errorf("value at path %s is a float, not an integer", path)
	default:
		return 0, fmt.Errorf("value at path %s is not an integer (got %T)", path, val)
	}
}

// GetFloat64 retrieves a float64 value at the given path
func (d *document) GetFloat64(path string) (float64, error) {
	val, err := d.Get(path)
	if err != nil {
		return 0, err
	}
	
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("value at path %s is not a number (got %T)", path, val)
	}
}


// GetStringSlice retrieves a string slice value at the given path
func (d *document) GetStringSlice(path string) ([]string, error) {
	val, err := d.Get(path)
	if err != nil {
		return nil, err
	}
	
	slice, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value at path %s is not a slice (got %T)", path, val)
	}
	
	result := make([]string, 0, len(slice))
	for i, item := range slice {
		str, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("item at index %d in slice at path %s is not a string (got %T)", i, path, item)
		}
		result = append(result, str)
	}
	return result, nil
}

// GetMapStringString retrieves a string-to-string map at the given path
func (d *document) GetMapStringString(path string) (map[string]string, error) {
	val, err := d.Get(path)
	if err != nil {
		return nil, err
	}
	
	var rawMap map[interface{}]interface{}
	switch v := val.(type) {
	case map[string]interface{}:
		// Convert to interface{} keyed map for uniform processing
		rawMap = make(map[interface{}]interface{})
		for k, v := range v {
			rawMap[k] = v
		}
	case map[interface{}]interface{}:
		rawMap = v
	default:
		return nil, fmt.Errorf("value at path %s is not a map (got %T)", path, val)
	}
	
	result := make(map[string]string)
	for k, v := range rawMap {
		key, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("map at path %s contains non-string key: %v", path, k)
		}
		value, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("map at path %s contains non-string value for key %s: %v", path, key, v)
		}
		result[key] = value
	}
	return result, nil
}