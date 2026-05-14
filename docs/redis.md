# ntl:redis

Redis client for caching, pub/sub, queues, and key-value storage.

## Import

```ntl
use redis
```

## Connection

### `redis.connect(url)`

Connects to a Redis server. Returns a client object.

```ntl
val client = redis.connect("redis://localhost:6379")
```

The URL format is `redis://[:password@]host[:port][/db]`.

## String operations

| Method | Description |
|---|---|
| `client.set(key, value, ttlMs?)` | Sets a key. Optional TTL in milliseconds |
| `client.get(key)` | Gets a value; returns `null` if not found |
| `client.del(key)` | Deletes a key; returns `true`/`false` |
| `client.exists(key)` | Returns `true` if the key exists |
| `client.expire(key, ttlMs)` | Sets TTL in milliseconds on an existing key |
| `client.ttl(key)` | Returns remaining TTL in milliseconds (`-1` = no expiry, `-2` = not found) |
| `client.keys(pattern)` | Returns an array of keys matching a glob pattern (e.g. `"user:*"`) |

## Counters

| Method | Description |
|---|---|
| `client.incr(key)` | Increments an integer key by 1; returns new value |
| `client.incrBy(key, amount)` | Increments by `amount`; returns new value |
| `client.decr(key)` | Decrements by 1; returns new value |

## Hashes

| Method | Description |
|---|---|
| `client.hset(key, field, value)` | Sets a hash field |
| `client.hget(key, field)` | Gets a hash field value |
| `client.hgetall(key)` | Returns all fields as an object |
| `client.hdel(key, field)` | Deletes a hash field |

## Lists

| Method | Description |
|---|---|
| `client.lpush(key, value)` | Prepends a value to a list |
| `client.rpush(key, value)` | Appends a value to a list |
| `client.lpop(key)` | Removes and returns the first element |
| `client.rpop(key)` | Removes and returns the last element |
| `client.lrange(key, start, stop)` | Returns a slice of the list (`0, -1` = all) |

## Sets

| Method | Description |
|---|---|
| `client.sadd(key, member)` | Adds a member to a set |
| `client.smembers(key)` | Returns all members of a set |
| `client.sismember(key, member)` | Returns `true` if member is in the set |
| `client.srem(key, member)` | Removes a member from a set |

## Example

```ntl
use redis

val r = redis.connect("redis://localhost:6379")

r.set("session:abc", "user123", 3600000)
val user = r.get("session:abc")

r.incr("page:views")
r.hset("user:1", "name", "Alice")
val name = r.hget("user:1", "name")
```
