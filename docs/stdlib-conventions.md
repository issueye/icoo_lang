# Icoo Standard Library Conventions

> **Status**: Living document â€?all new and refactored stdlib code should follow these rules.
> **Target**: v0.3.0 full alignment.

This document defines the conventions that every `std.*` module should follow. It is based on an audit of the current 30+ modules and captures both existing good patterns and areas that need convergence.

---

## 1. Error Model

### 1.1 Go â†?Icoo error bridge

All native functions return `(runtime.Value, error)`. The VM converts Go errors to Icoo `ErrorValue` objects automatically.

**Rule**: Use `fmt.Errorf` for all validation and operational errors. Do NOT return error values as part of the return Value (e.g. `{ok: false, error: "..."}`) unless the domain explicitly calls for it.

**Exception**: Protocol-level responses (HTTP, WebSocket) may embed errors in structured response objects because the transport requires it.

```go
// âœ?Good
func httpGet(args []runtime.Value) (runtime.Value, error) {
    url, err := utils.RequireStringArg("get", args[0])
    if err != nil {
        return nil, err
    }
    // ...
}

// âœ?Avoid (for non-transport functions)
func myFunc(args []runtime.Value) (runtime.Value, error) {
    return &runtime.ObjectValue{Fields: map[string]runtime.Value{
        "ok": runtime.BoolValue{Value: false},
        "error": runtime.StringValue{Value: "bad input"},
    }}, nil
}
```

### 1.2 Sentinel absence: use null

When a value is legitimately absent (key not found, header not set, optional field missing), return `runtime.NullValue{}` â€?not an error.

```go
// âœ?Good â€?missing key is not an error
func redisGet(args []runtime.Value) (runtime.Value, error) {
    // ...
    if err == redis.Nil {
        return runtime.NullValue{}, nil
    }
    // ...
}
```

**Rule**: If the operation succeeded but produced no value, return `null`. If the operation could not be performed, return an error.

### 1.3 Error messages

Error messages should be lowercase, no trailing punctuation, and include the function name:

```go
return nil, fmt.Errorf("get expects string argument")
return nil, fmt.Errorf("listen options require non-empty addr")
```

---

## 2. Return Value Style

### 2.1 Resource handles

Resources that need cleanup (connections, servers, tickers, file handles) MUST expose a `close()` method.

```go
// Required pattern
obj.Fields["close"] = &runtime.NativeFunction{Name: "close", Arity: 0, Fn: binding.close}
```

`close()` MUST:
- Be idempotent (safe to call multiple times)
- Return `runtime.NullValue{}, nil` on success
- Return `nil, err` only for unexpected I/O errors during teardown

### 2.2 Collection returns

When returning multiple items, use an array of values:

```go
// âœ?Good
return &runtime.ArrayValue{Elements: items}, nil

// âœ?Good â€?single value when only one header value
func headerGetter(args []runtime.Value) (runtime.Value, error) {
    // returns StringValue for single value, ArrayValue for multiples
}
```

### 2.3 Status / result objects

When a function returns structured results, use an Object with consistent field naming:

| Field | Type | Meaning |
|-------|------|---------|
| `ok` | Bool | Operation succeeded |
| `status` | Int | Numeric status code |
| `body` | String | Raw response body |
| `items` | Array | Paginated result items |
| `total` | Int | Total count for pagination |

HTTP response objects already follow this pattern. New modules should replicate it.

### 2.4 Optional / defaulted parameters

When a function accepts optional arguments beyond its minimum arity:

- Set `Arity: -1` in NativeFunction
- Validate `len(args)` at the top of the function
- Use clear error messages: `"set expects 2 or 3 arguments"`

```go
// âœ?Good pattern
&runtime.NativeFunction{Name: "set", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
    if len(args) < 2 || len(args) > 3 {
        return nil, fmt.Errorf("set expects 2 or 3 arguments")
    }
    // ...
}}
```

---

## 3. Naming Conventions

### 3.1 Function names: camelCase

All exported function names use camelCase.

| Domain | âœ?Correct | âœ?Avoid |
|--------|----------|---------|
| HTTP client | `getJSON`, `requestJSON`, `requestStream` | `get_json`, `request_json` |
| File system | `readFile`, `writeFile`, `readDir` | `read_file`, `write_file` |
| Time | `fromUnix`, `weekdayName` | `from_unix`, `weekday_name` |
| Crypto | `sha256`, `hmac256`, `randomBytes` | `SHA256`, `HMAC256` |

