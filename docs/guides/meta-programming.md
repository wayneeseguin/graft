# Generating graft with graft

Sometimes you need to generate graft templates dynamically. This guide covers techniques for meta-programming with graft.

## The defer Operator

The `(( defer ))` operator postpones operator evaluation to a subsequent `graft merge` run.

### Basic Example

```yaml
# template.yml
api_endpoint: (( defer grab config.api.endpoint ))
auth_token: (( defer vault "secret/api:token" ))

config:
  api:
    endpoint: "https://api.example.com"
```

Running `graft merge template.yml` produces:

```yaml
api_endpoint: (( grab config.api.endpoint ))
auth_token: (( vault "secret/api:token" ))

config:
  api:
    endpoint: "https://api.example.com"
```

The deferred operators remain as operators in the output, ready for evaluation in a later merge.

### Multiple Deferrals

You can chain multiple `defer` operators:

```yaml
# Triple defer - evaluates after three merges
future_value: (( defer defer defer grab data ))
data: my_value
```

Each merge removes one `defer`:
1. First merge: `(( defer defer grab data ))`
2. Second merge: `(( defer grab data ))`
3. Third merge: `(( grab data ))`
4. Fourth merge: `my_value`

## Skip Evaluation Mode

The `--skip-eval` flag skips all operator evaluation while still performing merges.

### Example

```yaml
# base.yml
name: (( grab meta.name ))
environment: (( grab meta.env ))
url: (( concat "https://" meta.domain ))

meta:
  name: myapp
  env: production
  domain: example.com
```

Running `graft merge --skip-eval base.yml` preserves all operators:

```yaml
name: (( grab meta.name ))
environment: (( grab meta.env ))
url: (( concat "https://" meta.domain ))

meta:
  name: myapp
  env: production
  domain: example.com
```

## Use Cases

### 1. Template Generation

Generate reusable templates with placeholders:

```yaml
# generator.yml
kubernetes:
  deployment:
    metadata:
      name: (( defer grab app.name ))
      namespace: (( defer grab app.namespace ))
    spec:
      replicas: (( defer grab app.replicas ))
      template:
        spec:
          containers:
          - name: (( defer grab app.name ))
            image: (( defer concat app.image ":" app.version ))
            env:
            - name: DATABASE_URL
              value: (( defer grab database.url ))
```

### 2. Multi-Stage Pipelines

Create templates that are processed in stages:

```yaml
# stage1-template.yml
base:
  name: (( grab $APP_NAME ))
  environment: (( grab $ENVIRONMENT ))

config:
  database:
    host: (( defer grab databases.$ENVIRONMENT.host ))
    credentials: (( defer vault (concat "secret/" base.environment "/db") ))
```

First merge captures environment variables:
```bash
graft merge stage1-template.yml > stage2-template.yml
```

Second merge evaluates deferred operators:
```bash
graft merge stage2-template.yml databases.yml > final.yml
```

### 3. Conditional Template Generation

Generate different templates based on conditions:

```yaml
# template-generator.yml
app_type: microservice

templates:
  microservice:
    service:
      name: (( defer grab service.name ))
      port: (( defer grab service.port || 8080 ))
      health_check: (( defer concat "/health" ))
    
  monolith:
    application:
      name: (( defer grab application.name ))
      modules: (( defer grab application.modules ))

# Select template based on app_type
output: (( grab templates.$app_type ))
```

### 4. Dynamic Operator Construction

Build operators dynamically:

```yaml
# Build vault paths dynamically
vault_paths:
  base: "secret"
  environment: production
  service: myapp

secrets:
  database:
    path: (( concat vault_paths.base "/" vault_paths.environment "/" vault_paths.service "/db" ))
    password: (( defer vault (grab secrets.database.path) ))
  
  api:
    path: (( concat vault_paths.base "/" vault_paths.environment "/" vault_paths.service "/api" ))
    key: (( defer vault (grab secrets.api.path) ))
```

## Advanced Patterns

### Recursive Templates

Generate templates that reference themselves:

```yaml
# recursive-template.yml
components:
  - name: web
    depends_on: (( defer grab components.[1].name ))
  - name: api
    depends_on: (( defer grab components.[2].name ))  
  - name: database
    depends_on: null

# After first merge, creates circular dependencies for next evaluation
```

### Template Libraries

Build a library of reusable template fragments:

