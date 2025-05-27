# Data Manipulation Operators

These operators transform and manipulate data values in your YAML/JSON structures.

## (( concat ))

Usage: `(( concat LITERAL|REFERENCE ... ))`

The `(( concat ))` operator concatenates values together into a string. You can pass it any number of arguments, literal or reference, as long as the reference is not an array/map.

### Examples:

```yaml
# Basic string concatenation
greeting: (( concat "Hello, " "World!" ))
# Result: "Hello, World!"

# Using references
meta:
  first_name: "John"
  last_name: "Doe"
full_name: (( concat meta.first_name " " meta.last_name ))
# Result: "John Doe"

# Building URLs
domain: "example.com"
protocol: "https"
path: "/api/v1"
url: (( concat protocol "://" domain path ))
# Result: "https://example.com/api/v1"

# With numbers (automatically converted to strings)
version: 3
build: 142
version_string: (( concat "v" version "." build ))
# Result: "v3.142"
```

See also: [concat examples](/examples/concat/)

## (( join ))

Usage: `(( join SEPARATOR ARRAY|REFERENCE ))`

The `(( join ))` operator concatenates array elements into a single string using the specified separator.

### Examples:

```yaml
# Join array elements
tags: ["production", "web", "frontend"]
tag_string: (( join ", " tags ))
# Result: "production, web, frontend"

# Building paths
path_parts: ["home", "user", "documents", "file.txt"]
file_path: (( join "/" path_parts ))
# Result: "home/user/documents/file.txt"

# Creating command arguments
options: ["--verbose", "--color=auto", "--jobs=4"]
command_args: (( join " " options ))
# Result: "--verbose --color=auto --jobs=4"

# Empty separator
words: ["s", "p", "r", "u", "c", "e"]
word: (( join "" words ))
# Result: "graft"
```

See also: [join examples](/examples/join/)

## (( stringify ))

Usage: `(( stringify REFERENCE ))`

The `(( stringify ))` operator converts a data structure (map, array, etc.) into a properly formatted YAML string. This is especially useful for embedding YAML as a string value, such as in Kubernetes ConfigMaps.

### Examples:

```yaml
# Convert a map to YAML string
app_config:
  database:
    host: localhost
    port: 5432
  cache:
    enabled: true
    ttl: 3600

config_string: (( stringify app_config ))
# Result: |
#   database:
#     host: localhost
#     port: 5432
#   cache:
#     enabled: true
#     ttl: 3600

# Kubernetes ConfigMap example
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config.yaml: (( stringify app_config ))
```

See also: [stringify examples](/examples/stringify/)

## (( base64 ))

Usage: `(( base64 LITERAL|REFERENCE ))`

The `(( base64 ))` operator encodes strings in base64 format. This is useful for secrets, certificates, and any data that needs to be base64-encoded.

### Examples:

```yaml
# Encode a literal string
encoded_password: (( base64 "my-secret-password" ))
# Result: "bXktc2VjcmV0LXBhc3N3b3Jk"

# Encode from reference
credentials:
  username: admin
  password: supersecret
encoded_creds: (( base64 credentials.password ))
# Result: "c3VwZXJzZWNyZXQ="

# Kubernetes Secret
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
data:
  username: (( base64 "dbuser" ))
  password: (( base64 "dbpass123" ))

# Certificate encoding
tls:
  cert: |
    -----BEGIN CERTIFICATE-----
    MIIDtTCCAp2gAwIBAgIJAKg...
    -----END CERTIFICATE-----
  encoded_cert: (( base64 tls.cert ))
```

See also: [base64 examples](/examples/base64/)

## (( base64-decode ))

Usage: `(( base64-decode LITERAL|REFERENCE ))`

The `(( base64-decode ))` operator decodes base64-encoded strings back to their original form.

### Examples:

