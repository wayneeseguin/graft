# Graft

Grafting YAML Trees for plentiful fruits 🤪

```
                                       ┌── 🍃 (simple value)
                             ┌── 🌿 ───┤
                   ┌── 🌲 ───┤         └── 🍂 ─── 🍁 (chained)
                   │         │
                   │         └── 🌱 ───┬── 🌸
                   │                   └── 🌼 ─── 🌺 ─── 🌻 (triple chain)
YAML <── graft <───┤
                   │         ┌── 🌾 ───┬── 🍃
                   ├── 🌳 ───┤         └── 🍂 ─┬─ 🍁 ─┬─ 🌸 (complex chain)
                   │         │                 │      └─ 🌼
                   │         │                 └─ 🌺
                   │         │
                   │         └── 🌿 ───── 🌻 (simple)
                   │
                   ├── 🌴 ───┬── 🌱 ───┬── 🍃 ─── 🍂 ─── 🍁 ─── 🌸 (long chain)
                   │         │         │
                   │         │         └── 🌼 (simple)
                   │         │
                   │         └── 🌾 ───┬── 🌺
                   │                   ├── 🌻 ─── 🍃 (double)
                   │                   └── 🍂 ─┬─ 🍁 ─── 🌸 (nested chain)
                   │                           └─ 🌼
                   │
                   └── 🌵 ───┬── 🌺 ───── 🌻 (simple pair)
                             │
                             └── 🌿 ───┬── 🍃
                                       ├── 🍂 ─── 🍁
                                       └── 🌸 ─┬─ 🌼 ─── 🌺 (branched chain)
                                               └─ 🌻
```

> **Note**: graft supports additional operators including boolean logic (`&&`, `||`), comparisons (`==`, `!=`, `<`, `>`, `<=`, `>=`), ternary operator (`? :`), and negation (`!`).

