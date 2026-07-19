# Service: API Gateway

**Path:** `services/api-gateway/`  
**Language:** Go  
**Role:** Edge BFF — only public HTTP/WebSocket surface for browsers and Stripe.

---

## 1. Responsibilities

- Accept trip **preview** and **start** over REST.
- Maintain **WebSocket** connections for riders and drivers.
- Bridge RabbitMQ notification queues → correct browser connection (`ownerId` = userID).
- Forward driver accept/decline WS messages into RabbitMQ.
- Receive **Stripe webhooks** and emit `payment.event.success`.
- Dial trip-service and driver-service over **gRPC**.

It does **not** store trips or drivers; it orchestrates IO.

---

## 2. Runtime

| Setting | Default / compose |
|---------|-------------------|
| Listen | `HTTP_ADDR` → compose `:8080`, code default `:8081` |
| Image | Multi-stage root `Dockerfile` with `SERVICE=api-gateway` |
| K8s | Helm `backend` deployment + HPA |

### Environment variables

| Variable | Purpose |
|----------|---------|
| `HTTP_ADDR` | Bind address (`:8080`) |
| `RABBITMQ_URI` | AMQP connection |
| `TRIP_SERVICE_URL` | `trip-service:9093` |
| `DRIVER_SERVICE_URL` | `driver-service:9092` |
| `JAEGER_ENDPOINT` | Trace exporter |
| `ENVIRONMENT` | Trace resource attr |
| `STRIPE_WEBHOOK_KEY` | Webhook signing secret (optional locally) |

---

## 3. HTTP API

| Method | Path | Handler | Downstream |
|--------|------|---------|------------|
| `POST` | `/trip/preview` | `handleTripPreview` | gRPC `PreviewTrip` |
| `POST` | `/trip/start` | `handleTripStart` | gRPC `CreateTrip` |
| `POST` | `/webhook/stripe` | `handleStripeWebhook` | Publish `payment.event.success` |

Preview / start request bodies come from the frontend (`rideFareID`, `userID`, coordinates). Responses are wrapped with shared HTTP helpers.

> Root `GET /` is not a health API (often 404). Probes use TCP; ALB success codes allow `200–399`.

---

## 4. WebSocket API

### Riders — `/ws/riders?userID=`

On connect:

1. Upgrade WS; register connection in `ConnectionManager`.
2. Start queue consumers:
   - `notify_driver_no_drivers_found`
   - `notify_driver_assign`
   - `notify_payment_session_created`
3. Messages with matching `ownerId` are pushed as `contracts.WSMessage`.

### Drivers — `/ws/drivers?userID=&packageSlug=`

On connect:

1. gRPC `RegisterDriver` on driver-service.
2. Send `driver.cmd.register` with driver profile to the browser.
3. Consume `driver_cmd_trip_request` → push trip offers.
4. Read loop handles:
   - `driver.cmd.location` — currently ignored (stub)
   - `driver.cmd.trip_accept` / `trip_decline` — republished to RabbitMQ
5. On disconnect: `UnregisterDriver`.

---

## 5. Internal structure

```
services/api-gateway/
  main.go              # mux, RabbitMQ, graceful shutdown
  http.go              # REST + Stripe webhook
  ws.go                # rider/driver sockets
  grpc_clients/        # trip + driver clients
  middleware.go
  types.go
```

Shared deps: `shared/messaging`, `shared/contracts`, `shared/tracing`, `shared/env`.

---

## 6. Failure modes

| Scenario | Behavior |
|----------|----------|
| RabbitMQ down | Process exits on startup (`log.Fatal`) |
| Trip gRPC down | Preview/start return errors to client |
| Driver WS without `packageSlug` | Connection rejected |
| Missing Stripe webhook key | Webhook verify may fail / skip depending on impl |

---

## 7. How to run alone (dev)

Prefer full compose. Isolated binary needs RabbitMQ + trip + driver reachable:

```bash
HTTP_ADDR=:8080 \
RABBITMQ_URI=amqp://guest:guest@localhost:5672/ \
TRIP_SERVICE_URL=localhost:9093 \
DRIVER_SERVICE_URL=localhost:9092 \
go run ./services/api-gateway
```

See also: [data-flow](../architecture/data-flow.md), [local E2E](../simulation/local-e2e-test.md).
