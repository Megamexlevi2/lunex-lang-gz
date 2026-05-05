# ws — WebSocket Module

The `ws` module provides a WebSocket server and client for real-time bidirectional communication.

## Import

```ntl
use ws
```

---

## WebSocket Server

### `ws.server([options])`
Create a WebSocket server.

```ntl
val server = ws.server({ port: 8080 })
```

#### Options

| Option | Default | Description |
|---|---|---|
| `port` | `8080` | Port to listen on |
| `path` | `"/"` | WebSocket endpoint path |
| `maxSize` | `1MB` | Max message size in bytes |

### Server Events

```ntl
server.on("connect", fn(client) {
  io.log("Client connected:", client.id)

  client.on("message", fn(msg) {
    io.log("Received:", msg)
    client.send("Echo: " + msg)
  })

  client.on("close", fn() {
    io.log("Client disconnected:", client.id)
  })
})

server.listen()
io.log("WebSocket server on :8080")
```

### `server.broadcast(message)`
Send a message to all connected clients.

```ntl
server.broadcast("Server announcement")
```

### `server.clients()`
Return a list of all connected clients.

```ntl
val count = server.clients().length
io.log(count, "clients connected")
```

### `server.close()`
Stop the server.

---

## WebSocket Client

### `ws.connect(url, [options])`
Connect to a WebSocket server.

```ntl
val client = ws.connect("ws://localhost:8080")
```

### Client Methods

```ntl
client.on("open", fn() {
  io.log("Connected")
  client.send("Hello!")
})

client.on("message", fn(data) {
  io.log("Got:", data)
})

client.on("close", fn(code, reason) {
  io.log("Disconnected:", code, reason)
})

client.on("error", fn(err) {
  io.error("Error:", err)
})
```

### `client.send(message)`
Send a string or JSON-serializable message.

```ntl
client.send("Hello")
client.send({ type: "ping", ts: now() })
```

### `client.close([code], [reason])`
Close the connection.

```ntl
client.close(1000, "Done")
```

---

## Chat Room Example

```ntl
use ws
use io

val server = ws.server({ port: 8080 })
val rooms = {}

server.on("connect", fn(client) {
  client.send({ type: "welcome", id: client.id })

  client.on("message", fn(msg) {
    val data = JSON.parse(msg)
    if data.type == "join" {
      client.room = data.room
      if rooms[data.room] == null {
        rooms[data.room] = []
      }
      rooms[data.room].push(client.id)
    } else if data.type == "message" {
      val room = rooms[client.room] ?? []
      for id in room {
        server.sendTo(id, { type: "message", from: client.id, text: data.text })
      }
    }
  })

  client.on("close", fn() {
    io.log("Client", client.id, "left")
  })
})

fn main() {
  server.listen()
  io.success("Chat server running on ws://localhost:8080")
}
```