```yaml
# Decode a literal
encoded: "SGVsbG8gV29ybGQh"
decoded: (( base64-decode encoded ))
# Result: "Hello World!"

# Decode multiple values
secrets:
  db_user: "ZGJ1c2Vy"
  db_pass: "ZGJwYXNzMTIz"

credentials:
  username: (( base64-decode secrets.db_user ))
  password: (( base64-decode secrets.db_pass ))
# Result:
#   username: "dbuser"
#   password: "dbpass123"

# Decode and use in connection string
encoded_host: "ZGIuZXhhbXBsZS5jb20="
connection: (( concat "postgresql://user:pass@" (base64-decode encoded_host) ":5432/mydb" ))
# Result: "postgresql://user:pass@db.example.com:5432/mydb"
```

## (( empty ))

Usage: `(( empty VALUE|REFERENCE ))`

The `(( empty ))` operator checks if a value is empty (null, "", [], or {}) and returns a boolean. It's commonly used in conditional expressions.

### Examples:

```yaml
# Check various empty values
values:
  null_value: null
  empty_string: ""
  empty_array: []
  empty_map: {}
  non_empty: "hello"

checks:
  is_null: (( empty values.null_value ))        # true
  is_empty_str: (( empty values.empty_string )) # true
  is_empty_arr: (( empty values.empty_array ))  # true
  is_empty_map: (( empty values.empty_map ))    # true
  is_not_empty: (( empty values.non_empty ))    # false

# Conditional logic
user_input: ""
display_name: (( empty user_input ? "Anonymous" : user_input ))
# Result: "Anonymous"

# Validation
config:
  required_field: ""
  optional_field: ""

validation:
  is_valid: (( ! empty config.required_field ))
  error: (( empty config.required_field ? "Required field missing" : null ))
```

See also: [empty examples](/examples/empty/)

## (( null ))

Usage: `(( null [VALUE] ))`

The `(( null ))` operator serves two purposes:
1. Without arguments: Returns a null value
2. With an argument: Checks if the value is null, returning true or false

**Note:** This operator is only available when using the enhanced parser (default in v1.31.0+).

### Examples:

```yaml
# Set a value to null
cleared_value: (( null ))
# Result: null

# Check if values are null
values:
  missing: null
  empty_string: ""
  zero: 0
  present: "hello"

checks:
  is_missing_null: (( null values.missing ))     # true
  is_empty_null: (( null values.empty_string ))  # false (empty string is not null)
  is_zero_null: (( null values.zero ))           # false (zero is not null)
  is_present_null: (( null values.present ))     # false

# Conditional based on null check
config:
  optional_value: null
  use_default: (( null config.optional_value ? "default" : config.optional_value ))
  # Result: "default"

# Combined with grab and ||
settings:
  # If the grab returns null, use default
  timeout: (( grab config.timeout || (null (grab config.timeout) ? 30 : config.timeout) ))
```

## (( negate ))

Usage: `(( negate BOOLEAN|REFERENCE ))`

The `(( negate ))` operator returns the logical NOT of a boolean value. It's useful for inverting boolean flags and conditions.

### Examples:

```yaml
# Basic negation
flags:
  debug_enabled: true
  production_mode: false

inverted:
  debug_disabled: (( negate flags.debug_enabled ))      # false
  development_mode: (( negate flags.production_mode ))  # true

# Feature flags
features:
  new_ui: true
  legacy_ui: (( negate features.new_ui ))  # false

# Conditional logic
user:
  is_admin: false
  is_regular_user: (( negate user.is_admin ))  # true

# Combined with empty
value: ""
has_value: (( negate (empty value) ))  # false
```

## Common Patterns

### Building Dynamic Strings
```yaml
environment: production
region: us-east-1
service: api

# Build resource names
bucket_name: (( concat service "-" environment "-" region ))
# Result: "api-production-us-east-1"
```

### Data Transformation Pipeline
```yaml
# Original data
raw_tags: ["Dev", "Test", "Prod"]

# Transform pipeline
lowercase_tags: ["dev", "test", "prod"]  # (would need custom operator)
tag_string: (( join "," lowercase_tags ))
encoded_tags: (( base64 tag_string ))
# Result: "ZGV2LHRlc3QscHJvZA=="
```

### Validation and Defaults
```yaml
input:
  name: ""
  email: "user@example.com"

processed:
  name: (( empty input.name ? "Unknown User" : input.name ))
  email: (( empty input.email ? "noreply@example.com" : input.email ))
  has_email: (( negate (empty input.email) ))
```