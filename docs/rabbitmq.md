# RabbitMQ Module

RabbitMQ client for message publishing, consuming, and queue management.

**Use case:** Integrate with RabbitMQ for distributed messaging.

---

## Import

```lunex
val rabbitmq = @import("std.rabbitmq")
```

---

## Available Functions

### `connect(url)`

Executes the `connect` operation with the given parameter (url).

**Signature:**
```lunex
fn connect(url)
```

