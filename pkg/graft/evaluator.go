package graft

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	
	"github.com/wayneeseguin/graft/internal/utils/ansi"
	"github.com/wayneeseguin/graft/internal/utils/tree"
	
	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft/merger"
)

// Evaluator ...
type Evaluator struct {
	Tree     map[interface{}]interface{}
	Deps     map[string][]tree.Cursor
	SkipEval bool
	Here     *tree.Cursor

	CheckOps []*Opcall

	Only []string

	pointer *interface{}
	
	// Reference to the engine (for accessing registries and state)
	engine interface{} // Using interface{} to avoid circular dependency
	
	// DataflowOrder controls the ordering of operations in dataflow output
	// "alphabetical" (default) - sort operations alphabetically by path
	// "insertion" - maintain the order operations were discovered
	DataflowOrder string
	
	// CherryPickPaths contains the paths to cherry-pick during evaluation.
	// When set, only operators under these paths and their dependencies will be evaluated.
	// This enables selective evaluation, significantly improving performance for large documents
	// when only specific parts are needed.
	//
	// Selective Evaluation Behavior:
	// - Only operators whose paths match or are under cherry-pick paths are evaluated
	// - Dependencies of cherry-picked operators are automatically included (transitive)
	// - Path matching supports both exact indices and named array entries
	// - Empty cherry-pick paths means evaluate everything (default behavior)
	//
	// Example: If cherry-picking "services.web", these operators will be evaluated:
	//   - services.web.port: (( grab defaults.port ))     // Under cherry-pick path
	//   - defaults.port: 8080                             // Dependency of above
	// But this won't be evaluated:
	//   - services.api.port: (( grab defaults.api_port )) // Not under cherry-pick path
	CherryPickPaths []string
}

// SetEngine sets the engine for the evaluator
func (ev *Evaluator) SetEngine(engine interface{}) {
	ev.engine = engine
}

func nameOfObj(o interface{}, def string) string {
	for _, field := range tree.NameFields {
		switch o.(type) {
		case map[string]interface{}:
			if value, ok := o.(map[string]interface{})[field]; ok {
				if s, ok := value.(string); ok {
					return s
				}
			}
		case map[interface{}]interface{}:
			if value, ok := o.(map[interface{}]interface{})[field]; ok {
				if s, ok := value.(string); ok {
					return s
				}
			}
		}
	}
	return def
}

