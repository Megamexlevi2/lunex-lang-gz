# crypto — Cryptography Module

The `crypto` module provides hashing, encoding, JWT handling, random generation, and symmetric encryption.

## Import

```ntl
use crypto
```

---

## Hashing

### `crypto.hash(algorithm, data)`
Hash data using the specified algorithm. Returns a hex string.

Supported algorithms: `md5`, `sha1`, `sha256`, `sha512`, `sha3_256`, `sha3_512`

```ntl
val h = crypto.hash("sha256", "hello world")
io.log(h)
// b94d27b9934d3e08a52e52d7da7dabfac484efe04294e576f07e1a8a...
```

### `crypto.hmac(algorithm, key, data)`
Compute an HMAC.

```ntl
val sig = crypto.hmac("sha256", "secret-key", "message")
```

### `crypto.md5(data)`
Shorthand for MD5 hash.

### `crypto.sha256(data)`
Shorthand for SHA-256 hash.

### `crypto.sha512(data)`
Shorthand for SHA-512 hash.

---

## Encoding

### `crypto.base64encode(data)`
Encode a string to Base64.

```ntl
val encoded = crypto.base64encode("Hello, World!")
// SGVsbG8sIFdvcmxkIQ==
```

### `crypto.base64decode(data)`
Decode a Base64 string.

```ntl
val decoded = crypto.base64decode("SGVsbG8sIFdvcmxkIQ==")
// Hello, World!
```

### `crypto.hexencode(data)`
Encode to hex.

### `crypto.hexdecode(hex)`
Decode from hex.

---

## Random

### `crypto.random(n)`
Generate `n` cryptographically random bytes as a hex string.

```ntl
val token = crypto.random(32)    // 64-char hex string
```

### `crypto.randomInt(min, max)`
Generate a random integer in `[min, max)`.

```ntl
val n = crypto.randomInt(1, 100)
```

### `crypto.uuid()`
Generate a random UUID v4.

```ntl
val id = crypto.uuid()
// "550e8400-e29b-41d4-a716-446655440000"
```

---

## JWT

### `crypto.jwtSign(payload, secret, [options])`
Sign a JWT token.

```ntl
val token = crypto.jwtSign(
  { userId: 42, role: "admin" },
  "my-secret-key",
  { expiresIn: "24h" }
)
```

### `crypto.jwtVerify(token, secret)`
Verify and decode a JWT. Returns the payload, or `null` if invalid/expired.

```ntl
val payload = crypto.jwtVerify(token, "my-secret-key")
if payload == null {
  io.error("Invalid or expired token")
  return
}
io.log("User ID:", payload.userId)
```

### `crypto.jwtDecode(token)`
Decode a JWT without verifying the signature.

```ntl
val header = crypto.jwtDecode(token)
```

---

## Encryption

### `crypto.encrypt(data, key)`
Encrypt a string using AES-256-GCM.

```ntl
val encrypted = crypto.encrypt("sensitive data", "32-byte-secret-key-0123456789ab")
```

### `crypto.decrypt(encrypted, key)`
Decrypt an encrypted string.

```ntl
val plain = crypto.decrypt(encrypted, "32-byte-secret-key-0123456789ab")
```

---

## Password Hashing

### `crypto.bcryptHash(password, [cost])`
Hash a password with bcrypt. Default cost is 10.

```ntl
val hashed = crypto.bcryptHash("my-password", 12)
```

### `crypto.bcryptVerify(password, hash)`
Verify a password against a bcrypt hash.

```ntl
val ok = crypto.bcryptVerify("my-password", hashed)
if ok {
  io.success("Password correct")
}
```

---

## Example: Auth Service

```ntl
use crypto
use io

val SECRET = "super-secret-key"

fn createToken(userId, role) {
  return crypto.jwtSign({ userId: userId, role: role }, SECRET, { expiresIn: "1h" })
}

fn verifyToken(token) {
  val payload = crypto.jwtVerify(token, SECRET)
  if payload == null {
    return null
  }
  return payload
}

fn main() {
  val token = createToken(1, "admin")
  io.log("Token:", token)

  val payload = verifyToken(token)
  io.log("User:", payload.userId, "Role:", payload.role)
}
```
