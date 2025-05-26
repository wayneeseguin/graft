# Utility and Metadata Operators

These operators provide utility functions for metadata extraction, data cleanup, and specialized operations like static IP allocation.

## (( keys ))

Usage: `(( keys MAP|REFERENCE ))`

The `(( keys ))` operator extracts all keys from a map/hash and returns them as an array. This is useful for iterating over dynamic structures or generating lists based on map keys.

### Examples:

```yaml
# Basic key extraction
services:
  web:
    port: 80
    protocol: http
  api:
    port: 8080
    protocol: http
  database:
    port: 5432
    protocol: tcp

service_names: (( keys services ))
# Result: ["web", "api", "database"]

# Dynamic configuration based on keys
environments:
  development:
    replicas: 1
  staging:
    replicas: 2
  production:
    replicas: 5

env_list: (( keys environments ))
# Result: ["development", "staging", "production"]

# Generate resources for each key
deployment:
  environments: (( keys environments ))
  # Can now iterate over environment names

# Nested structures
config:
  databases:
    primary:
      host: db1.example.com
    secondary:
      host: db2.example.com
  caches:
    redis:
      host: redis.example.com
    memcached:
      host: memcached.example.com

resource_types: (( keys config ))          # ["databases", "caches"]
database_names: (( keys config.databases )) # ["primary", "secondary"]
cache_names: (( keys config.caches ))       # ["redis", "memcached"]

# Using with other operators
users:
  alice: { role: admin }
  bob: { role: user }
  charlie: { role: user }

user_list: (( keys users ))
user_count: (( len (keys users) ))  # Would need len operator
first_user: (( grab (keys users).0 ))  # "alice"

# Validation example
required_fields:
  name: true
  email: true
  password: true

provided_fields:
  name: "John"
  email: "john@example.com"

provided_keys: (( keys provided_fields ))
# Can compare against required_fields keys
```

See also: [keys examples](/examples/keys/)

## (( prune ))

Usage: `(( prune ))`

The `(( prune ))` operator marks content for removal from the final output. This is useful for temporary values, build-time configuration, or any data that shouldn't appear in the merged result.

### Examples:

```yaml
# Remove temporary build data
meta:
  build_config: (( prune ))
  version: "1.2.3"
  timestamp: "2024-01-15"

application:
  name: my-app
  version: (( grab meta.version ))
  # build_config won't appear in final output

# Remove sensitive defaults
defaults:
  admin_password: (( prune ))  # Don't include default password
  admin_user: admin
  
config:
  admin:
    username: (( grab defaults.admin_user ))
    password: (( vault "secret/admin:password" ))

# Conditional pruning with grab
temporary_data:
  calculations: (( prune ))
  intermediate_values:
    - value1
    - value2
  
final_result: 42

# Template metadata removal
_template_info: (( prune ))
_author: (( prune ))
_version: (( prune ))

# Arrays with prune
build_artifacts:
  - name: app.jar
    size: 10485760
  - name: temp.log
    size: 1024
    temporary: (( prune ))
  - name: config.yml
    size: 2048

# Complex structure pruning
development:
  debug_endpoints: (( prune ))
  test_users: (( prune ))
  mock_data: (( prune ))
  
  # These remain
  endpoints:
    api: /api/v1
    health: /health

# Working with static_ips
networks:
  static_ranges: (( prune ))  # Used by static_ips but not needed in output
  - name: private
    static: [10.0.0.10 - 10.0.0.20]

# Prune with parameters
params:
  provided_by_user: (( param "User must provide this" ))
  internal_only: (( prune ))

# Note: Paths with dots in names don't work
# This will NOT work:
problematic:
  "10.0.0.1": (( prune ))  # Dots in key name not supported
  
# Use go-patch format instead for such cases
```

See also: [prune examples](/examples/prune/)

## (( static_ips ))

Usage: `(( static_ips INDEX1 [INDEX2 ...] ))`

The `(( static_ips ))` operator generates static IP assignments for BOSH deployments. It automatically calculates IPs based on network definitions, availability zones, and instance counts.

### How it works:
1. Looks for network definitions in the deployment
2. Finds static IP ranges
3. Allocates IPs based on provided indexes
4. Handles availability zones automatically
5. Adjusts for instance counts

### Examples:

