# Nested Expressions in graft

> **Feature**: graft supports nesting operators within other operators, enabling powerful expression composition.

## Overview

You can use operators as arguments to other operators. This allows you to build complex expressions without intermediate variables.

## Basic Syntax

Instead of:
```yaml
# Old way - requires intermediate steps
meta:
  name: john
  greeting: (( grab meta.name ))
  message: (( concat "Hello, " meta.greeting "!" ))
```

You can now write:
```yaml
# New way - direct composition
meta:
  name: john
  message: (( concat "Hello, " (grab meta.name) "!" ))
```

## Common Patterns

### 1. Dynamic String Building

```yaml
# Concatenate with grabbed values
welcome: (( concat "Welcome " (grab user.firstName) " " (grab user.lastName) ))

# Join lists with custom separator
path: (( join "/" (grab pathComponents) ))

# Conditional messages
status: (( concat "Status: " (grab isActive ? "Active" : "Inactive") ))
```

### 2. Dynamic File Loading

```yaml
# Load environment-specific config
config: (( load (concat "configs/" (grab environment) ".yml") ))

# Read file content dynamically
script: (( file (concat (grab scriptDir) "/" (grab scriptName)) ))
```

### 3. Conditional Operations

```yaml
# Use empty with conditions
database:
  host: (( grab production ? prodDB.host : devDB.host ))
  credentials: (( empty (grab useVault) ? (vault "secret/db:password") : "default" ))

# Conditional grab with defaults
port: (( grab (concat "ports." (grab service)) || 8080 ))
```

### 4. List and Map Operations

```yaml
# Extract and join keys
services: (( join ", " (keys (grab enabledFeatures)) ))

# Transform and encode
authHeader: (( concat "Basic " (base64 (concat (grab username) ":" (grab password))) ))
```

### 5. Mathematical Expressions

```yaml
# Calculate with nested expressions
total: (( (grab basePrice) * (grab quantity) + (grab tax) ))

# Percentage calculations
discount: (( (grab price) * (grab discountRate) / 100 ))
finalPrice: (( (grab price) - (grab discount) ))
```

### 6. Complex Data Transformations

```yaml
# Build connection strings
connectionString: (( 
  concat 
    (grab dbType) "://" 
    (grab dbUser) ":" 
    (grab dbPass) "@" 
    (grab dbHost) ":" 
    (grab dbPort) "/" 
    (grab dbName) 
))

# Multi-level grabbing
setting: (( grab (concat "profiles." (grab activeProfile) "." (grab settingName)) ))
```

## Supported Nesting Combinations

Most operators can be nested, but some have restrictions:

### ✅ Can be nested anywhere:
- `grab`, `concat`, `join`, `base64`, `base64-decode`
- Arithmetic operators: `+`, `-`, `*`, `/`, `%`
- Comparison operators: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Boolean operators: `&&`, `||`, `!`
- `keys`, `stringify`, `negate`
- `empty`, `null`

### ⚠️ Limited nesting:
- `static_ips`, `ips` - Can use nested expressions for arguments but not as the operator itself
- `vault`, `awsparam`, `awssecret` - Can use nested expressions to build paths
- `calc` - Can use nested expressions in the calculation string

### ❌ Cannot be nested:
- Array merge operators (`append`, `prepend`, `merge`, etc.) - Top-level only
- `prune`, `param` - Special operators that don't return values
- `defer` - By design, defers evaluation

## Advanced Examples

### Example 1: Environment-Aware Configuration

```yaml
meta:
  env: production
  region: us-east-1

database:
  host: (( grab (concat "hosts." (grab meta.env) "." (grab meta.region)) ))
  port: (( grab (concat "ports." (grab meta.env)) || 5432 ))
  url: (( concat "postgresql://" (grab database.host) ":" (grab database.port) ))
```

### Example 2: Feature Flags with Nested Logic

