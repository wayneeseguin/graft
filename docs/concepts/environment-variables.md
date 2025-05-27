# Environment Variables in graft

graft provides two powerful ways to work with environment variables: using them as literal values and expanding them within reference paths.

## Using Environment Variables as Values

The simplest way to use environment variables is with the `$` prefix:

```yaml
database_url: (( grab $DATABASE_URL ))
api_key: (( grab $API_KEY ))
debug_mode: (( grab $DEBUG_ENABLED ))
```

If the environment variable is not set, graft will error. To provide defaults, use the `||` operator:

```yaml
port: (( grab $PORT || 8080 ))
environment: (( grab $APP_ENV || "development" ))
log_level: (( grab $LOG_LEVEL || "info" ))
```

## Environment Variable Expansion in References

A more advanced feature allows environment variables to be expanded within reference paths using the `$VAR` syntax:

```yaml
# If ENVIRONMENT="production", this grabs config.production.database
database: (( grab config.$ENVIRONMENT.database ))

# Multiple variables can be used
setting: (( grab $TIER.$REGION.$SERVICE.endpoint ))
```

### How Expansion Works

1. Any path segment starting with `$` is treated as an environment variable
2. The variable name extends to the next `.` or end of path
3. If the environment variable is not set, it expands to an empty string
4. This works in any operator that accepts references

### Common Patterns

#### Environment-Specific Configuration

```yaml
environments:
  development:
    api_url: http://localhost:8080
    debug: true
  staging:
    api_url: https://staging.example.com
    debug: true
  production:
    api_url: https://api.example.com
    debug: false

# Select config based on APP_ENV
current: (( grab environments.$APP_ENV ))
api_url: (( grab environments.$APP_ENV.api_url ))
```

#### Multi-Tenant Systems

```yaml
tenants:
  acme:
    api_key: abc123
    plan: enterprise
  globex:
    api_key: xyz789
    plan: starter

# Use TENANT environment variable
tenant_config: (( grab tenants.$TENANT ))
api_key: (( grab tenants.$TENANT.api_key ))
```

#### Regional Deployments

```yaml
regions:
  us-east-1:
    ami: ami-12345678
    vpc: vpc-abcd1234
  eu-west-1:
    ami: ami-87654321
    vpc: vpc-efgh5678

# AWS_REGION from CI/CD
deployment:
  ami: (( grab regions.$AWS_REGION.ami ))
  vpc: (( grab regions.$AWS_REGION.vpc ))
```

## Default Values with `||` Operator

The logical-or operator provides fallback values:

```yaml
# Simple default
value: (( grab original.value || "default-value" ))

# Reference as default
value: (( grab primary.value || secondary.value ))

# Multiple fallbacks
value: (( grab primary || secondary || tertiary || "default" ))

# With environment variables
database_host: (( grab $DB_HOST || config.database.host || "localhost" ))
```

### Chaining Defaults

```yaml
# Try multiple sources in order
api_endpoint: (( 
  grab $API_ENDPOINT_OVERRIDE ||
  grab endpoints.$ENVIRONMENT ||
  grab endpoints.default ||
  "http://localhost:8080"
))
```

## Advanced Usage

### Dynamic Path Construction

```yaml
# Build paths from multiple variables
base_path: (( concat "services." $NAMESPACE "." $SERVICE ))
endpoint: (( grab (grab base_path).url ))

# Conditional paths
config: (( grab $USE_CUSTOM == "true" ? custom.$PROFILE : defaults ))
```

### Combining with Other Operators

```yaml
# With vault
secrets:
  path: (( concat "secret/" $ENVIRONMENT "/" $APP_NAME ))
  password: (( vault (concat (grab secrets.path) ":password") ))

# With concat
connection_string: (( concat 
  (grab $DB_TYPE || "postgresql") "://"
  (grab $DB_USER || "app") ":"
  (grab $DB_PASS || "password") "@"
  (grab $DB_HOST || "localhost") ":"
  (grab $DB_PORT || "5432") "/"
  (grab $DB_NAME || "myapp")
))
```

### Lists and Environment Variables

