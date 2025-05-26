package spruce

// ExtendedExpr adds new fields to support operator calls
// This extends the functionality of the base Expr struct
// without modifying the original definition

// Op field for operator name in OperatorCall expressions
func (e *Expr) Op() string {
	if e.Type == OperatorCall {
		return e.Name
	}
	return ""
}

// SetOp sets the operator name for OperatorCall expressions
func (e *Expr) SetOp(op string) {
	if e.Type == OperatorCall {
		// Intern operator names
		e.Name = InternString(op)
	}
}

// Args returns the arguments for an OperatorCall expression
// For now, we'll use Left and Right to store arguments in a linked list
func (e *Expr) Args() []*Expr {
	if e.Type != OperatorCall {
		return nil
	}
	
	// If we have arguments stored in Left as a special ArgsList expression
	if e.Left != nil && e.Left.Type == ArgsList {
		return e.Left.extractArgs()
	}
	
	return nil
}

// SetArgs sets the arguments for an OperatorCall expression
func (e *Expr) SetArgs(args []*Expr) {
	if e.Type == OperatorCall {
		e.Left = &Expr{
			Type: ArgsList,
		}
		e.Left.storeArgs(args)
	}
}

// ArgsList is a special expression type used internally to store operator arguments
const ArgsList ExprType = 999

// extractArgs extracts arguments from an ArgsList expression
func (e *Expr) extractArgs() []*Expr {
	if e.Type != ArgsList {
		return nil
	}
	
	var args []*Expr
	current := e
	for current != nil {
		if current.Left != nil {
			args = append(args, current.Left)
		}
		current = current.Right
	}
	return args
}

// storeArgs stores arguments in an ArgsList expression using a linked list
func (e *Expr) storeArgs(args []*Expr) {
	if e.Type != ArgsList || len(args) == 0 {
		return
	}
	
	e.Left = args[0]
	current := e
	
	for i := 1; i < len(args); i++ {
		current.Right = &Expr{
			Type: ArgsList,
			Left: args[i],
		}
		current = current.Right
	}
}

// Helper to create an OperatorCall expression
func NewOperatorCall(op string, args []*Expr) *Expr {
	expr := &Expr{
		Type: OperatorCall,
		Name: InternString(op), // Intern operator names
	}
	expr.SetArgs(args)
	return expr
}

// NewOperatorCallWithPos creates an OperatorCall expression with position info
func NewOperatorCallWithPos(op string, args []*Expr, pos Position) *Expr {
	expr := &Expr{
		Type: OperatorCall,
		Name: InternString(op), // Intern operator names
		Pos:  pos,
	}
	expr.SetArgs(args)
	return expr
}

// Helper to get operator call fields for compatibility with parser
func (e *Expr) GetOperatorCallFields() (op string, args []*Expr) {
	if e.Type == OperatorCall {
		return e.Op(), e.Args()
	}
	return "", nil
}