# AWS Integration Examples

This directory contains examples of using Graft with AWS services:
- AWS Systems Manager Parameter Store (`awsparam`)
- AWS Secrets Manager (`awssecret`)

## Prerequisites

1. AWS CLI configured with appropriate credentials:
   ```bash
   aws configure
   ```

2. Environment variables (alternative to AWS CLI):
   ```bash
   export AWS_ACCESS_KEY_ID=your-access-key
   export AWS_SECRET_ACCESS_KEY=your-secret-key
   export AWS_REGION=us-east-1
   ```

3. IAM permissions required:
   - For Parameter Store: `ssm:GetParameter`, `ssm:GetParameters`
   - For Secrets Manager: `secretsmanager:GetSecretValue`

## Files in this directory:

1. **parameter-store.yml** - AWS SSM Parameter Store examples
2. **secrets-manager.yml** - AWS Secrets Manager examples
3. **multi-region.yml** - Multi-region configuration
4. **dynamic-paths.yml** - Dynamic path construction
5. **with-fallbacks.yml** - Error handling and defaults

## Setting up test data

### Parameter Store
```bash
# Create test parameters
aws ssm put-parameter --name "/myapp/dev/database_host" --value "dev.db.example.com" --type String
aws ssm put-parameter --name "/myapp/dev/api_key" --value "dev-api-key-123" --type SecureString
aws ssm put-parameter --name "/myapp/config" --value '{"port": 8080, "timeout": 30}' --type String
```

### Secrets Manager
```bash
# Create test secrets
aws secretsmanager create-secret --name "myapp/dev/database" \
  --secret-string '{"username":"dbuser","password":"dbpass123","host":"db.example.com"}'

aws secretsmanager create-secret --name "myapp/dev/api-key" \
  --secret-string "secret-api-key-456"
```

## Running Examples

```bash
# Test Parameter Store
graft merge parameter-store.yml

# Test Secrets Manager
graft merge secrets-manager.yml

# Test with environment-specific paths
ENV=prod graft merge dynamic-paths.yml
```

## Important Notes

- AWS operators require network access to AWS services
- Costs may apply for AWS API calls
- Use IAM roles when running on EC2/ECS/Lambda
- Consider caching for frequently accessed parameters
- Both operators support JSON value extraction with `?key=` syntax