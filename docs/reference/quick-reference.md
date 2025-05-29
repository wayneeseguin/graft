# graft Quick Reference

## Command Line

```bash
# Basic merge
graft merge base.yml override.yml

# With options
graft merge --prune meta --cherry-pick jobs base.yml prod.yml

# Control color output
graft --color on merge file.yml    # Force color
graft --color off merge file.yml   # No color
graft merge file.yml              # Auto-detect (default)

# Other commands
graft diff file1.yml file2.yml
graft json config.yml
graft fan source.yml target1.yml target2.yml
graft vaultinfo manifest.yml
```

## Common Operators

### Data References
```yaml
# Reference another value
value: (( grab path.to.value ))

# With default
value: (( grab path.to.value || "default" ))

# Environment variable
value: (( grab $ENV_VAR ))

# Dynamic path
value: (( grab config.$ENVIRONMENT.key ))
```

### String Operations
```yaml
# Concatenation
url: (( concat "https://" domain "/api" ))

# Join array
path: (( join segments "/" ))

# Base64 encode/decode
encoded: (( base64 "secret-text" ))
decoded: (( base64-decode encoded_value ))
```

### Array Operations
```yaml
# Append
items:
  - (( append ))
  - new-item

# Merge by name
jobs:
  - (( merge ))
  - name: existing-job
    new-property: value

# Replace
items:
  - (( replace ))
  - only-item
```

### Conditionals
```yaml
# Ternary operator
debug: (( environment == "production" ? false : true ))

# Boolean logic
enabled: (( feature_a && feature_b ))
allowed: (( is_admin || is_owner ))
disabled: (( ! enabled ))

# Comparisons
valid: (( age >= 18 && age <= 65 ))
match: (( status == "active" ))
```

### Math Operations
```yaml
# Arithmetic
total: (( price + tax ))
discounted: (( price * 0.9 ))
average: (( sum / count ))

# Complex calculations
memory_mb: (( calc "nodes * memory_per_node * 1024" ))
```

### External Data
```yaml
# Vault
password: (( vault "secret/db:password" ))

# AWS Parameter Store
config: (( awsparam "/myapp/config" ))

# AWS Secrets Manager
api_key: (( awssecret "myapp/api-key" ))

# File
content: (( file "config.txt" ))
```

### Special Operations
```yaml
# Static IPs
networks:
  - name: web
    static_ips: (( static_ips 0 1 2 ))

# Cartesian product
combinations: (( cartesian-product sizes colors ))

# Keys of a map
map_keys: (( keys my_map ))

# Prune
temporary: (( prune ))
```

## Common Patterns

### Environment Configuration
```yaml
environment: (( grab $ENVIRONMENT || "development" ))
config: (( grab environments.$ENVIRONMENT ))

environments:
  development:
    debug: true
    replicas: 1
  production:
    debug: false
    replicas: 3
```

### Required Parameters
```yaml
database:
  host: (( param "Please set database.host" ))
  port: 5432
  password: (( param "Please set database.password" ))
```

### Building URLs
```yaml
protocol: https
domain: example.com
port: 443
path: /api/v1

url: (( concat protocol "://" domain ":" port path ))
```

### Conditional Features
```yaml
features:
  monitoring: (( environment == "production" ))
  debug_api: (( ! features.monitoring ))
  cache_enabled: (( environment != "development" ))
```

### List Operations
```yaml
# Filter and transform
all_hosts:
  - host1.example.com
  - host2.example.com
  - host3.example.com

# Reference specific elements
primary: (( grab all_hosts.[0] ))
```

## Flags Reference

| Flag | Description |
|------|-------------|
| `--skip-eval` | Don't evaluate operators |
| `--prune KEY` | Remove keys from output |
| `--cherry-pick KEY` | Only output specific keys |
| `--multi-doc` | Handle multi-document YAML |
| `--go-patch` | Use go-patch format |
| `--fallback-append` | Default to append for arrays |
| `-d, --debug` | Debug output |
| `--trace` | Verbose trace output |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VAULT_ADDR` | Vault server address |
| `VAULT_TOKEN` | Vault auth token |
| `VAULT_VERSION` | Vault KV version (1 or 2) |
| `AWS_REGION` | AWS region |
| `AWS_PROFILE` | AWS credentials profile |
| `AWS_ROLE` | AWS role to assume |
| `REDACT` | Show REDACTED for secrets |
| `GRAFT_DEBUG` | Enable debug mode |

## Type Checking

```yaml
# Check if null/empty
is_null: (( null value ))
is_empty: (( empty value ))

# Type conversion
as_string: (( stringify 123 ))
parsed: (( parse json_string ))
```

## Error Handling

```yaml
# Provide defaults for missing values
value: (( grab optional.path || "default" ))

# Multiple fallbacks
endpoint: (( 
  grab overrides.endpoint ||
  grab config.endpoint ||
  grab defaults.endpoint ||
  "http://localhost"
))

# Required values
critical: (( param "This value is required!" ))
```

## Debugging Tips

1. **Use debug mode**: `graft merge -d file.yml`
2. **Check specific paths**: `graft merge file.yml --cherry-pick path.to.check`
3. **Validate without operators**: `graft merge --skip-eval file.yml`
4. **Show differences**: `graft diff before.yml after.yml`
5. **Check vault paths**: `graft vaultinfo file.yml`

## Common Errors

**Reference not found:**
```yaml
# Error: $.missing.path could not be found
value: (( grab missing.path ))

# Fix:
value: (( grab missing.path || "default" ))
```

**Circular reference:**
```yaml
# Error: Circular reference detected
a: (( grab b ))
b: (( grab a ))
```

**Type mismatch:**
```yaml
# Error: Cannot concatenate non-string
result: (( concat "text" 123 ))

# Fix:
result: (( concat "text" (stringify 123) ))
```