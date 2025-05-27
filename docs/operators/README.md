# graft Operators Documentation

graft operators are special expressions that allow you to manipulate and transform your YAML/JSON data during the merge process. They are written in the form `(( operator arguments ))`.

## Operator Categories

### 1. [Data Manipulation](data-manipulation.md)
Transform and manipulate data values:
- `concat` - Concatenate strings and values
- `join` - Join array elements with a delimiter
- `stringify` - Convert any value to a string
- `base64` - Base64 encode a string
- `base64-decode` - Decode a base64 string
- `empty` - Check if a value is empty
- `negate` - Negate a boolean value

### 2. [Math and Calculations](math-calculations.md)
Perform arithmetic and mathematical operations:
- `+` - Addition
- `-` - Subtraction
- `*` - Multiplication
- `/` - Division
- `%` - Modulo
- `calc` - Complex arithmetic expressions
- `ips` - Calculate IP addresses from CIDR ranges

### 3. [Data References and Flow](data-references.md)
Reference and control data flow:
- `grab` - Reference values from elsewhere in the document
- `inject` - Inject data into specific paths
- `defer` - Defer operator evaluation for template generation
- `param` - Require parameters to be provided

### 4. [Array Operations](array-operations.md)
Manipulate arrays and perform array-specific operations:
- **Array Merge Operators:**
  - `append` - Add items to the end of an array
  - `prepend` - Add items to the beginning of an array
  - `insert` - Insert items at a specific index
  - `merge` - Merge arrays element by element
  - `inline` - Merge array contents inline
  - `replace` - Replace entire array
  - `delete` - Remove items from array
- **Array Manipulation:**
  - `cartesian-product` - Generate cartesian product of arrays
  - `shuffle` - Randomly shuffle array elements
  - `sort` - Sort array elements

### 5. [External Data Sources](external-data.md)
Load data from external sources:
- `vault` - Retrieve secrets from HashiCorp Vault
- `vault-try` - Try multiple Vault paths
- `awsparam` - Get AWS SSM parameters
- `awssecret` - Get AWS Secrets Manager secrets
- `file` - Read local files
- `load` - Load and merge external YAML/JSON files

### 6. [Utility and Metadata](utility-metadata.md)
Utility functions and metadata operations:
- `keys` - Get keys from a map
- `prune` - Remove keys from output
- `static_ips` - Generate static IPs for BOSH deployments

### 7. [Expression Operators](expression-operators.md) *(Enhanced Parser)*
Boolean logic, comparisons, and conditionals:
- **Boolean**: `&&`, `||`, `!`
- **Comparison**: `==`, `!=`, `<`, `>`, `<=`, `>=`
- **Conditional**: `?:` (ternary operator)

## Quick Reference

| Operator | Purpose | Example |
|----------|---------|---------|
| `grab` | Reference other values | `(( grab meta.name ))` |
| `concat` | Join strings | `(( concat "Hello " name ))` |
| `vault` | Get secrets | `(( vault "secret/db:password" ))` |
| `empty` | Check if empty | `(( empty value ))` |
| `calc` | Math expressions | `(( calc "2 * (3 + 4)" ))` |
| `join` | Join array | `(( join ", " items ))` |

## Advanced Features

### [Nested Expressions](../nested-expressions.md)
Use operators within other operators:
```yaml
message: (( concat "User: " (grab user.name) " (ID: " (grab user.id) ")" ))
```

### [Environment Variables](../environment-variable-expansion.md)
Expand environment variables in references:
```yaml
config: (( grab settings.$ENVIRONMENT.database ))
```

## See Also

- [Examples Directory](/examples) - Practical examples for each operator
- [Operator API](/doc/operator-api.md) - Creating custom operators
- [QUICK-REFERENCE.md](/QUICK-REFERENCE.md) - One-page operator reference