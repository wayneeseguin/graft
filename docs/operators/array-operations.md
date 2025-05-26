# Array Operations

These operators provide powerful array manipulation capabilities, including merging strategies, transformations, and sorting.

## Array Merge Operators

Spruce provides several operators to control how arrays are merged between files. By default, arrays of maps merge by `name` key, but you can customize this behavior.

### (( append ))

Adds elements to the end of an existing array.

```yaml
# base.yml
fruits:
  - apple
  - banana

# override.yml
fruits:
  - (( append ))
  - orange
  - grape

# Result:
# fruits:
#   - apple
#   - banana
#   - orange
#   - grape
```

### (( prepend ))

Adds elements to the beginning of an existing array.

```yaml
# base.yml
tasks:
  - test
  - build
  - deploy

# override.yml
tasks:
  - (( prepend ))
  - lint
  - validate

# Result:
# tasks:
#   - lint
#   - validate
#   - test
#   - build
#   - deploy
```

### (( replace ))

Completely replaces the existing array with new content.

```yaml
# base.yml
environments:
  - dev
  - staging
  - prod

# override.yml
environments:
  - (( replace ))
  - development
  - production

# Result:
# environments:
#   - development
#   - production
```

### (( inline ))

Merges arrays element by element based on index position.

```yaml
# base.yml
servers:
  - host: server1
    port: 8080
  - host: server2
    port: 8081

# override.yml
servers:
  - (( inline ))
  - port: 9090  # Merges with first element
  - port: 9091  # Merges with second element

# Result:
# servers:
#   - host: server1
#     port: 9090
#   - host: server2
#     port: 9091
```

### (( merge ))

Merges arrays of maps based on a key field (default: `name`).

```yaml
# base.yml
services:
  - name: web
    port: 80
    replicas: 2
  - name: api
    port: 8080
    replicas: 3

# override.yml
services:
  - (( merge ))  # Optional, this is default behavior
  - name: web
    replicas: 4  # Only updates replicas for web
  - name: cache
    port: 6379
    replicas: 1  # Adds new service

# Result:
# services:
#   - name: web
#     port: 80
#     replicas: 4
#   - name: api
#     port: 8080
#     replicas: 3
#   - name: cache
#     port: 6379
#     replicas: 1

# Custom merge key
items:
  - (( merge on id ))
  - id: item-001
    quantity: 10
  - id: item-002
    quantity: 5
```

### (( insert ))

Inserts elements at specific positions in an array.

```yaml
# base.yml
pipeline:
  - name: compile
  - name: test
  - name: deploy

# Insert by name
pipeline:
  - (( insert after name "test" ))
  - name: security-scan
  - name: quality-check

# Result:
# pipeline:
#   - name: compile
#   - name: test
#   - name: security-scan
#   - name: quality-check
#   - name: deploy

# Insert by index
numbers: [1, 2, 4, 5]
numbers:
  - (( insert after 1 ))
  - 3

# Result: [1, 2, 3, 4, 5]

# Insert before
items:
  - (( insert before 0 ))  # Same as prepend
  - first-item
```

### (( delete ))

Removes elements from an array.

```yaml
# base.yml
features:
  - name: feature-a
    enabled: true
  - name: feature-b
    enabled: false
  - name: feature-c
    enabled: true

# Delete by name
features:
  - (( delete "feature-b" ))

# Result:
# features:
#   - name: feature-a
#     enabled: true
#   - name: feature-c
#     enabled: true

# Delete by index
values: [10, 20, 30, 40, 50]
values:
  - (( delete 2 ))  # Removes 30

# Result: [10, 20, 40, 50]
```

## Array Transformation Operators

### (( cartesian-product ))

Usage: `(( cartesian-product ARRAY1 ARRAY2 ... ))`

Generates the cartesian product of multiple arrays.

