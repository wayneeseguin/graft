# graft Operators Documentation

graft operators are special expressions that allow you to manipulate and transform your YAML/JSON data during the merge process. They are written in the form `(( operator arguments ))`.

> **Enhanced Parser Note**: graft now uses an enhanced parser by default that supports additional expression operators. See the [Expression Operators](expression-operators.md) section for details on boolean logic, comparisons, ternary operator, and more.

> **New Features**: 
> - **[Nested Expressions](../concepts/nested-expressions.md)** - Operators can now be nested within other operators for powerful compositions
> - **[Environment Variables in References](../concepts/environment-variables.md)** - Use `$VAR` syntax in reference paths for dynamic lookups

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

## Complete Operator Reference

### Core Operators

- **[Arithmetic Operators](#arithmetic-operators)**
  - [+ (addition)](#addition)
  - [- (subtraction)](#subtraction)  
  - [* (multiplication)](#multiplication)
  - [/ (division)](#division)
  - [% (modulo)](#modulo)
- [calc](#calc) - Complex arithmetic expressions
- [cartesian-product](#cartesian-product) - Generate cartesian product of arrays
- [concat](#concat) - Concatenate strings and values
- [defer](#defer) - Defer operator evaluation for template generation
- [empty](#empty) - Check if a value is empty
- [file](#file) - Read local files
- [grab](#grab) - Reference values from elsewhere in the document
- [inject](#inject) - Inject data into specific paths
- [ips](#ips) - Calculate IP addresses from CIDR ranges
- [join](#join) - Join array elements with a delimiter
- [keys](#keys) - Get keys from a map
- [load](#load) - Load and merge external YAML/JSON files
- [negate](#negate) - Negate a boolean value
- [param](#param) - Require parameters to be provided
- [prune](#prune) - Remove keys from output
- [shuffle](#shuffle) - Randomly shuffle array elements
- [sort](#sort) - Sort array elements
- [static_ips](#static_ips) - Generate static IPs for BOSH deployments
- [stringify](#stringify) - Convert any value to a string
- [vault](#vault) - Retrieve secrets from HashiCorp Vault
- [vault-try](#vault-try) - Try multiple Vault paths (deprecated, use vault with multiple paths)
- [awsparam](#awsparam) - Get AWS SSM parameters
- [awssecret](#awssecret) - Get AWS Secrets Manager secrets
- [base64](#base64) - Base64 encode a string
- [base64-decode](#base64-decode) - Decode a base64 string

### Array Merge Operators

These operators are specific to merging arrays. For more detail see the [array merging documentation](../concepts/array-merging.md):

- `(( append ))` - Adds the data to the end of the corresponding array in the root document.
- `(( prepend ))` - Inserts the data at the beginning of the corresponding array in the root document.
- `(( insert ))` - Inserts the data before or after a specified index, or object.
- `(( merge ))` - Merges the data on top of an existing array based on a common key. This requires each element to be an object, all with the common key used for merging.
- `(( inline ))` - Merges the data on top of an existing array, based on the indices of the array.
- `(( replace ))` - Removes the existing array, and replaces it with the new one.
- `(( delete ))` - Deletes data at a specific index, or objects identified by the value of a specified key.

## Operator Arguments

Most `graft` operators have arguments. There are three basic types to the arguments:
- **Literal values** (strings/numbers/booleans)
- **References** (paths defining a datastructure in the root document)
- **Environment variables**

Arguments can also make use of a logical-or (`||`) to failover to other values. See our notes on [environment variables and default values](../concepts/environment-variables.md) for more information on environment variables and the logical-or.

## Important Notes

**Path Syntax Limitation**: You cannot use the convenient graft path syntax (`path.to.your.property`) in case one of the elements (e.g. named entry element) contains a dot as part of the actual key. The dot is in line with the YAML syntax, however it cannot be used since graft uses it as a separator internally. This also applies to operators, where it is not immediately obvious that a path is used like with the `(( prune ))` operator. As a workaround, depending on the actual use-case, it is often possible to replace the graft operator with a equivalent [go-patch] operator file.

## See Also

- [Examples Directory](../../examples) - Practical examples for each operator
- [Quick Reference](../reference/operator-quick-reference.md) - One-page operator reference
- [Expression Operators](expression-operators.md) - Boolean logic and comparisons
- [Array Operations](array-operations.md) - Detailed array manipulation
- [External Data](external-data.md) - Working with external data sources