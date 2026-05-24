# GraphQL Module

GraphQL client for querying GraphQL APIs with support for queries, mutations, and subscriptions.

**Use case:** Consume GraphQL APIs and build GraphQL clients.

---

## Import

```lunex
val graphql = @import("std.graphql")
```

---

## Available Functions

### `buildSchema(options)`

Executes the `buildSchema` operation with the given parameter (options).

**Signature:**
```lunex
fn buildSchema(options)
```

### `execute(schema, query, variables)`

Executes the `execute` operation with the given parameters (schema, query, variables).

**Signature:**
```lunex
fn execute(schema, query, variables)
```

