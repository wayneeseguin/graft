# AWS Integration Guide

graft integrates with AWS services to retrieve configuration values and secrets from Parameter Store and Secrets Manager.

## AWS Parameter Store

The `(( awsparam ))` operator fetches values from AWS Systems Manager Parameter Store.

### Basic Usage

```yaml
# Simple parameter
api_endpoint: (( awsparam "/myapp/api/endpoint" ))

# JSON parameter with key extraction
database:
  username: (( awsparam "/myapp/database/config?key=username" ))
  password: (( awsparam "/myapp/database/config?key=password" ))
  host: (( awsparam "/myapp/database/config?key=host" ))
```

### Setting Up Parameters

```bash
# Store a simple string
aws ssm put-parameter \
  --name "/myapp/api/endpoint" \
  --value "https://api.example.com" \
  --type String

# Store secure string
aws ssm put-parameter \
  --name "/myapp/database/password" \
  --value "secret-password" \
  --type SecureString \
  --key-id alias/aws/ssm

# Store JSON with multiple values
aws ssm put-parameter \
  --name "/myapp/database/config" \
  --value '{"host":"db.example.com","username":"dbuser","password":"dbpass"}' \
  --type SecureString
```

### Parameter Types

All SSM parameter types are supported:
- **String** - Plain text values
- **SecureString** - Encrypted values (requires KMS permissions)
- **StringList** - Comma-separated values (returned as single string)

### JSON Key Extraction

Use the `?key=` syntax to extract values from JSON parameters:

```yaml
# Parameter contains: {"user": "myuser", "pass": "mypass", "host": "db.example.com"}
config:
  db_user: (( awsparam "/services/myapp/db?key=user" ))
  db_pass: (( awsparam "/services/myapp/db?key=pass" ))
  db_host: (( awsparam "/services/myapp/db?key=host" ))
  
  # Without key, returns raw JSON
  db_raw: (( awsparam "/services/myapp/db" ))
```

## AWS Secrets Manager

The `(( awssecret ))` operator retrieves secrets from AWS Secrets Manager.

### Basic Usage

```yaml
# Simple secret
api_key: (( awssecret "myapp/api-key" ))

# JSON secret with key extraction
database:
  username: (( awssecret "myapp/database?key=username" ))
  password: (( awssecret "myapp/database?key=password" ))
```

### Creating Secrets

```bash
# Store a simple string secret
aws secretsmanager create-secret \
  --name "myapp/api-key" \
  --secret-string "abc123xyz"

# Store JSON secret
aws secretsmanager create-secret \
  --name "myapp/database" \
  --secret-string '{"username":"dbuser","password":"dbpass","host":"db.example.com"}'

# With KMS encryption
aws secretsmanager create-secret \
  --name "myapp/sensitive" \
  --secret-string "very-secret" \
  --kms-key-id alias/aws/secretsmanager
```

### Version and Stage Support

Retrieve specific versions or stages of secrets:

```yaml
# Current version (default)
current_key: (( awssecret "myapp/api-key" ))

# Specific version
pinned_key: (( awssecret "myapp/api-key?version=a1b2c3d4-5678-90ab-cdef-EXAMPLE11111" ))

# Previous version
previous_key: (( awssecret "myapp/api-key?stage=AWSPREVIOUS" ))

# Combine with key extraction
old_password: (( awssecret "myapp/database?stage=AWSPREVIOUS&key=password" ))
```

## Authentication and Configuration

### Environment Variables

Both operators use these environment variables:

- `AWS_REGION` - AWS region (e.g., `us-east-1`)
- `AWS_PROFILE` - AWS profile name from `~/.aws/credentials`
- `AWS_ROLE` - IAM role to assume

### IAM Permissions

#### Parameter Store Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameter",
        "ssm:GetParameters"
      ],
      "Resource": "arn:aws:ssm:*:*:parameter/myapp/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt"
      ],
      "Resource": "arn:aws:kms:*:*:key/*",
      "Condition": {
        "StringEquals": {
          "kms:ViaService": "ssm.*.amazonaws.com"
        }
      }
    }
  ]
}
```

#### Secrets Manager Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:*:*:secret:myapp/*"
    }
  ]
}
```

### Using AWS Profiles

```bash
# Set profile for the session
export AWS_PROFILE=production
export AWS_REGION=us-east-1

# Or specify role to assume
export AWS_ROLE=arn:aws:iam::123456789012:role/MyRole

graft merge config.yml
```

## Best Practices

### 1. Organize Parameters/Secrets Hierarchically

```yaml
# Good structure
parameters:
  base: "/myapp/production"
  
database:
  host: (( awsparam (concat parameters.base "/database/host") ))
  port: (( awsparam (concat parameters.base "/database/port") ))
  
redis:
  host: (( awsparam (concat parameters.base "/redis/host") ))
  port: (( awsparam (concat parameters.base "/redis/port") ))
```

### 2. Use JSON for Related Values

Instead of multiple parameters:
```yaml
# Less efficient - multiple API calls
db_host: (( awsparam "/myapp/db_host" ))
db_user: (( awsparam "/myapp/db_user" ))
db_pass: (( awsparam "/myapp/db_pass" ))
```

Use one JSON parameter:
```yaml
# More efficient - single API call
db_host: (( awsparam "/myapp/database?key=host" ))
db_user: (( awsparam "/myapp/database?key=user" ))
db_pass: (( awsparam "/myapp/database?key=pass" ))
```

### 3. Environment-Specific Paths

