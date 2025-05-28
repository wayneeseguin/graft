# Expression Operators

These operators enable complex expressions with boolean logic, comparisons, and conditional evaluation.

## Boolean Operators

### (( && )) - Logical AND

Usage: `(( EXPR1 && EXPR2 ))`

Returns true if both expressions are true.

```yaml
# Basic AND operation
flags:
  feature_a: true
  feature_b: false
  feature_c: true

checks:
  both_a_and_b: (( flags.feature_a && flags.feature_b ))  # false
  both_a_and_c: (( flags.feature_a && flags.feature_c ))  # true

# Multiple conditions
validation:
  has_name: true
  has_email: true
  has_password: false
  
  is_valid: (( validation.has_name && validation.has_email && validation.has_password ))  # false

# With comparisons
user:
  age: 25
  score: 85
  
  qualified: (( user.age >= 18 && user.score >= 80 ))  # true

# Short-circuit evaluation
# Second expression not evaluated if first is false
result: (( false && (grab nonexistent.path) ))  # false (no error)
```

### (( || )) - Logical OR (in expressions)

Usage: `(( EXPR1 || EXPR2 ))`

**Note:** The `||` operator serves dual purposes:
1. In expressions: Logical OR (returns true if either is true)
2. With operators: Fallback/default values

```yaml
# Logical OR in expressions
permissions:
  is_admin: false
  is_moderator: true
  is_owner: false

access:
  can_edit: (( permissions.is_admin || permissions.is_owner ))  # false
  can_moderate: (( permissions.is_admin || permissions.is_moderator ))  # true
  can_view: (( permissions.is_admin || permissions.is_moderator || permissions.is_owner ))  # true

# With comparisons
metrics:
  cpu_usage: 85
  memory_usage: 45
  
  needs_attention: (( metrics.cpu_usage > 80 || metrics.memory_usage > 90 ))  # true

# Fallback values (different from logical OR)
config:
  port: (( grab env.port || 8080 ))  # If env.port is null/missing, use 8080
  host: (( grab env.host || "localhost" ))
```

### (( ! )) - Logical NOT

Usage: `(( ! EXPR ))`

Returns the opposite boolean value.

```yaml
# Basic negation
settings:
  debug_enabled: true
  production_mode: false

computed:
  debug_disabled: (( ! settings.debug_enabled ))  # false
  development_mode: (( ! settings.production_mode ))  # true

# With empty checks
data:
  required_field: ""
  optional_field: "value"

validation:
  has_required: (( ! empty data.required_field ))  # false
  has_optional: (( ! empty data.optional_field ))  # true

# Complex conditions
user:
  is_guest: true
  is_verified: false
  
  can_post: (( ! user.is_guest && user.is_verified ))  # false
  needs_verification: (( ! user.is_guest && ! user.is_verified ))  # false
```

## Comparison Operators

### (( == )) - Equality

Usage: `(( EXPR1 == EXPR2 ))`

Checks if two values are equal.

```yaml
# String comparison
environment: "production"
is_prod: (( environment == "production" ))  # true
is_dev: (( environment == "development" ))  # false

# Number comparison
config:
  max_retries: 3
  current_retry: 3
  
  at_max_retries: (( config.current_retry == config.max_retries ))  # true

# Boolean comparison
flags:
  enabled: true
  active: true
  
  both_on: (( flags.enabled == flags.active ))  # true

# With grab
meta:
  version: "1.2.3"

deployment:
  needs_migration: (( grab meta.version == "1.0.0" ))  # false
```

### (( != )) - Inequality

Usage: `(( EXPR1 != EXPR2 ))`

Checks if two values are not equal.

```yaml
# Basic inequality
status: "pending"
is_complete: (( status != "pending" ))  # false
not_failed: (( status != "failed" ))  # true

# Multiple checks
user:
  role: "editor"
  
  not_admin: (( user.role != "admin" ))  # true
  not_viewer: (( user.role != "viewer" ))  # true
  has_write_access: (( user.role != "viewer" ))  # true
```

### (( < )) - Less Than

Usage: `(( EXPR1 < EXPR2 ))`

```yaml
# Number comparisons
thresholds:
  warning: 80
  critical: 95

metrics:
  cpu: 75
  memory: 92
  
status:
  cpu_ok: (( metrics.cpu < thresholds.warning ))  # true
  memory_ok: (( metrics.memory < thresholds.critical ))  # true

# String comparisons (lexicographic)
version: "1.5.0"
needs_update: (( version < "2.0.0" ))  # true
```

### (( <= )) - Less Than or Equal

Usage: `(( EXPR1 <= EXPR2 ))`

```yaml
# Inclusive thresholds
limits:
  max_connections: 100
  max_retries: 3

current:
  connections: 100
  retries: 2
  
checks:
  within_connection_limit: (( current.connections <= limits.max_connections ))  # true
  can_retry: (( current.retries <= limits.max_retries ))  # true
```

### (( > )) - Greater Than

Usage: `(( EXPR1 > EXPR2 ))`

```yaml
# Performance checks
requirements:
  min_memory_gb: 8
  min_cpu_cores: 4

system:
  memory_gb: 16
  cpu_cores: 8
  
adequate:
  memory: (( system.memory_gb > requirements.min_memory_gb ))  # true
  cpu: (( system.cpu_cores > requirements.min_cpu_cores ))  # true
```

### (( >= )) - Greater Than or Equal

