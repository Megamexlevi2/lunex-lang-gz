# http — HTTP Module

The `http` module provides an HTTP client for making requests and an HTTP server for building web APIs.

## Import

```ntl
use http
```

---

## HTTP Client

### `http.get(url, [headers])`
Perform a GET request. Returns a response object.

```ntl
val res = http.get("https://api.example.com/users")
io.log(res.status)    // 200
io.log(res.body)      // raw body string
val data = res.json() // parse body as JSON
```

### `http.post(url, body, [headers])`
Perform a POST request with a body.

```ntl
val res = http.post("https://api.example.com/users", {
  name: "Alice",
  email: "alice@example.com",
})
io.log(res.status)
```

### `http.put(url, body, [headers])`
Perform a PUT request.

```ntl
http.put("https://api.example.com/users/1", { name: "Alice Updated" })
```

### `http.patch(url, body, [headers])`
Perform a PATCH request.

### `http.delete(url, [headers])`
Perform a DELETE request.

```ntl
http.delete("https://api.example.com/users/1")
```

### `http.request(options)`
Full-control request with an options object.

```ntl
val res = http.request({
  method: "POST",
  url: "https://api.example.com/auth",
  headers: { "Content-Type": "application/json" },
  body: { username: "alice", password: "secret" },
  timeout: 5000,
})
```

### Response Object

| Property | Type | Description |
|---|---|---|
| `status` | number | HTTP status code |
| `ok` | boolean | `true` if status is 2xx |
| `body` | string | Raw response body |
| `headers` | object | Response headers |
| `json()` | fn | Parse body as JSON |

---

## HTTP Server

### `http.serve(port, handler)`
Start an HTTP server on the given port.

```ntl
http.serve(3000, fn(req, res) {
  res.json({ message: "Hello, World!" })
})
```

### `http.router()`
Create a router for defining multiple routes.

```ntl
val app = http.router()

app.get("/", fn(req, res) {
  res.html("<h1>Home</h1>")
})

app.get("/users/:id", fn(req, res) {
  val id = req.params.id
  res.json({ id: id, name: "Alice" })
})

app.post("/users", fn(req, res) {
  val body = req.body
  res.status(201).json({ created: body.name })
})

http.serve(3000, app.handler())
io.log("Server running on :3000")
```

### Request Object (`req`)

| Property | Type | Description |
|---|---|---|
| `method` | string | HTTP method (GET, POST, etc.) |
| `path` | string | Request path |
| `params` | object | URL parameters (`:id` → `params.id`) |
| `query` | object | Query string parameters |
| `headers` | object | Request headers |
| `body` | object/string | Parsed request body |

### Response Object (`res`)

| Method | Description |
|---|---|
| `res.send(text)` | Send plain text response |
| `res.json(value)` | Send JSON response |
| `res.html(html)` | Send HTML response |
| `res.status(code)` | Set status code (chainable) |
| `res.header(name, value)` | Set a header (chainable) |
| `res.redirect(url)` | Send redirect response |

---

## Middleware

```ntl
app.use(fn(req, res, next) {
  io.log(req.method, req.path)
  next()
})
```

### Built-in Middleware

```ntl
app.use(http.cors())          // Enable CORS
app.use(http.json())          // Parse JSON bodies
app.use(http.logger())        // Log all requests
app.use(http.static("public")) // Serve static files
```

---

## Example: REST API

```ntl
use http
use io

val users = [
  { id: 1, name: "Alice" },
  { id: 2, name: "Bob" },
]

val app = http.router()

app.get("/users", fn(req, res) {
  res.json(users)
})

app.get("/users/:id", fn(req, res) {
  val id = req.params.id
  val user = users.find(fn(u) { u.id == id })
  if user == null {
    res.status(404).json({ error: "Not found" })
    return
  }
  res.json(user)
})

fn main() {
  http.serve(3000, app.handler())
  io.success("API running on http://localhost:3000")
}
```
