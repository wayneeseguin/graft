package operators

// BooleanTypeHandler handles operations for boolean types
type BooleanTypeHandler struct {
	*BaseTypeHandler
}

// NewBooleanTypeHandler creates a new handler for boolean operations
func NewBooleanTypeHandler() *BooleanTypeHandler {
	handler := &BooleanTypeHandler{
		BaseTypeHandler: NewBaseTypeHandler(80), // Medium-high priority
	}
	
	// Support boolean-boolean operations and boolean with other types
	handler.AddSupportedTypes(
		TypePair{A: TypeBool, B: TypeBool},
		TypePair{A: TypeBool, B: TypeInt},
		TypePair{A: TypeBool, B: TypeString},
		TypePair{A: TypeBool, B: TypeNull},
	)
	
	return handler
}

// Add performs logical OR for booleans (non-standard but useful)
func (h *BooleanTypeHandler) Add(a, b interface{}) (interface{}, error) {
	// Boolean OR operation: true + true = true, true + false = true, etc.
	aBool, aOk := toBool(a)
	bBool, bOk := toBool(b)
	
	if aOk && bOk {
		return aBool || bBool, nil
	}
	
	return nil, NotImplementedError("add", a, b)
}

// Subtract is not supported for booleans
func (h *BooleanTypeHandler) Subtract(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("subtract", a, b)
}

// Multiply performs logical AND for booleans (non-standard but useful)
func (h *BooleanTypeHandler) Multiply(a, b interface{}) (interface{}, error) {
	// Boolean AND operation: true * true = true, true * false = false, etc.
	aBool, aOk := toBool(a)
	bBool, bOk := toBool(b)
	
	if aOk && bOk {
		return aBool && bBool, nil
	}
	
	return nil, NotImplementedError("multiply", a, b)
}

// Divide is not supported for booleans
func (h *BooleanTypeHandler) Divide(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("divide", a, b)
}

// Modulo is not supported for booleans
func (h *BooleanTypeHandler) Modulo(a, b interface{}) (interface{}, error) {
	return nil, NotImplementedError("modulo", a, b)
}

// Equal performs boolean equality comparison
func (h *BooleanTypeHandler) Equal(a, b interface{}) (bool, error) {
	aBool, aOk := toBool(a)
	bBool, bOk := toBool(b)
	
	if aOk && bOk {
		return aBool == bBool, nil
	}
	
	// If one is bool and other isn't, check truthiness equality
	if aOk {
		return aBool == isTruthy(b), nil
	}
	if bOk {
		return isTruthy(a) == bBool, nil
	}
	
	return false, NotImplementedError("equal", a, b)
}

// NotEqual performs boolean inequality comparison
func (h *BooleanTypeHandler) NotEqual(a, b interface{}) (bool, error) {
	equal, err := h.Equal(a, b)
	return !equal, err
}

// Less treats false < true
func (h *BooleanTypeHandler) Less(a, b interface{}) (bool, error) {
	aBool, aOk := toBool(a)
	bBool, bOk := toBool(b)
	
	if aOk && bOk {
		// false < true, true < false is false
		return !aBool && bBool, nil
	}
	
	return false, NotImplementedError("less", a, b)
}

// Greater treats true > false
func (h *BooleanTypeHandler) Greater(a, b interface{}) (bool, error) {
	aBool, aOk := toBool(a)
	bBool, bOk := toBool(b)
	
	if aOk && bOk {
		// true > false, false > true is false
		return aBool && !bBool, nil
	}
	
	return false, NotImplementedError("greater", a, b)
}

// LessOrEqual performs boolean comparison
func (h *BooleanTypeHandler) LessOrEqual(a, b interface{}) (bool, error) {
	greater, err := h.Greater(a, b)
	return !greater, err
}

// GreaterOrEqual performs boolean comparison
func (h *BooleanTypeHandler) GreaterOrEqual(a, b interface{}) (bool, error) {
	less, err := h.Less(a, b)
	return !less, err
}

// CanHandle checks if this handler can handle the given type combination
func (h *BooleanTypeHandler) CanHandle(aType, bType OperandType) bool {
	// Handle boolean with any type for truthiness operations
	if aType == TypeBool || bType == TypeBool {
		return true
	}
	return h.BaseTypeHandler.CanHandle(aType, bType)
}

// toBool converts a value to boolean if possible
func toBool(val interface{}) (bool, bool) {
	if b, ok := val.(bool); ok {
		return b, true
	}
	return false, false
}

// LogicalOperations provides specialized boolean operations
type LogicalOperations interface {
	And(a, b interface{}) (bool, error)
	Or(a, b interface{}) (bool, error)
	Not(a interface{}) (bool, error)
	Xor(a, b interface{}) (bool, error)
}

// BooleanLogicalOps implements logical operations for boolean types
type BooleanLogicalOps struct {
	handler *BooleanTypeHandler
}

// NewBooleanLogicalOps creates a new logical operations handler
func NewBooleanLogicalOps(handler *BooleanTypeHandler) *BooleanLogicalOps {
	return &BooleanLogicalOps{handler: handler}
}

// And performs logical AND operation
func (ops *BooleanLogicalOps) And(a, b interface{}) (bool, error) {
	// Use truthiness for all types
	return isTruthy(a) && isTruthy(b), nil
}

// Or performs logical OR operation
func (ops *BooleanLogicalOps) Or(a, b interface{}) (bool, error) {
	// Use truthiness for all types
	return isTruthy(a) || isTruthy(b), nil
}

// Not performs logical NOT operation
func (ops *BooleanLogicalOps) Not(a interface{}) (bool, error) {
	// Use truthiness for all types
	return !isTruthy(a), nil
}

// Xor performs logical XOR operation
func (ops *BooleanLogicalOps) Xor(a, b interface{}) (bool, error) {
	// Use truthiness for all types
	aTruthy := isTruthy(a)
	bTruthy := isTruthy(b)
	return aTruthy != bTruthy, nil
}