// DataFlow ...
func (ev *Evaluator) DataFlow(phase OperatorPhase) ([]*Opcall, error) {
	ev.Here = &tree.Cursor{}
	
	log.DEBUG("DataFlow: starting phase %v", phase)

	all := map[string]*Opcall{}
	insertionOrder := []string{} // Track insertion order
	locs := []*tree.Cursor{}
	errors := MultiError{Errors: []error{}}

	// forward decls of co-recursive function
	var check func(interface{})
	var scan func(interface{})

	check = func(v interface{}) {
		if s, ok := v.(string); ok {
			if strings.Contains(s, "grab base") {
				log.DEBUG("evaluator.check: found string with 'grab base': %s", s)
			}
			op, err := ParseOpcallCompat(phase, s)
			if err != nil {
				errors.Append(err)
			} else if op != nil {
				op.where = ev.Here.Copy()
				if canon, err := op.where.Canonical(ev.Tree); err == nil {
					op.canonical = canon
				} else {
					op.canonical = op.where
				}
				if _, exists := all[op.canonical.String()]; !exists {
					insertionOrder = append(insertionOrder, op.canonical.String())
				}
				all[op.canonical.String()] = op
				log.TRACE("found an operation at %s: %s", op.where.String(), op.src)
				log.TRACE("        (canonical at %s)", op.canonical.String())
				locs = append(locs, op.canonical)
			}
		} else {
			scan(v)
		}
	}

	// Track visited nodes to prevent infinite loops
	visited := make(map[uintptr]bool)
	
	scan = func(o interface{}) {
		switch v := o.(type) {
		case map[interface{}]interface{}:
			// Check if we've seen this map before (circular reference)
			ptr := reflect.ValueOf(v).Pointer()
			if visited[ptr] {
				return // Skip already visited nodes
			}
			visited[ptr] = true
			
			// Sort keys for deterministic iteration order (for reproducible tests)
			keys := make([]string, 0, len(v))
			keyToInterface := make(map[string]interface{})
			for k := range v {
				keyStr := fmt.Sprintf("%v", k)
				keys = append(keys, keyStr)
				keyToInterface[keyStr] = k
			}
			sort.Strings(keys)
			
			for _, keyStr := range keys {
				originalKey := keyToInterface[keyStr]
				val := v[originalKey]
				ev.Here.Push(keyStr)
				check(val)
				ev.Here.Pop()
			}
			
			delete(visited, ptr) // Clean up after visiting

		case map[string]interface{}:
			// Check if we've seen this map before (circular reference)
			ptr := reflect.ValueOf(v).Pointer()
			if visited[ptr] {
				return // Skip already visited nodes
			}
			visited[ptr] = true
			
			// Sort keys for deterministic iteration order (for reproducible tests)
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			
			for _, k := range keys {
				val := v[k]
				ev.Here.Push(k)
				check(val)
				ev.Here.Pop()
			}
			
			delete(visited, ptr) // Clean up after visiting

		case []interface{}:
			// Check if we've seen this slice before (circular reference)
			ptr := reflect.ValueOf(v).Pointer()
			if visited[ptr] {
				return // Skip already visited nodes
			}
			visited[ptr] = true
			
			for i, val := range v {
				name := nameOfObj(val, fmt.Sprintf("%d", i))
				op, _ := ParseOpcallCompat(phase, name)
				if op == nil {
					ev.Here.Push(name)
				} else {
					ev.Here.Push(fmt.Sprintf("%d", i))
				}
				check(val)
				ev.Here.Pop()
			}
			
			delete(visited, ptr) // Clean up after visiting
		}
	}

	scan(ev.Tree)

	// Filter operators if cherry-pick paths are specified
	// Apply selective evaluation if cherry-pick paths are specified
	// This filters operators to only those under cherry-picked paths and their dependencies
	if len(ev.CherryPickPaths) > 0 {
		log.DEBUG("DataFlow: Filtering operators for cherry-pick paths: %v", ev.CherryPickPaths)
		all = ev.filterOperatorsForCherryPick(all)
	}

	// construct the data flow graph, where a -> b means 'b' calls or requires 'a'
	// represent the graph as list of adjancies, where [a,b] = a -> b
	// []{ []*Opcall{ grabStaticValue, grabTheThingThatGrabsTheStaticValue}}
	var g [][]*Opcall
	for _, a := range all {
		for _, path := range a.Dependencies(ev, locs) {
			// First try the path as-is
			if b, found := all[path.String()]; found {
				g = append(g, []*Opcall{b, a})
			} else {
				// If not found, try to resolve to canonical path
				if canon, err := path.Canonical(ev.Tree); err == nil {
					if b, found := all[canon.String()]; found {
						g = append(g, []*Opcall{b, a})
					}
				} else {
					// If still not found, check parent paths for operators
					// This handles cases like meta.third.0 depending on meta.third
					parent := path.Copy()
					for len(parent.Nodes) > 0 {
						parent.Pop()
						if len(parent.Nodes) == 0 {
							break
						}
						
						// Try parent path as-is
						if b, found := all[parent.String()]; found {
							g = append(g, []*Opcall{b, a})
							break
						}
						
						// Try canonical parent path
						if canon, err := parent.Canonical(ev.Tree); err == nil {
							if b, found := all[canon.String()]; found {
								g = append(g, []*Opcall{b, a})
								break
							}
						}
					}
				}
			}
		}
	}

	if len(ev.Only) > 0 {
		/*
			[],
			[
			  { name:(( concat env "-" type )), list.1:(( grab name )) }
			  { name:(( concat env "-" type )), list.2:(( grab name )) }
			  { name:(( concat env "-" type )), list.3:(( grab name )) }
			  { name:(( concat env "-" type )), list.4:(( grab name )) }
			  { name:(( concat env "-" type )), params.bosh_username:(( grab name )) }
			  { type:(( grab meta.type )), name:(( concat env "-" type )) }
			  { name:(( concat env "-" type )), list.0:(( grab name )) }
			]

			pass 1:
			[
			  # add this one, because it is under `params`
			  { name:(( concat env "-" type )), params.bosh_username:(( grab name )) }
			], [
			  { name:(( concat env "-" type )), list.1:(( grab name )) }
			  { name:(( concat env "-" type )), list.2:(( grab name )) }
			  { name:(( concat env "-" type )), list.3:(( grab name )) }
			  { name:(( concat env "-" type )), list.4:(( grab name )) }
			  { type:(( grab meta.type )), name:(( concat env "-" type )) }
			  { name:(( concat env "-" type )), list.0:(( grab name )) }
			]

			pass2:
			[
			  { name:(( concat env "-" type )), params.bosh_username:(( grab name )) }

			  # add this one, because in[1] is a out[0] of a previously partitioned element.
			  { type:(( grab meta.type )), name:(( concat env "-" type )) }
			], [
			  { name:(( concat env "-" type )), list.1:(( grab name )) }
			  { name:(( concat env "-" type )), list.2:(( grab name )) }
			  { name:(( concat env "-" type )), list.3:(( grab name )) }
			  { name:(( concat env "-" type )), list.4:(( grab name )) }
			  { name:(( concat env "-" type )), list.0:(( grab name )) }
			]

			pass3:
			[
			  { name:(( concat env "-" type )), params.bosh_username:(( grab name )) }
			  { type:(( grab meta.type )), name:(( concat env "-" type )) }

			  # add nothing, because there is no [1] in the second list that is also a [0]
			  # in the first list.  partitioning is complete, and we use just the first list.
			], [
			  { name:(( concat env "-" type )), list.1:(( grab name )) }
			  { name:(( concat env "-" type )), list.2:(( grab name )) }
			  { name:(( concat env "-" type )), list.3:(( grab name )) }
			  { name:(( concat env "-" type )), list.4:(( grab name )) }
			  { name:(( concat env "-" type )), list.0:(( grab name )) }
			]
		*/

		// filter `in`, migrating elements to `out` if they are
		// dependencies of anything already in `out`.
		filter := func(out, in *[][]*Opcall) int {
			l := make([][]*Opcall, 0)

			for i, candidate := range *in {
				if candidate == nil {
					continue
				}
				for _, op := range *out {
					if candidate[1] == op[0] {
						log.TRACE("data flow - adding [%s: %s, %s: %s] to data flow set (it matched {%s})",
							candidate[0].canonical, candidate[0].src,
							candidate[1].canonical, candidate[1].src,
							op[0].canonical)
						l = append(l, candidate)
						(*in)[i] = nil
						break
					}
				}
			}

			*out = append(*out, l...)
			return len(l)
		}

		// return a subset of `ops` that is strictly related to
		// the processing of the top-levels listed in `picks`
		firsts := func(ops [][]*Opcall, picks []*tree.Cursor) [][]*Opcall {
			final := make([][]*Opcall, 0)
			for i, op := range ops {
				// check to see if this op.src is underneath
				// any of the paths in `picks` -- if so, we
				// want that opcall adjacency in `final`

				for _, pick := range picks {
					if pick.Contains(op[1].canonical) {
						final = append(final, op)
						ops[i] = nil
						log.TRACE("data flow - adding [%s: %s, %s: %s] to data flow set (it matched --cherry-pick %s)",
							op[0].canonical, op[0].src,
							op[1].canonical, op[1].src,
							pick)
						break
					}
				}
			}

			for filter(&final, &ops) > 0 {
			}

			return final
		}

		picks := make([]*tree.Cursor, len(ev.Only))
		for i, s := range ev.Only {
			c, err := tree.ParseCursor(s)
			if err != nil {
				return nil, ansi.Errorf("@*{invalid --cherry-pick path '%s': %s}", s, err)
			}
			picks[i] = c
		}
		g = firsts(g, picks)

		// repackage `all`, since follow-on logic needs it
		newAll := map[string]*Opcall{}
		// findall ops underneath cherry-picked paths
		for path, op := range all {
			for _, pickedPath := range ev.Only {
				cursor, err := tree.ParseCursor(pickedPath)
				if err != nil {
					return nil, ansi.Errorf("@*{invalid --cherry-pick path '%s': %s}", pickedPath, err)
				}
				if cursor.Contains(op.canonical) {
					newAll[path] = op
				}
			}
		}
		all = newAll
		// add in any dependencies of things cherry-picked
		for _, ops := range g {
			if _, exists := all[ops[0].canonical.String()]; !exists {
				insertionOrder = append(insertionOrder, ops[0].canonical.String())
			}
			all[ops[0].canonical.String()] = ops[0]
			if _, exists := all[ops[1].canonical.String()]; !exists {
				insertionOrder = append(insertionOrder, ops[1].canonical.String())
			}
			all[ops[1].canonical.String()] = ops[1]
		}
	}

	for i, node := range g {
		log.TRACE("data flow -- g[%d] is { %s:%s, %s:%s }\n", i, node[0].where, node[0].src, node[1].where, node[1].src)
	}

	// construct a list of keys in $all
	// Order depends on DataflowOrder setting
	var sortedKeys []string
	if ev.DataflowOrder == "insertion" {
		// Use the tracked insertion order
		sortedKeys = insertionOrder
	} else {
		// Default to alphabetical order for deterministic output
		for k := range all {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)
	}

	// find all nodes in g that are free (no further dependencies)
	freeNodes := func(g [][]*Opcall) []*Opcall {
		// First, collect all free nodes (those with no incoming dependencies)
		freeNodeMap := make(map[string]*Opcall)
		for k, node := range all {
			called := false
			for _, pair := range g {
				if pair[1] == node {
					called = true
					break
				}
			}
			
			if !called {
				freeNodeMap[k] = node
			}
		}
		
		// Then return them in a deterministic order based on sortedKeys
		// but only include nodes that are actually free
		l := []*Opcall{}
		for _, k := range sortedKeys {
			if node, isFree := freeNodeMap[k]; isFree {
				delete(all, k)
				l = append(l, node)
			}
		}

		return l
	}

	// removes (nullifies) all dependencies on n in g
	remove := func(old [][]*Opcall, n *Opcall) [][]*Opcall {
		l := [][]*Opcall{}
		for _, pair := range old {
			if pair[0] != n {
				l = append(l, pair)
			}
		}
		return l
	}

	// Kahn topological sort
	ops := []*Opcall{} // order in which to call the ops
	wave := 0
	for len(all) > 0 {
		wave++
		free := freeNodes(g)
		if len(free) == 0 {
			return nil, ansi.Errorf("@*{cycle detected in operator data-flow graph}")
		}

		for _, node := range free {
			log.TRACE("data flow: [%d] wave %d, op %s: %s", len(ops), wave, node.where, node.src)
			ops = append(ops, node)
			g = remove(g, node)
		}
	}

	if len(errors.Errors) > 0 {
		return nil, errors
	}
	return ops, nil
}

