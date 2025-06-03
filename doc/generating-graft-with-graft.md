## How can I generate graft templates with graft itself?

There are a couple of ways to generate `graft` templates using `graft`.

### Defferring Execution

The `(( defer ))` operator will allow you to defer an operation to the next
`graft merge` invocation. For example:

```
do_this_next: (( defer grab data ))
data: my_value
```

Would produce the following when merged:

```
data: my_value
do_this_next: (( grab data ))
```

If necessary, you could chain multiple defers in a row:

```
defer_a_while: (( defer defer defer grab data ))
data: my_value
```

### Skipping all Evaluation

Specifying `--skip-eval` when running `graft` will result in the entire
**Eval Phase** being skipped. Data is merged together normally. If any params
are missing, they will still throw errors. If any pruning or cherry-picking
is requested, that will also still occur.