```yaml
# lib/templates.yml
templates:
  standard_service:
    metadata:
      name: (( defer grab service.name ))
      labels:
        app: (( defer grab service.name ))
        version: (( defer grab service.version ))
    spec:
      replicas: (( defer grab service.replicas || 1 ))
      
  database_config:
    host: (( defer grab database.host ))
    port: (( defer grab database.port || 5432 ))
    ssl: (( defer grab database.ssl || true ))
    credentials:
      username: (( defer grab database.username ))
      password: (( defer vault (concat "secret/" service.name "/db:password") ))
```

Use the templates:

```yaml
# my-service.yml
(( grab templates.standard_service ))

service:
  name: my-api
  version: 1.2.3
  replicas: 3

database:
  host: db.example.com
  username: apiuser
```

### Operator Injection

Inject operators into existing structures:

```yaml
# injector.yml
inject_operators:
  vault_prefix: "secret/myapp"
  
  # Transform static config to use vault
  transform:
    - path: "database.password"
      operator: (( defer concat "(( vault \"" inject_operators.vault_prefix "/db:password\" ))" ))
    - path: "api.key"  
      operator: (( defer concat "(( vault \"" inject_operators.vault_prefix "/api:key\" ))" ))

# Original config
database:
  host: localhost
  password: "static-password"  # Will be replaced

api:
  endpoint: https://api.example.com
  key: "static-key"  # Will be replaced
```

## Best Practices

### 1. Document Deferred Operations

```yaml
# This template expects the following to be provided:
# - service.name (string): Name of the service
# - service.port (int): Port number (default: 8080)
# - database.url (string): Database connection URL

config:
  name: (( defer grab service.name ))
  port: (( defer grab service.port || 8080 ))
  database: (( defer grab database.url ))
```

### 2. Use Meaningful Defer Chains

```yaml
# Clear naming for multi-stage processing
stage1_value: (( defer grab input.value ))
stage2_value: (( defer defer grab processed.value ))
final_value: (( defer defer defer grab final.value ))
```

### 3. Validate Templates

Test generated templates:

```bash
# Generate template
graft merge --skip-eval template-generator.yml > generated.yml

# Validate by attempting to merge
graft merge generated.yml test-data.yml
```

### 4. Combine with Other Features

```yaml
# Defer with conditionals
optional_feature: (( defer environment == "production" ? (grab prod.config) : (grab dev.config) ))

# Defer with defaults
api_timeout: (( defer grab config.timeout || 30 ))

# Defer with concatenation
connection_string: (( defer concat "mongodb://" (grab db.host) ":" (grab db.port) ))
```

## Example: Complete Template System

```yaml
# template-system.yml
meta:
  app: myapp
  environments: [dev, staging, prod]
  
# Template definitions
templates:
  base:
    name: (( defer grab app.name ))
    environment: (( defer grab app.environment ))
    
  kubernetes:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: (( defer grab templates.base.name ))
      namespace: (( defer grab templates.base.environment ))
    spec:
      replicas: (( defer grab app.replicas ))
      selector:
        matchLabels:
          app: (( defer grab templates.base.name ))
      template:
        spec:
          containers:
          - name: app
            image: (( defer concat app.image ":" app.tag ))
            env:
            - name: APP_ENV
              value: (( defer grab templates.base.environment ))
            - name: DATABASE_URL
              value: (( defer vault (concat "secret/" templates.base.environment "/db:url") ))

# Generate environment-specific templates
outputs:
  - (( defer grab templates.kubernetes ))
```

Usage:
```bash
# Generate template
graft merge template-system.yml > k8s-template.yml

# Use template with actual values
cat <<EOF | graft merge k8s-template.yml -
app:
  name: myservice
  environment: production
  replicas: 3
  image: myregistry/myservice
  tag: v1.2.3
EOF
```

## Debugging

### View Deferred Operations

```bash
# See what will be deferred
graft merge template.yml | grep "(( defer"
```

### Step Through Deferrals

```bash
# Original
graft merge template.yml > step1.yml

# Remove one defer
graft merge step1.yml > step2.yml

# Continue until fully evaluated
graft merge step2.yml > final.yml
```

## See Also

- [defer Operator Reference](../operators/data-references.md#defer)
- [Operators Overview](../operators/README.md)
- [Advanced Examples](../../examples/meta-programming/)