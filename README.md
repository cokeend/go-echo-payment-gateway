# Go Payment Gateway API

Payment Gateway REST API built with Go, Echo v5, Stripe, and PostgreSQL.

## Project Structure

```
├── cmd/api/main.go                          # Entrypoint & dependency injection
├── internal
│   ├── domain/payment.go                    # Entities & interfaces
│   ├── payment
│   │   ├── usecase/payment_uc.go            # Business logic
│   │   ├── repository/postgres_repo.go      # PostgreSQL persistence
│   │   ├── delivery/http/payment_handler.go # Echo HTTP handlers
│   │   └── gateway/stripe_gw.go             # Stripe integration
├── pkg/stripe_util/                         # Stripe helper utilities
├── Dockerfile
└── docker-compose.yml
```

## Tech Stack

- **Go 1.25** with Echo v5
- **Stripe** Checkout Session + PaymentIntent
- **PostgreSQL 17** for persistence
- **Docker Compose** for local development

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Stripe account with API keys ([dashboard](https://dashboard.stripe.com/apikeys))

### Run with Docker Compose

```bash
# 1. Create .env from example
cp .env.example .env

# 2. Set your Stripe keys in .env
#    STRIPE_SECRET_KEY=sk_test_xxx
#    STRIPE_WEBHOOK_SECRET=whsec_xxx

# 3. Start services
docker compose up --build
```

API will be available at `http://localhost:8080`.

### Run locally (without Docker)

```bash
# Start PostgreSQL (port 5432)
# Then:
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments?sslmode=disable"
export STRIPE_SECRET_KEY="sk_test_xxx"
export STRIPE_WEBHOOK_SECRET="whsec_xxx"

go run ./cmd/api
```

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/payments?sslmode=disable` |
| `STRIPE_SECRET_KEY` | Stripe secret API key | (required) |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret | (optional) |
| `PORT` | HTTP server port | `8080` |

## API Endpoints

### Health Check

```
GET /health
```

### Checkout (Stripe Hosted Page)

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/checkout` | Create checkout session with Stripe payment URL |

### Payments

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/payments` | Create a payment (PaymentIntent, for Stripe.js) |
| `GET` | `/api/v1/payments` | List payments |
| `GET` | `/api/v1/payments/:id` | Get payment by ID |
| `POST` | `/api/v1/payments/:id/confirm` | Confirm a payment |
| `POST` | `/api/v1/payments/:id/cancel` | Cancel a payment |
| `POST` | `/api/v1/payments/:id/refund` | Refund a payment |

### Webhook

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/webhook/stripe` | Stripe webhook receiver |

## API Usage Examples

### Create Checkout (Recommended)

Creates a Stripe Checkout Session and returns a `checkout_url` that redirects the user to Stripe's hosted payment page.

The `payment_methods` field is optional. If omitted, Stripe will show all available methods for the currency.

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 1000,
    "currency": "thb",
    "customer_email": "customer@example.com",
    "description": "Order #1234",
    "success_url": "http://localhost:3000/success",
    "cancel_url": "http://localhost:3000/cancel",
    "payment_methods": ["card", "promptpay"]
  }'
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "a1b2c3d4-...",
    "stripe_payment_id": "pi_xxx",
    "amount": 1000,
    "currency": "usd",
    "status": "pending",
    "customer_email": "customer@example.com",
    "description": "Order #1234",
    "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_xxx",
    "created_at": "2026-02-28T12:00:00Z",
    "updated_at": "2026-02-28T12:00:00Z"
  }
}
```

Open the `checkout_url` in a browser to complete the payment on Stripe's hosted page.

**Test card:** `4242 4242 4242 4242`, any future expiry, any CVC.

### Create Payment (PaymentIntent)

For custom frontend integrations using Stripe.js / Stripe Elements. Returns a `client_secret` to confirm payment client-side.

```bash
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 1000,
    "currency": "thb",
    "customer_email": "customer@example.com",
    "description": "Order #1234",
    "payment_methods": ["card", "promptpay", "mobile_banking_scb"]
  }'
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "a1b2c3d4-...",
    "stripe_payment_id": "pi_xxx",
    "amount": 1000,
    "currency": "usd",
    "status": "pending",
    "customer_email": "customer@example.com",
    "description": "Order #1234",
    "client_secret": "pi_xxx_secret_xxx",
    "created_at": "2026-02-28T12:00:00Z",
    "updated_at": "2026-02-28T12:00:00Z"
  }
}
```

### Get Payment

```bash
curl http://localhost:8080/api/v1/payments/{id}
```

### List Payments

```bash
curl "http://localhost:8080/api/v1/payments?limit=10&offset=0"
```

### Confirm Payment

```bash
curl -X POST http://localhost:8080/api/v1/payments/{id}/confirm
```

### Cancel Payment

```bash
curl -X POST http://localhost:8080/api/v1/payments/{id}/cancel
```

### Refund Payment

```bash
curl -X POST http://localhost:8080/api/v1/payments/{id}/refund \
  -H "Content-Type: application/json" \
  -d '{"amount": 500}'