```yaml
# Basic cartesian product
colors: ["red", "blue"]
sizes: ["small", "large"]
products: (( cartesian-product colors sizes ))
# Result:
# products:
#   - ["red", "small"]
#   - ["red", "large"]
#   - ["blue", "small"]
#   - ["blue", "large"]

# Three-way product
environments: ["dev", "prod"]
regions: ["us-east", "us-west"]
services: ["api", "web"]
combinations: (( cartesian-product environments regions services ))
# Result: 8 combinations (2 x 2 x 2)

# Practical example: Test matrix
browsers: ["chrome", "firefox", "safari"]
os_versions: ["windows-10", "macos-12", "ubuntu-20"]
test_matrix: (( cartesian-product browsers os_versions ))
# Result: 9 test combinations
```

See also: [cartesian-product examples](/examples/cartesian-product/)

### (( shuffle ))

Usage: `(( shuffle ARRAY|VALUES... ))`

Randomly shuffles array elements.

```yaml
# Shuffle a single array
original: [1, 2, 3, 4, 5]
shuffled: (( shuffle original ))
# Result: [3, 1, 5, 2, 4] (random order)

# Shuffle multiple arrays into one
list1: ["a", "b", "c"]
list2: ["x", "y", "z"]
combined_shuffled: (( shuffle list1 list2 ))
# Result: ["y", "a", "z", "c", "b", "x"] (random order)

# Availability zone distribution
availability_zones: ["us-east-1a", "us-east-1b", "us-east-1c"]
instances:
  - name: web-1
    az: (( grab (shuffle availability_zones).0 ))
  - name: web-2
    az: (( grab (shuffle availability_zones).1 ))
  - name: web-3
    az: (( grab (shuffle availability_zones).2 ))
```

### (( sort ))

Usage: `(( sort [by KEY] ))`

Sorts arrays in post-processing phase.

```yaml
# Sort simple arrays
numbers: [3, 1, 4, 1, 5, 9]
sorted_numbers: (( sort ))
# Result: [1, 1, 3, 4, 5, 9]

strings: ["zebra", "alpha", "beta", "gamma"]
sorted_strings: (( sort ))
# Result: ["alpha", "beta", "gamma", "zebra"]

# Sort array of maps
users:
  - name: "charlie"
    age: 30
  - name: "alice"
    age: 25
  - name: "bob"
    age: 35

sorted_users: (( sort ))  # Sorts by 'name' (default)
# Result: alice, bob, charlie

# Sort by custom key
sorted_by_age: (( sort by age ))
# Result: alice (25), charlie (30), bob (35)

# Complex sorting example
inventory:
  - id: "PROD-003"
    name: "Widget C"
    price: 15.99
  - id: "PROD-001"
    name: "Widget A"
    price: 29.99
  - id: "PROD-002"
    name: "Widget B"
    price: 19.99

products_by_price: (( sort by price ))
products_by_id: (( sort by id ))
```

See also: [sort examples](/examples/sort/)

## Common Patterns

### Dynamic Array Building
```yaml
# Build array conditionally
base_services: ["web", "api"]
optional_services: []

production:
  services: (( append ))
  - (( grab optional_services ))
  - "monitoring"
  - "logging"
```

### Array Manipulation Pipeline
```yaml
# Start with raw data
raw_hosts: ["host-3", "host-1", "host-2", "host-1"]

# Process through multiple operations
processed:
  unique: ["host-1", "host-2", "host-3"]  # Would need custom dedup
  sorted: (( sort ))
  prefixed: 
    - (( prepend ))
    - "primary-host"
```

### Test Matrix Generation
```yaml
# Generate comprehensive test combinations
test_config:
  browsers: ["chrome", "firefox"]
  versions: ["latest", "latest-1"]
  platforms: ["windows", "mac", "linux"]
  
  # Generate all combinations
  full_matrix: (( cartesian-product test_config.browsers test_config.versions test_config.platforms ))
  
  # Shuffle for distributed testing
  test_queue: (( shuffle test_config.full_matrix ))
```

### Service Discovery Configuration
```yaml
# Merge service definitions from multiple sources
services:
  defaults:
    - name: database
      port: 5432
      health_check: "/health"
  
  overrides:
    - (( merge ))
    - name: database
      port: 5433  # Override port
    - name: cache
      port: 6379
      health_check: "/ping"
```