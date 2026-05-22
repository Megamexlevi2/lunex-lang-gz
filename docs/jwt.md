# JWT Module

JSON Web Token creation, verification, signing, and claims management.

**Use case:** Implement token-based authentication and authorization.

---

## Import

```ntl
val jwt = @import("std.jwt")
```

---

## Available Functions

### `sign(payload, secret, options)`

Executes the `sign` operation with the given parameters (payload, secret, options).

**Signature:**
```ntl
fn sign(payload, secret, options)
```

### `verify(token, secret)`

Executes the `verify` operation with the given parameters (token, secret).

**Signature:**
```ntl
fn verify(token, secret)
```

### `decode(token)`

Executes the `decode` operation with the given parameter (token).

**Signature:**
```ntl
fn decode(token)
```

### `isExpired(token)`

Executes the `isExpired` operation with the given parameter (token).

**Signature:**
```ntl
fn isExpired(token)
```

### `refresh(token, secret, expiresIn)`

Executes the `refresh` operation with the given parameters (token, secret, expiresIn).

**Signature:**
```ntl
fn refresh(token, secret, expiresIn)
```

