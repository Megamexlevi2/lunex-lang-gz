# env — Environment Variables Module

The `env` module provides access to OS environment variables, `.env` file loading, and type-safe getters.

## Import

```ntl
val env = @import("std.env")
```

---

## Getting Variables

### `env.get(name, [default])`
Get an environment variable. Returns the default (or `null`) if not set.

```ntl
val port = env.get("PORT", "3000")
val secret = env.get("JWT_SECRET")
```

### `env.require(name)`
Get a variable or throw if it is not set.

```ntl
val dbUrl = env.require("DATABASE_URL")
```

### `env.getString(name, [default])`
Get as string.

### `env.getInt(name, [default])`
Get as integer.

```ntl
val port = env.getInt("PORT", 3000)
```

### `env.getBool(name, [default])`
Get as boolean (`"true"`, `"1"`, `"yes"` → `true`).

```ntl
val debug = env.getBool("DEBUG", false)
```

### `env.getFloat(name, [default])`
Get as float.

---

## Setting Variables

### `env.set(name, value)`
Set an environment variable for the current process.

```ntl
env.set("LOG_LEVEL", "debug")
```

### `env.delete(name)`
Unset a variable.

```ntl
env.delete("TEMP_TOKEN")
```

---

## Listing Variables

### `env.all()`
Return all environment variables as an object.

```ntl
val vars = env.all()
io.log(vars.PATH)
```

### `env.has(name)`
Returns `true` if the variable is set.

```ntl
if env.has("CI") {
  io.log("Running in CI")
}
```

---

## .env Files

### `env.load([path])`
Load variables from a `.env` file into the process environment. Default path is `.env`.

```ntl
env.load()          // loads .env
env.load(".env.production")
```

### .env file format

```
PORT=3000
DATABASE_URL=postgres://localhost/mydb
JWT_SECRET=my-secret-key
DEBUG=true
```

---

## Example

```ntl
val env = @import("std.env")
val io = @import("std.io")

env.load()

val port    = env.getInt("PORT", 3000)
val debug   = env.getBool("DEBUG", false)
val secret  = env.require("JWT_SECRET")

fn main() {
  io.log("Port:", port)
  io.log("Debug mode:", debug)
  io.log("Secret loaded:", secret.length > 0)
}
```