```

## Payment Flows

### Flow 1: Checkout Session (Stripe Hosted Page)

Easiest integration -- no frontend code needed, just redirect.

```
Client                    API                     Stripe
  |                        |                        |
  |-- POST /checkout ----->|                        |
  |                        |-- CreateCheckoutSession -->|
  |                        |<-- checkout_url -----------|
  |<-- checkout_url -------|                        |
  |                        |                        |
  |-- open checkout_url --------------------------->|
  |        (user pays on Stripe's page)             |
  |<-- redirect to success_url --------------------|
  |                        |                        |
  |                        |<-- webhook event -------|
  |                        |-- update status         |
```

### Flow 2: PaymentIntent (Custom Frontend with Stripe.js)

For full control over the payment UI.

```
Client                    API                     Stripe
  |                        |                        |
  |-- POST /payments ----->|                        |
  |                        |-- CreatePaymentIntent ->|
  |                        |<-- client_secret -------|
  |<-- client_secret ------|                        |
  |                        |                        |
  |-- stripe.confirmCardPayment(client_secret) ---->|
  |                        |                        |
  |                        |<-- webhook event -------|
  |                        |-- update status         |
```

## Supported Payment Methods

The `payment_methods` field is **optional**. If omitted, Stripe automatically shows all methods available for the currency.

| Method | Value | Currency |
|---|---|---|
| Credit/Debit Card | `card` | All (includes Apple Pay / Google Pay) |
| PromptPay QR | `promptpay` | THB only |
| SCB Mobile Banking | `mobile_banking_scb` | THB only |
| KBank Mobile Banking | `mobile_banking_kbank` | THB only |
| Bangkok Bank Mobile Banking | `mobile_banking_bbl` | THB only |
| Krungsri Mobile Banking | `mobile_banking_bay` | THB only |
| Krungthai Mobile Banking | `mobile_banking_ktb` | THB only |
| Alipay | `alipay` | Multiple |
| WeChat Pay | `wechat_pay` | Multiple |
| GrabPay | `grabpay` | SGD, MYR |

### Examples by Use Case

**Card only:**
```json
{ "payment_methods": ["card"] }
```

**PromptPay QR (Thailand):**
```json
{ "currency": "thb", "payment_methods": ["promptpay"] }
```

**Thai payment methods (all):**
```json
{ "currency": "thb", "payment_methods": ["card", "promptpay", "mobile_banking_scb", "mobile_banking_kbank", "mobile_banking_bbl", "mobile_banking_bay", "mobile_banking_ktb"] }
```

**Let Stripe decide (recommended):**
```json
{ "currency": "thb" }
```

## Supported Currencies

`usd`, `eur`, `gbp`, `jpy`, `thb`, `sgd`, `aud`, `cad`, `myr`, `cny`, `hkd`