```yaml
# Select from lists using environment variables
environments: [dev, staging, prod]
current_env: (( grab environments.[$ENV_INDEX] ))

# Build lists dynamically
allowed_origins:
  - (( grab $ORIGIN_1 || "http://localhost:3000" ))
  - (( grab $ORIGIN_2 || "http://localhost:8080" ))
  - (( grab $ORIGIN_3 || "" ))
```

## Best Practices

### 1. Document Required Variables

Always document which environment variables your configuration expects:

```yaml
# Required Environment Variables:
# - APP_ENV: Application environment (development|staging|production)
# - DB_HOST: Database hostname
# - API_KEY: External API key
#
# Optional:
# - PORT: Server port (default: 8080)
# - LOG_LEVEL: Logging level (default: info)
```

### 2. Validate Early

Check for required variables at the start:

```yaml
# Will fail fast if required vars are missing
required:
  environment: (( grab $APP_ENV ))
  database_url: (( grab $DATABASE_URL ))
  api_key: (( grab $API_KEY ))

# Then use them with confidence
config:
  env: (( grab required.environment ))
```

### 3. Use Meaningful Names

Establish naming conventions:
- `APP_` prefix for application settings
- `DB_` prefix for database settings
- `AWS_` prefix for AWS-related settings

### 4. Avoid Secrets in Environment Variables

Use Vault for sensitive data:

```yaml
# Bad - password in environment variable
password: (( grab $DB_PASSWORD ))

# Good - password from Vault
password: (( vault "secret/db:password" ))

# OK - Vault path from environment
password: (( vault (concat $VAULT_PATH ":password") ))
```

### 5. Provide Sensible Defaults

```yaml
config:
  port: (( grab $PORT || 8080 ))
  workers: (( grab $WORKER_COUNT || 4 ))
  timeout: (( grab $TIMEOUT_SECONDS || 30 ))
  retries: (( grab $MAX_RETRIES || 3 ))
```

## Debugging

### Testing Variable Expansion

```bash
# Test specific expansion
echo 'value: (( grab config.$MY_VAR.key ))' | MY_VAR=test graft merge -

# Debug mode shows expansion
graft merge --debug config.yml

# List all environment variables
env | sort
```

### Common Issues

1. **Variable not set**: Use `||` for defaults
2. **Empty expansion**: Check variable spelling and case
3. **Path not found**: Verify the expanded path exists
4. **Special characters**: Stick to alphanumeric and underscore

## Security Considerations

1. Environment variables are visible in process listings
2. Don't store secrets directly in environment variables
3. Be cautious with user-supplied environment variables
4. Validate and sanitize when possible
5. Use Vault or other secret management for sensitive data

## Complete Example

```yaml
# config.yml
app:
  name: (( grab $APP_NAME || "my-app" ))
  version: (( grab $APP_VERSION || "1.0.0" ))
  environment: (( grab $APP_ENV || "development" ))

# Environment-specific settings
environments:
  development:
    debug: true
    database:
      host: localhost
      port: 5432
    cache:
      enabled: false
  production:
    debug: false
    database:
      host: (( grab $DB_HOST ))
      port: (( grab $DB_PORT || 5432 ))
    cache:
      enabled: true
      ttl: (( grab $CACHE_TTL || 3600 ))

# Select current environment
current: (( grab environments.$APP_ENV ))

# Build connection strings
database:
  url: (( concat 
    "postgresql://"
    (grab $DB_USER || "postgres") ":"
    (grab $DB_PASS || "password") "@"
    (grab current.database.host) ":"
    (grab current.database.port) "/"
    (grab $DB_NAME || app.name)
  ))

# Feature flags from environment
features:
  new_ui: (( grab $FEATURE_NEW_UI || current.debug ))
  analytics: (( grab $FEATURE_ANALYTICS || false ))
  beta: (( grab $FEATURE_BETA || current.environment == "development" ))
```

## See Also

- [Operators Reference](../operators/data-references.md) - grab and other operators
- [Vault Integration](../guides/vault-integration.md) - For sensitive data
- [Examples](../../examples/README.md) - More usage examples