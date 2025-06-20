package operators

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// pathsToSort moved to engine context - keeping this for backward compatibility
var pathsToSort = map[string]string{}

type itemType int

const (
	stringItems itemType = iota
	floatItems
	intItems
	mapItems
	otherItems
)

// SortOperator ...
type SortOperator struct{}

// Setup ...
func (SortOperator) Setup() error {
	return nil
}

// Phase ...
func (SortOperator) Phase() OperatorPhase {
	return MergePhase
}

// Dependencies ...
func (SortOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (SortOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	// The sort operator is handled as a post-processing step after evaluation,
	// so during evaluation it just returns the current value unchanged.
	// The actual sorting happens in the engine's evaluate method.
	val, err := ev.Here.Resolve(ev.Tree)
	if err != nil {
		return nil, err
	}
	return &Response{
		Type:  Replace,
		Value: val,
	}, nil
}

func init() {
	RegisterOp("sort", SortOperator{})
}

func addToSortListIfNecessary(operator string, path string) {
	if opcall, err := ParseOpcall(MergePhase, operator); err == nil {
		var byKey string
		args := opcall.Args()
		if len(args) == 2 {
			byKey = args[1].String()
		}

		DEBUG("adding sort by '%s' of path '%s' to the list of paths to sort", byKey, path)
		if _, ok := pathsToSort[path]; !ok {
			pathsToSort[path] = byKey
		}
	}
}

// AddToSortListIfNecessaryWithEngine is the engine-aware version
func AddToSortListIfNecessaryWithEngine(operator string, path string, engine graft.Engine) {
	if opcall, err := ParseOpcall(MergePhase, operator); err == nil {
		var byKey string
		args := opcall.Args()
		if len(args) == 2 {
			byKey = args[1].String()
		}

		DEBUG("adding sort by '%s' of path '%s' to the list of paths to sort", byKey, path)
		if engine != nil {
			engine.GetOperatorState().AddPathToSort(path, byKey)
		} else {
			// Fallback to global state for backward compatibility
			if _, ok := pathsToSort[path]; !ok {
				pathsToSort[path] = byKey
			}
		}
	}
}

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

func sortList(path string, list []interface{}, key string) error {
	typeCheckMap := map[string]struct{}{}
	for _, entry := range list {
		reflectType := reflect.TypeOf(entry)

		var typeName string
		if reflectType != nil {
			typeName = reflectType.Kind().String()
		} else {
			typeName = "nil"
		}

		if _, ok := typeCheckMap[typeName]; !ok {
			typeCheckMap[typeName] = struct{}{}
		}
	}

	if length := len(typeCheckMap); length > 0 && length != 1 {
		return tree.TypeMismatchError{
			Path:   []string{path},
			Wanted: "a list with homogeneous entry types",
			Got:    "a list with different types",
		}
	}

	for kind := range typeCheckMap {
		switch kind {
		case reflect.Map.String():
			if key == "" {
				key = "name" // default identifier key
			}

			// Check if all maps have the key
			for _, item := range list {
				if m, ok := item.(map[interface{}]interface{}); ok {
					if _, hasKey := m[key]; !hasKey {
						return tree.TypeMismatchError{
							Path:   []string{path},
							Wanted: fmt.Sprintf("a list with map entries each containing %s", key),
							Got:    fmt.Sprintf("a list with map entries, where some do not contain %s", key),
						}
					}
				}
			}

		case reflect.Slice.String():
			return tree.TypeMismatchError{
				Path:   []string{path},
				Wanted: fmt.Sprintf("a list with maps, strings or numbers"),
				Got:    fmt.Sprintf("a list with list entries"),
			}
		}
	}

	sort.Slice(list, func(i int, j int) bool {
		return universalLess(list[i], list[j], key)
	})

	return nil
}
