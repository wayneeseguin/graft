package operators


// BooleanAndOperator implements logical AND (&&)
type BooleanAndOperator struct{}

// Setup initializes the operator
func (BooleanAndOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (BooleanAndOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns operator dependencies
func (BooleanAndOperator) Dependencies(_ *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the logical AND
func (BooleanAndOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("&& operator requires exactly 2 arguments, got %d", len(args))
	}

	// Short-circuit evaluation: evaluate left first
	leftResp, err := EvaluateExpr(args[0], ev)
	if err != nil {
		return nil, err
	}

	if !isTruthy(leftResp.Value) {
		// Left is falsy, return false without evaluating right
		return &Response{
			Type:  Replace,
			Value: false,
		}, nil
	}

	// Left is truthy, evaluate right
	rightResp, err := EvaluateExpr(args[1], ev)
	if err != nil {
		return nil, err
	}

	return &Response{
		Type:  Replace,
		Value: isTruthy(rightResp.Value),
	}, nil
}

// BooleanOrOperator implements logical OR (||) - different from the fallback || operator
type BooleanOrOperator struct{}

// Setup initializes the operator
func (BooleanOrOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (BooleanOrOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns operator dependencies
func (BooleanOrOperator) Dependencies(_ *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the logical OR
func (BooleanOrOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("|| operator requires exactly 2 arguments, got %d", len(args))
	}

	// Short-circuit evaluation: evaluate left first
	leftResp, err := EvaluateExpr(args[0], ev)
	if err != nil {
		return nil, err
	}

	if isTruthy(leftResp.Value) {
		// Left is truthy, return true without evaluating right
		return &Response{
			Type:  Replace,
			Value: true,
		}, nil
	}

	// Left is falsy, evaluate right
	rightResp, err := EvaluateExpr(args[1], ev)
	if err != nil {
		return nil, err
	}

	return &Response{
		Type:  Replace,
		Value: isTruthy(rightResp.Value),
	}, nil
}

// isTruthy determines if a value is truthy
// false, nil, 0, "", [], {} are falsy
// Everything else is truthy
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}

	// Check for boolean
	if b, ok := v.(bool); ok {
		return b
	}

	// Check for numeric zero
	if num, ok := toFloat64(v); ok {
		return num != 0
	}

	// Check for empty string
	if s, ok := v.(string); ok {
		return s != ""
	}

	// Check for empty slice/array
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len() > 0
	case reflect.Map:
		return rv.Len() > 0
	}

	// Everything else is truthy
	return true
}

// Register boolean operators
func init() {
	RegisterOp("&&", BooleanAndOperator{})
	RegisterOp("||", BooleanOrOperator{})
}