// RunOps ...
func (ev *Evaluator) RunOps(ops []*Opcall) error {
	log.DEBUG("patching up YAML by evaluating outstanding operators\n")

	errors := MultiError{Errors: []error{}}
	for _, op := range ops {
		err := ev.RunOp(op)
		if err != nil {
			errors.Append(err)
		}
	}

	if len(errors.Errors) > 0 {
		return errors
	}
	return nil
}

// Prune ...
func (ev *Evaluator) Prune(paths []string) error {
	log.DEBUG("pruning %d paths from the final YAML structure", len(paths))
	for _, path := range paths {
		c, err := tree.ParseCursor(path)
		if err != nil {
			return err
		}

		key := c.Component(-1)
		parent := c.Copy()
		parent.Pop()
		o, err := parent.Resolve(ev.Tree)
		if err != nil {
			continue
		}

		switch o.(type) {
		case map[interface{}]interface{}:
			if _, ok := o.(map[interface{}]interface{}); ok {
				log.DEBUG("  pruning %s", path)
				delete(o.(map[interface{}]interface{}), key)
			}

		case []interface{}:
			if list, ok := o.([]interface{}); ok {
				if idx, err := strconv.Atoi(key); err == nil {
					parent.Pop()
					if s, err := parent.Resolve(ev.Tree); err == nil {
						if reflect.TypeOf(s).Kind() == reflect.Map {
							parentName := fmt.Sprintf("%s", c.Component(-2))
							log.DEBUG("  pruning index %d of array '%s'", idx, parentName)

							length := len(list) - 1
							replacement := make([]interface{}, length)
							copy(replacement, append(list[:idx], list[idx+1:]...))

							delete(s.(map[interface{}]interface{}), parentName)
							s.(map[interface{}]interface{})[parentName] = replacement
						}
					}
				}
			}

		default:
			log.DEBUG("  I don't know how to prune %s\n    value=%v\n", path, o)
		}
	}
	log.DEBUG("")
	return nil
}

