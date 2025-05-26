# Getting Started with Spruce

Spruce is a domain-specific YAML merging tool, designed to make it easy to manage and merge multi-part YAML configurations.

## Installation

### macOS (Homebrew)

```bash
brew install spruce
```

### Linux/macOS (Binary)

```bash
# Download the latest release
curl -L https://github.com/geofffranks/spruce/releases/latest/download/spruce-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64 -o spruce

# Make it executable
chmod +x spruce

# Move to PATH
sudo mv spruce /usr/local/bin/
```

### From Source

```bash
go get github.com/geofffranks/spruce/cmd/spruce
```

### Verify Installation

```bash
spruce --version
```

## Basic Usage

### Your First Merge

Create two YAML files:

**base.yml:**
```yaml
name: my-app
version: 1.0.0
server:
  port: 8080
  host: localhost
```

**production.yml:**
```yaml
server:
  port: 443
  host: app.example.com
  ssl: true
```

Merge them:
```bash
spruce merge base.yml production.yml
```

Output:
```yaml
name: my-app
version: 1.0.0
server:
  port: 443
  host: app.example.com
  ssl: true
```

### Understanding Merge Order

Files are merged left-to-right. Later files override values from earlier files:

```bash
spruce merge first.yml second.yml third.yml
```

- `first.yml` is the base
- `second.yml` overrides/adds to first
- `third.yml` overrides/adds to the result

## Using Operators

Spruce's power comes from its operators - special expressions that perform operations during merging.

### References with grab

Reference other parts of the document:

```yaml
# config.yml
domain: example.com
api:
  url: (( concat "https://api." domain ))
web:
  url: (( concat "https://www." domain ))
```

Result:
```yaml
domain: example.com
api:
  url: https://api.example.com
web:
  url: https://www.example.com
```

### Concatenation

Build strings from parts:

```yaml
# Example
environment: production
region: us-east-1
resource_name: (( concat "myapp-" environment "-" region ))
```

Result:
```yaml
resource_name: myapp-production-us-east-1
```

### Conditional Values

Use the ternary operator for conditionals:

```yaml
environment: production
debug_mode: (( environment == "production" ? false : true ))
replicas: (( environment == "production" ? 3 : 1 ))
```

## Working with Arrays

### Default Array Merging

Arrays of maps with `name` fields merge by name:

```yaml
# base.yml
servers:
  - name: web
    port: 8080
  - name: db
    port: 5432

# override.yml
servers:
  - name: web
    port: 80
    ssl: true
```

Result:
```yaml
servers:
  - name: web
    port: 80
    ssl: true
  - name: db
    port: 5432
```

### Array Operators

Control array merging explicitly:

```yaml
# Append to array
items:
  - (( append ))
  - new-item

# Replace array
items:
  - (( replace ))
  - only-item

# Prepend to array
items:
  - (( prepend ))
  - first-item
```

## Environment Variables

### Using Environment Variables

```yaml
# Direct usage
database_url: (( grab $DATABASE_URL ))

# With defaults
port: (( grab $PORT || 8080 ))

# In paths (environment variable expansion)
config: (( grab settings.$ENVIRONMENT.database ))
```

### Example with Defaults

```yaml
app:
  name: (( grab $APP_NAME || "my-app" ))
  port: (( grab $PORT || 8080 ))
  debug: (( grab $DEBUG || false ))
  database:
    host: (( grab $DB_HOST || "localhost" ))
    port: (( grab $DB_PORT || 5432 ))
```

## Common Patterns

### Configuration Layering

Structure your configurations in layers:

```bash
config/
  base.yml           # Shared configuration
  development.yml    # Development overrides
  staging.yml        # Staging overrides  
  production.yml     # Production overrides
```

Usage:
```bash
# Development
spruce merge config/base.yml config/development.yml

# Production
spruce merge config/base.yml config/production.yml
```

### Pruning Temporary Data

Remove temporary scaffolding:

```yaml
# base.yml
meta:
  environment: production
  region: us-east-1

app:
  name: myapp
  full_name: (( concat meta.environment "-" app.name "-" meta.region ))
```

```bash
spruce merge base.yml --prune meta
```

### Using Parameters

Require values to be provided:

```yaml
# base.yml
database:
  host: (( param "Please provide database.host" ))
  port: 5432
  username: (( param "Please provide database.username" ))
  password: (( param "Please provide database.password" ))
```

## Advanced Features

### Multi-Document YAML

Process files with multiple documents:

```yaml
---
name: doc1
---
name: doc2
```

```bash
spruce merge --multi-doc multi.yml
```

### Cherry-Picking

Extract specific parts:

```bash
spruce merge large-manifest.yml --cherry-pick jobs.web
```

### Vault Integration

Pull secrets from HashiCorp Vault:

```yaml
database:
  password: (( vault "secret/db:password" ))
api:
  key: (( vault "secret/api:key" ))
```

## Best Practices

### 1. Organize Your Files

```
manifests/
  base/
    app.yml
    database.yml
    networking.yml
  environments/
    dev.yml
    staging.yml
    prod.yml
```

### 2. Use Meaningful Names

```yaml
# Good
database_connection_timeout: 30
api_rate_limit: 1000

# Less clear
db_timeout: 30
limit: 1000
```

### 3. Document Your Operators

```yaml
# Calculate replica count based on environment
# Production: 3, Others: 1
replicas: (( environment == "production" ? 3 : 1 ))
```

### 4. Validate Early

```yaml
required:
  api_key: (( param "API_KEY is required" ))
  database_url: (( param "DATABASE_URL is required" ))
```

### 5. Keep Secrets Secure

```yaml
# Don't do this
password: supersecret

# Do this
password: (( vault "secret/app:password" ))
# Or
password: (( grab $PASSWORD ))
```

## Troubleshooting

### Debug Mode

See what Spruce is doing:

```bash
spruce merge --debug base.yml override.yml
```

### Common Issues

**Missing references:**
```yaml
# Error: $.does.not.exist could not be found
value: (( grab does.not.exist ))

# Fix: Provide a default
value: (( grab does.not.exist || "default" ))
```

**Circular references:**
```yaml
# Error: Circular reference detected
a: (( grab b ))
b: (( grab a ))
```

**Type mismatches:**
```yaml
# Error: Cannot concatenate non-string values
result: (( concat "prefix" 123 ))  # 123 is not a string

# Fix: Convert to string first
result: (( concat "prefix" (stringify 123) ))
```

## Next Steps

- Explore [Operators Reference](operators/README.md) for all available operators
- See [Examples](../examples/README.md) for real-world usage
- Read about [Advanced Concepts](concepts/README.md)
- Check out [Integration Guides](guides/README.md)

## Getting Help

- Run `spruce -h` for command help
- Check the [FAQ](reference/faq.md)
- Visit the [GitHub repository](https://github.com/geofffranks/spruce)
- Read the [Troubleshooting Guide](reference/troubleshooting.md)