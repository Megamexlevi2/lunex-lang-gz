# db — In-Memory Database Module

The `db` module provides a lightweight in-memory key-value store with support for tables, querying, sorting, and basic persistence.

## Import

```ntl
use db
```

---

## Key-Value Store

### `db.set(key, value)`
Store any value under a key.

```ntl
db.set("user:1", { name: "Alice", age: 30 })
```

### `db.get(key)`
Retrieve a value by key. Returns `null` if not found.

```ntl
val user = db.get("user:1")
io.log(user.name)
```

### `db.has(key)`
Returns `true` if the key exists.

```ntl
if db.has("user:1") {
  // ...
}
```

### `db.delete(key)`
Remove a key.

```ntl
db.delete("user:1")
```

### `db.keys()`
Return all stored keys as an array.

```ntl
val keys = db.keys()
```

### `db.size()`
Return the number of stored keys.

### `db.clear()`
Delete all keys.

### `db.all()`
Return all key-value pairs as an object.

```ntl
val store = db.all()
```

---

## Tables

### `db.table(name)`
Create or access a table (like a simple collection).

```ntl
val users = db.table("users")
```

### Table Methods

#### `table.insert(record)`
Insert a record. Auto-assigns an `id` if not provided.

```ntl
users.insert({ name: "Alice", email: "alice@example.com" })
users.insert({ name: "Bob",   email: "bob@example.com" })
```

#### `table.find(id)` / `table.findById(id)`
Find a record by its ID.

```ntl
val user = users.find(1)
```

#### `table.where(predicate)`
Filter records by a predicate function.

```ntl
val admins = users.where(fn(u) { u.role == "admin" })
```

#### `table.all()`
Return all records.

```ntl
val all = users.all()
```

#### `table.update(id, changes)`
Update a record by ID.

```ntl
users.update(1, { name: "Alice Smith" })
```

#### `table.delete(id)` / `table.remove(id)`
Delete a record by ID.

```ntl
users.delete(1)
```

#### `table.count()`
Return the number of records.

```ntl
io.log(users.count(), "users")
```

#### `table.clear()`
Remove all records from the table.

#### `table.sort(key, [direction])`
Sort records by a field. Direction is `"asc"` (default) or `"desc"`.

```ntl
val sorted = users.sort("name", "asc")
```

#### `table.first()`
Return the first record or `null`.

#### `table.last()`
Return the last record or `null`.

#### `table.limit(n)`
Return the first `n` records.

```ntl
val top5 = users.limit(5)
```

---

## Persistence (optional)

### `db.save(path)`
Serialize the entire store to a JSON file.

```ntl
db.save("data.json")
```

### `db.load(path)`
Load data from a JSON file.

```ntl
db.load("data.json")
```

---

## Example: Simple User Store

```ntl
use db
use io

val users = db.table("users")

fn addUser(name, email) {
  users.insert({ name: name, email: email, createdAt: now() })
}

fn listUsers() {
  val all = users.all()
  each u in all {
    io.log(io.cyan(u.name), gray(u.email))
  }
}

fn main() {
  addUser("Alice", "alice@example.com")
  addUser("Bob",   "bob@example.com")
  addUser("Carol", "carol@example.com")

  io.log("Total users:", users.count())
  listUsers()

  val found = users.where(fn(u) { u.name == "Alice" })
  io.log("Found:", found[0].name)
}
```
