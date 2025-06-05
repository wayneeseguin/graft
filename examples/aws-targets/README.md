# AWS Operators with Targets

This example demonstrates how to use the AWS operators with different targets to connect to multiple AWS accounts/regions.

## Overview

The AWS operators now support target syntax to specify which AWS account/region to connect to:

```yaml
# Connect to production AWS account
production_password: (( awsparam@production "/app/database/password" ))

# Connect to staging AWS account
staging_password: (( awsparam@staging "/app/database/password" ))

# Connect to production Secrets Manager
production_secret: (( awssecret@production "database-credentials" ))

# Use default AWS configuration
default_password: (( awsparam "/app/database/password" ))
```

## Configuration

### Environment Variables

For each target, you need to set environment variables:

```bash
# Production AWS
export AWS_PRODUCTION_REGION="us-east-1"
export AWS_PRODUCTION_PROFILE="production"
export AWS_PRODUCTION_ROLE="arn:aws:iam::123456789012:role/GraftRole"
export AWS_PRODUCTION_ACCESS_KEY_ID="AKIAPRODUCTION123"
export AWS_PRODUCTION_SECRET_ACCESS_KEY="production-secret-key"
export AWS_PRODUCTION_MAX_RETRIES="5"
export AWS_PRODUCTION_HTTP_TIMEOUT="60s"
export AWS_PRODUCTION_CACHE_TTL="15m"
export AWS_PRODUCTION_AUDIT_LOGGING="true"
export AWS_PRODUCTION_ASSUME_ROLE_DURATION="2h"
export AWS_PRODUCTION_EXTERNAL_ID="external-id-prod"
export AWS_PRODUCTION_SESSION_NAME="graft-production"

# Staging AWS
export AWS_STAGING_REGION="us-west-2"
export AWS_STAGING_PROFILE="staging"
export AWS_STAGING_ACCESS_KEY_ID="AKIASTAGING456"
export AWS_STAGING_SECRET_ACCESS_KEY="staging-secret-key"
export AWS_STAGING_MAX_RETRIES="3"
export AWS_STAGING_HTTP_TIMEOUT="30s"
export AWS_STAGING_CACHE_TTL="5m"

# Development AWS (using custom endpoint for LocalStack)
export AWS_DEV_REGION="us-east-1"
export AWS_DEV_ACCESS_KEY_ID="AKIADEV789"
export AWS_DEV_SECRET_ACCESS_KEY="dev-secret-key"
export AWS_DEV_ENDPOINT="http://localhost:4566"
export AWS_DEV_DISABLE_SSL="true"
export AWS_DEV_S3_FORCE_PATH_STYLE="true"

# Default AWS (existing behavior)
export AWS_REGION="us-east-1"
export AWS_PROFILE="default"
```

### Configuration File (Future)

In future versions, targets will be configurable via a configuration file:

```yaml
# ~/.graft/config.yml
targets:
  aws:
    production:
      region: "${AWS_PROD_REGION}"
      profile: "production"
      role: "${AWS_PROD_ROLE}"
      max_retries: 5
      http_timeout: "60s"
      cache_ttl: "15m"
      audit_logging: true
      assume_role_duration: "2h"
      external_id: "${AWS_PROD_EXTERNAL_ID}"
      session_name: "graft-production"
    staging:
      region: "us-west-2"
      profile: "staging"
      access_key_id: "${AWS_STAGING_ACCESS_KEY_ID}"
      secret_access_key: "${AWS_STAGING_SECRET_ACCESS_KEY}"
      max_retries: 3
      http_timeout: "30s"
      cache_ttl: "5m"
    dev:
      region: "us-east-1"
      endpoint: "http://localhost:4566"
      disable_ssl: true
      s3_force_path_style: true
      access_key_id: "test"
      secret_access_key: "test"
```

## Usage Examples

### Basic Target Usage

```yaml
# main.yml
app:
  database:
    # Production database credentials
    prod_password: (( awsparam@production "/app/prod/database/password" ))
    prod_username: (( awsparam@production "/app/prod/database/username" ))
    
    # Staging database credentials
    staging_password: (( awsparam@staging "/app/staging/database/password" ))
    staging_username: (( awsparam@staging "/app/staging/database/username" ))
    
    # Development database credentials (default configuration)
    dev_password: (( awsparam "/app/dev/database/password" ))
```

### Secrets Manager Access

```yaml
# Fetch JSON secrets and extract specific keys
database:
  production:
    credentials: (( awssecret@production "prod/database/credentials?key=password" ))
    connection_string: (( awssecret@production "prod/database/connection" ))
    
  staging:
    credentials: (( awssecret@staging "staging/database/credentials?key=password" ))
    connection_string: (( awssecret@staging "staging/database/connection" ))
```

### Cross-Account Access

```yaml
# Access resources across different AWS accounts
api:
  # Production API keys from production account
  production_key: (( awsparam@production "/api/keys/production" ))
  
  # Partner integration keys from partner account
  partner_key: (( awsparam@partner "/api/keys/integration" ))
  
  # Shared services from shared account
  shared_service_url: (( awsparam@shared "/services/shared/url" ))
```

### With Defaults and Fallbacks

```yaml
# Use target with fallback to default
api_key: (( awsparam@production "/api/key" || awsparam "/api/fallback_key" ))

# Multiple fallbacks across targets
secret: (( awssecret@production "app-secret" || awssecret@staging "app-secret" || "default-secret" ))
```

### Complex Expressions

