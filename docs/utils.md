# Utilities Module

General-purpose utility functions and helpers for common operations.

**Use case:** Access helper functions for common tasks.

---

## Import

```ntl
val utils = @import("std.utils")
```

---

## Available Functions

### `sleep(ms)`

Executes the `sleep` operation with the given parameter (ms).

**Signature:**
```ntl
fn sleep(ms)
```

### `now()`

Executes the `now` operation with the given no arguments.

**Signature:**
```ntl
fn now()
```

### `timestamp()`

Executes the `timestamp` operation with the given no arguments.

**Signature:**
```ntl
fn timestamp()
```

### `uuid()`

Executes the `uuid` operation with the given no arguments.

**Signature:**
```ntl
fn uuid()
```

### `range(start, end, step)`

Executes the `range` operation with the given parameters (start, end, step).

**Signature:**
```ntl
fn range(start, end, step)
```

### `chunk(arr, size)`

Executes the `chunk` operation with the given parameters (arr, size).

**Signature:**
```ntl
fn chunk(arr, size)
```

### `flatten(arr, depth)`

Executes the `flatten` operation with the given parameters (arr, depth).

**Signature:**
```ntl
fn flatten(arr, depth)
```

### `flatMap(arr, cb)`

Executes the `flatMap` operation with the given parameters (arr, cb).

**Signature:**
```ntl
fn flatMap(arr, cb)
```

### `zip(...arrays)`

Executes the `zip` operation with the given parameter (...arrays).

**Signature:**
```ntl
fn zip(...arrays)
```

### `unzip(arr)`

Executes the `unzip` operation with the given parameter (arr).

**Signature:**
```ntl
fn unzip(arr)
```

### `intersection(a, b)`

Executes the `intersection` operation with the given parameters (a, b).

**Signature:**
```ntl
fn intersection(a, b)
```

### `difference(a, b)`

Executes the `difference` operation with the given parameters (a, b).

**Signature:**
```ntl
fn difference(a, b)
```

### `union(...arrays)`

Executes the `union` operation with the given parameter (...arrays).

**Signature:**
```ntl
fn union(...arrays)
```

### `uniq(arr)`

Executes the `uniq` operation with the given parameter (arr).

**Signature:**
```ntl
fn uniq(arr)
```

### `uniqBy(arr, key)`

Executes the `uniqBy` operation with the given parameters (arr, key).

**Signature:**
```ntl
fn uniqBy(arr, key)
```

### `groupBy(arr, key)`

Executes the `groupBy` operation with the given parameters (arr, key).

**Signature:**
```ntl
fn groupBy(arr, key)
```

### `countBy(arr, key)`

Executes the `countBy` operation with the given parameters (arr, key).

**Signature:**
```ntl
fn countBy(arr, key)
```

### `partition(arr, predicate)`

Executes the `partition` operation with the given parameters (arr, predicate).

**Signature:**
```ntl
fn partition(arr, predicate)
```

### `sortBy(arr, key, order)`

Executes the `sortBy` operation with the given parameters (arr, key, order).

**Signature:**
```ntl
fn sortBy(arr, key, order)
```

### `pick(obj, ...keys)`

Executes the `pick` operation with the given parameters (obj, ...keys).

**Signature:**
```ntl
fn pick(obj, ...keys)
```

### `omit(obj, ...keys)`

Executes the `omit` operation with the given parameters (obj, ...keys).

**Signature:**
```ntl
fn omit(obj, ...keys)
```

### `merge(...objects)`

Executes the `merge` operation with the given parameter (...objects).

**Signature:**
```ntl
fn merge(...objects)
```

### `assign(...objects)`

Executes the `assign` operation with the given parameter (...objects).

**Signature:**
```ntl
fn assign(...objects)
```

### `keys(obj)`

Executes the `keys` operation with the given parameter (obj).

**Signature:**
```ntl
fn keys(obj)
```

### `values(obj)`

Executes the `values` operation with the given parameter (obj).

**Signature:**
```ntl
fn values(obj)
```

### `entries(obj)`

Executes the `entries` operation with the given parameter (obj).

**Signature:**
```ntl
fn entries(obj)
```

### `fromEntries(arr)`

Executes the `fromEntries` operation with the given parameter (arr).

**Signature:**
```ntl
fn fromEntries(arr)
```

### `hasKey(obj, key)`

Executes the `hasKey` operation with the given parameters (obj, key).

**Signature:**
```ntl
fn hasKey(obj, key)
```

### `invert(obj)`

Executes the `invert` operation with the given parameter (obj).

**Signature:**
```ntl
fn invert(obj)
```

### `mapValues(obj, cb)`

Executes the `mapValues` operation with the given parameters (obj, cb).

**Signature:**
```ntl
fn mapValues(obj, cb)
```

### `sum(arr)`

Executes the `sum` operation with the given parameter (arr).

**Signature:**
```ntl
fn sum(arr)
```

### `mean(arr)`

Executes the `mean` operation with the given parameter (arr).

**Signature:**
```ntl
fn mean(arr)
```

### `median(arr)`

Executes the `median` operation with the given parameter (arr).

**Signature:**
```ntl
fn median(arr)
```

### `min(...args)`

Executes the `min` operation with the given parameter (...args).

**Signature:**
```ntl
fn min(...args)
```

### `max(...args)`

Executes the `max` operation with the given parameter (...args).

**Signature:**
```ntl
fn max(...args)
```

