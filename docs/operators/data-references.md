# Data References and Flow Operators

These operators control how data flows through your YAML structures, enabling references, injections, and template parameters.

## (( grab ))

Usage: `(( grab REFERENCE|LITERAL ))`

The `(( grab ))` operator retrieves values from elsewhere in the document and assigns them to the current key. It's the primary way to reference and reuse values, helping you DRY (Don't Repeat Yourself) up your configuration.

### Examples:

```yaml
# Basic reference
meta:
  name: "my-app"
  version: "1.2.3"

application:
  name: (( grab meta.name ))        # "my-app"
  version: (( grab meta.version ))  # "1.2.3"

# With defaults (using literal values)
database:
  host: (( grab meta.db_host || "localhost" ))
  port: (( grab meta.db_port || 5432 ))

# Deep references
config:
  server:
    production:
      host: "prod.example.com"
      port: 443
  
deployment:
  endpoint: (( grab config.server.production.host ))  # "prod.example.com"

# Array access
environments: ["dev", "staging", "prod"]
current_env: (( grab environments.2 ))  # "prod"

# Reference with concatenation
base_domain: "example.com"
subdomain: "api"
full_domain: (( concat (grab subdomain) "." (grab base_domain) ))  # "api.example.com"

# Grabbing complex structures
database_config:
  primary:
    host: "db1.example.com"
    port: 5432
    ssl: true
  
app:
  db: (( grab database_config.primary ))
  # Result: entire primary object is copied
```

See also: [grab examples](/examples/grab/)

## (( inject ))

Usage: `(( inject REFERENCE ))`

The `(( inject ))` operator inserts the contents of a referenced structure at the current level, then removes the key containing the operator. This is useful for template inheritance and composition.

### Examples:

```yaml
# Template injection
templates:
  base_server:
    port: 8080
    protocol: "http"
    timeout: 30

services:
  web:
    inject_base: (( inject templates.base_server ))
    hostname: "web.example.com"
    # After merge:
    # port: 8080
    # protocol: "http"
    # timeout: 30
    # hostname: "web.example.com"

# Partial override pattern
defaults:
  logging:
    level: "info"
    format: "json"
    output: "stdout"

applications:
  api:
    _: (( inject defaults.logging ))
    level: "debug"  # Override just this field
    # Result:
    # level: "debug"
    # format: "json"
    # output: "stdout"

# Multiple injections
common:
  metadata:
    created_by: "spruce"
    version: "1.0"
  security:
    ssl_enabled: true
    auth_required: true

service:
  meta_inject: (( inject common.metadata ))
  sec_inject: (( inject common.security ))
  name: "my-service"
  # Result includes all metadata and security fields

# Nested injection
base_configs:
  web:
    server:
      port: 80
      workers: 4
    cache:
      enabled: true
      ttl: 3600

my_app:
  config:
    _: (( inject base_configs.web ))
    server:
      port: 8080  # Override specific nested value
  # Result has all web config with port overridden
```

See also: [inject examples](/examples/inject/)

## (( defer ))

Usage: `(( defer EXPRESSION ))`

The `(( defer ))` operator prevents evaluation of spruce operators, outputting them as literal strings. This is useful when generating configuration files that themselves contain spruce operators.

### Examples:

```yaml
# Generate a spruce template
template:
  name: (( defer (( grab meta.name )) ))
  version: (( defer (( grab meta.version )) ))
  # Output:
  # name: (( grab meta.name ))
  # version: (( grab meta.version ))

# CredHub integration example
credhub_manifest:
  credentials:
    - name: cf-admin-password
      type: password
    - name: db-password
      type: password
  
  properties:
    admin_password: (( defer ((credhub-get "cf-admin-password")) ))
    database:
      password: (( defer ((credhub-get "db-password")) ))
  # Output will contain the literal CredHub operators

# Multi-level defer
pipeline:
  jobs:
    - name: deploy
      params:
        MANIFEST: (( defer |
          name: (( grab deployment.name ))
          instances: (( grab deployment.instances ))
          properties:
            domain: (( grab deployment.domain ))
        ))
  # The entire block remains unevaluated

# Conditional defer
meta:
  use_credhub: true

credentials:
  password: (( grab meta.use_credhub ? (defer ((credhub-get "password"))) : "hardcoded" ))
  # If use_credhub is true: password: ((credhub-get "password"))
  # If false: password: "hardcoded"

# Generating operator documentation
examples:
  grab_usage: (( defer (( grab some.value )) ))
  concat_usage: (( defer (( concat "Hello " name )) ))
  vault_usage: (( defer (( vault "secret/path:key" )) ))
```

