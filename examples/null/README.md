# Null Operator Examples

The `null` operator in Spruce is used to check if a value is null, empty, or undefined. It returns true if the value is considered "null-like" and false otherwise.

## Overview

The `(( null <reference> ))` operator evaluates to:
- `true` if the reference is:
  - `null` or `nil`
  - An empty string `""`
  - An empty array `[]`
  - An empty map `{}`
  - A non-existent key (undefined)
- `false` for any other value

## Files in this Directory

1. **basic.yml** - Basic null checking examples
2. **conditional-config.yml** - Using null checks in conditional configurations
3. **default-values.yml** - Providing defaults for null values
4. **validation.yml** - Data validation using null checks

## Common Use Cases

- Checking if optional configuration is provided
- Validating required fields
- Providing default values when configuration is missing
- Conditional resource creation based on presence of values
- Data validation in complex configurations

## Running the Examples

```bash
# Check basic null operations
spruce merge basic.yml

# Conditional configuration based on null checks
spruce merge conditional-config.yml

# Default value handling
spruce merge default-values.yml

# Data validation examples
spruce merge validation.yml
```

## Key Concepts

1. **Empty vs Null**: The operator treats empty collections the same as null
2. **Undefined References**: Non-existent keys are considered null
3. **Type Coercion**: No type coercion occurs - only truly empty/null values return true
4. **Combination with Ternary**: Often used with `?:` operator for conditional logic