```yaml
features:
  authentication: true
  advanced_search: false
  beta_features: (( grab environment == "staging" || grab environment == "dev" ))

config:
  searchEndpoint: (( 
    grab features.advanced_search 
      ? (concat (grab apiBase) "/search/v2") 
      : (concat (grab apiBase) "/search/v1") 
  ))
```

### Example 3: Dynamic Vault Paths

```yaml
secrets:
  path: (( concat "secret/" (grab environment) "/" (grab service) ))
  dbPassword: (( vault (concat (grab secrets.path) "/db:password") ))
  apiKey: (( vault (concat (grab secrets.path) "/api:key") || (grab defaults.apiKey) ))
```

### Example 4: Combining with Environment Variables

```yaml
# Build paths with environment variable expansion
config:
  basePath: (( concat "configs/" $ENVIRONMENT "/" $REGION ))
  setting: (( grab (concat (grab config.basePath) "." (grab settingName)) ))
  
# Conditional with environment variables
database:
  host: (( grab $USE_LOCAL == "true" ? "localhost" : (grab remoteHosts.$ENVIRONMENT) ))
```

## Best Practices

### 1. Readability First

While nesting is powerful, don't sacrifice readability:

```yaml
# ❌ Too complex
result: (( concat (grab (concat "data." (grab (join "." (grab pathParts))))) "suffix" ))

# ✅ Better - use intermediate values
dataPath: (( join "." (grab pathParts) ))
dataKey: (( concat "data." (grab dataPath) ))
result: (( concat (grab (grab dataKey)) "suffix" ))
```

### 2. Use Parentheses

Always use parentheses to make nesting clear:

```yaml
# ❌ Ambiguous
value: (( grab config.key || grab defaults.key ))

# ✅ Clear
value: (( (grab config.key) || (grab defaults.key) ))
```

### 3. Test Complex Expressions

Use `graft merge` with `--debug` to verify behavior:

```bash
# Enable debug output to see evaluation order
graft merge --debug config.yml

# Test expressions in isolation
echo 'test: (( concat "Hello " (grab name) ))' | graft merge --debug -
```

### 4. Consider Performance

Deeply nested expressions are evaluated recursively. For very complex operations, consider breaking them into steps.

## Troubleshooting

### Common Errors

1. **"Unknown expression type"**
   - Check your graft syntax is correct
   - Check parentheses are balanced
   - Verify operator names are correct

2. **"Cannot resolve operator"**
   - Verify the nested operator returns a compatible type
   - Some operators expect specific types (e.g., `join` needs a list)
   - Check that all referenced paths exist

3. **Circular Dependencies**
   - Avoid expressions that reference themselves
   - Use `--debug` flag to trace evaluation order
   - Break complex circular references into steps

### Type Mismatches

Different operators expect different types:

```yaml
# ❌ Wrong - concat expects strings
result: (( concat (grab myList) "suffix" ))

# ✅ Correct - stringify the list first
result: (( concat (stringify (grab myList)) "suffix" ))

# ❌ Wrong - join expects a list
result: (( join ", " (grab myString) ))

# ✅ Correct - provide a list
result: (( join ", " (grab myList) ))
```

## Migration from Legacy Parser

If you have existing configurations:

1. **No breaking changes** - Old syntax continues to work
2. **Incremental adoption** - Add nested expressions as needed
3. **Full backward compatibility** - All existing graft files continue to work

## Performance Considerations

1. **Evaluation Order** - Expressions are evaluated depth-first
2. **Caching** - Results are cached during a single merge operation
3. **Memory Usage** - Deep nesting increases memory usage
4. **Debugging** - Complex expressions are harder to debug

## See Also

- [Expression Operators](../operators/expression-operators.md) - Comparison, boolean, and ternary operators
- [Environment Variables](environment-variables.md) - Using environment variables in expressions
- [Operator Quick Reference](../reference/operator-quick-reference.md) - Complete operator list
- [Examples Directory](../../examples/) - Working examples of nested expressions