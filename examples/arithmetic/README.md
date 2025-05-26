# Arithmetic Operators in Spruce

This directory contains examples demonstrating Spruce's arithmetic operators for performing mathematical calculations in YAML configurations.

## Available Operators

### Addition (+)
Adds two or more numeric values together.
```yaml
total: (( 10 + 20 + 30 ))  # Result: 60
```

### Subtraction (-)
Subtracts one or more values from the first value.
```yaml
difference: (( 100 - 25 - 15 ))  # Result: 60
```

### Multiplication (*)
Multiplies two or more numeric values.
```yaml
product: (( 5 * 4 * 3 ))  # Result: 60
```

### Division (/)
Divides the first value by subsequent values.
```yaml
quotient: (( 100 / 4 / 5 ))  # Result: 5
```

### Modulo (%)
Returns the remainder after division.
```yaml
remainder: (( 17 % 5 ))  # Result: 2
```

## Key Features

1. **Type Support**: Works with both integers and floating-point numbers
2. **Order of Operations**: Follows standard mathematical precedence (*, /, % before +, -)
3. **Parentheses**: Use parentheses to control evaluation order
4. **Reference Support**: Can use references to other values in calculations
5. **Error Handling**: Gracefully handles division by zero and type mismatches

## Example Files

- **basic.yml**: Demonstrates basic usage of each arithmetic operator
- **calculations.yml**: Shows complex calculations with multiple operators
- **resource-calculations.yml**: Practical examples for computing resource allocations
- **percentage-scaling.yml**: Examples of percentage-based calculations and scaling

## Usage

To run these examples:
```bash
# Single file
spruce merge basic.yml

# Combine with base configuration
spruce merge base.yml calculations.yml

# See the results
spruce merge resource-calculations.yml | spruce json
```

## Common Patterns

### Dynamic Resource Allocation
```yaml
resources:
  cpu_per_instance: 2
  instance_count: (( grab meta.scaling.instances ))
  total_cpu: (( resources.cpu_per_instance * resources.instance_count ))
```

### Percentage Calculations
```yaml
scaling:
  base_capacity: 100
  scale_factor: 1.5
  new_capacity: (( scaling.base_capacity * scaling.scale_factor ))
  increase_percent: (( (scaling.new_capacity - scaling.base_capacity) / scaling.base_capacity * 100 ))
```

### Resource Distribution
```yaml
cluster:
  total_memory_gb: 256
  node_count: 8
  memory_per_node: (( cluster.total_memory_gb / cluster.node_count ))
  overhead_percent: 10
  usable_memory_per_node: (( cluster.memory_per_node * (100 - cluster.overhead_percent) / 100 ))
```