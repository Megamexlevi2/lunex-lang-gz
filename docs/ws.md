# WebSocket Module

WebSocket client and server for real-time bidirectional communication.

**Use case:** Build real-time applications with WebSocket support.

---

## Import

```lunex
val ws = @import("std.ws")
```

---

## Available Functions

### `createServer(port, options)`

Executes the `createServer` operation with the given parameters (port, options).

**Signature:**
```lunex
fn createServer(port, options)
```

### `on(event, handler)`

Executes the `on` operation with the given parameters (event, handler).

**Signature:**
```lunex
fn on(event, handler)
```

### `send(client, message)`

Executes the `send` operation with the given parameters (client, message).

**Signature:**
```lunex
fn send(client, message)
```

### `broadcast(message)`

Executes the `broadcast` operation with the given parameter (message).

**Signature:**
```lunex
fn broadcast(message)
```

### `start(callback)`

Executes the `start` operation with the given parameter (callback).

**Signature:**
```lunex
fn start(callback)
```

### `clientCount()`

Executes the `clientCount` operation with the given no arguments.

**Signature:**
```lunex
fn clientCount()
```

### `stop()`

Executes the `stop` operation with the given no arguments.

**Signature:**
```lunex
fn stop()
```

### `createClient(url)`

Executes the `createClient` operation with the given parameter (url).

**Signature:**
```lunex
fn createClient(url)
```

### `on(event, handler)`

Executes the `on` operation with the given parameters (event, handler).

**Signature:**
```lunex
fn on(event, handler)
```

### `connect()`

Executes the `connect` operation with the given no arguments.

**Signature:**
```lunex
fn connect()
```

### `send(message)`

Executes the `send` operation with the given parameter (message).

**Signature:**
```lunex
fn send(message)
```

### `close()`

Executes the `close` operation with the given no arguments.

**Signature:**
```lunex
fn close()
```

