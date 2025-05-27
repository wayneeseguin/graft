# graft Documentation

This directory contains the comprehensive documentation for graft, organized into clear categories for easy navigation.

## Documentation Structure

### 📚 [Main Index](index.md)
The starting point for all graft documentation, with links to all sections.

### 🚀 [Getting Started](getting-started.md)
Quick introduction to graft, installation instructions, and basic usage examples.

### 📖 Concepts
Core concepts and theory behind graft:
- [Merging Rules](concepts/merging.md) - How graft merges YAML files
- [Nested Expressions](concepts/nested-expressions.md) - Composing operators within operators
- [Array Merging](concepts/array-merging.md) - Special handling for arrays
- [Environment Variables](concepts/environment-variables.md) - Using environment variables

### 📋 Operators
Detailed reference for all graft operators, organized by category:
- [Data Manipulation](operators/data-manipulation.md) - String and data transformation
- [Data References](operators/data-references.md) - Referencing and flow control
- [Array Operations](operators/array-operations.md) - Array-specific operators
- [Math Calculations](operators/math-calculations.md) - Arithmetic and calculations
- [Expression Operators](operators/expression-operators.md) - Boolean, comparison, conditional
- [External Data](operators/external-data.md) - Vault, AWS, file operations
- [Utility & Metadata](operators/utility-metadata.md) - Helper functions

### 🛠️ Guides
Practical how-to guides:
- [Vault Integration](guides/vault-integration.md) - Using HashiCorp Vault
- [AWS Integration](guides/aws-integration.md) - AWS Parameter Store & Secrets Manager
- [Meta Programming](guides/meta-programming.md) - Generating graft with graft

### 📋 Reference
Technical reference documentation:
- [Commands](reference/commands.md) - All graft commands
- [Operator Quick Reference](reference/operator-quick-reference.md) - Concise operator syntax reference
- [Use Cases Guide](reference/use-cases.md) - Common patterns and solutions
- [Quick Reference](reference/quick-reference.md) - Original quick reference guide

### 🧩 [Examples](../examples/)
Extensive examples demonstrating real-world usage of each operator.

## Migration from Old Documentation

This new documentation structure replaces the previous `doc/` directory. Key improvements:

1. **Better Organization** - Content is categorized by type (concepts, guides, reference)
2. **Operator Categories** - Operators are grouped by function rather than one long list
3. **More Examples** - Each operator has dedicated example files
4. **Improved Navigation** - Clear hierarchy and cross-references
5. **Modern Format** - Updated for better readability and maintenance

## Documentation Standards

### File Naming
- Use lowercase with hyphens: `vault-integration.md`
- Be descriptive but concise
- Group related content in subdirectories

### Content Structure
- Start with a clear title and overview
- Use practical examples throughout
- Include "See Also" sections for related topics
- Provide troubleshooting sections where appropriate

### Code Examples
- Use complete, runnable examples
- Include expected output
- Show both simple and advanced usage
- Add comments explaining complex operations

## Contributing

When adding new documentation:

1. Place it in the appropriate category
2. Update the index.md with a link
3. Add cross-references to related documents
4. Include practical examples
5. Test all code examples

## Quick Links

- [Installation](getting-started.md#installation)
- [Basic Usage](getting-started.md#basic-usage)
- [All Operators](operators/README.md)
- [Command Reference](reference/commands.md)
- [FAQ](reference/faq.md)

## Version

This documentation is for graft v1.31.0+, which includes:
- Enhanced parser with nested expressions
- Environment variable expansion in references
- Default values for vault operator
- Improved error messages