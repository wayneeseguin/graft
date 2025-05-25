# Expression Operators Guide

This guide covers the enhanced expression operators available in Spruce, including arithmetic, comparison, boolean logic, and conditional expressions.

## Table of Contents

- [Overview](#overview)
- [Arithmetic Operators](#arithmetic-operators)
- [Comparison Operators](#comparison-operators)
- [Boolean Operators](#boolean-operators)
- [Ternary Operator](#ternary-operator)
- [Operator Precedence](#operator-precedence)
- [Complex Expressions](#complex-expressions)
- [Migration Guide](#migration-guide)

## Overview

Spruce's enhanced parser (enabled by default) supports a rich set of operators for building complex expressions:

```yaml
# Simple arithmetic
total: (( 10 + 5 ))                    # 15

# Comparisons
is_valid: (( count > 0 ))              # true/false

# Boolean logic
can_proceed: (( is_valid && !error ))  # true/false

# Conditional expressions
status: (( success ? "ok" : "failed" )) # "ok" or "failed"

# Complex nested expressions
result: (( (base * multiplier) + (bonus > 0 ? bonus : 0) ))
```

## Arithmetic Operators

### Basic Arithmetic

```yaml
# Addition
sum: (( 10 + 5 ))              # 15
total: (( price + tax ))        # adds two values

# Subtraction
difference: (( 20 - 8 ))        # 12
remaining: (( total - used ))   # subtracts used from total

# Multiplication
product: (( 4 * 5 ))            # 20
area: (( width * height ))      # multiplies dimensions

# Division
quotient: (( 20 / 4 ))          # 5
average: (( sum / count ))      # divides sum by count

# Modulo (remainder)
remainder: (( 17 % 5 ))         # 2
is_even: (( number % 2 == 0 )) # true if even
```

### Type Coercion

Arithmetic operators handle mixed types intelligently:

```yaml
# Integer + Float = Float
result1: (( 10 + 3.5 ))         # 13.5

# String concatenation with +
greeting: (( "Hello, " + name )) # "Hello, Alice"

# Automatic type conversion
value: "42"
doubled: (( value * 2 ))        # 84 (string "42" → int 42)
```

### Using References

```yaml
prices:
  item: 100
  tax_rate: 0.08

totals:
  subtotal: (( grab prices.item ))
  tax: (( prices.item * prices.tax_rate ))
  total: (( prices.item + (prices.item * prices.tax_rate) ))
  # or more simply:
  total_alt: (( prices.item * (1 + prices.tax_rate) ))
```

## Comparison Operators

### Equality Operators

```yaml
# Equality
is_equal: (( value == 42 ))
same_name: (( user.name == "admin" ))

# Inequality
not_equal: (( status != "error" ))
different: (( old_value != new_value ))
```

### Relational Operators

```yaml
# Greater than
is_adult: (( age > 18 ))
overdue: (( days_elapsed > grace_period ))

# Less than
needs_refill: (( fuel_level < 0.25 ))
under_budget: (( spent < budget ))

# Greater than or equal
can_vote: (( age >= 18 ))
at_capacity: (( count >= max_allowed ))

# Less than or equal
within_limit: (( usage <= quota ))
valid_score: (( score <= 100 ))
```

### Type-Aware Comparisons

```yaml
# Numeric comparison
result: (( "10" > 9 ))          # true (string "10" → int 10)

# String comparison (lexicographic)
alphabetical: (( "apple" < "banana" )) # true

# Mixed arrays/maps use deep equality
list1: [1, 2, 3]
list2: [1, 2, 3]
lists_equal: (( list1 == list2 ))     # true
```

## Boolean Operators

### Logical AND (&&)

```yaml
# Both conditions must be true
can_proceed: (( is_ready && !has_errors ))
is_valid: (( age >= 18 && age <= 65 ))

# Short-circuit evaluation (right side not evaluated if left is false)
safe_check: (( obj != nil && obj.value > 0 ))
```

### Logical OR (||)

The new `||` operator performs boolean OR, distinct from the fallback operator:

```yaml
# Boolean OR - at least one condition must be true
has_access: (( is_admin || is_owner ))
needs_attention: (( is_urgent || is_expired ))

# Comparison with fallback operator
# Fallback: returns first non-null value
fallback_example: (( grab optional.value || "default" ))

# Boolean OR: returns true/false
boolean_or_example: (( has_feature_a || has_feature_b ))
```

### Logical NOT (!)

```yaml
# Negation
is_invalid: (( !is_valid ))
not_empty: (( !empty ))
should_proceed: (( !cancelled && !error ))
```

### Truthiness

The following values are considered "falsy":
- `false` (boolean)
- `nil` or `null`
- `0` (numeric zero)
- `""` (empty string)
- `[]` (empty array)
- `{}` (empty map)

Everything else is "truthy":

```yaml
# Examples of truthiness
examples:
  truthy1: (( !false ))        # true
  truthy2: (( !"" ))          # true
  truthy3: (( !0 ))           # true
  truthy4: (( ![] ))          # true
  truthy5: (( !{} ))          # true
  truthy6: (( !nil ))         # true
  
  falsy1: (( !"hello" ))      # false (non-empty string is truthy)
  falsy2: (( !42 ))           # false (non-zero is truthy)
  falsy3: (( ![1, 2] ))       # false (non-empty array is truthy)
```

## Ternary Operator

The ternary operator (`? :`) provides inline conditional expressions:

```yaml
# Basic syntax: condition ? true_value : false_value
status: (( is_active ? "running" : "stopped" ))
message: (( error ? error_text : "Success" ))

# Numeric conditionals
discount: (( is_member ? 0.2 : 0.0 ))
final_price: (( base_price * (1 - (is_member ? 0.2 : 0.0)) ))

# Nested ternary (use parentheses for clarity)
grade: (( score >= 90 ? "A" : (score >= 80 ? "B" : (score >= 70 ? "C" : "F")) ))

# Conditional references
config_source: (( use_default ? (grab defaults.config) : (grab custom.config) ))
```

## Operator Precedence

Operators are evaluated in the following order (highest to lowest precedence):

1. **Parentheses** `( )` - Explicit grouping
2. **Unary** `!` - Logical NOT
3. **Multiplicative** `*`, `/`, `%`
4. **Additive** `+`, `-`
5. **Comparison** `<`, `>`, `<=`, `>=`
6. **Equality** `==`, `!=`
7. **Logical AND** `&&`
8. **Logical OR** `||`
9. **Ternary** `? :`

### Examples

```yaml
# Without parentheses (follows precedence)
result1: (( 2 + 3 * 4 ))        # 14 (not 20)
result2: (( 10 > 5 && 20 < 30 )) # true

# With parentheses (explicit grouping)
result3: (( (2 + 3) * 4 ))      # 20
result4: (( 10 > (5 + 2) ))     # true

# Complex expression
is_valid: (( !error && (count > min && count < max) || override ))
```

## Complex Expressions

### Nested Operator Calls

Combine operators with Spruce's built-in operators:

```yaml
# Arithmetic with grab
total: (( (grab prices.base) + (grab prices.tax) ))

# Conditional with concat
message: (( concat "Status: " (success ? "OK" : "Failed") ))

# Boolean logic with list operations
has_required: (( (grab features.required || []) != [] ))
```

### Real-World Examples

#### Configuration Based on Environment

```yaml
environments:
  dev:
    instances: 1
    debug: true
  prod:
    instances: 10
    debug: false

deployment:
  env: prod
  instances: (( grab environments.(( grab deployment.env )).instances ))
  memory: (( deployment.env == "prod" ? 4096 : 1024 ))
  debug_mode: (( deployment.env != "prod" && (grab override_debug || false) ))
```

#### Dynamic Scaling

```yaml
scaling:
  base_instances: 2
  load_multiplier: 1.5
  current_load: 0.8
  max_instances: 10
  
  calculated_instances: (( scaling.base_instances * scaling.load_multiplier * scaling.current_load ))
  final_instances: (( scaling.calculated_instances > scaling.max_instances ? scaling.max_instances : scaling.calculated_instances ))
```

#### Conditional Features

```yaml
features:
  premium: true
  trial_days_left: 5
  
settings:
  max_uploads: (( features.premium ? 1000 : 10 ))
  can_export: (( features.premium || features.trial_days_left > 0 ))
  show_upgrade: (( !features.premium && features.trial_days_left <= 3 ))
```

## Migration Guide

### From Legacy to Enhanced Parser

The enhanced parser is backward compatible, but if you need to use the legacy parser:

```bash
# Via environment variable
export SPRUCE_LEGACY_PARSER=true
spruce merge file.yml

# Via command line flag
spruce merge --legacy-parser file.yml
```

### Common Patterns

#### Fallback vs Boolean OR

```yaml
# Old style (still works) - fallback operator
value: (( grab optional.path || "default" ))

# New style - explicit boolean OR for conditions
condition: (( has_feature_a || has_feature_b ))
```

#### Complex Conditions

```yaml
# Old style - nested grab and conditions
valid: (( grab meta.ready ))
flag: (( grab meta.enabled ))
# Then check both separately

# New style - inline boolean expression
can_proceed: (( (grab meta.ready || false) && (grab meta.enabled || false) ))
```

#### Calculated Values

```yaml
# Old style - store intermediate values
base: 100
multiplier: 1.5
intermediate: (( concat base "*" multiplier )) # String manipulation

# New style - direct arithmetic
result: (( base * multiplier ))  # 150
```

### Best Practices

1. **Use parentheses for clarity** in complex expressions
2. **Leverage short-circuit evaluation** for safety checks
3. **Prefer arithmetic operators** over string concatenation for math
4. **Use ternary operator** for simple conditionals instead of complex YAML structures
5. **Test expressions** with `spruce eval` for debugging

### Debugging Expressions

Use `spruce eval` to test expressions:

```bash
# Test an expression directly
echo 'result: (( 10 + 5 * 2 ))' | spruce eval

# With references
cat <<EOF | spruce eval
values:
  a: 10
  b: 5
result: (( values.a > values.b ? "a is bigger" : "b is bigger" ))
EOF
```

## See Also

- [Spruce Operators Documentation](operators.md) - Complete list of all Spruce operators
- [Merging Documentation](merging.md) - How Spruce merges YAML files
- [Environment Variables](environment-variables-and-defaults.md) - Using environment variables in expressions