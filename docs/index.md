# graft Documentation

Welcome to the graft documentation! graft is a general purpose YAML & JSON merging tool with complex expression operators.

## Documentation Structure

### 🚀 [Getting Started](getting-started.md)
New to graft? Start here to learn the basics of YAML merging and simple operators.

### 📚 Concepts
Understand the core concepts behind graft:
- [Merging Rules](concepts/merging.md) - How graft merges YAML files
- [Expression Evaluation](concepts/expression-evaluation.md) - How operators are processed
- [Nested Expressions](concepts/nested-expressions.md) - Composing operators within operators
- [Array Merging](concepts/array-merging.md) - Special array merge operators
- [Environment Variables](concepts/environment-variables.md) - Using environment variables

### 📖 Operator Reference
Detailed reference for all graft operators, organized by category:

#### Data Manipulation
- [concat](operators/data-manipulation.md#concat) - String concatenation
- [join](operators/data-manipulation.md#join) - Join arrays with separator
- [split](operators/data-manipulation.md#split) - Split strings into arrays
- [regexp](operators/data-manipulation.md#regexp) - Regular expression matching
- [base64](operators/data-manipulation.md#base64) - Base64 encoding
- [base64-decode](operators/data-manipulation.md#base64-decode) - Base64 decoding
- [stringify](operators/data-manipulation.md#stringify) - Convert to string
- [parse](operators/data-manipulation.md#parse) - Parse strings to data structures

#### Data References
- [grab](operators/data-references.md#grab) - Reference other values
- [concat](operators/data-references.md#concat) - Concatenate multiple references
- [defer](operators/data-references.md#defer) - Defer evaluation
- [inject](operators/data-references.md#inject) - Inject values into templates
- [param](operators/data-references.md#param) - Parameter references

#### Array Operations
- [append](operators/array-operations.md#append) - Append to arrays
- [prepend](operators/array-operations.md#prepend) - Prepend to arrays
- [merge](operators/array-operations.md#merge) - Deep merge arrays
- [inline](operators/array-operations.md#inline) - Inline array elements
- [flatten](operators/array-operations.md#flatten) - Flatten nested arrays
- [uniq](operators/array-operations.md#uniq) - Remove duplicates
- [cartesian-product](operators/array-operations.md#cartesian-product) - Cartesian product
- [sort](operators/array-operations.md#sort) - Sort arrays
- [shuffle](operators/array-operations.md#shuffle) - Randomize arrays

#### Math & Calculations
- [calc](operators/math-calculations.md#calc) - Mathematical calculations
- [ips](operators/math-calculations.md#ips) - IP math operations
- [Arithmetic Operators](operators/math-calculations.md#arithmetic) - +, -, *, /, %

#### Expression Operators
- [Comparison](operators/expression-operators.md#comparison) - ==, !=, <, >, <=, >=
- [Boolean Logic](operators/expression-operators.md#boolean) - &&, ||, !
- [Ternary](operators/expression-operators.md#ternary) - Conditional operator ?:

#### External Data Sources
- [vault](operators/external-data.md#vault) - HashiCorp Vault integration
- [awsparam](operators/external-data.md#awsparam) - AWS Parameter Store
- [awssecret](operators/external-data.md#awssecret) - AWS Secrets Manager
- [file](operators/external-data.md#file) - Read from files
- [load](operators/external-data.md#load) - Load YAML/JSON files

#### Utility & Metadata
- [keys](operators/utility-metadata.md#keys) - Get map keys
- [prune](operators/utility-metadata.md#prune) - Remove keys
- [empty](operators/utility-metadata.md#empty) - Check if empty
- [null](operators/utility-metadata.md#null) - Check if null
- [static_ips](operators/utility-metadata.md#static_ips) - Static IP allocation
- [type](operators/utility-metadata.md#type) - Get value type

### 🛠️ How-To Guides
Practical guides for common tasks:
- [Integrating with BOSH](guides/integrating-with-cloud-config.md)
- [Using with CredHub](guides/integrating-with-credhub.md) 
- [Vault Integration](guides/vault-integration.md)
- [AWS Integration](guides/aws-integration.md)
- [Generating graft with graft](guides/meta-programming.md)
- [Working with Go Patches](guides/go-patch.md)

### 📋 Command Reference
- [graft merge](reference/commands.md#merge) - Merge YAML files
- [graft diff](reference/commands.md#diff) - Diff YAML files
- [graft json](reference/commands.md#json) - Convert to JSON
- [graft fan](reference/commands.md#fan) - Process multi-document YAML
- [graft vaultinfo](reference/commands.md#vaultinfo) - Extract Vault paths

### 📖 Reference Documents
- [Operator Quick Reference](reference/operator-quick-reference.md) - Concise operator syntax reference
- [Use Cases Guide](reference/use-cases.md) - Common patterns and solutions
- [Quick Reference](reference/quick-reference.md) - Original quick reference guide

### 🧩 Examples
- [Basic Examples](../examples/README.md) - Simple usage examples
- [Operator Examples](operators/README.md) - Examples for each operator
- [Real-World Scenarios](guides/examples.md) - Complex use cases

## Quick Links

- [Installation](getting-started.md#installation)
- [Basic Usage](getting-started.md#basic-usage)
- [Operator Quick Reference](reference/quick-reference.md)
- [FAQ](reference/faq.md)
- [Troubleshooting](reference/troubleshooting.md)

## Advanced Features

graft supports:
- Nested operator expressions
- Environment variable expansion in references
- Complex arithmetic and boolean expressions
- Type coercion and validation

See [Expression Evaluation](concepts/expression-evaluation.md) for details.

## Contributing

See our [Contributing Guide](../CONTRIBUTING.md) for information on contributing to graft.

## License

graft is released under the [MIT License](../LICENSE).