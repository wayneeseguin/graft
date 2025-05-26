# Math and Calculations Operators

These operators perform arithmetic operations and mathematical calculations in your YAML/JSON structures.

## Basic Arithmetic Operators

Spruce supports infix arithmetic operators that follow standard mathematical precedence rules (multiplication and division before addition and subtraction) and support parentheses for grouping.

### (( + )) - Addition

Usage: `(( OPERAND1 + OPERAND2 ))`

The addition operator adds two numeric values or concatenates strings.

```yaml
# Numeric addition
simple: (( 1 + 2 ))                    # 3
float_add: (( 3.14 + 2.86 ))          # 6.0
mixed: (( 10 + 2.5 ))                 # 12.5

# String concatenation
greeting: (( "Hello, " + "World!" ))   # "Hello, World!"
mixed_concat: (( "Value: " + 42 ))     # "Value: 42"

# With references
values:
  a: 10
  b: 25
result: (( values.a + values.b ))      # 35

# In complex expressions
total: (( base_price + tax + shipping ))
```

### (( - )) - Subtraction

Usage: `(( OPERAND1 - OPERAND2 ))`

The subtraction operator subtracts the second operand from the first.

```yaml
# Basic subtraction
simple: (( 10 - 3 ))                   # 7
negative: (( 5 - 8 ))                  # -3
float_sub: (( 10.5 - 2.5 ))           # 8.0

# With references
budget:
  total: 1000
  spent: 650
remaining: (( budget.total - budget.spent ))  # 350

# Calculate differences
metrics:
  current: 85.5
  previous: 72.3
change: (( metrics.current - metrics.previous ))  # 13.2
```

### (( * )) - Multiplication

Usage: `(( OPERAND1 * OPERAND2 ))`

The multiplication operator multiplies two numbers or repeats a string.

```yaml
# Numeric multiplication
simple: (( 6 * 7 ))                    # 42
float_mult: (( 2.5 * 4 ))             # 10.0

# String repetition
repeat: (( "ha" * 3 ))                 # "hahaha"
pattern: (( "-" * 10 ))                # "----------"
separator: (( "=" * 50 ))              # 50 equals signs

# With references
pricing:
  unit_price: 29.99
  quantity: 5
total: (( pricing.unit_price * pricing.quantity ))  # 149.95

# Scaling values
base_memory: 512
instances: 4
total_memory: (( base_memory * instances ))  # 2048
```

### (( / )) - Division

Usage: `(( OPERAND1 / OPERAND2 ))`

The division operator divides the first operand by the second. Division by zero results in an error.

```yaml
# Basic division
simple: (( 10 / 2 ))                   # 5.0
float_div: (( 7 / 2 ))                # 3.5
precise: (( 1 / 3 ))                  # 0.3333333...

# With references
totals:
  sum: 150
  count: 6
average: (( totals.sum / totals.count ))  # 25.0

# Calculate ratios
performance:
  successful: 950
  total: 1000
success_rate: (( performance.successful / performance.total ))  # 0.95

# Note: Division always returns a float
int_division: (( 10 / 5 ))  # 2.0 (not 2)
```

### (( % )) - Modulo

Usage: `(( OPERAND1 % OPERAND2 ))`

The modulo operator returns the remainder of integer division. Both operands must be integers.

```yaml
# Basic modulo
simple: (( 10 % 3 ))                   # 1
even_check: (( 8 % 2 ))               # 0 (even number)
odd_check: (( 7 % 2 ))                # 1 (odd number)

# With negative numbers
negative: (( -7 % 3 ))                # -1

# Practical uses
items: 17
per_page: 5
pages: (( items / per_page ))         # 3.4
full_pages: 3                         # Would need floor function
remaining: (( items % per_page ))     # 2

# Cycling through values
index: 8
colors: 3
color_index: (( index % colors ))     # 2 (cycles 0, 1, 2)
```

### Complex Arithmetic Expressions

Operators can be combined with proper precedence:

```yaml
# Precedence examples
no_parens: (( 2 + 3 * 4 ))           # 14 (multiplication first)
with_parens: (( (2 + 3) * 4 ))       # 20 (parentheses first)

# Complex formulas
values:
  base: 100
  rate: 0.15
  bonus: 50
total: (( values.base * (1 + values.rate) + values.bonus ))  # 165.0

# Multi-step calculations
order:
  subtotal: 85.00
  tax_rate: 0.08
  discount_percent: 10
  shipping: 5.00

  discount: (( order.subtotal * order.discount_percent / 100 ))  # 8.50
  taxable: (( order.subtotal - order.discount ))                 # 76.50
  tax: (( order.taxable * order.tax_rate ))                      # 6.12
  total: (( order.taxable + order.tax + order.shipping ))        # 87.62
```

## (( calc ))

Usage: `(( calc "EXPRESSION" ))`

The `(( calc ))` operator evaluates mathematical expressions provided as strings. It supports advanced functions and can reference values from the data structure.

