# Cryptography Module

Cryptographic operations including hashing, symmetric/asymmetric encryption, digital signatures, and JWT handling.

**Use case:** Secure data, verify integrity, implement authentication, and manage credentials.

---

## Import

```lunex
val crypto = @import("std.crypto")
```

---

## Available Functions

### `hash(algorithm, data)`

Executes the `hash` operation with the given parameters (algorithm, data).

**Signature:**
```lunex
fn hash(algorithm, data)
```

### `md5(data)`

Executes the `md5` operation with the given parameter (data).

**Signature:**
```lunex
fn md5(data)
```

### `sha1(data)`

Executes the `sha1` operation with the given parameter (data).

**Signature:**
```lunex
fn sha1(data)
```

### `sha256(data)`

Executes the `sha256` operation with the given parameter (data).

**Signature:**
```lunex
fn sha256(data)
```

### `sha512(data)`

Executes the `sha512` operation with the given parameter (data).

**Signature:**
```lunex
fn sha512(data)
```

### `hmac(algorithm, key, data)`

Executes the `hmac` operation with the given parameters (algorithm, key, data).

**Signature:**
```lunex
fn hmac(algorithm, key, data)
```

### `hmacSha256(key, data)`

Executes the `hmacSha256` operation with the given parameters (key, data).

**Signature:**
```lunex
fn hmacSha256(key, data)
```

### `hmacSha512(key, data)`

Executes the `hmacSha512` operation with the given parameters (key, data).

**Signature:**
```lunex
fn hmacSha512(key, data)
```

### `randomBytes(n)`

Executes the `randomBytes` operation with the given parameter (n).

**Signature:**
```lunex
fn randomBytes(n)
```

### `randomHex(n)`

Executes the `randomHex` operation with the given parameter (n).

**Signature:**
```lunex
fn randomHex(n)
```

### `randomUUID()`

Executes the `randomUUID` operation with the given no arguments.

**Signature:**
```lunex
fn randomUUID()
```

### `uuid()`

Executes the `uuid` operation with the given no arguments.

**Signature:**
```lunex
fn uuid()
```

### `token(n)`

Executes the `token` operation with the given parameter (n).

**Signature:**
```lunex
fn token(n)
```

### `encrypt(data, key)`

Executes the `encrypt` operation with the given parameters (data, key).

**Signature:**
```lunex
fn encrypt(data, key)
```

### `decrypt(data, key)`

Executes the `decrypt` operation with the given parameters (data, key).

**Signature:**
```lunex
fn decrypt(data, key)
```

### `encryptAES(data, key)`

Executes the `encryptAES` operation with the given parameters (data, key).

**Signature:**
```lunex
fn encryptAES(data, key)
```

### `decryptAES(data, key)`

Executes the `decryptAES` operation with the given parameters (data, key).

**Signature:**
```lunex
fn decryptAES(data, key)
```

### `pbkdf2(password, salt, iterations, keyLen)`

Executes the `pbkdf2` operation with the given parameters (password, salt, iterations, keyLen).

**Signature:**
```lunex
fn pbkdf2(password, salt, iterations, keyLen)
```

### `hashPassword(password, cost)`

Executes the `hashPassword` operation with the given parameters (password, cost).

**Signature:**
```lunex
fn hashPassword(password, cost)
```

### `verifyPassword(password, hash)`

Executes the `verifyPassword` operation with the given parameters (password, hash).

**Signature:**
```lunex
fn verifyPassword(password, hash)
```

### `signJWT(payload, secret, expiresIn)`

Executes the `signJWT` operation with the given parameters (payload, secret, expiresIn).

**Signature:**
```lunex
fn signJWT(payload, secret, expiresIn)
```

### `verifyJWT(tok, secret)`

Executes the `verifyJWT` operation with the given parameters (tok, secret).

**Signature:**
```lunex
fn verifyJWT(tok, secret)
```

### `jwtDecode(tok)`

Executes the `jwtDecode` operation with the given parameter (tok).

**Signature:**
```lunex
fn jwtDecode(tok)
```

### `base64Encode(data)`

Executes the `base64Encode` operation with the given parameter (data).

**Signature:**
```lunex
fn base64Encode(data)
```

### `base64Decode(data)`

Executes the `base64Decode` operation with the given parameter (data).

**Signature:**
```lunex
fn base64Decode(data)
```

### `base64UrlEncode(data)`

Executes the `base64UrlEncode` operation with the given parameter (data).

**Signature:**
```lunex
fn base64UrlEncode(data)
```

### `base64UrlDecode(data)`

Executes the `base64UrlDecode` operation with the given parameter (data).

**Signature:**
```lunex
fn base64UrlDecode(data)
```

### `toHex(data)`

Executes the `toHex` operation with the given parameter (data).

**Signature:**
```lunex
fn toHex(data)
```

### `fromHex(data)`

Executes the `fromHex` operation with the given parameter (data).

**Signature:**
```lunex
fn fromHex(data)
```

### `compare(a, b)`

Executes the `compare` operation with the given parameters (a, b).

**Signature:**
```lunex
fn compare(a, b)
```

### `timingSafeEqual(a, b)`

Executes the `timingSafeEqual` operation with the given parameters (a, b).

**Signature:**
```lunex
fn timingSafeEqual(a, b)
```

