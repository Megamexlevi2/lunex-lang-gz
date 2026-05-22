# GraphQL Module

GraphQL client for querying GraphQL APIs with support for queries, mutations, and subscriptions.

**Use case:** Consume GraphQL APIs and build GraphQL clients.

---

## Import

```ntl
val graphql = @import("std.graphql")
```

---

## Available Functions

### `buildSchema(options)`

Executes the `buildSchema` operation with the given parameter (options).

**Signature:**
```ntl
fn buildSchema(options)
```

### `execute(schema, query, variables)`

Executes the `execute` operation with the given parameters (schema, query, variables).

**Signature:**
```ntl
fn execute(schema, query, variables)
```

