# ntl:mysql

MySQL/MariaDB database client.

## Import

```ntl
use mysql
```

## Connection

### `mysql.connect(options)`

Connects to a MySQL server. Options:

| Field | Description |
|---|---|
| `host` | Hostname (default `"localhost"`) |
| `port` | Port (default `3306`) |
| `user` | Username |
| `password` | Password |
| `database` | Database name |
| `maxConns` | Max connection pool size (default `10`) |

Returns a connection object.

```ntl
val db = mysql.connect({
  host: "localhost",
  user: "root",
  password: "secret",
  database: "myapp"
})
```

Alternatively, pass a DSN string: `mysql.connect("user:pass@tcp(host:3306)/db")`.

## Querying

### `db.query(sql, params?)`

Executes a SELECT query and returns an array of row objects.

```ntl
val users = db.query("SELECT * FROM users WHERE active = ?", [1])
each user in users {
  io.log(user.name)
}
```

### `db.queryOne(sql, params?)`

Like `query` but returns a single row object or `null`.

```ntl
val user = db.queryOne("SELECT * FROM users WHERE id = ?", [42])
```

### `db.exec(sql, params?)`

Executes an INSERT, UPDATE, DELETE, or DDL statement. Returns `{ rowsAffected, lastInsertId }`.

```ntl
val result = db.exec("INSERT INTO users (name, email) VALUES (?, ?)", ["Alice", "alice@example.com"])
io.log(result.lastInsertId)
```

### `db.transaction(fn)`

Runs `fn` inside a transaction. The function receives a `tx` object with the same `query`, `queryOne`, and `exec` methods. If `fn` throws, the transaction is rolled back automatically.

```ntl
db.transaction(fn(tx) {
  tx.exec("UPDATE accounts SET balance = balance - 100 WHERE id = ?", [1])
  tx.exec("UPDATE accounts SET balance = balance + 100 WHERE id = ?", [2])
})
```

### `db.ping()`

Tests the connection. Returns `true` on success.

### `db.close()`

Closes the connection pool.

## Example

```ntl
use mysql
use env
use io

val db = mysql.connect({
  host: env.DB_HOST,
  user: env.DB_USER,
  password: env.DB_PASS,
  database: env.DB_NAME
})

val products = db.query("SELECT id, name, price FROM products WHERE price < ?", [50])
each p in products {
  io.log(p.name + " $" + p.price)
}

db.close()
```
