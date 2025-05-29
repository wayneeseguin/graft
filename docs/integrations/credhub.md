# CredHub Integration

graft can work seamlessly with [BOSH CredHub](https://docs.cloudfoundry.org/credhub/) credential references by using a special syntax that avoids conflicts with graft operators.

## The Problem

CredHub uses a syntax that looks similar to graft operators:

```yaml
# CredHub syntax (conflicts with graft)
secret:
  password: ((from_credhub))
```

This causes graft to treat `from_credhub` as an unknown operator and throw an error.

## The Solution

Use the `((!...))` notation instead of `((...))` for CredHub references:

```yaml
# graft-compatible CredHub syntax
secret:
  password: ((!from_credhub))
```

## How It Works

1. graft sees `((!...))` and ignores it (the `!` indicates it's not a graft operator)
2. graft outputs `((!from_credhub))` in the final YAML
3. BOSH reads the manifest and automatically drops the `!` 
4. BOSH processes `((from_credhub))` as a normal CredHub reference

## Basic Examples

### Simple Credential Reference

```yaml
# Input (graft template)
database:
  username: ((!db-username))
  password: ((!db-password))

# Output (after graft processing)  
database:
  username: ((!db-username))
  password: ((!db-password))

# Final (processed by BOSH)
database:
  username: ((db-username))
  password: ((db-password))
```

### Mixed graft and CredHub

```yaml
# Template combining graft operators and CredHub references
meta:
  environment: production
  
database:
  # graft operator
  host: (( concat "db-" meta.environment ".example.com" ))
  port: 5432
  
  # CredHub references  
  username: ((!db-username))
  password: ((!db-password))

# Output
database:
  host: db-production.example.com
  port: 5432
  username: ((!db-username))
  password: ((!db-password))
```

## Advanced Patterns

### Environment-Specific Credentials

```yaml
meta:
  environment: production

# Use graft to build CredHub credential names dynamically
database:
  # This creates: ((!prod-db-password))
  password: ((!{{ meta.environment }}-db-password))
  
# Alternative approach using concat and defer
database:
  password: (( defer concat "((!db-" meta.environment "-password))" ))
```

### Conditional CredHub References

```yaml
meta:
  use_credhub: true
  environment: production

database:
  # Use CredHub in production, default values elsewhere
  password: (( meta.use_credhub ? 
    defer concat "((!db-" meta.environment "-password))" : 
    "default-password" 
  ))
```

### Generated CredHub Names

```yaml
meta:
  app_name: myapp
  environment: production
  
# Build CredHub reference names from metadata
services:
  api:
    # Results in: ((!myapp-production-api-key))
    api_key: (( defer concat "((!{{ meta.app_name }}-{{ meta.environment }}-api-key))" ))
  
  database:
    # Results in: ((!myapp-production-db-password))  
    password: (( defer concat "((!{{ meta.app_name }}-{{ meta.environment }}-db-password))" ))
```

## Migration from CredHub Templates

If you have existing templates written for CredHub, you can easily migrate them:

### Manual Migration

Replace `((` with `((!` throughout your templates:

```bash
# Before
secret:
  password: ((db-password))
  api_key: ((api-key))

# After  
secret:
  password: ((!db-password))
  api_key: ((!api-key))
```

### Automated Migration

Use `sed` to automatically convert CredHub templates:

```bash
# Convert single file
sed 's/((/((!/g' credhub-template.yml > graft-template.yml

# Convert multiple files
for file in templates/*.yml; do
  sed 's/((/((!/g' "$file" > "graft-$(basename "$file")"
done

# In-place conversion (be careful!)
sed -i 's/((/((!/g' template.yml
```

### Batch Processing

For large numbers of files:

```bash
#!/bin/bash
# migrate-credhub-templates.sh

find . -name "*.yml" -exec sed -i 's/((/((!/g' {} \;
echo "Converted CredHub templates to graft-compatible format"
```

## Best Practices

### 1. Consistent Naming

Use consistent CredHub credential naming:

```yaml
# Good - consistent pattern
database:
  username: ((!{{ app_name }}-{{ env }}-db-username))
  password: ((!{{ app_name }}-{{ env }}-db-password))

# Avoid - inconsistent naming
database:
  username: ((!db_user))
  password: ((!database-secret-password))
```

### 2. Document CredHub Dependencies

Document required CredHub credentials:

```yaml
# Required CredHub credentials:
# - myapp-prod-db-username: Database username
# - myapp-prod-db-password: Database password  
# - myapp-prod-api-key: External API key

meta:
  app_name: myapp
  environment: prod

database:
  username: ((!{{ meta.app_name }}-{{ meta.environment }}-db-username))
  password: ((!{{ meta.app_name }}-{{ meta.environment }}-db-password))
```

### 3. Use defer for Dynamic Names

When building CredHub names dynamically, use defer:

```yaml
meta:
  service: api
  environment: production

# Use defer to ensure proper string construction
credentials:
  key: (( defer concat "((!service-" meta.service "-" meta.environment "-key))" ))
```

### 4. Validate Output

Always check that your CredHub references are correctly formatted in the output:

```bash
# Check the final output
graft merge template.yml | grep -E '\(\(\![^)]+\)\)'

# Should show lines like:
# password: ((!db-password))
# api_key: ((!api-key))
```

## Common Issues and Solutions

### Issue: Double Processing

**Problem**: CredHub references get processed twice
```yaml
# Wrong - this doesn't work
password: (( defer "((!db-password))" ))
```

**Solution**: Use proper defer syntax
```yaml
# Correct
password: (( defer concat "((!db-password))" ))
```

### Issue: Invalid CredHub Names

**Problem**: Generated names contain invalid characters
```yaml
# This might create: ((!my.app-prod-key))
app_key: ((!{{ meta.app.name }}-{{ environment }}-key))
```

**Solution**: Sanitize names in metadata
```yaml
meta:
  app_name: (( grab meta.app.name | replace "." "-" ))
  
app_key: ((!{{ meta.app_name }}-{{ environment }}-key))
```

### Issue: Missing Exclamation Mark

**Problem**: Forgot the `!` and graft treats it as an operator
```yaml
# Wrong - graft error
password: ((db-password))
```

**Solution**: Always use `!` for CredHub references
```yaml
# Correct
password: ((!db-password))
```

## Integration with BOSH

### Deployment Workflow

1. **Template Processing**: graft processes templates, preserves CredHub references
2. **Deployment**: BOSH deploys with CredHub references intact
3. **Runtime**: BOSH resolves CredHub references during deployment

```bash
# Step 1: Process with graft
graft merge base.yml prod.yml > manifest.yml

# Step 2: Deploy with BOSH (CredHub references resolved automatically)
bosh deploy manifest.yml
```

### Credential Storage

Store credentials in CredHub before deployment:

```bash
# Store credentials in CredHub
credhub set -t password -n /bosh/myapp-prod-db-password -v secret123
credhub set -t value -n /bosh/myapp-prod-api-key -v abc123def456

# Deploy (references will be resolved)
graft merge template.yml | bosh deploy -
```

## Complete Example

### Template File

```yaml
# manifest-template.yml
meta:
  app_name: webapp
  environment: production
  domain: example.com

name: (( concat meta.app_name "-" meta.environment ))

instance_groups:
- name: web
  instances: 3
  jobs:
  - name: webapp
    properties:
      # graft operator
      domain: (( meta.domain ))
      
      # CredHub references
      database:
        username: ((!{{ meta.app_name }}-{{ meta.environment }}-db-username))
        password: ((!{{ meta.app_name }}-{{ meta.environment }}-db-password))
      
      api:
        key: ((!{{ meta.app_name }}-{{ meta.environment }}-api-key))
        endpoint: (( concat "https://api." meta.domain ))
```

### Processing and Output

```bash
graft merge manifest-template.yml --prune meta
```

Output:
```yaml
name: webapp-production

instance_groups:
- name: web  
  instances: 3
  jobs:
  - name: webapp
    properties:
      domain: example.com
      
      database:
        username: ((!webapp-production-db-username))
        password: ((!webapp-production-db-password))
      
      api:
        key: ((!webapp-production-api-key))
        endpoint: https://api.example.com
```

## See Also

- [BOSH CredHub Documentation](https://docs.cloudfoundry.org/credhub/)
- [BOSH Integration](bosh.md) - Using graft with BOSH
- [Vault Integration](../guides/vault-integration.md) - Alternative secret management
- [defer Operator](../operators/data-references.md#defer) - Template generation