### 3.2 Redis-style prefixes

Redis commands that map directly to Redis operations may keep Redis naming conventions (e.g. `hGet`, `hSet`, `lPush`, `zAdd`). This is an intentional exception â€?the prefix indicates the data structure type.

### 3.3 Module names: dot-separated

```
std.io
std.time.basic
std.math.basic
std.net.http.client
std.net.websocket.server
std.crypto.hash
std.db.sql
```

No underscores, no hyphens.

### 3.4 Response object field names

Response object fields use camelCase:

```go
// âœ?Good
Fields: map[string]runtime.Value{
    "statusCode":    ...,
    "contentLength": ...,
    "requestId":     ...,
    "remoteAddr":    ...,
}

// âœ?Avoid
Fields: map[string]runtime.Value{
    "status_code":     ...,
    "content_length":  ...,
}
```

---

## 4. Resource Lifecycle

### 4.1 Construction

Resources that connect to external systems should offer both:
- `open(url)` â€?simple string-based construction
- `connect(options)` â€?full options-object construction

```go
// Current pattern (std.db.redis)
"connect": &runtime.NativeFunction{Name: "connect", Arity: 1, Fn: redisConnect},
"open":    &runtime.NativeFunction{Name: "open", Arity: 1, Fn: redisOpen},
```

`open` takes a URL string. `connect` takes an options object.

### 4.2 Cleanup

Every resource handle returned to Icoo code MUST:
1. Expose `close()`
2. Be safe to call after the underlying resource is already closed (nil-guard)
3. Set the internal handle to nil after closing

```go
func (h *handle) close(args []runtime.Value) (runtime.Value, error) {
    if h.client == nil {
        return runtime.NullValue{}, nil  // idempotent
    }
    err := h.client.Close()
    h.client = nil
    return runtime.NullValue{}, err
}
```

### 4.3 Streaming resources

Resources that provide streaming access (SSE connections, HTTP streams, subscriptions) MUST:
1. Expose `close()` 
2. Have `close()` stop any background goroutines
3. Return `null` (not error) when the stream ends naturally

---

## 5. Validation Helpers

### 5.1 Use `utils` package

The `internal/stdlib/utils` package provides standard validators:

```go
// String validation
url, err := utils.RequireStringArg("get", args[0])

// Type conversion
plain, err := utils.RuntimeToPlainValue(jsonValue)
converted := utils.PlainToRuntimeValue(decoded)

// ID generation
requestID := utils.GenerateRequestID()
```

Do NOT duplicate these in individual modules.

### 5.2 Type checking

For functions that accept multiple types, check with a type switch:

```go
switch typed := value.(type) {
case runtime.StringValue:
    return typed.Value, nil
case runtime.IntValue:
    return strconv.FormatInt(typed.Value, 10), nil
case runtime.NullValue:
    return "", nil
default:
    return "", fmt.Errorf("set does not support %s value", runtime.KindName(value))
}
```

---

## 6. Module Registration

### 6.1 Module structure

Every module is defined in a `LoadStd{X}Module()` function that returns a `*runtime.Module`:

```go
func LoadStdTimeModule() *runtime.Module {
    return &runtime.Module{
        Name: "std.time.basic",
        Path: "std.time.basic",
        Exports: map[string]runtime.Value{...},
        Done: true,
    }
}
```

### 6.2 Registration

All modules are registered in `internal/stdlib/modules.go` via a single switch statement.

---

## 7. Current Gap Summary (as of analysis)

| Area | Status | Priority |
|------|--------|----------|
| Error model | Mostly consistent; some HTTP functions embed errors in objects (acceptable for transport) | Low |
| Return null vs error | Consistent â€?`null` for absence, error for failure | â€?|
| camelCase naming | Consistent across all modules | â€?|
| `close()` idempotency | Consistent â€?redis, http, db all follow pattern | â€?|
| `open`/`connect` pattern | Only redis has both; db only has `open` | Medium |
| Response object fields | HTTP uses camelCase; Express follows same pattern | â€?|
| Streaming resource cleanup | HTTP stream, SSE, redis subscription all have close | â€?|
| Validation helpers | All modules use `utils` package | â€?|

### Recommended actions (v0.3.0):

1. **Add `connect` to `std.db.sql`** â€?currently only has `open(sqlite_path)`. A `connect(options)` variant would align with `std.db.redis`.
2. **Document the conventions** â€?this document serves that purpose.
3. **No urgent renames needed** â€?the current naming is already largely consistent.
