# WebSocket Module

WebSocket client and server for real-time bidirectional communication.

**Use case:** Build real-time applications with WebSocket support.

---

## Import

```ntl
val ws = @import("std.ws")
```

---

## Available Functions

### `createServer(port, options)`

Executes the `createServer` operation with the given parameters (port, options).

**Signature:**
```ntl
fn createServer(port, options)
```

### `on(event, handler)`

Executes the `on` operation with the given parameters (event, handler).

**Signature:**
```ntl
fn on(event, handler)
```

### `send(client, message)`

Executes the `send` operation with the given parameters (client, message).

**Signature:**
```ntl
fn send(client, message)
```

### `broadcast(message)`

Executes the `broadcast` operation with the given parameter (message).

**Signature:**
```ntl
fn broadcast(message)
```

### `start(callback)`

Executes the `start` operation with the given parameter (callback).

**Signature:**
```ntl
fn start(callback)
```

### `clientCount()`

Executes the `clientCount` operation with the given no arguments.

**Signature:**
```ntl
fn clientCount()
```

### `stop()`

Executes the `stop` operation with the given no arguments.

**Signature:**
```ntl
fn stop()
```

### `createClient(url)`

Executes the `createClient` operation with the given parameter (url).

**Signature:**
```ntl
fn createClient(url)
```

### `on(event, handler)`

Executes the `on` operation with the given parameters (event, handler).

**Signature:**
```ntl
fn on(event, handler)
```

### `connect()`

Executes the `connect` operation with the given no arguments.

**Signature:**
```ntl
fn connect()
```

### `send(message)`

Executes the `send` operation with the given parameter (message).

**Signature:**
```ntl
fn send(message)
```

### `close()`

Executes the `close` operation with the given no arguments.

**Signature:**
```ntl
fn close()
```