// SortPaths sorts all paths (keys in map) using the provided sort-key (respective value)
func (ev *Evaluator) SortPaths(pathKeyMap map[string]string) error {
	log.DEBUG("sorting %d paths in the final YAML structure", len(pathKeyMap))
	for path, sortBy := range pathKeyMap {
		log.DEBUG("  sorting path %s (sort-key %s)", path, sortBy)

		cursor, err := tree.ParseCursor(path)
		if err != nil {
			return err
		}

		value, err := cursor.Resolve(ev.Tree)
		if err != nil {
			return err
		}

		switch value.(type) {
		case []interface{}:
			// no-op, that's what we want ...

		case map[interface{}]interface{}:
			return tree.TypeMismatchError{
				Path:   []string{path},
				Wanted: "a list",
				Got:    "a map",
			}

		default:
			return tree.TypeMismatchError{
				Path:   []string{path},
				Wanted: "a list",
				Got:    "a scalar",
			}
		}

		if err := SortList(path, value.([]interface{}), sortBy); err != nil {
			return err
		}
	}

	log.DEBUG("")
	return nil
}

// Cherry-pick ...
func (ev *Evaluator) CherryPick(paths []string) error {
	log.DEBUG("cherry-picking %d paths from the final YAML structure", len(paths))

	if len(paths) > 0 {
		// This will serve as the replacement tree ...
		replacement := make(map[interface{}]interface{})

		for _, path := range paths {
			cursor, err := tree.ParseCursor(path)
			if err != nil {
				return err
			}

			// These variables will potentially be modified (depending on the structure)
			var cherryName string
			var cherryValue interface{}

			// Resolve the value that needs to be cherry picked
			cherryValue, err = cursor.Resolve(ev.Tree)
			if err != nil {
				return err
			}

			// Name of the parameter of the to-be-picked value
			cherryName = cursor.Nodes[len(cursor.Nodes)-1]

			// Since the cherry can be deep down the structure, we need to go down
			// (or up, depending how you read it) the structure to include the parent
			// names of the respective cherry. The pointer will be reassigned with
			// each level.
			pointer := cursor
			for pointer != nil {
				parent := pointer.Copy()
				parent.Pop()

				if parent.String() == "" {
					// Empty parent string means we reached the root, setting the pointer nil to stop processing ...
					pointer = nil

					// ... create the final cherry wrapped in its container ...
					tmp := make(map[interface{}]interface{})
					tmp[cherryName] = cherryValue

					// ... and add it to the replacement map
					log.DEBUG("Merging '%s' into the replacement tree", path)
					merger := &merger.Merger{AppendByDefault: true}
					merged := merger.MergeObj(tmp, replacement, path)
					if err := merger.Error(); err != nil {
						return err
					}

					replacement = merged.(map[interface{}]interface{})

				} else {
					// Reassign the pointer to the parent and restructre the current cherry value to address the parent structure and name
					pointer = parent

					// Depending on the type of the parent, either a map or a list is created for the new parent of the cherry value
					if obj, err := parent.Resolve(ev.Tree); err == nil {
						switch obj.(type) {
						case map[interface{}]interface{}:
							tmp := make(map[interface{}]interface{})
							tmp[cherryName] = cherryValue

							cherryName = parent.Nodes[len(parent.Nodes)-1]
							cherryValue = tmp

						case []interface{}:
							tmp := make([]interface{}, 0, 0)
							tmp = append(tmp, cherryValue)

							cherryName = parent.Nodes[len(parent.Nodes)-1]
							cherryValue = tmp

						default:
							return ansi.Errorf("@*{Unsupported type detected, %s is neither a map nor a list}", parent.String())
						}

					} else {
						return err
					}
				}
			}
		}

		// replace the existing tree with a new one that contain the cherry-picks
		ev.Tree = replacement
	}

	log.DEBUG("")
	return nil
}

