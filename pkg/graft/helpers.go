package graft

import (
	"fmt"
	"reflect"
	"strings"
)

// deepMerge recursively merges src into dst and returns the result
func deepMerge(dst, src map[interface{}]interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})
	
	// Copy dst first
	for k, v := range dst {
		result[k] = deepCopy(v)
	}
	
	// Then merge src
	for key, srcVal := range src {
		if dstVal, exists := result[key]; exists {
			// If both are maps, merge recursively
			if srcMap, srcOk := srcVal.(map[interface{}]interface{}); srcOk {
				if dstMap, dstOk := dstVal.(map[interface{}]interface{}); dstOk {
					result[key] = deepMerge(dstMap, srcMap)
					continue
				}
			}
		}
		// Otherwise, overwrite the value
		result[key] = deepCopy(srcVal)
	}
	
	return result
}

// deepEqual performs a deep comparison of two values
func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// deepCopy creates a deep copy of the given value
func deepCopy(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[interface{}]interface{}:
		copy := make(map[interface{}]interface{})
		for k, v := range val {
			copy[deepCopy(k)] = deepCopy(v)
		}
		return copy
	case []interface{}:
		copy := make([]interface{}, len(val))
		for i, v := range val {
			copy[i] = deepCopy(v)
		}
		return copy
	case map[string]interface{}:
		copy := make(map[string]interface{})
		for k, v := range val {
			copy[k] = deepCopy(v)
		}
		return copy
	default:
		// For primitive types, return as-is
		return v
	}
}

// joinPath joins path segments with dots
func joinPath(segments ...string) string {
	var nonEmpty []string
	for _, s := range segments {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return strings.Join(nonEmpty, ".")
}

// parsePath splits a dot-separated path into segments
func parsePath(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, ".")
}

// splitPath is an alias for parsePath for compatibility
func splitPath(path string) []string {
	return parsePath(path)
}

// getValueAtPath retrieves a value from a nested map using a dot-separated path
func getValueAtPath(data interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}

	segments := parsePath(path)
	current := data

	for _, segment := range segments {
		switch v := current.(type) {
		case map[interface{}]interface{}:
			val, ok := v[segment]
			if !ok {
				return nil, fmt.Errorf("key %s not found", segment)
			}
			current = val
		case map[string]interface{}:
			val, ok := v[segment]
			if !ok {
				return nil, fmt.Errorf("key %s not found", segment)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot index %T with string key", v)
		}
	}

	return current, nil
}

// setValueAtPath sets a value in a nested map using a dot-separated path
func setValueAtPath(data interface{}, path string, value interface{}) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	segments := parsePath(path)
	if len(segments) == 0 {
		return fmt.Errorf("invalid path")
	}

	// Navigate to the parent of the target
	current := data
	for i := 0; i < len(segments)-1; i++ {
		segment := segments[i]
		switch v := current.(type) {
		case map[interface{}]interface{}:
			if next, ok := v[segment]; ok {
				current = next
			} else {
				// Create intermediate map
				newMap := make(map[interface{}]interface{})
				v[segment] = newMap
				current = newMap
			}
		case map[string]interface{}:
			if next, ok := v[segment]; ok {
				current = next
			} else {
				// Create intermediate map
				newMap := make(map[string]interface{})
				v[segment] = newMap
				current = newMap
			}
		default:
			return fmt.Errorf("cannot index %T with string key", v)
		}
	}

	// Set the final value
	lastSegment := segments[len(segments)-1]
	switch v := current.(type) {
	case map[interface{}]interface{}:
		v[lastSegment] = value
	case map[string]interface{}:
		v[lastSegment] = value
	default:
		return fmt.Errorf("cannot set value in %T", v)
	}

	return nil
}

// sortList sorts a list of items based on a sort key
func sortList(path string, list []interface{}, sortKey string) error {
	// TODO: Implement proper sorting logic
	// For now, this is a placeholder
	return nil
}