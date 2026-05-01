# Icoo Language Examples

These examples are small, runnable `.ic` programs that introduce the language from basics to common standard-library usage.

Run a single example with:

```powershell
go run ./cmd/icoo run examples/01_data_types.ic
```

Suggested order:

1. `01_data_types.ic` - null, bool, int, float, string, array, object
2. `02_functions_and_closures.ic` - functions, anonymous functions, closures
3. `03_control_flow.ic` - `if`, ternary, `for`, `break`, `continue`, logical operators
4. `04_collections_and_iteration.ic` - arrays, objects, strings, iterators
5. `05_classes_and_methods.ic` - classes, `this`, methods, state
6. `06_errors_and_try.ic` - `throw`, `try/catch/finally`, `error()`, postfix `?`
7. `07_types_and_interfaces.ic` - type aliases, interfaces, `satisfies`
8. `08_modules_and_formats.ic` - local modules plus JSON/YAML/TOML/XML codecs
9. `09_http_client_server.ic` - `std.http.client` and `std.http.server`
10. `10_concurrency.ic` - channels, `go`, `select`

Helper modules used by the examples live under `examples/lib`.
