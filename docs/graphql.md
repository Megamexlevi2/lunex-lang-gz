# ntl:graphql

GraphQL schema builder and executor for building APIs.

## Import

```ntl
use graphql
```

## `graphql.buildSchema(options)`

Builds a GraphQL schema. Returns a schema object.

| Field | Type | Description |
|---|---|---|
| `query` | object | Map of field names to resolver functions |
| `mutation` | object? | Map of field names to resolver functions |

Resolver functions receive `(args)` and must return the field value.

```ntl
val schema = graphql.buildSchema({
  query: {
    hello: fn(args) { return "Hello, World!" },
    user: fn(args) { return { id: args.id, name: "Alice" } }
  },
  mutation: {
    createUser: fn(args) { return { id: 1, name: args.name } }
  }
})
```

## Schema methods

### `schema.execute(query, variables?)`

Executes a GraphQL query string against the schema. Returns `{ data, errors? }`.

```ntl
val result = schema.execute(`{ hello }`)
io.log(result.data.hello)
```

With variables:

```ntl
val result = schema.execute(`query GetUser($id: String) { user(id: $id) { id name } }`, { id: "42" })
```

### `schema.executeRaw(query)`

Like `execute` but always returns raw JSON as a string.

## HTTP integration

Use with `ntl:http` to expose a `/graphql` endpoint:

```ntl
use graphql
use http

val schema = graphql.buildSchema({
  query: {
    ping: fn(_) { return "pong" }
  }
})

http.post("/graphql", fn(req, res) {
  val body = req.json()
  val result = schema.execute(body.query, body.variables)
  res.json(result)
})

http.listen(4000)
```

## Notes

- Resolver return values are automatically serialized: NTL objects become GraphQL objects, arrays become lists, strings and numbers are returned as-is.
- All fields are typed as `String` by default in the current implementation. For production use with complex type systems, consider generating a schema SDL string and using a dedicated GraphQL server.
- Mutations work identically to queries — the distinction is semantic only.
