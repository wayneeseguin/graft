# Operator API Documentation

This guide explains how to create new operators for graft and integrate them with the enhanced parser system.

## Overview

graft operators are special functions that transform data during YAML/JSON merging. They are invoked using the `(( operator args ))` syntax and can perform operations ranging from simple value transformations to complex data manipulation.

## The Operator Interface

Every operator must implement the `Operator` interface defined in `operator.go`:

```go
type Operator interface {
    // Run executes the operator with given arguments
    Run(ev *Evaluator, args []*Expr) (*Response, error)
    
    // Phase returns when this operator should be evaluated
    Phase() OperatorPhase
}
```

### OperatorPhase

Operators can run in different phases:

```go
type OperatorPhase int

const (
    MergePhase     OperatorPhase = iota // During initial merge
    EvalPhase                           // During evaluation
    ParamPhase                         // During parameter resolution
)
```

Most operators run in `EvalPhase`. Special operators like `param` run earlier.

## Creating a New Operator

### Step 1: Create the Operator File

Create a new file `op_myoperator.go`:

```go
package graft

import (
    "fmt"
    "github.com/starkandwayne/graft/tree"
)

// MyOperator implements a custom transformation
type MyOperator struct{}

// Run executes the operator logic
func (m MyOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
    // Validate argument count
    if len(args) != 1 {
        return nil, fmt.Errorf("myoperator expects exactly 1 argument")
    }
    
    // Evaluate the argument
    resp, err := EvaluateExpr(args[0], ev)
    if err != nil {
        return nil, err
    }
    
    // Transform the value
    result := transformValue(resp.Value)
    
    // Return the response
    return &Response{
        Type:  Replace,
        Value: result,
    }, nil
}

// Phase returns when this operator runs
func (m MyOperator) Phase() OperatorPhase {
    return EvalPhase
}

func transformValue(v interface{}) interface{} {
    // Your transformation logic here
    return v
}
```

### Step 2: Register the Operator

Add registration in `operator.go` or create an init function:

```go
func init() {
    RegisterOp("myoperator", MyOperator{})
}
```

Or register conditionally for enhanced parser only:

```go
func init() {
    if UseEnhancedParser {
        RegisterOp("myoperator", MyOperator{})
    }
}
```

## Operator Implementation Guidelines

### 1. Argument Handling

Always validate argument counts and types:

```go
func (o MyOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
    // Check argument count
    if len(args) < 1 || len(args) > 2 {
        return nil, fmt.Errorf("myoperator expects 1 or 2 arguments, got %d", len(args))
    }
    
    // Evaluate arguments
    arg1, err := EvaluateExpr(args[0], ev)
    if err != nil {
        return nil, fmt.Errorf("error evaluating first argument: %v", err)
    }
    
    // Type checking
    str, ok := arg1.Value.(string)
    if !ok {
        return nil, fmt.Errorf("myoperator expects string argument, got %T", arg1.Value)
    }
    
    // Process optional arguments
    var opt string
    if len(args) > 1 {
        optResp, err := EvaluateExpr(args[1], ev)
        if err != nil {
            return nil, err
        }
        opt, _ = optResp.Value.(string)
    }
    
    // Continue with operation...
}
```

### 2. Expression Evaluation

Use the `EvaluateExpr` function to evaluate argument expressions:

```go
// For simple evaluation
resp, err := EvaluateExpr(args[0], ev)

// For expressions that might contain operators
resp, err := ev.Evaluate(args[0])

// For raw values without evaluation
if args[0].Type == Literal {
    value := args[0].Literal.Value
}
```

### 3. Response Types

Choose the appropriate response type:

```go
// Replace - replaces the entire node
return &Response{Type: Replace, Value: result}, nil

// Inject - injects values into a map
return &Response{Type: Inject, Value: map[interface{}]interface{}{...}}, nil

// Delete - removes the node
return &Response{Type: Delete}, nil
```

### 4. Error Handling

Provide clear, actionable error messages:

```go
return nil, fmt.Errorf("myoperator: cannot process value of type %T", value)
return nil, fmt.Errorf("myoperator: path '%s' not found in document", path)
return nil, fmt.Errorf("myoperator: %v", err) // Wrap underlying errors
```

## Binary Operators for Enhanced Parser

To create operators that work with the enhanced parser's expression syntax:

### 1. Define the Operator Structure

```go
type MyBinaryOperator struct {
    op string // The operator symbol (e.g., "==", "+", "&&")
}
```

### 2. Implement Binary Operation Logic

```go
func (m MyBinaryOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
    if len(args) != 2 {
        return nil, fmt.Errorf("%s operator requires exactly 2 operands", m.op)
    }
    
    // Evaluate left operand
    left, err := EvaluateExpr(args[0], ev)
    if err != nil {
        return nil, err
    }
    
    // For short-circuit operators, check left value first
    if m.op == "&&" && !isTruthy(left.Value) {
        return &Response{Type: Replace, Value: false}, nil
    }
    
    // Evaluate right operand
    right, err := EvaluateExpr(args[1], ev)
    if err != nil {
        return nil, err
    }
    
    // Perform operation
    result := performOperation(m.op, left.Value, right.Value)
    
    return &Response{Type: Replace, Value: result}, nil
}
```

### 3. Register with Appropriate Symbol

```go
func init() {
    // For operators parsed by enhanced parser
    RegisterOp("==", ComparisonOperator{op: "=="})
    RegisterOp("+", ArithmeticOperator{op: "+"})
    RegisterOp("&&", BooleanOperator{op: "&&"})
}
```

## Operator Precedence

When creating operators for the enhanced parser, consider precedence levels:

1. **Lowest**: Ternary (`?:`)
2. **Logical OR**: `||`
3. **Logical AND**: `&&`
4. **Equality**: `==`, `!=`
5. **Comparison**: `<`, `>`, `<=`, `>=`
6. **Additive**: `+`, `-`
7. **Multiplicative**: `*`, `/`, `%`
8. **Unary**: `!`, `-` (prefix)
9. **Highest**: Primary expressions, parentheses

The parser handles precedence automatically based on token types.

## Testing Your Operator

### 1. Unit Tests

Create `op_myoperator_test.go`:

```go
func TestMyOperator(t *testing.T) {
    RunOperatorTests(t, MyOperator{}, []OperatorTest{
        {
            desc: "basic transformation",
            yaml: `value: (( myoperator "input" ))`,
            expect: `value: output`,
        },
        {
            desc: "error on invalid input",
            yaml: `value: (( myoperator ))`,
            err: "myoperator expects exactly 1 argument",
        },
    })
}
```

### 2. Integration Tests

Add test cases to `assets/myoperator/` directory:

```yaml
# assets/myoperator/basic.yml
input:
  value: (( myoperator "test" ))
expected:
  value: "TEST"
```

### 3. Enhanced Parser Tests

For operators that work with expressions:

```go
func TestMyOperatorInExpressions(t *testing.T) {
    UseEnhancedParser = true
    defer func() { UseEnhancedParser = false }()
    
    RunOperatorTests(t, nil, []OperatorTest{
        {
            desc: "in complex expression",
            yaml: `result: (( myoperator("a") + myoperator("b") ))`,
            expect: `result: AB`,
        },
    })
}
```

## Best Practices

### 1. Type Safety

Use type assertions carefully and handle all expected types:

```go
switch v := value.(type) {
case string:
    return processString(v)
case int, int64:
    return processNumber(v)
case []interface{}:
    return processArray(v)
case map[interface{}]interface{}:
    return processMap(v)
default:
    return nil, fmt.Errorf("unsupported type: %T", v)
}
```

### 2. Nil Handling

Always check for nil values:

```go
if resp.Value == nil {
    return &Response{Type: Replace, Value: nil}, nil
}
```

### 3. Memory Efficiency

For large data structures, avoid unnecessary copying:

```go
// Bad: Creates a copy
result := make([]interface{}, len(input))
copy(result, input)

// Good: Modify in place if safe
for i, v := range input {
    input[i] = transform(v)
}
```

### 4. Concurrency

Operators should be stateless and thread-safe:

```go
// Bad: Stores state
type BadOperator struct {
    cache map[string]interface{} // Not thread-safe!
}

// Good: Stateless
type GoodOperator struct{}
```

## Advanced Topics

### 1. Context-Aware Operations

Access document context through the Evaluator:

```go
func (o MyOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
    // Access the current document tree
    tree := ev.Tree
    
    // Resolve references
    cursor, err := tree.FindRef(path)
    if err != nil {
        return nil, err
    }
    
    // Continue processing...
}
```

### 2. Deferred Evaluation

For operators that should preserve expressions:

```go
if ev.IsDeferring() {
    // Return an AST representation instead of evaluating
    return &Response{
        Type: Replace,
        Value: &tree.Meta{
            Type: tree.MetaOp,
            Name: "myoperator",
            Args: args,
        },
    }, nil
}
```

### 3. Custom Expression Types

Create specialized expression types:

```go
type MyCustomExpr struct {
    Base
    Special string
}

func (e *MyCustomExpr) String() string {
    return fmt.Sprintf("custom:%s", e.Special)
}
```

## Debugging Tips

### 1. Enable Debug Logging

```go
import "github.com/starkandwayne/graft/log"

func (o MyOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
    log.DEBUG("MyOperator called with %d args", len(args))
    
    for i, arg := range args {
        log.DEBUG("  arg[%d]: %v", i, arg)
    }
    
    // ...
}
```

### 2. Trace Expression Evaluation

```go
resp, err := EvaluateExpr(args[0], ev)
log.DEBUG("Evaluated expression: %v -> %v (err: %v)", args[0], resp, err)
```

### 3. Test with Enhanced Parser

```bash
# Test with enhanced parser
GRAFT_ENHANCED_PARSER=1 graft merge test.yml

# Or use the legacy parser to compare
graft merge --legacy-parser test.yml
```

## Common Patterns

### 1. Path Resolution

```go
// Resolve a path reference
path := args[0].Reference.String()
cursor, err := ev.Tree.FindRef(path)
if err != nil {
    return nil, fmt.Errorf("path '%s' not found", path)
}
```

### 2. Type Coercion

```go
// Convert to number
func toNumber(v interface{}) (float64, error) {
    switch n := v.(type) {
    case int:
        return float64(n), nil
    case int64:
        return float64(n), nil
    case float64:
        return n, nil
    case string:
        return strconv.ParseFloat(n, 64)
    default:
        return 0, fmt.Errorf("cannot convert %T to number", v)
    }
}
```

### 3. List Processing

```go
// Process a list argument
list, ok := args[0].Value.([]interface{})
if !ok {
    return nil, fmt.Errorf("expected list, got %T", args[0].Value)
}

result := make([]interface{}, 0, len(list))
for _, item := range list {
    processed := processItem(item)
    result = append(result, processed)
}
```

## Migration from Legacy Operators

When updating operators for the enhanced parser:

1. **Support Both Syntaxes**: Keep backward compatibility
2. **Update Tests**: Add tests for new expression syntax
3. **Document Changes**: Update operator documentation
4. **Gradual Migration**: Use feature flags during transition

Example migration:

```go
// Legacy: (( concat "a" "b" "c" ))
// Enhanced: (( "a" + "b" + "c" ))

func init() {
    RegisterOp("concat", ConcatOperator{})
    if UseEnhancedParser {
        RegisterOp("+", ConcatOperator{}) // Support both
    }
}
```

## Resources

- [Expression Operators Guide](expression-operators.md) - User documentation
- [Operator Examples](../examples/expression-operators/) - Example YAML files
- [Test Suite](../operator_test.go) - Comprehensive test examples
- [Parser Documentation](parser-enhanced.go) - Enhanced parser implementation