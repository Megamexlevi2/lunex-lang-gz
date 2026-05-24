# HTTP Module

HTTP client for making requests to web services with support for headers, cookies, and authentication.

**Use case:** Call REST APIs and interact with web services.

---

## Import

```lunex
val http = @import("std.http")
```

---

## Available Functions

### `createServer(handler)`

Executes the `createServer` operation with the given parameter (handler).

**Signature:**
```lunex
fn createServer(handler)
```

### `listen(server, port, host, callback)`

Executes the `listen` operation with the given parameters (server, port, host, callback).

**Signature:**
```lunex
fn listen(server, port, host, callback)
```

### `get(url, options)`

Executes the `get` operation with the given parameters (url, options).

**Signature:**
```lunex
fn get(url, options)
```

### `post(url, options)`

Executes the `post` operation with the given parameters (url, options).

**Signature:**
```lunex
fn post(url, options)
```

### `put(url, options)`

Executes the `put` operation with the given parameters (url, options).

**Signature:**
```lunex
fn put(url, options)
```

### `patch(url, options)`

Executes the `patch` operation with the given parameters (url, options).

**Signature:**
```lunex
fn patch(url, options)
```

### `del(url, options)`

Executes the `del` operation with the given parameters (url, options).

**Signature:**
```lunex
fn del(url, options)
```

### `head(url, options)`

Executes the `head` operation with the given parameters (url, options).

**Signature:**
```lunex
fn head(url, options)
```

### `json(res, data, status)`

Executes the `json` operation with the given parameters (res, data, status).

**Signature:**
```lunex
fn json(res, data, status)
```

### `text(res, data, status)`

Executes the `text` operation with the given parameters (res, data, status).

**Signature:**
```lunex
fn text(res, data, status)
```

### `redirect(res, url, status)`

Executes the `redirect` operation with the given parameters (res, url, status).

**Signature:**
```lunex
fn redirect(res, url, status)
```

### `router()`

Executes the `router` operation with the given no arguments.

**Signature:**
```lunex
fn router()
```

### `serve(port, handler)`

Executes the `serve` operation with the given parameters (port, handler).

**Signature:**
```lunex
fn serve(port, handler)
```

