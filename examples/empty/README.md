# Empty Operator Examples

The `empty` operator returns true if a value is null, an empty string, empty array, or empty map. It's particularly useful for conditional logic and validation.

## Files in this directory:

1. **basic.yml** - Basic empty checks for different data types
2. **conditionals.yml** - Using empty in conditional expressions
3. **validation.yml** - Data validation patterns
4. **defaults.yml** - Setting defaults for empty values
5. **complex-structures.yml** - Checking emptiness in nested structures

## Key Concepts:

- `empty` returns true for: null, "", [], {}
- `empty` returns false for: non-empty strings, non-empty arrays/maps, numbers, booleans
- Often used with ternary operator `?:` for conditional logic
- Can be combined with `!` (not) operator for non-empty checks

## Common Patterns:

```yaml
# Check if value is empty
is_empty: (( empty some.value ))

# Set default if empty
value: (( empty input ? "default" : input ))

# Require non-empty value
validated: (( ! empty required.field ? required.field : "error" ))

# Multiple empty checks
all_empty: (( empty a && empty b && empty c ))
any_empty: (( empty a || empty b || empty c ))
```

## Running Examples:

```bash
# Test individual examples
spruce merge basic.yml

# See how empty interacts with parameters
spruce merge validation.yml <(echo "name: John")

# Test with different input values
echo "items: []" | spruce merge -
```