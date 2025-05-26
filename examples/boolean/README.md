# Boolean Operator Examples

The boolean operators in Spruce provide logical operations for combining conditions and controlling flow.

## Overview

Spruce provides the following boolean operators:
- `&&` - Logical AND
- `||` - Logical OR  
- `!` - Logical NOT (negation)

These operators are essential for creating complex conditions and implementing business logic.

## Files in this Directory

1. **basic.yml** - Basic boolean operations with all operators
2. **access-control.yml** - Using boolean logic for access control
3. **validation-rules.yml** - Complex validation with boolean operators
4. **feature-combinations.yml** - Feature flag combinations and dependencies

## Common Use Cases

- Combining multiple conditions
- Access control and permissions
- Feature flag dependencies
- Validation rules
- Configuration constraints
- Conditional resource allocation

## Running the Examples

```bash
# Basic boolean operations
spruce merge basic.yml

# Access control examples
spruce merge access-control.yml

# Validation rule examples
spruce merge validation-rules.yml

# Feature combination examples
spruce merge feature-combinations.yml
```

## Key Concepts

1. **Short-circuit evaluation**: Operations stop as soon as result is determined
2. **Truthiness**: Understanding what values are considered true/false
3. **Operator precedence**: NOT > AND > OR
4. **Complex expressions**: Combining multiple operators
5. **De Morgan's Laws**: Useful for simplifying boolean expressions