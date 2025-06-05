# Target-Aware Operators

This guide explains how to use target-aware operators in graft to connect to multiple instances of external services like Vault, AWS, and NATS.

## Overview

Target-aware operators allow you to specify which instance of an external service to connect to using the `@target` syntax:

```yaml
# Connect to production Vault
production_password: (( vault@production "secret/database:password" ))

# Connect to staging AWS
staging_config: (( awsparam@staging "/app/config" ))

# Connect to development NATS
dev_setting: (( nats@dev "kv:settings/app" ))
```

## Supported Operators

The following operators support targets:

- **vault** - HashiCorp Vault
- **awsparam** - AWS Systems Manager Parameter Store
- **awssecret** - AWS Secrets Manager
- **nats** - NATS JetStream KV/Object stores

## Configuration

Target configuration is currently managed through environment variables. Each target requires specific environment variables following the pattern `{SERVICE}_{TARGET}_*`.

### Vault Targets

```bash
# Production Vault
export VAULT_PRODUCTION_ADDR="https://vault.prod.example.com"
export VAULT_PRODUCTION_TOKEN="prod-token-12345"
export VAULT_PRODUCTION_NAMESPACE="production"
export VAULT_PRODUCTION_SKIP_VERIFY="false"

# Staging Vault
export VAULT_STAGING_ADDR="https://vault.staging.example.com"
export VAULT_STAGING_TOKEN="staging-token-67890"
export VAULT_STAGING_NAMESPACE="staging"
```

### AWS Targets

```bash
# Production AWS
export AWS_PRODUCTION_REGION="us-east-1"
export AWS_PRODUCTION_PROFILE="production"
export AWS_PRODUCTION_ROLE="arn:aws:iam::123456789012:role/GraftRole"
export AWS_PRODUCTION_AUDIT_LOGGING="true"

# Staging AWS
export AWS_STAGING_REGION="us-west-2"
export AWS_STAGING_ACCESS_KEY_ID="AKIASTAGING456"
export AWS_STAGING_SECRET_ACCESS_KEY="staging-secret-key"
```

### NATS Targets

```bash
# Production NATS
export NATS_PRODUCTION_URL="nats://nats.prod.example.com:4222"
export NATS_PRODUCTION_TLS="true"
export NATS_PRODUCTION_CERT_FILE="/etc/ssl/certs/nats-prod.crt"
export NATS_PRODUCTION_KEY_FILE="/etc/ssl/private/nats-prod.key"

# Development NATS
export NATS_DEV_URL="nats://localhost:4222"
export NATS_DEV_TLS="false"
```

## Usage Examples

### Basic Usage

```yaml
# database.yml
database:
  production:
    host: (( awsparam@production "/db/prod/host" ))
    port: (( awsparam@production "/db/prod/port" ))
    password: (( vault@production "secret/db/prod:password" ))
    
  staging:
    host: (( awsparam@staging "/db/staging/host" ))
    port: (( awsparam@staging "/db/staging/port" ))
    password: (( vault@staging "secret/db/staging:password" ))
```

### Cross-Environment Configuration

```yaml
# app-config.yml
app:
  # Try production first, fall back to staging
  api_key: (( vault@production "secret/api:key" || vault@staging "secret/api:key" ))
  
  # Build connection string from multiple sources
  redis_url: (( concat "redis://:" (vault@production "secret/redis:password") "@" (awsparam@production "/redis/host") ":6379" ))
```

### Multi-Cloud Setup

```yaml
# multi-cloud.yml
infrastructure:
  # AWS resources
  aws:
    database_url: (( awsparam@aws_prod "/rds/connection_string" ))
    s3_bucket: (( awsparam@aws_prod "/s3/bucket_name" ))
    
  # Azure resources (via Vault)
  azure:
    storage_key: (( vault@azure_prod "secret/storage:key" ))
    cosmos_url: (( vault@azure_prod "secret/cosmos:connection_string" ))
    
  # GCP resources (via NATS)
  gcp:
    project_id: (( nats@gcp_prod "kv:config/project_id" ))
    service_account: (( nats@gcp_prod "obj:credentials/service-account.json" ))
```

## Features

### Connection Pooling

All target-aware operators implement connection pooling to efficiently reuse connections:

- Connections are cached and reused across operator calls
- Automatic cleanup of idle connections
- Thread-safe connection management

### Target-Aware Caching

Each target maintains its own cache to prevent data conflicts:

- Cache keys include target information
- Configurable cache TTL per target
- Independent cache invalidation

### Audit Logging

Enable audit logging for compliance and debugging:

```bash
export VAULT_PRODUCTION_AUDIT_LOGGING="true"
export AWS_PRODUCTION_AUDIT_LOGGING="true"
export NATS_PRODUCTION_AUDIT_LOGGING="true"
```

When enabled, you'll see audit logs like:
```
AUDIT: Accessing Vault secret: secret/database (target: production)
AUDIT: Successfully retrieved secret: secret/database (target: production)
```

### Error Handling

Clear error messages help identify configuration issues:

```
Error: vault target 'production' configuration not found (expected VAULT_PRODUCTION_ADDR and VAULT_PRODUCTION_TOKEN environment variables)
```

## Advanced Configuration

