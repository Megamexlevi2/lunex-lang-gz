# ntl:oauth2

OAuth 2.0 authorization code flow — Google, GitHub, and custom providers.

## Import

```ntl
use oauth2
```

## Built-in providers

### `oauth2.google(options)`

Creates an OAuth2 config for Google.

```ntl
val gauth = oauth2.google({
  clientId: env.GOOGLE_CLIENT_ID,
  clientSecret: env.GOOGLE_CLIENT_SECRET,
  redirectUrl: "https://myapp.com/auth/google/callback",
  scopes: ["openid", "email", "profile"]
})
```

### `oauth2.github(options)`

Creates an OAuth2 config for GitHub.

```ntl
val ghauth = oauth2.github({
  clientId: env.GITHUB_CLIENT_ID,
  clientSecret: env.GITHUB_CLIENT_SECRET,
  redirectUrl: "https://myapp.com/auth/github/callback",
  scopes: ["read:user", "user:email"]
})
```

## Custom provider

### `oauth2.create(options)`

Creates a config for any OAuth2-compatible provider.

```ntl
val auth = oauth2.create({
  clientId: "...",
  clientSecret: "...",
  redirectUrl: "...",
  scopes: ["..."],
  authURL: "https://provider.com/oauth/authorize",
  tokenURL: "https://provider.com/oauth/token"
})
```

## Config methods

All three functions return a config object with the following methods:

### `config.authURL(state?)`

Returns the authorization URL to redirect the user to. The optional `state` parameter is included in the URL for CSRF protection.

```ntl
val url = gauth.authURL("random_state_value")
res.redirect(url)
```

### `config.exchange(code)`

Exchanges an authorization code (from the callback query parameter) for an access token. Returns `{ accessToken, refreshToken, expiry }`.

```ntl
val tokens = gauth.exchange(req.query.code)
```

### `config.refresh(refreshToken)`

Refreshes an expired access token. Returns a new `{ accessToken, refreshToken, expiry }`.

```ntl
val newTokens = gauth.refresh(tokens.refreshToken)
```

### `config.fetchUser(accessToken)`

Fetches the authenticated user's profile from the provider. Returns a provider-specific user object.

```ntl
val user = gauth.fetchUser(tokens.accessToken)
io.log(user.email)
```

## Example

```ntl
use oauth2
use http
use env

val gauth = oauth2.google({
  clientId: env.GOOGLE_CLIENT_ID,
  clientSecret: env.GOOGLE_CLIENT_SECRET,
  redirectUrl: "http://localhost:3000/callback",
  scopes: ["openid", "email", "profile"]
})

http.get("/login", fn(req, res) {
  res.redirect(gauth.authURL("state123"))
})

http.get("/callback", fn(req, res) {
  val tokens = gauth.exchange(req.query.code)
  val user = gauth.fetchUser(tokens.accessToken)
  res.json({ email: user.email })
})

http.listen(3000)
```
