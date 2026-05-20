# mail — Email Module

The `mail` module provides SMTP email sending with support for HTML, attachments, and templating.

## Import

```ntl
val mail = @import("std.mail")
```

---

## Configuration

### `mail.configure(options)`
Configure the SMTP connection. Call this once before sending.

```ntl
mail.configure({
  host:     "smtp.gmail.com",
  port:     587,
  secure:   false,
  username: "you@gmail.com",
  password: env.get("SMTP_PASSWORD"),
  from:     "My App <noreply@example.com>",
})
```

#### Options

| Option | Description |
|---|---|
| `host` | SMTP server hostname |
| `port` | SMTP port (`465` for SSL, `587` for TLS, `25` for plain) |
| `secure` | Use TLS (`true` for port 465) |
| `username` | SMTP authentication username |
| `password` | SMTP authentication password |
| `from` | Default sender address |

---

## Sending Email

### `mail.send(message)`
Send an email. Returns `{ ok: bool, id: string }`.

```ntl
val result = mail.send({
  to:      "alice@example.com",
  subject: "Welcome!",
  text:    "Hello Alice, welcome to the app.",
  html:    "<h1>Welcome!</h1><p>Hello Alice.</p>",
})

if result.ok {
  io.success("Email sent:", result.id)
} else {
  io.error("Failed to send email")
}
```

#### Message Fields

| Field | Type | Description |
|---|---|---|
| `to` | string \| array | Recipient(s) |
| `cc` | string \| array | CC recipient(s) |
| `bcc` | string \| array | BCC recipient(s) |
| `subject` | string | Email subject |
| `text` | string | Plain text body |
| `html` | string | HTML body |
| `from` | string | Override sender |
| `replyTo` | string | Reply-To address |
| `attachments` | array | File attachments |

---

## Multiple Recipients

```ntl
mail.send({
  to: ["alice@example.com", "bob@example.com"],
  subject: "Team Update",
  text: "See the attached report.",
})
```

---

## Attachments

```ntl
mail.send({
  to: "alice@example.com",
  subject: "Your Report",
  text: "Please find the report attached.",
  attachments: [
    { filename: "report.pdf", path: "/tmp/report.pdf" },
    { filename: "data.csv",  content: "col1,col2\n1,2\n" },
  ],
})
```

---

## Email Templates

### `mail.template(html, vars)`
Render a simple email template. Replaces `{{variable}}` placeholders.

```ntl
val html = mail.template(`
  <h1>Hello, {{name}}!</h1>
  <p>Your activation code is: <strong>{{code}}</strong></p>
`, {
  name: "Alice",
  code: "ABC-123",
})

mail.send({
  to: "alice@example.com",
  subject: "Activate your account",
  html: html,
})
```

---

## Batch Sending

### `mail.sendMany(messages)`
Send multiple emails efficiently.

```ntl
val emails = users.map(fn(u) {
  {
    to: u.email,
    subject: "Your weekly report",
    text: "Hi " + u.name + ", here is your report...",
  }
})

mail.sendMany(emails)
```

---

## Example: Registration Email

```ntl
val mail = @import("std.mail")
val env = @import("std.env")
val io = @import("std.io")

env.load()

mail.configure({
  host:     env.require("SMTP_HOST"),
  port:     env.getInt("SMTP_PORT", 587),
  username: env.require("SMTP_USER"),
  password: env.require("SMTP_PASS"),
  from:     "MyApp <noreply@myapp.com>",
})

fn sendWelcome(name, email, code) {
  val html = mail.template(`
    <h2>Welcome to MyApp, {{name}}!</h2>
    <p>Verify your email: <a href="https://myapp.com/verify?code={{code}}">Click here</a></p>
  `, { name: name, code: code })

  mail.send({
    to:      email,
    subject: "Welcome to MyApp",
    html:    html,
  })
}

fn main() {
  val result = sendWelcome("Alice", "alice@example.com", "XYZ-789")
  if result.ok {
    io.success("Welcome email sent")
  }
}
```