```yaml
environment: (( grab $ENVIRONMENT || "development" ))

# Build paths based on environment
database:
  password: (( awssecret (concat "myapp/" environment "/database?key=password") ))
  
api:
  key: (( awsparam (concat "/myapp/" environment "/api-key") ))
```

### 4. Fallback Values

```yaml
# With defaults
database:
  host: (( awsparam "/myapp/database/host" || "localhost" ))
  port: (( awsparam "/myapp/database/port" || 5432 ))
  
# With fallback to different source
api_key: (( awssecret "myapp/api-key" || awsparam "/myapp/api-key" || grab $API_KEY ))
```

## Complete Example

```yaml
# config.yml
meta:
  app: myapp
  environment: (( grab $ENVIRONMENT || "development" ))
  aws_region: (( grab $AWS_REGION || "us-east-1" ))
  
  # Path prefixes
  param_prefix: (( concat "/" meta.app "/" meta.environment ))
  secret_prefix: (( concat meta.app "/" meta.environment ))

# Application configuration from Parameter Store
app:
  name: (( meta.app ))
  environment: (( meta.environment ))
  version: (( awsparam (concat meta.param_prefix "/version") || "latest" ))
  
  # Feature flags
  features:
    new_ui: (( awsparam (concat meta.param_prefix "/features/new_ui") || false ))
    beta_api: (( awsparam (concat meta.param_prefix "/features/beta_api") || false ))

# Database configuration from Secrets Manager
database:
  primary:
    host: (( awssecret (concat meta.secret_prefix "/database/primary?key=host") ))
    port: (( awssecret (concat meta.secret_prefix "/database/primary?key=port") || 5432 ))
    username: (( awssecret (concat meta.secret_prefix "/database/primary?key=username") ))
    password: (( awssecret (concat meta.secret_prefix "/database/primary?key=password") ))
    
  read_replica:
    host: (( awssecret (concat meta.secret_prefix "/database/replica?key=host") ))
    port: (( awssecret (concat meta.secret_prefix "/database/replica?key=port") || 5432 ))

# External service credentials
services:
  stripe:
    public_key: (( awsparam (concat meta.param_prefix "/stripe/public_key") ))
    secret_key: (( awssecret (concat meta.secret_prefix "/stripe/secret_key") ))
    
  datadog:
    api_key: (( awssecret (concat meta.secret_prefix "/datadog/api_key") ))
    app_key: (( awssecret (concat meta.secret_prefix "/datadog/app_key") ))
    
  redis:
    # Using Parameter Store for non-sensitive config
    host: (( awsparam (concat meta.param_prefix "/redis/host") ))
    port: (( awsparam (concat meta.param_prefix "/redis/port") || 6379 ))
    # Using Secrets Manager for auth
    auth_token: (( awssecret (concat meta.secret_prefix "/redis/auth_token") || "" ))

# TLS certificates from Secrets Manager
tls:
  cert: (( awssecret (concat meta.secret_prefix "/tls/certificate") ))
  key: (( awssecret (concat meta.secret_prefix "/tls/private_key") ))
  ca_bundle: (( awssecret (concat meta.secret_prefix "/tls/ca_bundle") || "" ))

# Build connection strings
connections:
  database_url: (( concat 
    "postgresql://"
    database.primary.username ":"
    database.primary.password "@"
    database.primary.host ":"
    database.primary.port "/"
    meta.app "_" meta.environment
  ))
  
  redis_url: (( concat
    "redis://"
    (services.redis.auth_token ? ":" services.redis.auth_token "@" : "")
    services.redis.host ":"
    services.redis.port
  ))
```

## Usage Workflow

```bash
# 1. Set AWS credentials
export AWS_PROFILE=mycompany
export AWS_REGION=us-east-1

# 2. Development environment
ENVIRONMENT=development graft merge config.yml > dev-config.yml

# 3. Production environment with role assumption
export AWS_ROLE=arn:aws:iam::123456789012:role/ProductionRole
ENVIRONMENT=production graft merge config.yml > prod-config.yml

# 4. Clean up
rm -f dev-config.yml prod-config.yml
```

## Troubleshooting

### Access Denied

```bash
# Check your permissions
aws ssm get-parameter --name "/myapp/test"
aws secretsmanager get-secret-value --secret-id "myapp/test"

# Verify KMS key access for SecureString/encrypted secrets
aws kms decrypt --key-id alias/aws/ssm --ciphertext-blob ...
```

### Parameter Not Found

```bash
# List parameters to verify path
aws ssm get-parameters-by-path --path "/myapp/" --recursive

# List secrets
aws secretsmanager list-secrets --filters Key=name,Values=myapp
```

### Region Issues

```bash
# Ensure region is set correctly
echo $AWS_REGION

# Or specify in AWS CLI
aws ssm get-parameter --name "/myapp/test" --region us-east-1
```

## Migration Examples

### From Environment Variables

Before:
```yaml
database:
  host: (( grab $DB_HOST ))
  password: (( grab $DB_PASSWORD ))
```

After:
```yaml
database:
  host: (( awsparam "/myapp/database/host" ))
  password: (( awssecret "myapp/database?key=password" ))
```

### From Vault

Before:
```yaml
api_key: (( vault "secret/myapp:api_key" ))
```

After:
```yaml
api_key: (( awssecret "myapp/api_key" ))
```

## See Also

- [AWS Parameter Store Operator](../operators/external-data.md#awsparam)
- [AWS Secrets Manager Operator](../operators/external-data.md#awssecret)
- [Vault Integration](vault-integration.md) - Alternative secret management
- [Environment Variables](../concepts/environment-variables.md)