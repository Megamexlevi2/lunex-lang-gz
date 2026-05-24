# Mail Module

Email sending via SMTP with support for attachments, HTML content, and templates.

**Use case:** Send emails from your applications.

---

## Import

```lunex
val mail = @import("std.mail")
```

---

## Available Functions

### `createMailer(config)`

Executes the `createMailer` operation with the given parameter (config).

**Signature:**
```lunex
fn createMailer(config)
```

### `send(options)`

Executes the `send` operation with the given parameter (options).

**Signature:**
```lunex
fn send(options)
```

### `sendText(to, subject, text)`

Executes the `sendText` operation with the given parameters (to, subject, text).

**Signature:**
```lunex
fn sendText(to, subject, text)
```

### `sendHTML(to, subject, html)`

Executes the `sendHTML` operation with the given parameters (to, subject, html).

**Signature:**
```lunex
fn sendHTML(to, subject, html)
```

### `sendTemplate(to, subject, html, vars)`

Executes the `sendTemplate` operation with the given parameters (to, subject, html, vars).

**Signature:**
```lunex
fn sendTemplate(to, subject, html, vars)
```

### `send(config, options)`

Executes the `send` operation with the given parameters (config, options).

**Signature:**
```lunex
fn send(config, options)
```

### `sendText(config, to, subject, text)`

Executes the `sendText` operation with the given parameters (config, to, subject, text).

**Signature:**
```lunex
fn sendText(config, to, subject, text)
```

### `sendHTML(config, to, subject, html)`

Executes the `sendHTML` operation with the given parameters (config, to, subject, html).

**Signature:**
```lunex
fn sendHTML(config, to, subject, html)
```