```yaml
# Basic static IP allocation
networks:
  - name: private
    static: [10.0.0.10 - 10.0.0.20]

jobs:
  - name: web
    instances: 3
    networks:
      - name: private
        static_ips: (( static_ips 0 1 2 ))
    # Result: ["10.0.0.10", "10.0.0.11", "10.0.0.12"]

# With availability zones
networks:
  - name: private
    subnets:
      - az: z1
        static: [10.0.1.10 - 10.0.1.20]
      - az: z2
        static: [10.0.2.10 - 10.0.2.20]

jobs:
  - name: database
    instances: 2
    azs: [z1, z2]
    networks:
      - name: private
        static_ips: (( static_ips 0 1 ))
    # Result: ["10.0.1.10", "10.0.2.10"]

# Multiple jobs sharing IP space
jobs:
  - name: etcd
    instances: 3
    networks:
      - name: private
        static_ips: (( static_ips 0 1 2 ))
    # Gets: ["10.0.0.10", "10.0.0.11", "10.0.0.12"]
  
  - name: consul
    instances: 3
    networks:
      - name: private
        static_ips: (( static_ips 3 4 5 ))
    # Gets: ["10.0.0.13", "10.0.0.14", "10.0.0.15"]

# Dynamic index calculation
meta:
  etcd_servers: 3
  consul_servers: 3

jobs:
  - name: etcd
    instances: (( grab meta.etcd_servers ))
    networks:
      - name: private
        static_ips: (( static_ips 0 1 2 ))
  
  - name: consul
    instances: (( grab meta.consul_servers ))
    networks:
      - name: private
        # Start after etcd IPs
        static_ips: (( static_ips meta.etcd_servers 
                                  meta.etcd_servers + 1 
                                  meta.etcd_servers + 2 ))

# With network properties
networks:
  - name: private
    static: 
      - 10.0.0.10 - 10.0.0.19  # Web tier
      - 10.0.0.20 - 10.0.0.29  # App tier
      - 10.0.0.30 - 10.0.0.39  # Data tier

jobs:
  - name: web
    instances: 2
    networks:
      - name: private
        static_ips: (( static_ips 0 1 ))  # Uses first range
  
  - name: app
    instances: 3
    networks:
      - name: private
        static_ips: (( static_ips 10 11 12 ))  # Uses second range

# Complex multi-AZ setup
instance_groups:
  - name: zookeeper
    instances: 5
    azs: [z1, z2, z3]
    networks:
      - name: private
        static_ips: (( static_ips 0 1 2 3 4 ))
    # Distributes across AZs in round-robin fashion
    # z1: ["10.0.1.10", "10.0.1.13"]
    # z2: ["10.0.2.11", "10.0.2.14"]  
    # z3: ["10.0.3.12"]
```

See also: [static_ips examples](/examples/static-ips/)

## Common Patterns

### Dynamic Resource Generation
```yaml
# Generate resources based on keys
services:
  web: { port: 80 }
  api: { port: 8080 }
  admin: { port: 9090 }

ingress_rules:
  # Would need custom logic, but concept:
  # For each service in (keys services), create rule
  
service_ports: (( keys services ))  # Use for iteration
```

### Configuration Cleanup
```yaml
# Development configuration with pruning
meta:
  _comments: (( prune ))
  _todo: (( prune ))
  _debug_info: (( prune ))
  
  # Actual configuration
  environment: production
  region: us-east-1

# Build-time vs runtime config
build:
  source_repo: (( prune ))
  build_number: (( prune ))
  artifacts: (( prune ))

runtime:
  version: "1.2.3"
  config_hash: "abc123"
```

### BOSH Deployment Patterns
```yaml
# Standard BOSH deployment with static IPs
networks:
  - name: default
    type: manual
    subnets:
      - range: 10.0.0.0/24
        gateway: 10.0.0.1
        static: [10.0.0.10 - 10.0.0.50]
        az: z1

instance_groups:
  - name: nats
    instances: 2
    azs: [z1]
    networks:
      - name: default
        static_ips: (( static_ips 0 1 ))
  
  - name: postgres
    instances: 1
    azs: [z1]
    networks:
      - name: default
        static_ips: (( static_ips 5 ))  # Skip some IPs

# Calculate static IPs with offsets
meta:
  nats_count: 2
  postgres_count: 1
  postgres_offset: (( grab meta.nats_count ))

# Use calculated offsets
postgres_ips: (( static_ips meta.postgres_offset ))
```

### Metadata Extraction
```yaml
# Extract and process metadata
components:
  frontend:
    version: "2.1.0"
    dependencies: ["api", "auth"]
  backend:
    version: "3.0.0"
    dependencies: ["database", "cache"]
  auth:
    version: "1.5.0"
    dependencies: []

# Extract component names
all_components: (( keys components ))

# Build deployment order (would need custom logic)
deployment_metadata:
  components: (( grab all_components ))
  versions: 
    # Would extract version for each component
  total: 3
```