// CheckForCycles ...
func (ev *Evaluator) CheckForCycles(maxDepth int) error {
	log.DEBUG("checking for cycles in final YAML structure")

	var check func(o interface{}, depth int) error
	check = func(o interface{}, depth int) error {
		if depth == 0 {
			return ansi.Errorf("@*{Hit max recursion depth. You seem to have a self-referencing dataset}")
		}

		switch o.(type) {
		case []interface{}:
			for _, v := range o.([]interface{}) {
				if err := check(v, depth-1); err != nil {
					return err
				}
			}

		case map[interface{}]interface{}:
			for _, v := range o.(map[interface{}]interface{}) {
				if err := check(v, depth-1); err != nil {
					return err
				}
			}
		}

		return nil
	}

	err := check(ev.Tree, maxDepth)
	if err != nil {
		log.DEBUG("error: %s\n", err)
		return err
	}

	log.DEBUG("no cycles detected.\n")
	return nil
}

// RunOp ...
func (ev *Evaluator) RunOp(op *Opcall) error {

	resp, err := op.Run(ev)
	if err != nil {
		return err
	}

	switch resp.Type {
	case Replace:
		log.DEBUG("executing a Replace instruction on %s", op.where)
		key := op.where.Component(-1)
		parent := op.where.Copy()
		parent.Pop()

		o, err := parent.Resolve(ev.Tree)
		if err != nil {
			log.DEBUG("  error: %s\n  continuing\n", err)
			return err
		}
		switch o.(type) {
		case []interface{}:
			i, err := strconv.ParseUint(key, 10, 0)
			if err != nil {
				log.DEBUG("  error: %s\n  continuing\n", err)
				return err
			}
			o.([]interface{})[i] = resp.Value

		case map[interface{}]interface{}:
			o.(map[interface{}]interface{})[key] = resp.Value

		case map[string]interface{}:
			o.(map[string]interface{})[key] = resp.Value

		default:
			err := tree.TypeMismatchError{
				Path:   parent.Nodes,
				Wanted: "a map or a list",
				Got:    "a scalar",
			}
			log.DEBUG("  error: %s\n  continuing\n", err)
			return err
		}
		log.DEBUG("")

	case Inject:
		log.DEBUG("executing an Inject instruction on %s", op.where)
		key := op.where.Component(-1)
		parent := op.where.Copy()
		parent.Pop()

		o, err := parent.Resolve(ev.Tree)
		if err != nil {
			log.DEBUG("  error: %s\n  continuing\n", err)
			return err
		}

		m := o.(map[interface{}]interface{})
		delete(m, key)

		for k, v := range resp.Value.(map[interface{}]interface{}) {
			path := fmt.Sprintf("%s.%s", parent, k)
			log.DEBUG("Inject: parent=%s, k=%s, path=%s", parent, k, path)
			_, set := m[k]
			if !set {
				log.DEBUG("  %s is not set, using the injected value", path)
				m[k] = v
			} else {
				// Check type of existing and injected values for proper merging
				merger := &merger.Merger{AppendByDefault: true}
				merged := merger.MergeObj(v, m[k], path)
				if err := merger.Error(); err != nil {
					return err
				}
				m[k] = merged
			}
		}
	}
	return nil
}

