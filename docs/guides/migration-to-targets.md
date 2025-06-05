# Migration Guide: Adopting Target-Aware Operators

This guide helps you migrate from single-instance external operators to multi-target configurations.

## Overview

Target-aware operators allow you to connect to multiple instances of external services (Vault, AWS, NATS) from a single graft template. This is essential for:

- Multi-environment deployments
- Cross-account/region access
- Gradual migrations between systems
- Partner integrations
- Development/staging/production isolation

## Planning Your Migration

### 1. Assess Current Usage

First, identify all external operator usage in your templates:

```bash
# Find Vault operators
grep -r "(( vault " templates/

# Find AWS operators
grep -r "(( aws" templates/

# Find NATS operators
grep -r "(( nats " templates/
```

### 2. Define Target Structure

Common target patterns:

**By Environment:**
- `production`
- `staging`
- `development`

**By Region:**
- `us-east-1`
- `us-west-2`
- `eu-central-1`

**By Purpose:**
- `shared`
- `partner`
- `legacy`
- `migration`

**By Service:**
- `auth`
- `cache`
- `queue`

### 3. Choose Migration Strategy

#### Option A: Big Bang Migration
- Convert all operators at once
- Higher risk, faster completion
- Best for small codebases

#### Option B: Gradual Migration
- Migrate one service/environment at a time
- Lower risk, takes longer
- Recommended for production systems

#### Option C: Parallel Run
- Keep both old and new configurations
- Highest safety, requires more resources
- Best for critical systems

## Migration Steps

### Step 1: Configure Target Environments

Set up environment variables for each target:

```bash
# production.env
export VAULT_PRODUCTION_ADDR="https://vault.prod.example.com"
export VAULT_PRODUCTION_TOKEN="${PROD_VAULT_TOKEN}"
export AWS_PRODUCTION_REGION="us-east-1"
export AWS_PRODUCTION_ROLE="arn:aws:iam::123456789012:role/ProdRole"
export NATS_PRODUCTION_URL="nats://nats.prod.example.com:4222"

# staging.env
export VAULT_STAGING_ADDR="https://vault.staging.example.com"
export VAULT_STAGING_TOKEN="${STAGING_VAULT_TOKEN}"
export AWS_STAGING_REGION="us-west-2"
export AWS_STAGING_PROFILE="staging"
export NATS_STAGING_URL="nats://nats.staging.example.com:4222"
```

### Step 2: Update Templates

#### Vault Migration

Before:
```yaml
database:
  password: (( vault "secret/db:password" ))
  api_key: (( vault "secret/api:key" ))
```

After:
```yaml
database:
  password: (( vault@production "secret/db:password" ))
  api_key: (( vault@production "secret/api:key" ))
```

#### AWS Migration

Before:
```yaml
config:
  db_host: (( awsparam "/app/database/host" ))
  api_secret: (( awssecret "api-credentials" ))
```

After:
```yaml
config:
  db_host: (( awsparam@production "/app/database/host" ))
  api_secret: (( awssecret@production "api-credentials" ))
```

#### NATS Migration

Before:
```yaml
settings:
  config: (( nats "kv:app/config" ))
  template: (( nats "obj:templates/main.yml" ))
```

After:
```yaml
settings:
  config: (( nats@production "kv:app/config" ))
  template: (( nats@production "obj:templates/main.yml" ))
```

### Step 3: Implement Backward Compatibility

During migration, support both patterns:

```yaml
# Determine target from environment
meta:
  target: (( grab $TARGET || "" ))

database:
  # Use target if specified, otherwise fall back to default
  password: (( 
    grab meta.target && 
    vault (concat "@" meta.target) "secret/db:password" ||
    vault "secret/db:password" 
  ))
```

### Step 4: Test Thoroughly

Create test templates:

```yaml
# test-targets.yml
test_results:
  # Test each target
  vault_prod: (( vault@production "secret/test:value" || "FAIL" ))
  vault_staging: (( vault@staging "secret/test:value" || "FAIL" ))
  
  aws_prod: (( awsparam@production "/test/param" || "FAIL" ))
  aws_staging: (( awsparam@staging "/test/param" || "FAIL" ))
  
  nats_prod: (( nats@production "kv:test/value" || "FAIL" ))
  nats_staging: (( nats@staging "kv:test/value" || "FAIL" ))
```

Run tests:
```bash
graft merge test-targets.yml
```

## Common Patterns

### Multi-Environment with Fallback

```yaml
# Try production first, then staging, then default
database:
  password: (( 
    vault@production "secret/db:password" || 
    vault@staging "secret/db:password" || 
    vault "secret/db:password" 
  ))
```

### Environment-Specific Targets

```yaml
# Use environment variable to select target
environment: (( grab $ENVIRONMENT || "development" ))

database:
  # Dynamic target selection (requires template preprocessing)
  password: (( vault@${environment} "secret/db:password" ))
```

### Cross-Service Configuration

```yaml
# Different services from different targets
services:
  auth:
    url: (( awsparam@shared "/services/auth/url" ))
    token: (( vault@auth "secret/service:token" ))
  
  cache:
    host: (( nats@cache "kv:config/host" ))
    password: (( vault@cache "secret/redis:password" ))
```

### Migration Path Tracking

