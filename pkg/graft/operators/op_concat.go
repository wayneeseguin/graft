package operators


// ConcatOperator ...
type ConcatOperator struct{}

// Setup ...
func (ConcatOperator) Setup() error {
	return nil
}

// Phase ...
func (ConcatOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (ConcatOperator) Dependencies(ev *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := auto
	
	for _, arg := range args {
		if arg.Type == OperatorCall {
			// Get dependencies from nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), nil, nil)
				deps = append(deps, nestedDeps...)
			}
		} else if arg.Type == Reference {
			deps = append(deps, arg.Reference)
		}
	}
	
	return deps
}

// Run ...
func (ConcatOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( concat ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( concat ... )) operation at $%s\n", ev.Here)

	l := GetStringSlice()
	defer PutStringSlice(l)

	if len(args) < 2 {
		return nil, fmt.Errorf("concat operator requires at least two arguments")
	}

	for i, arg := range args {
		// Use ResolveOperatorArgument to support nested expressions
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
			DEBUG("     [%d]: error was: %s", i, err)
			// Wrap error to maintain backward compatibility
			if arg.Type == Reference {
				return nil, fmt.Errorf("Unable to resolve `%s`: %s", arg.Reference, err)
			}
			return nil, err
		}

		// Check if it's a non-scalar type
		switch val.(type) {
		case map[interface{}]interface{}, map[string]interface{}:
			DEBUG("  arg[%d]: %v is not a string scalar", i, val)
			return nil, ansi.Errorf("@R{tried to concat} @c{%v}@R{, which is not a string scalar}", val)
		case []interface{}:
			DEBUG("  arg[%d]: %v is not a string scalar", i, val)
			return nil, ansi.Errorf("@R{tried to concat} @c{%v}@R{, which is not a string scalar}", val)
		}

		// Convert to string
		str, err := AsString(val)
		if err != nil {
			DEBUG("  arg[%d]: %v cannot be converted to string", i, val)
			return nil, ansi.Errorf("@R{tried to concat} @c{%v}@R{, which cannot be converted to string}", val)
		}
		
		DEBUG("  arg[%d]: appending '%s' to resultant string", i, str)
		*l = append(*l, str)
	}

	final := strings.Join(*l, "")
	DEBUG("  resolved (( concat ... )) operation to the string:\n    \"%s\"", final)

	return &Response{
		Type:  Replace,
		Value: final,
	}, nil
}

func init() {
	RegisterOp("concat", ConcatOperator{})
}
