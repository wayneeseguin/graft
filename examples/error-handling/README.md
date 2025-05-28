# Error Handling Examples

This directory contains examples that demonstrate Graft's improved error handling capabilities.

## Error Messages

The parser provides detailed error messages with:

1. **Error Type Classification**
   - Syntax Error: Problems with expression syntax
   - Type Error: Type mismatches in operations
   - Reference Error: Invalid or undefined references
   - Evaluation Error: Runtime evaluation failures
   - Operator Error: Unknown or misused operators

2. **Position Tracking**
   - Line and column numbers for error location
   - Source code context with visual indicators
   - Error tracking through nested expressions

3. **Helpful Context**
   - Additional context explaining the error
   - Suggestions for common mistakes
   - Clear indication of what was expected

## Running the Examples

Try merging the syntax-errors.yml file to see error messages:

```bash
# This will show various syntax errors with detailed messages
graft merge syntax-errors.yml

# Example output:
# Syntax Error: 1:15: expected closing parenthesis to match opening parenthesis
# 
#    1 | unclosed: (( grab meta.name
#      |               ^ parentheses must be balanced
```

## Error Recovery

The parser can be configured to collect multiple errors instead of stopping at the first one:

```bash
# Set environment variable to collect all errors
GRAFT_COLLECT_ERRORS=1 graft merge syntax-errors.yml
```

This helps identify multiple issues in a single run rather than fixing them one at a time.

## Common Error Patterns

### 1. Unclosed Parentheses
```yaml
bad: (( grab meta.value
```
Error: "expected closing parenthesis to match opening parenthesis"

### 2. Operators Without Operands
```yaml
bad: (( + 5 ))
```
Error: "unexpected operator '+' - operators must appear between operands"

### 3. Invalid Ternary Syntax
```yaml
bad: (( condition ? true_value ))
```
Error: "expected ':' in ternary expression - ternary operator requires '?' followed by ':'"

### 4. Type Mismatches
```yaml
bad: (( "string" + 42 ))
```
Error: "Type Error: cannot add number to string"

### 5. Undefined References
```yaml
bad: (( grab meta.undefined.path ))
```
Error: "Reference Error: reference not found: meta.undefined.path"

## Best Practices

1. **Check parentheses balance** - Make sure all `((` have matching `))`
2. **Verify operator usage** - Operators need operands on both sides (except unary operators like `!`)
3. **Use quotes for strings** - String literals should be quoted: `"string"` not `string`
4. **Check reference paths** - Ensure referenced paths exist in the document
5. **Match types in operations** - Can't mix strings and numbers in arithmetic

## Debugging Tips

1. **Use --trace flag** for detailed parsing information:
   ```bash
   graft merge --trace syntax-errors.yml
   ```

2. **Test expressions in isolation** by creating small test files

3. **Check operator precedence** - Use parentheses to make precedence explicit:
   ```yaml
   clear: (( (a + b) * c ))
   ambiguous: (( a + b * c ))  # b * c happens first
   ```