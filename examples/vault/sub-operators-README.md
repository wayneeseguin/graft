# Vault Sub-Operators

This document describes the new paren `()` and bar `|` sub-operators for the vault operator, which provide enhanced expressiveness for vault path construction and choice handling.

## Overview

The vault operator now supports two new sub-operators that can be used within vault expressions:

- **Parentheses `()`**: For grouping expressions and controlling evaluation precedence
- **Bar `|`**: For trying multiple alternatives in sequence until one succeeds

These sub-operators work alongside the existing space concatenation and `||` logical OR default handling.

## Syntax

### Basic Syntax

```yaml
# Grouping with parentheses
vault_secret: (( vault ("secret/" env ":password") ))

# Choice with bar operator  
vault_secret: (( vault "secret/db:" ("password" | "pass") ))

# Combined grouping and choice
vault_secret: (( vault ("secret/" env ":" ("key1" | "key2")) ))

# With default fallback (existing || syntax)
vault_secret: (( vault ("secret/prod:key" | "secret/dev:key") || "default" ))
```

### Complex Example

The motivating example from the requirements:

```yaml
vault_key: (( vault ( meta.vault_path meta.stub  ":" ("key1" | "key2" ) | meta.exodus_path "subpath:key1") || "default"))
```

This expression:
1. Groups `meta.vault_path meta.stub ":" ("key1" | "key2")` together
2. Tries either "key1" or "key2" for the key name
3. If that entire path fails, tries `meta.exodus_path "subpath:key1"`
4. If all vault paths fail, falls back to "default"

## Operator Precedence

From highest to lowest precedence:

1. **`()`** - Parentheses/grouping
2. **`|`** - Choice alternatives  
3. **Space** - Concatenation (existing)
4. **`||`** - Logical OR defaults (existing)

## Sub-Operator Details

### Parentheses `()` - Grouping

Parentheses create expression groups that are evaluated as units:

```yaml
# Without grouping - might not work as expected
vault_secret: (( vault "secret/" env ":" "key1" | "key2" ))

# With grouping - ensures proper evaluation order
vault_secret: (( vault ("secret/" env ":" ("key1" | "key2")) ))
```

**Features:**
- Control evaluation precedence
- Support unlimited nesting depth
- Preserve existing space concatenation within groups
- Work with all existing vault operator features

### Bar `|` - Choice/Alternatives

The bar operator tries multiple values in sequence until one succeeds:

```yaml
# Try different key names
vault_secret: (( vault "secret/db:" ("password" | "pass" | "pwd") ))

# Try different paths
vault_secret: (( vault ("secret/prod/db:pass" | "secret/dev/db:pass") ))
```

**Features:**
- Left-to-right evaluation (short-circuit on first success)
- Stops at first non-nil, non-error result
- Different from `||` which is for final defaults
- Can be used at any level within vault expressions

## Use Cases

### 1. Key Name Variations

Try multiple key names for the same secret:

```yaml
database_password: (( vault "secret/db:" ("password" | "pass" | "pwd") ))
```

### 2. Environment Fallback

Try environment-specific paths with fallbacks:

```yaml
api_key: (( vault ("secret/" env "/api:key" | "secret/default/api:key") ))
```

### 3. Migration Scenarios

Support migration between secret stores:

```yaml
app_secret: (( vault ("new-secrets/app:key" | "old-secrets/app:key" | "legacy/app:key") ))
```

### 4. Multi-Tenant with Shared Fallback

```yaml
tenant_secret: (( vault ("secret/tenants/" tenant_id ":api_key" | "secret/shared:default_api_key") ))
```

### 5. KV Version Compatibility

Support both KV v1 and v2 paths:

```yaml
secret_value: (( vault ("secret/app:key" | "secret/data/app:key") ))
```

## Integration with Existing Features

Sub-operators work seamlessly with all existing vault operator features:

### With Environment Variables
```yaml
vault_secret: (( vault ("secret/" $ENV ":" ("key" | "secret")) ))
```

### With Reference Operators
```yaml
vault_secret: (( vault ("secret/" (grab env) ":" ("password" | "pass")) ))
```

### With Concat Operator
```yaml
vault_secret: (( vault (concat "secret/" env "/" ("app" | "service") ":key") ))
```

### With Logical OR Defaults
```yaml
vault_secret: (( vault ("secret/prod:key" | "secret/dev:key") || "fallback" ))
```

## Backward Compatibility

All existing vault operator syntax continues to work unchanged:

```yaml
# All of these still work exactly as before
basic: (( vault "secret/app:password" ))
concat: (( vault "secret/" env "/app:password" ))
with_default: (( vault "secret/app:password" || "default" ))
nested_op: (( vault "secret/" (grab env) "/app:password" ))
```

The new sub-operators are purely additive and don't change any existing behavior.

## Performance

Sub-operators are designed for efficiency:

- **Lazy evaluation**: Choice alternatives are only tried until one succeeds
- **Short-circuit behavior**: If the first choice succeeds, others aren't evaluated
- **Order optimization**: Place most likely choices first for best performance

```yaml
# Efficient: most common path first
vault_secret: (( vault ("secret/cache/frequent:key" | "secret/db/expensive:key") ))
```

## Error Handling

Sub-operators provide graceful error handling:

```yaml
# If any choice succeeds, no error is raised
vault_secret: (( vault ("secret/prod:key" | "secret/staging:key" | "secret/dev:key") ))

# Combined with || for ultimate fallback
vault_secret: (( vault ("secret/prod:key" | "secret/dev:key") || "default-value" ))
```

## Examples

See `sub-operators.yml` for comprehensive examples including:
- Basic grouping and choice examples
- Complex nested expressions
- Real-world use cases
- Multi-tenant patterns
- Integration with other operators
- Performance optimization patterns

## Migration Guide

To start using sub-operators:

1. **Identify opportunities**: Look for vault expressions that could benefit from choices or grouping
2. **Start simple**: Begin with basic choice operators for key name variations
3. **Add grouping**: Use parentheses when you need precise control over evaluation order
4. **Optimize order**: Place most likely successful choices first
5. **Test thoroughly**: Verify that expressions work as expected in your environment

The sub-operators are optional and don't require any changes to existing configurations.