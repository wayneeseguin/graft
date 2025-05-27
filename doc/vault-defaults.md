# Vault Operator Default Values

As of graft v1.31.0, the vault operator supports default values using the logical OR (`||`) syntax. This allows you to specify fallback values when secrets are not found in Vault, preventing graft from failing when optional secrets are missing.

## Basic Usage

### Simple Default Value

```yaml
password: (( vault "secret/myapp:password" || "default-password" ))
```

If the secret at `secret/myapp:password` doesn't exist in Vault, graft will use `"default-password"` instead of failing.

### Reference as Default

```yaml
defaults:
  password: my-default-password

password: (( vault "secret/myapp:password" || defaults.password ))
```

You can reference other values in your YAML as defaults.

### Environment Variable as Default

```yaml
password: (( vault "secret/myapp:password" || $DEFAULT_PASSWORD ))
```

Environment variables can be used as fallback values.

### Nil as Default

```yaml
password: (( vault "secret/myapp:password" || nil ))
```

You can explicitly set `nil` as the default value.

## Advanced Usage

### Concatenated Paths with Defaults

```yaml
meta:
  env: production

password: (( vault "secret/" meta.env ":password" || "default-password" ))
```

Default values work with concatenated vault paths.

### Multiple Vault Paths (vault-try)

For trying multiple vault paths before falling back to a default, use the `vault-try` operator:

```yaml
# Try production first, then staging, then use default
password: (( vault-try "secret/prod:password" "secret/staging:password" "default-password" ))

# With references
paths:
  primary: secret/prod:password
  secondary: secret/staging:password
  
password: (( vault-try paths.primary paths.secondary "fallback-password" ))
```

The `vault-try` operator:
- Requires at least 2 arguments (minimum 1 path + 1 default)
- Tries each vault path in order
- Uses the last argument as the default if all paths fail
- Falls back to default even for malformed paths (more forgiving than regular vault operator)

## Known Limitations

### Nested Operators in Defaults

Currently, nested operators within the default expression are not supported due to parser limitations:

```yaml
# This will NOT work as expected:
password: (( vault "secret/myapp:password" || grab defaults.password ))
password: (( vault "secret/myapp:password" || concat "prefix-" suffix ))
```

The parser interprets `grab defaults.password` as two separate tokens rather than a single operator expression.

**Recommended Workaround**: Use intermediate variables:

```yaml
# For grab operator:
defaults:
  password: my-secure-default
  
# Create an intermediate variable
default_from_grab: (( grab defaults.password ))
password: (( vault "secret/myapp:password" || default_from_grab ))

# For concat operator:
prefix: "app-"
suffix: "password"

# Create an intermediate variable  
default_from_concat: (( concat prefix suffix ))
password: (( vault "secret/myapp:password" || default_from_concat ))
```

This workaround is fully supported and tested. The intermediate variable approach also makes your configuration more readable by giving meaningful names to your default values.

### Chained Vault Lookups

Chaining vault lookups in defaults requires careful structuring:

```yaml
# This works:
password: (( vault "secret/primary:password" || vault "secret/secondary:password" || "default" ))

# For multiple paths, prefer vault-try:
password: (( vault-try "secret/primary:password" "secret/secondary:password" "default" ))
```

## Migration Guide

If you're currently using wrapper scripts or pre-processing to handle missing vault secrets, you can now simplify your configuration:

### Before (with wrapper script)
```bash
# wrapper.sh
value=$(vault read -field=password secret/myapp 2>/dev/null || echo "default-password")
graft merge --prune meta <(echo "password: $value") base.yml
```

### After (native support)
```yaml
# base.yml
password: (( vault "secret/myapp:password" || "default-password" ))
```

## Best Practices

1. **Use specific defaults**: Avoid using generic defaults like "PLACEHOLDER" that might accidentally make it to production.

2. **Document your defaults**: Add comments explaining why a default is acceptable:
   ```yaml
   # Default to standard port if not specified in vault
   port: (( vault "secret/myapp:port" || "5432" ))
   ```

3. **Use vault-try for multiple paths**: When you have multiple possible vault paths, `vault-try` provides cleaner syntax:
   ```yaml
   # Cleaner than chained || operators
   password: (( vault-try "secret/prod:pass" "secret/common:pass" "default" ))
   ```

4. **Consider nil for optional values**: For truly optional configuration, nil might be more appropriate than a string default:
   ```yaml
   # Optional feature flag
   feature_enabled: (( vault "secret/features:new_feature" || nil ))
   ```

## Error Handling

The vault operator with defaults will still fail in these cases:
- Invalid vault paths (e.g., missing colons): `(( vault "invalid-path" || "default" ))`
- Network/authentication errors when connecting to Vault
- Invalid YAML in the default value expression

Use `vault-try` if you want more forgiving behavior that falls back to the default even for malformed paths.