### AWS Cross-Account Access

Configure cross-account role assumption:

```bash
# Partner account access
export AWS_PARTNER_ROLE="arn:aws:iam::987654321098:role/PartnerRole"
export AWS_PARTNER_EXTERNAL_ID="shared-external-id-12345"
export AWS_PARTNER_SESSION_NAME="graft-partner-access"
export AWS_PARTNER_ASSUME_ROLE_DURATION="1h"
```

Usage:
```yaml
partner_data: (( awsparam@partner "/shared/configuration" ))
```

### NATS with TLS

Configure secure NATS connections:

```bash
export NATS_PRODUCTION_TLS="true"
export NATS_PRODUCTION_CERT_FILE="/etc/nats/client.crt"
export NATS_PRODUCTION_KEY_FILE="/etc/nats/client.key"
export NATS_PRODUCTION_CA_FILE="/etc/nats/ca.crt"
export NATS_PRODUCTION_INSECURE_SKIP_VERIFY="false"
```

### LocalStack for Development

Use LocalStack for local AWS development:

```bash
export AWS_DEV_ENDPOINT="http://localhost:4566"
export AWS_DEV_REGION="us-east-1"
export AWS_DEV_ACCESS_KEY_ID="test"
export AWS_DEV_SECRET_ACCESS_KEY="test"
export AWS_DEV_DISABLE_SSL="true"
export AWS_DEV_S3_FORCE_PATH_STYLE="true"
```

## Performance Tuning

### Cache Configuration

Optimize cache settings based on your use case:

```bash
# Long-lived production data
export VAULT_PRODUCTION_CACHE_TTL="30m"
export AWS_PRODUCTION_CACHE_TTL="15m"

# Frequently changing staging data
export VAULT_STAGING_CACHE_TTL="1m"
export AWS_STAGING_CACHE_TTL="30s"
```

### Connection Settings

Configure timeouts and retries:

```bash
# Production - high reliability
export AWS_PRODUCTION_MAX_RETRIES="5"
export AWS_PRODUCTION_HTTP_TIMEOUT="60s"
export NATS_PRODUCTION_RETRIES="5"
export NATS_PRODUCTION_RETRY_BACKOFF="2.0"

# Development - fast failures
export AWS_DEV_MAX_RETRIES="1"
export AWS_DEV_HTTP_TIMEOUT="5s"
```

## Security Best Practices

1. **Use Environment Variables Securely**
   - Store sensitive values in secure secret management systems
   - Use CI/CD secret injection
   - Avoid committing credentials to version control

2. **Implement Least Privilege**
   - Grant minimal required permissions
   - Use separate credentials per environment
   - Implement role-based access control

3. **Enable Audit Logging**
   - Track access to sensitive data
   - Monitor for unusual access patterns
   - Retain logs for compliance

4. **Rotate Credentials Regularly**
   - Implement automated credential rotation
   - Use short-lived tokens where possible
   - Monitor credential age

## Migration Guide

### From Single to Multi-Target

Before (single Vault):
```yaml
password: (( vault "secret/database:password" ))
```

After (multi-target):
```yaml
password: (( vault@production "secret/database:password" ))
```

### Backward Compatibility

Operators without targets continue to work:
```yaml
# These still work
password: (( vault "secret/database:password" ))
param: (( awsparam "/app/config" ))
```

### Gradual Migration

1. Start with non-critical environments
2. Test target configuration thoroughly
3. Update templates incrementally
4. Monitor for issues

## Troubleshooting

### Debug Mode

Enable debug logging:
```bash
export GRAFT_DEBUG=1
graft merge template.yml
```

### Common Issues

1. **Missing Configuration**
   ```
   Error: AWS target 'production' configuration incomplete
   ```
   Solution: Ensure all required environment variables are set

2. **Connection Failures**
   ```
   Error: failed to create NATS connection for target 'production'
   ```
   Solution: Verify network connectivity and credentials

3. **Cache Conflicts**
   ```
   Error: Unexpected value returned
   ```
   Solution: Clear caches or reduce cache TTL

### Verification Commands

Test target configuration:
```bash
# Test Vault connection
vault login -address=$VAULT_PRODUCTION_ADDR $VAULT_PRODUCTION_TOKEN
vault list -address=$VAULT_PRODUCTION_ADDR secret/

# Test AWS connection
aws sts get-caller-identity --profile production

# Test NATS connection
nats --server=$NATS_PRODUCTION_URL server check connection
```

## Future Enhancements

### Configuration File Support

Future versions will support configuration files:

```yaml
# ~/.graft/config.yml
targets:
  vault:
    production:
      url: "${VAULT_PROD_ADDR}"
      token: "${VAULT_PROD_TOKEN}"
    staging:
      url: "${VAULT_STAGING_ADDR}"
      token: "${VAULT_STAGING_TOKEN}"
```

### Dynamic Target Resolution

Planned support for dynamic target selection:

```yaml
# Use environment-based target
password: (( vault@${ENVIRONMENT} "secret/database:password" ))
```

### Target Discovery

Automatic target discovery from cloud provider APIs:

```yaml
# Discover targets from AWS tags
targets:
  aws:
    discover:
      filter: "tag:Environment=*"
      region: "us-east-1"
```