# Spruce Operator Quick Reference

A concise reference for all Spruce operators, organized by category.

## Most Common Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `grab` | Get value from path | `(( grab meta.name ))` |
| `concat` | Join strings | `(( concat "Hello " name ))` |
| `\|\|` | Fallback/default | `(( grab config.port \|\| 8080 ))` |
| `vault` | Vault secret | `(( vault "secret/path:key" ))` |

## String Operations

| Operator | Example | Result |
|----------|---------|--------|
| `concat` | `(( concat "a" "b" "c" ))` | `"abc"` |
| `join` | `(( join ", " mylist ))` | `"a, b, c"` |
| `split` | `(( split "a,b,c" "," ))` | `["a", "b", "c"]` |
| `base64` | `(( base64 "hello" ))` | `"aGVsbG8="` |
| `base64-decode` | `(( base64-decode "aGVsbG8=" ))` | `"hello"` |
| `stringify` | `(( stringify mydata ))` | YAML string |
| `parse` | `(( parse json_string ))` | Parsed data |

## Data Retrieval

| Operator | Example | Description |
|----------|---------|-------------|
| `grab` | `(( grab path.to.value ))` | Get value |
| `file` | `(( file "config.txt" ))` | Read file as string |
| `load` | `(( load "data.yml" ))` | Parse YAML/JSON file |
| `vault` | `(( vault "secret/db:pass" ))` | Vault lookup |
| `vault-try` | `(( vault-try "path1" "path2" \|\| "default" ))` | Try multiple paths |
| `awsparam` | `(( awsparam "/app/config" ))` | AWS SSM Parameter |
| `awssecret` | `(( awssecret "db-creds?key=password" ))` | AWS Secrets Manager |

## List Operations

| Operator | Example | Description |
|----------|---------|-------------|
| `keys` | `(( keys mymap ))` | Get map keys as list |
| `join` | `(( join "-" items ))` | Join list to string |
| `shuffle` | `(( shuffle mylist ))` | Randomize list |
| `sort` | `(( sort mylist ))` | Sort list (strings/numbers) |
| `cartesian-product` | `(( cartesian-product list1 list2 ))` | All combinations |

## Conditionals & Logic

| Operator | Example | Result |
|----------|---------|--------|
| `empty` | `(( empty myvalue ))` | Type-specific empty value |
| `null` | `(( null myvalue ))` | Check if null/empty |
| `\|\|` (or) | `(( grab a \|\| grab b \|\| "default" ))` | First non-null |
| `? :` (ternary) | `(( x > 5 ? "big" : "small" ))` | Conditional |
| `!` (not) | `(( ! grab config.disabled ))` | Boolean NOT |
| `&&` (and) | `(( grab a && grab b ))` | Boolean AND |
| `\|\|` (or) | `(( grab a \|\| grab b ))` | Boolean OR |

## Arithmetic

| Operator | Example | Result |
|----------|---------|--------|
| `+` | `(( 10 + 5 ))` | `15` |
| `-` | `(( 10 - 3 ))` | `7` |
| `*` | `(( 4 * 5 ))` | `20` |
| `/` | `(( 20 / 4 ))` | `5` |
| `%` | `(( 17 % 5 ))` | `2` |
| `calc` | `(( calc "x + y * 2" ))` | Complex math |

## Comparisons

| Operator | Example | Result |
|----------|---------|--------|
| `==` | `(( grab x == 5 ))` | `true/false` |
| `!=` | `(( grab x != 5 ))` | `true/false` |
| `<` | `(( grab x < 10 ))` | `true/false` |
| `>` | `(( grab x > 10 ))` | `true/false` |
| `<=` | `(( grab x <= 10 ))` | `true/false` |
| `>=` | `(( grab x >= 10 ))` | `true/false` |

## Array Merge Operators

