# ntl:rabbitmq

RabbitMQ / AMQP 0-9-1 message queue client.

## Import

```ntl
val rabbitmq = @import("std.rabbitmq")
```

## Connection

### `rabbitmq.connect(url?)`

Connects to a RabbitMQ broker. Default URL: `amqp://guest:guest@localhost:5672/`.

```ntl
val conn = rabbitmq.connect("amqp://user:pass@localhost:5672/")
```

Returns a connection object.

## Connection methods

| Method | Description |
|---|---|
| `conn.createChannel()` | Opens a new channel; returns a channel object |
| `conn.close()` | Closes the connection |
| `conn.isClosed()` | Returns `true` if the connection is closed |

## Channel methods

### Queue management

| Method | Description |
|---|---|
| `ch.declareQueue(name, options?)` | Declares a queue. Options: `{ durable, autoDelete, exclusive }` |
| `ch.declareExchange(name, type, options?)` | Declares an exchange. Types: `"direct"`, `"fanout"`, `"topic"`, `"headers"` |
| `ch.bindQueue(queue, exchange, routingKey)` | Binds a queue to an exchange with a routing key |

### Publishing

| Method | Description |
|---|---|
| `ch.publish(exchange, routingKey, body, options?)` | Publishes a raw string message |
| `ch.publishJSON(exchange, routingKey, object, options?)` | Publishes a JSON-serialized message |

Publish options: `{ persistent, priority, expiration, contentType }`.

### Consuming

#### `ch.consume(queue, handler, options?)`

Subscribes to a queue. The `handler` function receives a message object with:

| Field | Description |
|---|---|
| `body` | Raw message body string |
| `json()` | Parses `body` as JSON |
| `ack()` | Acknowledges the message |
| `nack(requeue?)` | Negatively acknowledges; optionally requeues |

Options: `{ autoAck, noLocal, exclusive, consumerTag }`.

Returns a consumer object with a `cancel()` method.

```ntl
val consumer = ch.consume("tasks", fn(msg) {
  val task = msg.json()
  io.log("processing: " + task.id)
  msg.ack()
})
```

### QoS

| Method | Description |
|---|---|
| `ch.qos(prefetchCount)` | Sets the maximum number of unacknowledged messages the broker sends per consumer |

### Close

| Method | Description |
|---|---|
| `ch.close()` | Closes the channel |

## Example

```ntl
val rabbitmq = @import("std.rabbitmq")

val conn = rabbitmq.connect("amqp://localhost")
val ch = conn.createChannel()

ch.declareQueue("jobs", { durable: true })

ch.publishJSON("", "jobs", { type: "email", to: "user@example.com" })

ch.consume("jobs", fn(msg) {
  val job = msg.json()
  io.log("sending email to " + job.to)
  msg.ack()
})
```
