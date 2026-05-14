# ntl:postgres

PostgreSQL client with connection pooling, parameterized queries, transactions, and a convenience insert helper.

## Import

```ntl
use postgres
```

## Connection

### `postgres.connect(dsn)`

Connects to a PostgreSQL server using a DSN string. Returns a connection object. Connections are pooled — calling `connect` with the same DSN reuses the existing pool.

```ntl
val db = postgres.connect("postgres://user:pass@localhost:5432/mydb")
```

DSN format: `postgres://[user[:password]@][host][:port]/[database][?param=value]`

## Connection methods

### `db.query(sql, params?)`

Executes a SELECT query and returns an array of row objects. Parameters are passed as an array and referenced with `$1`, `$2`, etc.

```ntl
val users = db.query("SELECT * FROM users WHERE active = $1", [true])
each user in users {
  io.log(user.name)
}
```

### `db.queryOne(sql, params?)`

Like `query` but returns a single row object, or `null` if no rows match.

```ntl
val user = db.queryOne("SELECT * FROM users WHERE id = $1", [42])
if user != null {
  io.log(user.email)
}
```

### `db.exec(sql, params?)`

Executes an INSERT, UPDATE, DELETE, or DDL statement. Returns `{ rowsAffected, command }`.

```ntl
val result = db.exec("UPDATE users SET active = $1 WHERE id = $2", [false, 7])
io.log(result.rowsAffected)
```

### `db.insert(table, data)`

Convenience helper that builds and executes an `INSERT ... RETURNING *` from a plain object. Returns the inserted row.

```ntl
val user = db.insert("users", {
  name:  "Alice",
  email: "alice@example.com",
  active: true
})
io.log(user.id)
```

### `db.transaction(fn)`

Runs `fn` inside a transaction. The function receives a `tx` object with `query` and `exec` methods. If `fn` throws, the transaction is rolled back automatically; otherwise it is committed.

```ntl
db.transaction(fn(tx) {
  tx.exec("UPDATE accounts SET balance = balance - $1 WHERE id = $2", [100, 1])
  tx.exec("UPDATE accounts SET balance = balance + $1 WHERE id = $2", [100, 2])
})
```

### `db.ping()`

Tests the connection. Returns `true` on success.

### `db.stats()`

Returns pool statistics as `{ totalConns, idleConns, acquiredConns }`.

```ntl
val s = db.stats()
io.log(s.totalConns)
```

### `db.close()`

Closes the connection pool and releases all resources.

## Example

```ntl
use postgres
use env
use io

val db = postgres.connect(env.DATABASE_URL)

val products = db.query("SELECT id, name, price FROM products WHERE price < $1", [50])
each p in products {
  io.log(p.name + " $" + p.price)
}

val newProduct = db.insert("products", { name: "Widget", price: 9.99 })
io.log("created: " + newProduct.id)

db.transaction(fn(tx) {
  tx.exec("INSERT INTO orders (product_id, qty) VALUES ($1, $2)", [newProduct.id, 3])
  tx.exec("UPDATE products SET stock = stock - $1 WHERE id = $2", [3, newProduct.id])
})

db.close()
```

## Notes

- Parameters use `$1`, `$2`, ... (PostgreSQL-style placeholders), not `?`.
- `insert` uses `RETURNING *` — the table must allow it (standard PostgreSQL behavior).
- The pool is shared per DSN within the same process; `close` shuts it down for all callers.
