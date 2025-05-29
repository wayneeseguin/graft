# Vault Operator Migration Guide

This example demonstrates the enhanced vault operator syntax and how to migrate from the older `vault-try` operator.

## New Features

The enhanced `vault` operator now supports multiple vault paths with fallback, eliminating the need for the separate `vault-try` operator.

### Semicolon-Separated Paths

You can now specify multiple vault paths separated by semicolons:

```yaml
# Try multiple paths in order
password: (( vault "secret/prod:password; secret/staging:password; secret/dev:password" ))

# With a default value using LogicalOr
password: (( vault "secret/prod:password; secret/dev:password" || "default-password" ))

# With concatenation and multiple paths
api_key: (( vault "secret/" environment ":api-key; secret/shared:api-key" || "dev-key" ))
```

### Multiple Arguments (vault-try style)

The vault operator also supports multiple arguments similar to vault-try:

```yaml
# Each argument is a separate path to try
password: (( vault "secret/prod:password" "secret/dev:password" "default-password" ))

# With LogicalOr for explicit default
password: (( vault "secret/prod:password" "secret/dev:password" || "default-password" ))
```

## Migration Examples

### From vault-try to vault

#### Basic Migration
```yaml
# Old (vault-try)
database:
  password: (( vault-try "secret/prod/db:password" "secret/dev/db:password" "changeme" ))

# New (vault with semicolons)
database:
  password: (( vault "secret/prod/db:password; secret/dev/db:password" || "changeme" ))

# New (vault with multiple args)
database:
  password: (( vault "secret/prod/db:password" "secret/dev/db:password" || "changeme" ))
```

#### With References
```yaml
# Old (vault-try)
api:
  key: (( vault-try (concat "secret/" meta.env "/api:key") "secret/shared/api:key" meta.default_key ))

# New (vault with semicolons)
api:
  key: (( vault "secret/" meta.env "/api:key; secret/shared/api:key" || meta.default_key ))
```

## Running the Examples

1. Set up your Vault environment:
   ```bash
   export VAULT_ADDR="https://vault.example.com"
   export VAULT_TOKEN="your-token"
   ```

2. Run the migration example:
   ```bash
   graft merge migration-example.yml
   ```

3. Compare with the legacy vault-try syntax:
   ```bash
   graft merge legacy-vault-try.yml
   ```

Both should produce identical results, demonstrating full backward compatibility.

## Best Practices

1. **Use semicolons for clarity**: When you have multiple paths, semicolon syntax is clearer than multiple arguments
2. **Always provide defaults**: Use the `||` operator to provide sensible defaults
3. **Test your migration**: The enhanced vault operator is fully backward compatible, but always test your specific use cases
4. **Consider path organization**: With easy fallback support, you can organize secrets by environment more effectively

## Advantages of the New Syntax

1. **Unified operator**: No need to remember two different operators
2. **Flexible syntax**: Choose between semicolons or multiple arguments based on your preference
3. **Better concatenation**: Semicolon syntax works naturally with dynamic path building
4. **Clearer intent**: The `||` operator clearly indicates the default value