### Supported Functions:
- `max(a, b)` - Maximum of two values
- `min(a, b)` - Minimum of two values
- `mod(a, b)` - Modulo operation
- `pow(a, b)` - Power (a^b)
- `sqrt(a)` - Square root
- `floor(a)` - Round down
- `ceil(a)` - Round up

### Examples:

```yaml
# Basic calculations
simple: (( calc "2 + 2" ))            # 4
complex: (( calc "10 * (5 + 3) / 2" ))  # 40

# Using functions
maximum: (( calc "max(10, 25)" ))     # 25
minimum: (( calc "min(10, 25)" ))     # 10
power: (( calc "pow(2, 8)" ))         # 256
square_root: (( calc "sqrt(16)" ))    # 4
rounded_up: (( calc "ceil(4.2)" ))    # 5
rounded_down: (( calc "floor(4.8)" )) # 4

# With references
values:
  radius: 5
  pi: 3.14159
area: (( calc "pi * pow(radius, 2)" ))  # 78.53975

# Complex formulas
formula:
  a: 3
  b: 4
  c: 5
# Pythagorean theorem
hypotenuse: (( calc "sqrt(pow(a, 2) + pow(b, 2))" ))  # 5

# Nested functions
result: (( calc "max(10, min(25, 15))" ))  # 15
```

See also: [calc examples](/examples/calc/)

## (( ips ))

Usage: `(( ips IP_OR_CIDR INDEX [COUNT] ))`

The `(( ips ))` operator performs IP address arithmetic, useful for network configuration and IP allocation.

### Parameters:
- `IP_OR_CIDR` - An IP address or CIDR block
- `INDEX` - Offset from the base (positive or negative)
- `COUNT` - Optional number of IPs to generate (returns array)

### Examples:

```yaml
# Add to an IP address
single_ip: (( ips "10.0.0.10" 2 ))    # "10.0.0.12"

# Start from network address with CIDR
network_ip: (( ips "10.0.0.0/24" 5 )) # "10.0.0.5"

# Generate multiple IPs
ip_list: (( ips "10.0.0.10" 0 5 ))
# Result: ["10.0.0.10", "10.0.0.11", "10.0.0.12", "10.0.0.13", "10.0.0.14"]

# Negative index from end of network
last_ip: (( ips "10.0.0.0/24" -1 ))   # "10.0.0.254"
from_end: (( ips "10.0.0.0/24" -5 3 ))
# Result: ["10.0.0.250", "10.0.0.251", "10.0.0.252"]

# Practical network configuration
network:
  subnet: "172.16.0.0/24"
  
  # Network addresses
  gateway: (( ips network.subnet 1 ))      # "172.16.0.1"
  dns: (( ips network.subnet 2 ))          # "172.16.0.2"
  
  # DHCP range
  dhcp_start: (( ips network.subnet 100 )) # "172.16.0.100"
  dhcp_pool: (( ips network.subnet 100 50 ))
  # Result: ["172.16.0.100", "172.16.0.101", ..., "172.16.0.149"]
  
  # Static assignments
  static_ips: (( ips network.subnet 10 5 ))
  # Result: ["172.16.0.10", "172.16.0.11", ..., "172.16.0.14"]

# Multiple subnets
subnets:
  - cidr: "10.1.0.0/24"
    gateway: (( ips subnets.0.cidr 1 ))
    reserved: (( ips subnets.0.cidr 1 10 ))
  - cidr: "10.2.0.0/24"
    gateway: (( ips subnets.1.cidr 1 ))
    reserved: (( ips subnets.1.cidr 1 10 ))
```

## Common Patterns

### Percentage Calculations
```yaml
stats:
  total: 1250
  completed: 1000
  percentage: (( stats.completed * 100 / stats.total ))  # 80.0
```

### Resource Allocation
```yaml
cluster:
  nodes: 5
  total_memory_gb: 128
  total_cpu_cores: 40
  
  per_node:
    memory_gb: (( cluster.total_memory_gb / cluster.nodes ))  # 25.6
    cpu_cores: (( cluster.total_cpu_cores / cluster.nodes ))  # 8.0
```

### Dynamic Scaling
```yaml
base_config:
  min_instances: 2
  max_instances: 10
  users_per_instance: 100
  
current:
  active_users: 750
  required_instances: (( calc "ceil(active_users / users_per_instance)" ))  # 8
  scaled_instances: (( calc "min(max_instances, max(min_instances, required_instances))" ))  # 8
```

### Network Planning
```yaml
datacenter:
  network: "10.0.0.0/16"
  
  # Subnet allocation
  management: (( ips datacenter.network 0 ))     # "10.0.0.0"
  production: (( ips "10.1.0.0/24" 0 ))         # "10.1.0.0"
  development: (( ips "10.2.0.0/24" 0 ))        # "10.2.0.0"
  
  # Service IPs
  services:
    lb_vip: (( ips datacenter.management 100 ))     # "10.0.0.100"
    db_cluster: (( ips datacenter.production 10 3 )) # ["10.1.0.10", "10.1.0.11", "10.1.0.12"]
```