Usage: `(( EXPR1 >= EXPR2 ))`

```yaml
# Age verification
user:
  age: 18
  
  is_adult: (( user.age >= 18 ))  # true
  can_vote: (( user.age >= 18 ))  # true
  senior_discount: (( user.age >= 65 ))  # false

# Score thresholds
grades:
  score: 85
  
  passed: (( grades.score >= 60 ))  # true
  grade_a: (( grades.score >= 90 ))  # false
  grade_b: (( grades.score >= 80 ))  # true
```

## Ternary Operator

### (( ?: )) - Conditional Expression

Usage: `(( CONDITION ? TRUE_VALUE : FALSE_VALUE ))`

Returns TRUE_VALUE if CONDITION is true, otherwise FALSE_VALUE.

```yaml
# Basic ternary
environment: "production"
config:
  debug: (( environment == "production" ? false : true ))  # false
  log_level: (( environment == "production" ? "error" : "debug" ))  # "error"
  replicas: (( environment == "production" ? 3 : 1 ))  # 3

# Nested ternary
score: 85
grade: (( score >= 90 ? "A" : score >= 80 ? "B" : score >= 70 ? "C" : "F" ))  # "B"

# With complex conditions
user:
  is_premium: true
  trial_days_left: 0
  
  access_level: (( user.is_premium ? "full" : user.trial_days_left > 0 ? "trial" : "basic" ))  # "full"

# Dynamic configuration
cluster:
  size: "large"
  
  nodes: (( cluster.size == "small" ? 1 : cluster.size == "medium" ? 3 : 5 ))  # 5
  memory_per_node: (( cluster.size == "large" ? "8Gi" : "4Gi" ))  # "8Gi"

# With grab and other operators
defaults:
  timeout: 30

service:
  custom_timeout: null
  timeout: (( grab service.custom_timeout ? service.custom_timeout : defaults.timeout ))  # 30
```

## Complex Expression Examples

### Combining Multiple Operators
```yaml
# Authentication logic
auth:
  user:
    authenticated: true
    verified: false
    role: "editor"
    subscription: "premium"
  
  # Complex access control
  can_publish: (( auth.user.authenticated && auth.user.verified && 
                  (auth.user.role == "admin" || auth.user.role == "editor") ))  # false
  
  # Premium features
  premium_access: (( auth.user.authenticated && 
                     (auth.user.subscription == "premium" || auth.user.role == "admin") ))  # true

# Validation with multiple conditions
form:
  email: "user@example.com"
  password: "short"
  age: 16
  terms_accepted: true
  
  validation:
    email_valid: (( ! empty form.email && form.email != "" ))  # true
    password_valid: (( ! empty form.password && len form.password >= 8 ))  # false (if len existed)
    age_valid: (( form.age >= 13 && form.age <= 120 ))  # true
    
    all_valid: (( form.validation.email_valid && 
                  form.validation.password_valid && 
                  form.validation.age_valid && 
                  form.terms_accepted ))  # false
```

### Environment-Based Configuration
```yaml
meta:
  env: (( grab $ENV || "development" ))
  region: (( grab $REGION || "us-east-1" ))

config:
  # Multi-condition feature flags
  enable_caching: (( meta.env == "production" || meta.env == "staging" ))
  enable_debug: (( meta.env != "production" && meta.env != "staging" ))
  
  # Regional configuration
  use_multi_az: (( meta.env == "production" && meta.region != "local" ))
  
  # Computed values
  database:
    host: (( meta.env == "production" ? "prod.db.example.com" : 
             meta.env == "staging" ? "stage.db.example.com" : 
             "localhost" ))
    
    pool_size: (( meta.env == "production" ? 20 : 5 ))
    ssl_required: (( meta.env == "production" || meta.env == "staging" ))
```

### Error Handling and Defaults
```yaml
# Safe navigation with defaults
external_config:
  timeout: null
  retries: 0
  
app_config:
  # Multiple fallback levels
  timeout: (( grab external_config.timeout || grab defaults.timeout || 30 ))
  
  # Conditional defaults based on other values
  retries: (( external_config.retries > 0 ? external_config.retries : 
              meta.env == "production" ? 5 : 3 ))
  
  # Validation before use
  valid_config: (( app_config.timeout > 0 && app_config.retries >= 0 ))
  
  # Safe usage with ternary
  final_timeout: (( app_config.valid_config ? app_config.timeout : 30 ))
```

## Operator Precedence

Operators follow standard precedence rules (highest to lowest):
1. Parentheses `()`
2. Unary operators: `!`
3. Multiplication/Division: `*`, `/`, `%`
4. Addition/Subtraction: `+`, `-`
5. Comparisons: `<`, `<=`, `>`, `>=`
6. Equality: `==`, `!=`
7. Logical AND: `&&`
8. Logical OR: `||`
9. Ternary: `?:`

```yaml
# Examples showing precedence
results:
  # Arithmetic before comparison
  a: (( 2 + 3 > 4 ))  # true (5 > 4)
  
  # Comparison before logical
  b: (( 5 > 3 && 2 < 4 ))  # true
  
  # AND before OR
  c: (( true || false && false ))  # true (true || (false && false))
  
  # Use parentheses to override
  d: (( (true || false) && false ))  # false
```

## See Also

- [Expression operators documentation](/doc/expression-operators.md)
- [Expression examples](/examples/expression-operators/)
- [Math operators](math-calculations.md)
- [Data manipulation operators](data-manipulation.md)