[![Slack][slack-badge]][slack-channel] ( We'll be in `#graft`)

## Introducing graft

`graft` is a general purpose YAML & JSON merging tool.

It is designed to be an intuitive utility for merging YAML/JSON templates together
to generate complicated YAML/JSON config files in a repeatable fashion. It can be used
to stitch together some generic/top level definitions for the config and pull in overrides
for site-specific configurations to [DRY][dry-definition] your configs up as much as possible.

## Origins

Graft was originally forked from the Geoff Franks's absolutely amazing work, the [Spruce](https://github.com/geofffranks/spruce/tree/main) tool in order to add a few features. After much refactoring and grafting of new features it grew to become it's own tool with a different focus and use cases. It still passes all of the original Spruce tests and as such should be considered a superset of Spruce.

## How do I get started?

`graft` is available via Homebrew, just `brew tap starkandwayne/cf; brew install graft`

Alternatively, you can download a [prebuilt binaries for 64-bit Linux, or Mac OS X][releases]

## 📚 Documentation

- **[Documentation Hub](docs/index.md)** - Complete documentation organized by topic
- **[Getting Started Guide](docs/getting-started.md)** - Installation and basic usage
- **[Operator Quick Reference](docs/reference/operator-quick-reference.md)** - Concise overview of all operators
- **[Use Cases Guide](docs/reference/use-cases.md)** - Find the right operators for your needs
- **[Examples Directory](examples/)** - Practical examples for every operator

## How do I compile from source?

1. [Install Go][install-go], e.g. on Ubuntu `sudo snap install --classic go`
1. Fetch sources via `go get github.com/wayneeseguin/graft`
1. Change current directory to the source root `cd ~/go/src/github.com/wayneeseguin/graft/`
1. Compile and execute tests `make all`

## A Quick Example

```sh
# Let's build the first yaml file we will merge
$ cat <<EOF first.yml
some_data: this will be overwritten later
a_random_map:
  key1: some data
heres_an_array:
- first element
EOF

# and now build the second yaml file to merge on top of it
$ cat <<EOF second.yml
some_data: 42
a_random_map:
  key2: adding more data
heres_an_array:
- (( prepend ))
- zeroth element
more_data: 84

# what happens when we graft merge?
$ graft merge first.yml second.yml
a_random_map:
  key1: some data
  key2: adding more data
heres_an_array:
- zeroth element
- first element
more_data: 84
some_data: 42
```

The data in `second.yml` is overlayed on top of the data in `first.yml`. Check out the
[merge semantics][merge-semantics] and [array merging][array-merge] for more info on how that was done. Or,
check out [this example on play.graft.cf][play.graft-example]

## Documentation

- [What are all the graft operators, and how do they work?][operator-docs]
- [How do I use expression operators (arithmetic, boolean, comparisons)?][expression-operators]
- [What are the merge semantics of graft?][merge-semantics]
- [How can I manipulate arrays with graft?][array-merge]
- [Can I specify defaults for an operation, or use environment variables?][env-var-defaults]
- [Can I use graft with go-patch files?][go-patch-support]
- [Can I use graft with CredHub?][credhub-support]
- [Can I use graft with Vault?][vault-support]
- [How can I use default values with Vault?][vault-defaults]
- [How can I generate graft templates with graft itself?][defer]
- [How can I use graft with BOSH's Cloud Config?][cloud-config-support]
- [How do I create new operators?][operator-api]

## What else can graft do for you?

`graft` doesn't just stop at merging datastructures together. It also has the following
helpful subcommands:

`graft diff` - Allows you to get a useful diff of two YAML files, to see where they differ
semantically. This is more than a simple diff tool, as it examines the functional differences,
rather than just textual (e.g. key-ordering differences would be ignored)

`graft json` - Allows you to convert a YAML document into JSON, for consumption by something
that requires a JSON input. `graft merge` will handle both YAML + JSON documents, but produce
only YAML output.

`graft vaultinfo` - Takes a list of files that would be merged together, and analyzes what paths
in Vault would be looked up. Useful for determining explicitly what access an automated process
might need to Vault to obtain the right credentials, and nothing more. Also useful if you need
to audit what credentials your configs are retrieving for a system..

## License

Licensed under [the MIT License][license]


[slack-channel]:        https://cloudfoundry.slack.com/messages/graft/
[slack-badge]:          http://slack.cloudfoundry.org/badge.svg
[dry-definition]:       https://en.wikipedia.org/wiki/Don%27t_repeat_yourself
[releases]:             https://github.com/wayneeseguin/graft/releases/
[operator-docs]:        https://github.com/wayneeseguin/graft/blob/master/docs/operators/README.md
[expression-operators]: https://github.com/wayneeseguin/graft/blob/master/docs/operators/expression-operators.md
[merge-semantics]:      https://github.com/wayneeseguin/graft/blob/master/docs/concepts/merging.md
[array-merge]:          https://github.com/wayneeseguin/graft/blob/master/docs/concepts/array-merging.md
[env-var-defaults]:     https://github.com/wayneeseguin/graft/blob/master/docs/concepts/environment-variables.md
[go-patch-support]:     https://github.com/wayneeseguin/graft/blob/master/docs/integrations/go-patch.md
[credhub-support]:      https://github.com/wayneeseguin/graft/blob/master/docs/integrations/credhub.md
[vault-support]:        https://github.com/wayneeseguin/graft/blob/master/docs/guides/vault-integration.md
[vault-defaults]:       https://github.com/wayneeseguin/graft/blob/master/docs/guides/vault-integration.md
[defer]:                https://github.com/wayneeseguin/graft/blob/master/docs/guides/meta-programming.md
[cloud-config-support]: https://github.com/wayneeseguin/graft/blob/master/docs/integrations/bosh.md
[operator-api]:         https://github.com/wayneeseguin/graft/blob/master/docs/development/operator-api.md
[license]:              https://github.com/wayneeseguin/graft/blob/master/LICENSE
[install-go]:           https://golang.org/doc/install