// RunPhase ...
func (ev *Evaluator) RunPhase(p OperatorPhase) error {
	err := SetupOperators(p)
	if err != nil {
		return err
	}

	op, err := ev.DataFlow(p)
	if err != nil {
		return err
	}

	return ev.RunOps(op)
}

// Run ...
func (ev *Evaluator) Run(prune []string, picks []string) error {
	errors := MultiError{Errors: []error{}}
	paramErrs := MultiError{Errors: []error{}}

	if os.Getenv("REDACT") != "" {
		log.DEBUG("Setting vault & aws operators to redact keys")
		SkipVault = true
		SkipAws = true
	}

	if !ev.SkipEval {
		ev.Only = picks
		errors.Append(ev.RunPhase(MergePhase))
		paramErrs.Append(ev.RunPhase(ParamPhase))
		if len(paramErrs.Errors) > 0 {
			return paramErrs
		}

		errors.Append(ev.RunPhase(EvalPhase))
	}

	// this is a big failure...
	if err := ev.CheckForCycles(4096); err != nil {
		return err
	}

	// post-processing: prune
	addToPruneListIfNecessary(prune...)
	log.DEBUG("Final prune list contains %d paths: %v", len(keysToPrune), keysToPrune)
	errors.Append(ev.Prune(keysToPrune))
	keysToPrune = nil

	// post-processing: sorting
	errors.Append(ev.SortPaths(pathsToSort))
	pathsToSort = map[string]string{}

	// post-processing: cherry-pick
	errors.Append(ev.CherryPick(picks))

	if len(errors.Errors) > 0 {
		return errors
	}
	return nil
}