See also: [defer examples](/examples/defer/)

## (( param ))

Usage: `(( param "DESCRIPTION" ))`

The `(( param ))` operator marks a value as required, causing spruce to fail if the value is not provided by a merge operation. This ensures critical values aren't accidentally omitted.

### Examples:

```yaml
# Basic parameter requirement
database:
  host: (( param "Please provide database host" ))
  port: 5432
  username: (( param "Please provide database username" ))
  password: (( param "Please provide database password" ))

# With descriptive messages
aws:
  region: (( param "AWS region (e.g., us-east-1)" ))
  access_key: (( param "AWS access key ID" ))
  secret_key: (( param "AWS secret access key" ))

# Conditional parameters
meta:
  environment: (( param "Environment name (dev/staging/prod)" ))

production_only:
  ssl_cert: (( grab meta.environment == "prod" ? (param "SSL certificate required for production") : null ))

# Nested parameter structures
cluster:
  name: (( param "Cluster name" ))
  nodes:
    count: (( param "Number of nodes in cluster" ))
    type: (( param "Node instance type (e.g., t3.medium)" ))
  network:
    vpc_id: (( param "VPC ID where cluster will be deployed" ))
    subnet_ids: (( param "List of subnet IDs for cluster nodes" ))

# Parameter with grab fallback
config:
  # Try to grab from meta, but require it if not present
  app_name: (( grab meta.app_name || (param "Application name required") ))
  
# Usage with other operators
connection_string: (( concat 
  "postgresql://" 
  (param "Database username") 
  ":" 
  (param "Database password") 
  "@" 
  (param "Database host") 
  ":5432/mydb" 
))
```

See also: [param examples](/examples/params/)

## Common Patterns

### Configuration Templates
```yaml
# base.yml - Template with parameters
application:
  name: (( param "Application name" ))
  port: (( grab defaults.port || 8080 ))
  
  database:
    host: (( param "Database hostname" ))
    credentials:
      username: (( param "Database username" ))
      password: (( param "Database password" ))

defaults:
  port: 3000
  timeout: 30

# prod.yml - Provides required values
application:
  name: "my-app"
  database:
    host: "db.prod.example.com"
    credentials:
      username: "app_user"
      password: "secret123"
```

### Hierarchical References
```yaml
# Global -> Environment -> Application specific
global:
  company: "TechCorp"
  domain: "example.com"

environments:
  production:
    subdomain: "prod"
    replicas: 3
  staging:
    subdomain: "stage"
    replicas: 1

applications:
  web:
    env: (( grab environments.production ))
    url: (( concat "https://" applications.web.env.subdomain "." global.domain ))
    config:
      company: (( grab global.company ))
      replicas: (( grab applications.web.env.replicas ))
```

### Template Generation
```yaml
# Generate CloudFormation template with deferred values
cloudformation:
  Parameters:
    InstanceType:
      Type: String
      Default: (( defer (( grab instance.type )) ))
  
  Resources:
    WebServer:
      Type: "AWS::EC2::Instance"
      Properties:
        InstanceType: (( defer !Ref InstanceType ))
        ImageId: (( defer (( grab ami.id )) ))
```

### Modular Configuration
```yaml
# modules/database.yml
database:
  engine: "postgres"
  version: "13"
  settings: (( inject database.overrides ))
  overrides: {}  # Placeholder for injection

# modules/cache.yml  
cache:
  engine: "redis"
  version: "6"
  settings: (( inject cache.overrides ))
  overrides: {}

# main.yml
imports:
  db: (( grab database ))
  cache: (( grab cache ))

database:
  overrides:
    max_connections: 100
    
cache:
  overrides:
    maxmemory: "1gb"
```