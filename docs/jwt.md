# ntl:jwt

JSON Web Token signing, verification, and decoding.

## Import

```ntl
val jwt = @import("std.jwt")
```

## `jwt.sign(payload, secret, options?)`

Creates a signed JWT string.

| Parameter | Type | Description |
|---|---|---|
| `payload` | object | Claims to encode (any key-value pairs) |
| `secret` | string | Signing secret |
| `options` | object? | Optional signing options |

Options:

| Field | Default | Description |
|---|---|---|
| `algorithm` | `"HS256"` | Algorithm: `"HS256"`, `"HS384"`, `"HS512"`, `"RS256"` |
| `expiresIn` | `3600` | Expiry in seconds |
| `issuer` | — | `iss` claim |
| `audience` | — | `aud` claim |
| `subject` | — | `sub` claim |

```ntl
val token = jwt.sign({ userId: 42, role: "admin" }, "mysecret", {
  expiresIn: 86400,
  algorithm: "HS256"
})
```

## `jwt.verify(token, secret, options?)`

Verifies a JWT and returns the decoded payload. Throws if the token is invalid or expired.

```ntl
val payload = jwt.verify(token, "mysecret")
io.log(payload.userId)
```

## `jwt.decode(token)`

Decodes a JWT **without** verifying the signature. Useful for reading claims from trusted internal tokens.

```ntl
val claims = jwt.decode(token)
```

## `jwt.isExpired(token)`

Returns `true` if the token's `exp` claim is in the past.

```ntl
if jwt.isExpired(token) {
  // refresh or reject
}
```

## `jwt.refresh(token, secret, options?)`

Verifies a token and issues a new one with a fresh expiry. Useful for sliding session windows.

```ntl
val newToken = jwt.refresh(oldToken, "mysecret", { expiresIn: 3600 })
```

## Example

```ntl
val jwt = @import("std.jwt")
val http = @import("std.http")
val env = @import("std.env")

http.get("/protected", fn(req, res) {
  val token = req.headers["authorization"]?.replace("Bearer ", "")
  if !token {
    res.status(401).send("unauthorized")
  }
  val user = jwt.verify(token, env.JWT_SECRET)
  res.json({ userId: user.userId })
})
```
