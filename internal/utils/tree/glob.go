package tree

import (
	"fmt"
	"strconv"
)

// Glob performs glob pattern matching on the cursor path
func (c *Cursor) Glob(tree interface{}) ([]*Cursor, error) {
	var resolver func(interface{}, []string, []string, int) ([]interface{}, error)
	resolver = func(o interface{}, here, path []string, pos int) ([]interface{}, error) {
		if pos == len(path) {
			return []interface{}{
				(&Cursor{Nodes: here}).Copy(),
			}, nil
		}

		paths := []interface{}{}
		k := path[pos]
		if k == "*" {
			switch o.(type) {
			case []interface{}:
				for i, v := range o.([]interface{}) {
					sub, err := resolver(v, append(here, fmt.Sprintf("%d", i)), path, pos+1)
					if err != nil {
						if _, ok := err.(NotFoundError); !ok {
							return nil, err
						}
					}
					paths = append(paths, sub...)
				}

			case map[string]interface{}:
				for k, v := range o.(map[string]interface{}) {
					sub, err := resolver(v, append(here, k), path, pos+1)
					if err != nil {
						if _, ok := err.(NotFoundError); !ok {
							return nil, err
						}
					}
					paths = append(paths, sub...)
				}

			case map[interface{}]interface{}:
				for k, v := range o.(map[interface{}]interface{}) {
					sub, err := resolver(v, append(here, fmt.Sprintf("%v", k)), path, pos+1)
					if err != nil {
						if _, ok := err.(NotFoundError); !ok {
							return nil, err
						}
					}
					paths = append(paths, sub...)
				}

			default:
				return nil, TypeMismatchError{
					Path:   path,
					Wanted: "a map or a list",
					Got:    "a scalar",
				}
			}

		} else {
			switch o.(type) {
			case []interface{}:
				i, err := strconv.ParseUint(k, 10, 0)
				if err == nil {
					// if k is an integer (in string form), go by index
					if int(i) >= len(o.([]interface{})) {
						return nil, NotFoundError{
							Path: path[0 : pos+1],
						}
					}
					return resolver(o.([]interface{})[i], append(here, k), path, pos+1)
				}

				// if k is a string, look for immediate map descendants who have
				//     'name', 'key' or 'id' fields matching k
				var found bool
				o, _, found = listFind(o.([]interface{}), NameFields, k)
				if !found {
					return nil, NotFoundError{
						Path: path[0 : pos+1],
					}
				}
				return resolver(o, append(here, k), path, pos+1)

			case map[string]interface{}:
				v, ok := o.(map[string]interface{})[k]
				if !ok {
					return nil, NotFoundError{
						Path: path[0 : pos+1],
					}
				}
				return resolver(v, append(here, k), path, pos+1)

			case map[interface{}]interface{}:
				v, ok := o.(map[interface{}]interface{})[k]
				if !ok {
					/* key might not actually be a string.  let's iterate */
					k2 := fmt.Sprintf("%v", k)
					for k1, v1 := range o.(map[interface{}]interface{}) {
						if fmt.Sprintf("%v", k1) == k2 {
							v, ok = v1, true
							break
						}
					}
					if !ok {
						return nil, NotFoundError{
							Path: path[0 : pos+1],
						}
					}
				}
				return resolver(v, append(here, k), path, pos+1)

			default:
				return nil, TypeMismatchError{
					Path:   path[0:pos],
					Wanted: "a map or a list",
					Got:    "a scalar",
				}
			}
		}

		return paths, nil
	}

	var path []string
	for _, s := range c.Nodes {
		path = append(path, s)
	}

	l, err := resolver(tree, []string{}, path, 0)
	if err != nil {
		return nil, err
	}

	cursors := []*Cursor{}
	for _, c := range l {
		cursors = append(cursors, c.(*Cursor))
	}
	return cursors, nil
}