```yaml
# Track which secrets have been migrated
migration:
  status:
    database:
      old_path: "secret/db:password"
      new_path: "secret/v2/db:password"
      migrated: true
      target: "production"
    
    api:
      old_path: "secret/api:key"
      new_path: "secret/v2/api:key"
      migrated: false
      target: "pending"
```

## Rollback Strategy

### Quick Rollback

Keep environment variables for quick rollback:

```bash
# rollback.sh
#!/bin/bash

# Point production target to old infrastructure
export VAULT_PRODUCTION_ADDR="${OLD_VAULT_ADDR}"
export AWS_PRODUCTION_REGION="${OLD_AWS_REGION}"
export NATS_PRODUCTION_URL="${OLD_NATS_URL}"
```

### Template Rollback

Use feature flags:

```yaml
features:
  use_targets: (( grab $USE_TARGETS || false ))

database:
  password: ((
    grab features.use_targets &&
    vault@production "secret/db:password" ||
    vault "secret/db:password"
  ))
```

## Verification Checklist

- [ ] All target environment variables are set
- [ ] Connection tests pass for each target
- [ ] No hardcoded default operators remain (unless intentional)
- [ ] Cache keys don't conflict between targets
- [ ] Audit logging captures target information
- [ ] Error messages clearly indicate which target failed
- [ ] Performance metrics show expected behavior
- [ ] Rollback procedure is documented and tested

## Performance Considerations

### Connection Pooling

Targets use connection pooling automatically:
- Connections are reused within the same target
- Each target maintains its own connection pool
- Idle connections are cleaned up after 5 minutes

### Caching

Configure cache TTL per target based on data volatility:

```bash
# Long-lived production data
export VAULT_PRODUCTION_CACHE_TTL="30m"

# Frequently changing development data
export VAULT_DEV_CACHE_TTL="1m"
```

### Parallel Execution

Operators from different targets can execute in parallel:

```yaml
# These execute in parallel
data:
  prod_secret: (( vault@production "secret/data" ))
  staging_secret: (( vault@staging "secret/data" ))
  dev_secret: (( vault@development "secret/data" ))
```

## Troubleshooting

### Debug Mode

Enable debug logging to trace target resolution:

```bash
export GRAFT_DEBUG=1
graft merge template.yml 2>&1 | grep -E "(target|@)"
```

### Common Issues

1. **Missing Target Configuration**
   ```
   Error: vault target 'production' configuration not found
   ```
   Solution: Ensure `VAULT_PRODUCTION_ADDR` and `VAULT_PRODUCTION_TOKEN` are set

2. **Wrong Target Name**
   ```
   Error: Unknown target 'prod' (did you mean 'production'?)
   ```
   Solution: Target names are case-sensitive and must match exactly

3. **Cache Conflicts**
   ```
   Getting staging data from production cache
   ```
   Solution: Clear caches when switching between targets rapidly

4. **Connection Limit**
   ```
   Error: too many connections
   ```
   Solution: Connection pooling should prevent this; check for connection leaks

## Best Practices

1. **Naming Conventions**
   - Use consistent target names across services
   - Document target naming scheme
   - Avoid special characters in target names

2. **Security**
   - Use separate credentials per target
   - Implement least-privilege access
   - Rotate credentials regularly
   - Never commit target credentials

3. **Testing**
   - Test each target independently
   - Verify cross-target isolation
   - Test failure scenarios
   - Monitor target performance

4. **Documentation**
   - Document all targets and their purposes
   - Maintain target configuration inventory
   - Document emergency procedures
   - Keep runbooks updated

## Example: Complete Migration

Here's a complete example migrating a microservices configuration:

### Before Migration

```yaml
# config.yml
microservices:
  auth:
    database:
      host: (( awsparam "/services/auth/db_host" ))
      password: (( vault "secret/auth/db:password" ))
    
  api:
    redis:
      host: (( awsparam "/services/api/redis_host" ))
      password: (( vault "secret/api/redis:password" ))
    
  worker:
    queue:
      url: (( nats "kv:config/queue_url" ))
      token: (( vault "secret/worker/queue:token" ))
```

### After Migration

```yaml
# config.yml
microservices:
  auth:
    database:
      host: (( awsparam@production "/services/auth/db_host" ))
      password: (( vault@production "secret/auth/db:password" ))
    
  api:
    redis:
      host: (( awsparam@production "/services/api/redis_host" ))
      password: (( vault@production "secret/api/redis:password" ))
    
  worker:
    queue:
      url: (( nats@production "kv:config/queue_url" ))
      token: (( vault@production "secret/worker/queue:token" ))

# staging-config.yml (overlay)
microservices:
  auth:
    database:
      host: (( awsparam@staging "/services/auth/db_host" ))
      password: (( vault@staging "secret/auth/db:password" ))
  # ... etc
```

### Environment Setup

```bash
# deploy.sh
#!/bin/bash

ENVIRONMENT=$1

case $ENVIRONMENT in
  production)
    source envs/production.env
    TARGET_SUFFIX="@production"
    ;;
  staging)
    source envs/staging.env
    TARGET_SUFFIX="@staging"
    ;;
  *)
    echo "Unknown environment: $ENVIRONMENT"
    exit 1
    ;;
esac

graft merge config.yml ${ENVIRONMENT}-config.yml > final-config.yml
```

This approach provides clear separation between environments while maintaining a single template structure.