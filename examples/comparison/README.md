# Comparison Operator Examples

The comparison operators in Spruce allow you to compare values and create conditional logic in your configurations.

## Overview

Spruce provides the following comparison operators:
- `==` - Equal to
- `!=` - Not equal to
- `<` - Less than
- `>` - Greater than
- `<=` - Less than or equal to
- `>=` - Greater than or equal to

All comparison operators return boolean values (`true` or `false`).

## Files in this Directory

1. **basic.yml** - Basic comparison operations with all operators
2. **threshold-checks.yml** - Using comparisons for threshold validation
3. **conditional-resources.yml** - Resource allocation based on comparisons
4. **version-comparisons.yml** - Comparing version numbers and strings

## Common Use Cases

- Validating configuration values
- Conditional resource allocation
- Environment-based decisions
- Threshold monitoring
- Version compatibility checks
- Access control logic

## Running the Examples

```bash
# Basic comparison operations
spruce merge basic.yml

# Threshold validation examples
spruce merge threshold-checks.yml

# Conditional resource allocation
spruce merge conditional-resources.yml

# Version comparison examples
spruce merge version-comparisons.yml
```

## Key Concepts

1. **Type Compatibility**: Comparisons work with numbers, strings, and booleans
2. **String Comparison**: Lexicographic (alphabetical) ordering
3. **Boolean Context**: Results are always true or false
4. **Chaining**: Can be combined with logical operators (&&, ||, !)
5. **Ternary Usage**: Often used as conditions in ternary expressions