// isUnderPath checks if an operator path is under a cherry-pick path.
// This is the core of selective evaluation - it determines whether an operator
// should be evaluated based on cherry-pick paths.
//
// The function handles special cases like:
// - Named array entries (e.g., "jobs.web" matching "jobs.0")
// - Exact path matches
// - Nested paths (e.g., "a.b.c" is under "a.b")
func (ev *Evaluator) isUnderPath(opPath, cherryPath string) bool {
	// Handle empty paths
	if opPath == "" || cherryPath == "" {
		return false
	}
	
	// Parse both paths into cursors
	opCursor, err := tree.ParseCursor(opPath)
	if err != nil {
		return false
	}
	cherryCursor, err := tree.ParseCursor(cherryPath)
	if err != nil {
		return false
	}
	
	// Check if either cursor has no nodes
	if len(opCursor.Nodes) == 0 || len(cherryCursor.Nodes) == 0 {
		return false
	}
	
	// Check if opCursor starts with cherryCursor
	if len(opCursor.Nodes) < len(cherryCursor.Nodes) {
		return false
	}
	
	// Compare each segment with context
	currentPath := &tree.Cursor{}
	for i, cherryNode := range cherryCursor.Nodes {
		if !ev.segmentsMatchWithContext(opCursor.Nodes[i], cherryNode, currentPath) {
			return false
		}
		// Build up the current path as we go
		currentPath.Push(opCursor.Nodes[i])
	}
	
	return true
}

