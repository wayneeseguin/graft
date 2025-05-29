package graft

import (
	"fmt"
	"reflect"
	"sort"
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

// deepCopyHelper creates a deep copy of the given value
func deepCopyHelper(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[interface{}]interface{}:
		copy := make(map[interface{}]interface{})
		for k, v := range val {
			copy[deepCopyHelper(k)] = deepCopyHelper(v)
		}
		return copy
	case []interface{}:
		copy := make([]interface{}, len(val))
		for i, v := range val {
			copy[i] = deepCopyHelper(v)
		}
		return copy
	case map[string]interface{}:
		copy := make(map[string]interface{})
		for k, v := range val {
			copy[k] = deepCopyHelper(v)
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

// DeepCopyMap creates a deep copy of a map[interface{}]interface{}
func DeepCopyMap(m map[interface{}]interface{}) map[interface{}]interface{} {
	if m == nil {
		return nil
	}
	return deepCopyHelper(m).(map[interface{}]interface{})
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

// SortList sorts a list of items based on a sort key
// This is a helper that delegates to the operators package implementation
func SortList(path string, list []interface{}, sortKey string) error {
	// Handle empty list
	if len(list) == 0 {
		return nil
	}
	
	// Type checking
	var commonType string
	hasInconsistentMaps := false
	
	for i, entry := range list {
		var typeName string
		
		if entry == nil {
			typeName = "nil"
		} else {
			switch v := entry.(type) {
			case string:
				typeName = "string"
			case int, int32, int64:
				typeName = "int"
			case float32, float64:
				typeName = "float64"
			case []interface{}:
				// Special error for lists of lists
				return fmt.Errorf("$.%s is a list with list entries (not a list with maps, strings or numbers)", path)
			case map[interface{}]interface{}:
				// Always consider maps as maps for type checking
				typeName = "map"
				
				// Check if it's a named-entry map
				if sortKey != "" {
					if _, hasKey := v[sortKey]; !hasKey {
						hasInconsistentMaps = true
					}
				} else {
					// Auto-detect sort key from first map
					if i == 0 {
						for _, field := range []string{"name", "key", "id"} {
							if _, ok := v[field]; ok {
								sortKey = field
								break
							}
						}
					}
				}
			default:
				// Get the reflect kind
				reflectType := reflect.TypeOf(entry)
				if reflectType != nil {
					typeName = reflectType.Kind().String()
				} else {
					typeName = "unknown"
				}
			}
		}
		
		// Set or check common type
		if i == 0 {
			commonType = typeName
		} else if commonType != typeName {
			// Different types detected
			if typeName == "nil" || commonType == "nil" {
				return fmt.Errorf("$.%s is a list with different types (not a list with homogeneous entry types)", path)
			}
			return fmt.Errorf("$.%s is a list with different types (not a list with homogeneous entry types)", path)
		}
	}
	
	// Check for inconsistent map entries
	if commonType == "map" && hasInconsistentMaps && sortKey != "" {
		return fmt.Errorf("$.%s is a list with map entries, where some do not contain %s (not a list with map entries each containing %s)", path, sortKey, sortKey)
	}
	
	// Sort the list
	sort.Slice(list, func(i, j int) bool {
		return universalLess(list[i], list[j], sortKey)
	})
	
	return nil
}

// universalLess compares two values for sorting
func universalLess(a interface{}, b interface{}, key string) bool {
	switch a.(type) {
	case string:
		return strings.Compare(a.(string), b.(string)) < 0
		
	case float64:
		return a.(float64) < b.(float64)
		
	case int:
		return a.(int) < b.(int)
		
	case map[interface{}]interface{}:
		entryA, entryB := a.(map[interface{}]interface{}), b.(map[interface{}]interface{})
		return universalLess(entryA[key], entryB[key], key)
	}
	
	return false
}