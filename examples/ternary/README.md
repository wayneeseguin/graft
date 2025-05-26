# Ternary Operator Examples

The ternary operator (`?:`) in Spruce provides conditional expression evaluation, similar to the ternary operator in many programming languages.

## Overview

The `(( condition ? true_value : false_value ))` operator evaluates:
- If `condition` is truthy, returns `true_value`
- If `condition` is falsy, returns `false_value`

Falsy values in Spruce:
- `false` (boolean)
- `null` or `nil`
- `0` (numeric zero)
- `""` (empty string)
- `[]` (empty array)
- `{}` (empty map)

## Files in this Directory

1. **basic.yml** - Basic ternary operator usage
2. **environment-config.yml** - Environment-specific configuration
3. **feature-flags.yml** - Feature flag management
4. **resource-sizing.yml** - Dynamic resource allocation

## Common Use Cases

- Conditional value selection
- Environment-specific settings
- Feature flag implementation
- Default value handling
- Dynamic configuration based on conditions
- Resource sizing based on environment

## Running the Examples

```bash
# Basic ternary operations
spruce merge basic.yml

# Environment-based configuration
spruce merge environment-config.yml

# Feature flag examples
spruce merge feature-flags.yml

# Resource sizing examples
spruce merge resource-sizing.yml
```

## Key Concepts

1. **Truthiness**: Understanding what values are considered true/false
2. **Nested Ternary**: Chaining multiple conditions
3. **Complex Conditions**: Using operators in the condition
4. **Type Flexibility**: Return values can be any type