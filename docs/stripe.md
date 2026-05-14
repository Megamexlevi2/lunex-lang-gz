# ntl:stripe

Stripe payments integration — payment intents, customers, products, subscriptions, checkout sessions, and webhooks.

## Import

```ntl
use stripe
```

## Initialization

### `stripe.init(secretKey)`

Sets your Stripe secret key. Must be called before any other method.

```ntl
stripe.init("sk_live_...")
```

## Payment intents

### `stripe.createPaymentIntent(options)`

Creates a payment intent and returns its details.

| Option | Type | Description |
|---|---|---|
| `amount` | number | Amount in smallest currency unit (e.g. cents) |
| `currency` | string | ISO currency code (e.g. `"usd"`) |
| `customer` | string? | Customer ID |
| `description` | string? | Payment description |

Returns `{ id, amount, currency, status, clientSecret }`.

### `stripe.getPaymentIntent(id)`

Retrieves a payment intent by ID. Returns `{ id, amount, currency, status }`.

## Customers

### `stripe.createCustomer(options)`

| Option | Type | Description |
|---|---|---|
| `email` | string | Customer email |
| `name` | string? | Customer name |
| `phone` | string? | Phone number |

Returns `{ id, email, name }`.

### `stripe.getCustomer(id)`

Retrieves a customer by ID.

## Products and prices

### `stripe.createProduct(options)`

| Option | Type | Description |
|---|---|---|
| `name` | string | Product name |
| `description` | string? | Product description |

Returns `{ id, name }`.

### `stripe.createPrice(options)`

| Option | Type | Description |
|---|---|---|
| `productId` | string | Product ID |
| `amount` | number | Unit amount in smallest currency unit |
| `currency` | string | ISO currency code |
| `recurring` | string? | `"month"` or `"year"` for subscriptions |

Returns `{ id, amount, currency }`.

## Checkout sessions

### `stripe.createCheckoutSession(options)`

| Option | Type | Description |
|---|---|---|
| `priceId` | string | Price ID |
| `successUrl` | string | Redirect URL on success |
| `cancelUrl` | string | Redirect URL on cancel |
| `mode` | string | `"payment"` or `"subscription"` |
| `quantity` | number? | Item quantity (default 1) |

Returns `{ id, url }`.

## Subscriptions

### `stripe.createSubscription(options)`

| Option | Type | Description |
|---|---|---|
| `customerId` | string | Customer ID |
| `priceId` | string | Price ID |

Returns `{ id, status, currentPeriodEnd }`.

### `stripe.cancelSubscription(id)`

Cancels a subscription. Returns `{ id, status }`.

## Invoices

### `stripe.listInvoices(customerId)`

Lists all invoices for a customer. Returns an array of `{ id, amount, status, date }`.

## Refunds

### `stripe.createRefund(options)`

| Option | Type | Description |
|---|---|---|
| `paymentIntentId` | string | Payment intent to refund |
| `amount` | number? | Partial refund amount; omit for full refund |

Returns `{ id, amount, status }`.

## Webhooks

### `stripe.constructEvent(payload, signature, secret)`

Verifies and parses a Stripe webhook payload. Returns `{ type, data }`.

```ntl
use http
use stripe
use env

stripe.init(env.STRIPE_SECRET_KEY)

http.post("/webhook", fn(req, res) {
  val event = stripe.constructEvent(
    req.rawBody,
    req.headers["stripe-signature"],
    env.STRIPE_WEBHOOK_SECRET
  )
  if event.type == "payment_intent.succeeded" {
    // handle
  }
  res.send("ok")
})
```
