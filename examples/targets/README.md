# Target-Aware Operators Examples

This directory contains comprehensive examples demonstrating the use of target-aware operators in graft for various real-world scenarios.

## Examples Overview

### 1. Multi-Cloud Infrastructure (`multi-cloud.yml`)

Demonstrates managing resources across multiple cloud providers:
- **AWS**: Multiple accounts (production, shared, partner)
- **Google Cloud**: Via Vault targets
- **Azure**: Via Vault targets
- **NATS**: Multiple clusters for different purposes

Key patterns shown:
- Cross-cloud service mesh configuration
- Multi-region disaster recovery
- Global CDN with fallbacks
- Compliance and data residency
- Cost allocation across clouds

### 2. Microservices Architecture (`microservices.yml`)

Shows a complete microservices setup with:
- **Authentication Service**: JWT, OAuth providers
- **User Service**: Database replicas, caching, storage
- **Notification Service**: Email, SMS, push notifications
- **API Gateway**: Rate limiting, service discovery

Key patterns shown:
- Service-specific configuration sources
- Shared infrastructure components
- Environment-specific overrides
- Feature flag management
- Observability stack integration

### 3. Migration Scenario (`migration.yml`)

Illustrates gradual migration between systems:
- Legacy to modern infrastructure
- Vault v1 to v2 migration
- Cross-region failover
- Blue-green deployments

### 4. Development Workflow (`development.yml`)

Shows development and testing patterns:
- LocalStack for AWS development
- Local NATS for messaging
- Mock services configuration
- Test data management

## Running the Examples

### Prerequisites

1. Set up environment variables for your targets:

```bash
# Source the appropriate environment file
source environments/production.env
# or
source environments/staging.env
# or
source environments/development.env
```

2. Verify target connectivity:

```bash
# Test script to verify all targets are accessible
./test-targets.sh
```

### Basic Usage

Merge a single example:
```bash
graft merge multi-cloud.yml
```

Merge with environment-specific overrides:
```bash
graft merge microservices.yml environment-overrides.yml
```

### Environment Files

Create environment files for each target configuration:

#### environments/production.env
```bash
# Vault targets
export VAULT_PRODUCTION_ADDR="https://vault.prod.example.com"
export VAULT_PRODUCTION_TOKEN="${PROD_VAULT_TOKEN}"
export VAULT_GCP_PROD_ADDR="https://vault-gcp.prod.example.com"
export VAULT_GCP_PROD_TOKEN="${GCP_VAULT_TOKEN}"
export VAULT_AZURE_PROD_ADDR="https://vault-azure.prod.example.com"
export VAULT_AZURE_PROD_TOKEN="${AZURE_VAULT_TOKEN}"

# AWS targets
export AWS_PRODUCTION_REGION="us-east-1"
export AWS_PRODUCTION_ROLE="arn:aws:iam::123456789012:role/GraftProdRole"
export AWS_SHARED_REGION="us-east-1"
export AWS_SHARED_ROLE="arn:aws:iam::111111111111:role/GraftSharedRole"
export AWS_PARTNER_REGION="us-east-1"
export AWS_PARTNER_ROLE="arn:aws:iam::222222222222:role/GraftPartnerRole"

# NATS targets
export NATS_EVENTS_URL="nats://events.prod.example.com:4222"
export NATS_CACHE_URL="nats://cache.prod.example.com:4222"
export NATS_ANALYTICS_URL="nats://analytics.prod.example.com:4222"
```

## Common Patterns

### 1. Service Discovery

```yaml
# Dynamic service discovery with fallback
service_endpoint: (( 
  awsparam@discovery "/services/api/endpoint" || 
  nats@config "kv:services/api/endpoint" ||
  "http://localhost:8080" 
))
```

### 2. Secrets Rotation

```yaml
# Check for rotated secrets
database:
  password: (( 
    vault@production "secret/db:password_new" || 
    vault@production "secret/db:password" 
  ))
```

### 3. Multi-Region Configuration

```yaml
# Region-specific configuration
regions:
  us_east_1:
    vpc_id: (( awsparam@us_east_1 "/infrastructure/vpc/id" ))
    subnets: (( awsparam@us_east_1 "/infrastructure/subnets" ))
  
  eu_west_1:
    vpc_id: (( awsparam@eu_west_1 "/infrastructure/vpc/id" ))
    subnets: (( awsparam@eu_west_1 "/infrastructure/subnets" ))
```

### 4. Feature Flag Management

```yaml
# Multi-source feature flags
features:
  # Critical flags from Vault (audit trail)
  payment_v2: (( vault@features "secret/flags:payment_v2" ))
  
  # Fast-changing flags from NATS
  ui_experiment: (( nats@features "kv:flags/ui_experiment" ))
  
  # Environment flags from AWS
  debug_mode: (( awsparam@config "/features/debug_mode" ))
```

### 5. Monitoring Configuration