| Operator | Usage | Description |
|----------|-------|-------------|
| `append` | `- (( append ))` | Add to end |
| `prepend` | `- (( prepend ))` | Add to beginning |
| `insert` | `- (( insert after 0 ))` | Insert at position |
| `merge` | `- (( merge on name ))` | Merge by key |
| `replace` | `- (( replace ))` | Replace entire array |
| `delete` | `- (( delete 2 ))` | Delete by index/key |
| `inline` | `- (( inline ))` | Merge by position |

## Special Operators

| Operator | Example | Description |
|----------|---------|-------------|
| `static_ips` | `(( static_ips 0 1 2 ))` | BOSH static IPs |
| `ips` | `(( ips "10.0.0.0/24" 5 3 ))` | Generate IP range |
| `inject` | `(( inject meta.template ))` | Inject map at current level |
| `defer` | `(( defer grab meta.name ))` | Don't evaluate (for templates) |
| `prune` | `(( prune ))` | Remove key from output |
| `param` | `(( param "Required!" ))` | Require override |

## Nested Expressions (v1.31.0+)

Operators can be nested for complex operations:

```yaml
# String building with nested grabs
message: (( concat "Hello " (grab user.name) "!" ))

# Dynamic file loading
config: (( load (concat "configs/" (grab env) ".yml") ))

# Conditional with calculations
timeout: (( grab count > 100 ? (grab base_timeout * 2) : grab base_timeout ))

# Base64 encoding concatenated values
auth: (( base64 (concat (grab user) ":" (grab pass)) ))
```

## Environment Variables in References

Use `$VAR` syntax within reference paths:

```yaml
# Direct expansion
value: (( grab configs.$ENVIRONMENT.setting ))

# Multiple variables
endpoint: (( grab services.$REGION.$SERVICE.url ))

# With defaults
config: (( grab $PROFILE || grab defaults ))
```

## Common Patterns

### Building URLs
```yaml
url: (( concat (grab protocol) "://" (grab host) ":" (grab port) (grab path) ))
```

### Conditional Defaults
```yaml
port: (( grab config.port || grab defaults.port || 8080 ))
debug: (( grab env == "prod" ? false : true ))
```

### Dynamic Configuration
```yaml
config: (( grab (concat "profiles." (grab profile) ".settings") ))
```

### Safe References with Defaults
```yaml
# Multiple fallbacks
value: (( grab overrides.value || grab config.value || grab defaults.value || "fallback" ))

# Conditional selection
database: (( grab use_replica ? grab db.replica : grab db.primary ))
```

### List Processing
```yaml
# Get all service names
service_names: (( keys services ))

# Join with custom separator
summary: (( join ", " (keys features) ))

# Count items (using calc)
count: (( calc "len($items)" ))
```

## Tips & Tricks

### Debugging
- `spruce merge --debug` - See evaluation order
- `spruce merge --trace` - Detailed trace output
- `spruce diff` - Compare merge results
- `spruce merge --cherry-pick path` - Extract specific paths

### Performance
- Use `||` for defaults instead of multiple merges
- Minimize vault/AWS calls by grouping in JSON
- Use `(( inject ))` for large template blocks

### Safety
- Always provide defaults for external data
- Use `param` for required values
- Set `REDACT=true` for sensitive data
- Test with `--skip-eval` to see structure

## Command Line Flags

| Flag | Description |
|------|-------------|
| `--skip-eval` | Don't evaluate operators |
| `--prune KEY` | Remove keys from output |
| `--cherry-pick KEY` | Only output specific keys |
| `--multi-doc` | Handle multi-document YAML |
| `--go-patch` | Use go-patch format |
| `--fallback-append` | Default to append for arrays |
| `-d, --debug` | Debug output |
| `--trace` | Verbose trace output |

## See Also

- [Full Operator Documentation](../operators/README.md)
- [Command Reference](commands.md)
- [Examples](../../examples/README.md)
- [Use Cases Guide](use-cases.md)