### `clamp(n, lo, hi)`

Executes the `clamp` operation with the given parameters (n, lo, hi).

**Signature:**
```ntl
fn clamp(n, lo, hi)
```

### `lerp(a, b, t)`

Executes the `lerp` operation with the given parameters (a, b, t).

**Signature:**
```ntl
fn lerp(a, b, t)
```

### `random(lo, hi)`

Executes the `random` operation with the given parameters (lo, hi).

**Signature:**
```ntl
fn random(lo, hi)
```

### `randInt(lo, hi)`

Executes the `randInt` operation with the given parameters (lo, hi).

**Signature:**
```ntl
fn randInt(lo, hi)
```

### `shuffle(arr)`

Executes the `shuffle` operation with the given parameter (arr).

**Signature:**
```ntl
fn shuffle(arr)
```

### `sample(arr)`

Executes the `sample` operation with the given parameter (arr).

**Signature:**
```ntl
fn sample(arr)
```

### `sampleSize(arr, n)`

Executes the `sampleSize` operation with the given parameters (arr, n).

**Signature:**
```ntl
fn sampleSize(arr, n)
```

### `camelCase(s)`

Executes the `camelCase` operation with the given parameter (s).

**Signature:**
```ntl
fn camelCase(s)
```

### `snakeCase(s)`

Executes the `snakeCase` operation with the given parameter (s).

**Signature:**
```ntl
fn snakeCase(s)
```

### `kebabCase(s)`

Executes the `kebabCase` operation with the given parameter (s).

**Signature:**
```ntl
fn kebabCase(s)
```

### `titleCase(s)`

Executes the `titleCase` operation with the given parameter (s).

**Signature:**
```ntl
fn titleCase(s)
```

### `slugify(s)`

Executes the `slugify` operation with the given parameter (s).

**Signature:**
```ntl
fn slugify(s)
```

### `truncate(s, max, suffix)`

Executes the `truncate` operation with the given parameters (s, max, suffix).

**Signature:**
```ntl
fn truncate(s, max, suffix)
```

### `pad(s, n, char)`

Executes the `pad` operation with the given parameters (s, n, char).

**Signature:**
```ntl
fn pad(s, n, char)
```

### `padStart(s, n, char)`

Executes the `padStart` operation with the given parameters (s, n, char).

**Signature:**
```ntl
fn padStart(s, n, char)
```

### `padEnd(s, n, char)`

Executes the `padEnd` operation with the given parameters (s, n, char).

**Signature:**
```ntl
fn padEnd(s, n, char)
```

### `repeat(s, n)`

Executes the `repeat` operation with the given parameters (s, n).

**Signature:**
```ntl
fn repeat(s, n)
```

### `template(s, vars)`

Executes the `template` operation with the given parameters (s, vars).

**Signature:**
```ntl
fn template(s, vars)
```

### `times(n, cb)`

Executes the `times` operation with the given parameters (n, cb).

**Signature:**
```ntl
fn times(n, cb)
```

### `pipe(...cbs)`

Executes the `pipe` operation with the given parameter (...cbs).

**Signature:**
```ntl
fn pipe(...cbs)
```

### `compose(...cbs)`

Executes the `compose` operation with the given parameter (...cbs).

**Signature:**
```ntl
fn compose(...cbs)
```

### `memoize(cb)`

Executes the `memoize` operation with the given parameter (cb).

**Signature:**
```ntl
fn memoize(cb)
```

### `once(cb)`

Executes the `once` operation with the given parameter (cb).

**Signature:**
```ntl
fn once(cb)
```

### `negate(cb)`

Executes the `negate` operation with the given parameter (cb).

**Signature:**
```ntl
fn negate(cb)
```

### `formatNumber(n, decimals)`

Executes the `formatNumber` operation with the given parameters (n, decimals).

**Signature:**
```ntl
fn formatNumber(n, decimals)
```

### `formatBytes(n)`

Executes the `formatBytes` operation with the given parameter (n).

**Signature:**
```ntl
fn formatBytes(n)
```

### `toNumber(v)`

Executes the `toNumber` operation with the given parameter (v).

**Signature:**
```ntl
fn toNumber(v)
```

### `toString(v)`

Executes the `toString` operation with the given parameter (v).

**Signature:**
```ntl
fn toString(v)
```

### `toJSON(v)`

Executes the `toJSON` operation with the given parameter (v).

**Signature:**
```ntl
fn toJSON(v)
```

### `fromJSON(s)`

Executes the `fromJSON` operation with the given parameter (s).

**Signature:**
```ntl
fn fromJSON(s)
```

### `clone(v)`

Executes the `clone` operation with the given parameter (v).

**Signature:**
```ntl
fn clone(v)
```

### `equal(a, b)`

Executes the `equal` operation with the given parameters (a, b).

**Signature:**
```ntl
fn equal(a, b)
```

### `isEmpty(v)`

Executes the `isEmpty` operation with the given parameter (v).

**Signature:**
```ntl
fn isEmpty(v)
```

### `isNil(v)`

Executes the `isNil` operation with the given parameter (v).

**Signature:**
```ntl
fn isNil(v)
```

### `deepClone(v)`

Executes the `deepClone` operation with the given parameter (v).

**Signature:**
```ntl
fn deepClone(v)
```

### `deepEqual(a, b)`

Executes the `deepEqual` operation with the given parameters (a, b).

**Signature:**
```ntl
fn deepEqual(a, b)
```