```yaml
# Observability configuration from multiple sources
monitoring:
  # Metrics endpoints
  prometheus:
    url: (( awsparam@monitoring "/prometheus/url" ))
    token: (( vault@monitoring "secret/prometheus:token" ))
  
  # Log aggregation
  logs:
    elasticsearch: (( awssecret@logging "elasticsearch/config" ))
    api_key: (( vault@logging "secret/elastic:api_key" ))
  
  # Distributed tracing
  tracing:
    enabled: (( nats@config "kv:tracing/enabled" ))
    sampling: (( nats@config "kv:tracing/sampling_rate" ))
```

## Best Practices

### 1. Target Naming

Use consistent, descriptive target names:
- Environment-based: `production`, `staging`, `development`
- Purpose-based: `auth`, `cache`, `messaging`, `monitoring`
- Region-based: `us_east_1`, `eu_west_1`, `ap_southeast_1`
- Account-based: `main`, `shared`, `audit`, `partner`

### 2. Configuration Hierarchy

Organize configuration by sensitivity and change frequency:
- **Vault**: Secrets, credentials, sensitive configuration
- **AWS Parameter Store**: Infrastructure configuration, endpoints
- **AWS Secrets Manager**: Rotating secrets, API keys
- **NATS**: Dynamic configuration, feature flags, templates

### 3. Error Handling

Always provide fallbacks for critical configuration:
```yaml
critical_config: (( 
  vault@primary "secret/critical" || 
  vault@secondary "secret/critical" || 
  panic "Critical configuration not found!" 
))
```

### 4. Performance Optimization

Cache configuration appropriately:
```bash
# Long-lived infrastructure config
export AWS_PRODUCTION_CACHE_TTL="30m"

# Frequently changing feature flags
export NATS_CONFIG_CACHE_TTL="1m"

# Sensitive data with shorter TTL
export VAULT_PRODUCTION_CACHE_TTL="5m"
```

### 5. Security Considerations

- Use separate credentials for each target
- Implement least-privilege access
- Enable audit logging for sensitive operations
- Rotate credentials regularly
- Never commit target credentials

## Troubleshooting

### Connection Issues

Test individual targets:
```bash
# Test Vault
vault login -address=$VAULT_PRODUCTION_ADDR
vault list -address=$VAULT_PRODUCTION_ADDR secret/

# Test AWS
aws sts get-caller-identity --profile production

# Test NATS
nats --server=$NATS_EVENTS_URL server check connection
```

### Debug Output

Enable debug mode to see target resolution:
```bash
export GRAFT_DEBUG=1
graft merge examples/targets/multi-cloud.yml 2>&1 | tee debug.log
```

### Common Errors

1. **Missing Target Configuration**
   - Ensure all required environment variables are set
   - Check for typos in target names

2. **Permission Denied**
   - Verify IAM roles and policies
   - Check Vault policies
   - Verify NATS authentication

3. **Cache Issues**
   - Clear local caches if seeing stale data
   - Reduce cache TTL for debugging

## Advanced Usage

### Dynamic Target Selection

Use shell scripts to select targets dynamically:

```bash
#!/bin/bash
# deploy.sh

ENVIRONMENT=$1
REGION=$2

# Set target suffix based on environment and region
export TARGET_SUFFIX="${ENVIRONMENT}_${REGION}"

# Generate config with dynamic targets
cat > dynamic-config.yml <<EOF
app:
  config: (( awsparam@${TARGET_SUFFIX} "/app/config" ))
  secret: (( vault@${TARGET_SUFFIX} "secret/app:key" ))
EOF

graft merge base.yml dynamic-config.yml
```

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Deploy with Targets

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: us-east-1
      
      - name: Set up targets
        run: |
          echo "AWS_PRODUCTION_REGION=us-east-1" >> $GITHUB_ENV
          echo "VAULT_PRODUCTION_ADDR=${{ secrets.VAULT_ADDR }}" >> $GITHUB_ENV
          echo "VAULT_PRODUCTION_TOKEN=${{ secrets.VAULT_TOKEN }}" >> $GITHUB_ENV
      
      - name: Generate configuration
        run: |
          graft merge config/base.yml config/production.yml > final-config.yml
      
      - name: Deploy
        run: |
          ./deploy.sh final-config.yml
```

## Contributing

When adding new examples:

1. Use meaningful target names
2. Include comments explaining the pattern
3. Provide fallback values where appropriate
4. Test with actual services when possible
5. Document any special requirements

## Additional Resources

- [Target-Aware Operators Guide](../../docs/guides/targets.md)
- [Migration Guide](../../docs/guides/migration-to-targets.md)
- [Operator Reference](../../docs/operators/external-data.md)
- Individual operator examples:
  - [Vault Examples](../vault-targets/)
  - [AWS Examples](../aws-targets/)
  - [NATS Examples](../nats-targets/)