```yaml
# Build connection strings using multiple parameters
database:
  url: (( concat "postgresql://" (awsparam@production "/db/user") ":" (awssecret@production "db-password") "@" (awsparam@production "/db/host") ":5432/app" ))
  
# Use versioned secrets
api:
  current_key: (( awssecret@production "api-key" ))
  previous_key: (( awssecret@production "api-key?version=1" ))
  staging_key: (( awssecret@production "api-key?stage=AWSPENDING" ))
```

## Features

### Supported Services

- **Parameter Store**: `(( awsparam@target "parameter-name" ))`
  - Supports hierarchical parameters
  - Automatic decryption of SecureString parameters
  - Cross-account parameter access

- **Secrets Manager**: `(( awssecret@target "secret-name" ))`
  - Supports JSON secrets with key extraction
  - Version and stage support
  - Cross-account secret access

### Authentication Methods

- **IAM Roles**: Assume roles across accounts
- **Access Keys**: Direct access key authentication
- **AWS Profiles**: Named profile configuration
- **Instance Profiles**: EC2 instance role authentication
- **Cross-Account Access**: Role assumption with external IDs

### Advanced Features

- **Session Management**: Efficient session reuse and credential caching
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **MFA Support**: Multi-factor authentication for role assumption
- **Custom Endpoints**: Support for LocalStack and custom AWS endpoints
- **Audit Logging**: Optional audit logging for compliance requirements
- **Caching**: Target-aware caching to prevent cross-contamination

### Error Handling

Clear error messages when targets are not configured:
```
Error: AWS target 'production' configuration incomplete (expected AWS_PRODUCTION_REGION, AWS_PRODUCTION_PROFILE, AWS_PRODUCTION_ROLE, or AWS_PRODUCTION_ACCESS_KEY_ID environment variable)
```

### Backward Compatibility

Existing AWS operators without targets continue to work unchanged:
```yaml
# Still works as before
password: (( awsparam "/app/database/password" ))
secret: (( awssecret "database-credentials" ))
```

## Security Considerations

### Cross-Account Access

When accessing resources across AWS accounts:

1. **Use IAM Roles**: Prefer role assumption over access keys
2. **External IDs**: Use external IDs for enhanced security
3. **Least Privilege**: Grant minimal required permissions
4. **Session Duration**: Use short-lived sessions when possible

### Credential Management

- **Environment Variables**: Use secure environment variable management
- **Secrets Storage**: Store sensitive credentials in encrypted storage
- **Rotation**: Implement regular credential rotation
- **Audit Logging**: Enable audit logging for compliance

## Migration Guide

### Step 1: Identify Current Usage

Find all AWS operators in your templates:
```bash
grep -r "(( aws" templates/
```

### Step 2: Plan Target Structure

Define your target structure based on:
- AWS accounts (production, staging, development)
- Regions (us-east-1, us-west-2, eu-west-1)
- Environments (prod, staging, dev)

### Step 3: Set Up Target Configuration

Configure environment variables for each target:

```bash
# Production
export AWS_PRODUCTION_REGION="us-east-1"
export AWS_PRODUCTION_ROLE="arn:aws:iam::123456789012:role/GraftRole"

# Staging  
export AWS_STAGING_REGION="us-west-2"
export AWS_STAGING_PROFILE="staging"
```

### Step 4: Update Templates

Replace AWS operators with target-specific versions:

```yaml
# Before
database_password: (( awsparam "/app/database/password" ))

# After
database_password: (( awsparam@production "/app/database/password" ))
```

### Step 5: Test

Verify that:
1. Target-specific AWS calls work correctly
2. Default AWS calls still work (backward compatibility)
3. Caching works independently for each target
4. Cross-account access works as expected

## Troubleshooting

### Common Issues

1. **Missing configuration**: Ensure required AWS target environment variables are set
2. **Wrong target name**: Target names are case-sensitive and must match environment variable suffixes
3. **Cache conflicts**: Different targets use separate caches automatically
4. **Permission errors**: Verify IAM permissions for cross-account access
5. **Session expiration**: Check assume role duration settings

### Debug Mode

Enable debug logging to see AWS target resolution:
```bash
export GRAFT_DEBUG=1
graft merge template.yml
```

Look for log messages like:
```
aws: using target 'production'
aws: using target-specific session for 'production'
AUDIT: Accessing AWS awsparam: /app/database/password (target: production)
aws: Cache hit for awsparam:/app/database/password (target: production)
```

### Performance Tuning

- **Cache TTL**: Adjust `AWS_{TARGET}_CACHE_TTL` based on parameter change frequency
- **Session Reuse**: Sessions are automatically cached and reused
- **Retry Settings**: Configure retries and timeouts for unreliable networks
- **Regional Optimization**: Use regions closest to your infrastructure

### LocalStack Integration

For local development with LocalStack:

```bash
export AWS_DEV_REGION="us-east-1"
export AWS_DEV_ENDPOINT="http://localhost:4566"
export AWS_DEV_ACCESS_KEY_ID="test"
export AWS_DEV_SECRET_ACCESS_KEY="test"
export AWS_DEV_DISABLE_SSL="true"
export AWS_DEV_S3_FORCE_PATH_STYLE="true"
```

## Implementation Notes

- Target extraction is currently a placeholder (returns empty string)
- Full target extraction from parsed expressions will be implemented in the next iteration
- Environment variable configuration is the current approach; configuration file support is planned
- Session pooling ensures efficient reuse of AWS sessions per target
- Cross-account access requires proper IAM role trust relationships