// segmentsMatchWithContext compares two path segments with access to the data structure
func (ev *Evaluator) segmentsMatchWithContext(opSegment, cherrySegment string, currentPath *tree.Cursor) bool {
	// Direct string comparison first
	if opSegment == cherrySegment {
		return true
	}
	
	// Check if both are numeric indices
	opIdx, opErr := strconv.Atoi(opSegment)
	cherryIdx, cherryErr := strconv.Atoi(cherrySegment)
	
	// Both are numeric - they should match exactly
	if opErr == nil && cherryErr == nil {
		return opIdx == cherryIdx
	}
	
	// One is numeric and one is not - check if they refer to the same array element
	if (opErr == nil) != (cherryErr == nil) {
		// Try to resolve the current path to get the actual array
		if len(currentPath.Nodes) > 0 {
			obj, err := currentPath.Resolve(ev.Tree)
			if err == nil {
				switch arr := obj.(type) {
				case []interface{}:
					// If we have a numeric index and a name, check if they match
					if opErr == nil {
						// opSegment is numeric, cherrySegment is a name
						if opIdx >= 0 && opIdx < len(arr) {
							// Check if the element at this index has the expected name
							if elem, ok := arr[opIdx].(map[interface{}]interface{}); ok {
								// Check common name fields
								for _, nameField := range tree.NameFields {
									if name, exists := elem[nameField]; exists {
										if nameStr, ok := name.(string); ok && nameStr == cherrySegment {
											return true
										}
									}
								}
							}
						}
					} else {
						// cherrySegment is numeric, opSegment is a name
						if cherryIdx >= 0 && cherryIdx < len(arr) {
							// Check if the element at the cherry index has the op name
							if elem, ok := arr[cherryIdx].(map[interface{}]interface{}); ok {
								for _, nameField := range tree.NameFields {
									if name, exists := elem[nameField]; exists {
										if nameStr, ok := name.(string); ok && nameStr == opSegment {
											return true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	return false
}

// segmentsMatch compares two path segments without context (for backward compatibility)
func (ev *Evaluator) segmentsMatch(opSegment, cherrySegment string) bool {
	// Direct string comparison
	return opSegment == cherrySegment
}

// filterOperatorsForCherryPick filters operators to only those needed for cherry-picked paths.
// This implements the selective evaluation strategy:
// 
// 1. First identifies all operators under cherry-picked paths
// 2. Then recursively collects all their dependencies (transitive closure)
// 3. Returns only the operators that are needed for evaluation
//
// This significantly reduces the number of operators evaluated in large documents,
// improving performance when only specific sections are needed.
func (ev *Evaluator) filterOperatorsForCherryPick(all map[string]*Opcall) map[string]*Opcall {
	if len(ev.CherryPickPaths) == 0 {
		return all // No filtering needed
	}
	
	needed := make(map[string]bool)
	result := make(map[string]*Opcall)
	
	log.DEBUG("filterOperatorsForCherryPick: Filtering operators for cherry-pick paths: %v", ev.CherryPickPaths)
	
	// Step 1: Mark operators under cherry-picked paths
	for path := range all {
		for _, cherryPath := range ev.CherryPickPaths {
			if ev.isUnderPath(path, cherryPath) {
				needed[path] = true
				log.DEBUG("filterOperatorsForCherryPick: Operator at %s is under cherry-pick path %s", path, cherryPath)
				break
			}
		}
	}
	
	// If no operators were found under cherry-pick paths, include all operators
	// This handles cases where cherry-pick paths don't contain operators directly
	if len(needed) == 0 {
		log.DEBUG("filterOperatorsForCherryPick: No operators found under cherry-pick paths, including all")
		return all
	}
	
	// Step 2: Collect transitive dependencies - but only check operators in the dependency list
	// We need to look at what the needed operators depend on, not what depends on them
	changed := true
	iterations := 0
	maxIterations := 100 // Prevent infinite loops
	
	for changed && iterations < maxIterations {
		changed = false
		iterations++
		
		// Create a snapshot of currently needed paths to iterate over
		currentNeeded := make([]string, 0, len(needed))
		for path := range needed {
			currentNeeded = append(currentNeeded, path)
		}
		
		// For each needed operator, add its dependencies
		for _, path := range currentNeeded {
			if op, exists := all[path]; exists {
				// Get dependencies for this operator
				deps := op.Dependencies(ev, nil)
				for _, dep := range deps {
					// Try to resolve the dependency to a canonical path
					depPath := dep.String()
					
					// Check if this dependency corresponds to an operator
					if _, isOp := all[depPath]; isOp && !needed[depPath] {
						needed[depPath] = true
						changed = true
						log.DEBUG("filterOperatorsForCherryPick: Added operator dependency %s for operator at %s", depPath, path)
					}
				}
			}
		}
	}
	
	if iterations >= maxIterations {
		log.DEBUG("filterOperatorsForCherryPick: Warning - reached maximum iterations while collecting dependencies")
	}
	
	// Step 3: Build filtered result
	for path, op := range all {
		if needed[path] {
			result[path] = op
		}
	}
	
	log.DEBUG("filterOperatorsForCherryPick: Filtered from %d to %d operators", len(all), len(result))
	
	return result
}
