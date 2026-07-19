# Uber-like product simulation

How CloudPulse mimics a real Uber booking experience in local and demo environments.

## User journey

```text
Rider opens map
  → sees nearby cars moving (Redis + WebSocket)
  → optionally moves pickup pin
  → taps dropoff
  → chooses Sedan / SUV / Van / Luxury
  → "Finding your driver"
  → Driver assigned / en route (car moves toward pickup)
  → Driver arrived
  → Trip in progress (car follows route)
  → Arrived / trip completed
  → Mock card checkout + receipt
```

## Modes

| Mode | Flags | Tabs needed |
|------|-------|-------------|
| **Rider-only demo** (default) | `LOCAL_SEED_DRIVERS=true`, `LOCAL_AUTO_ACCEPT=true`, `LOCAL_SIMULATE_TRIPS=true` | Rider only |
| **Two-sided live** | `LOCAL_AUTO_ACCEPT=false` (seeds optional) | Rider + Driver |
| **No drivers** | both seed/auto-accept false, no driver tab | Rider → "No drivers nearby" |

## Key components

| Layer | Role |
|-------|------|
| Redis GEO (`shared/tracking`) | Live driver positions |
| api-gateway rider WS | Polls nearby drivers every 2s; lifecycle event fan-out |
| driver-service seeds + wander | Cars appear and drift on the map |
| driver-service `trip_simulator` | After auto-accept: en_route → arrived → started → completed |
| trip-service lifecycle consumer | Persists statuses; creates payment **after** completed |
| Driver UI auto-simulate | Live driver tab animates the same lifecycle after Accept |
| `MockPaymentModal` | In-app demo checkout (no real Stripe charge) |

## Events (trip lifecycle)

| Event | Rider UI |
|-------|----------|
| `trip.event.driver_assigned` | Driver assigned |
| `trip.event.driver_en_route` | Driver is coming |
| `trip.event.driver_arrived` | Driver has arrived |
| `trip.event.started` | On the way |
| `trip.event.completed` | Preparing receipt |
| `payment.event.session_created` | Mock pay |
| `trip.event.cancelled` | Cancelled |

## Quick start

```bash
docker compose --env-file .env.example up --build
# Rider: http://localhost:3000 → I Need a Ride
# Driver (optional): I Want to Drive → Sedan
```

See [local-e2e-test.md](./local-e2e-test.md) § Simulation D for a step-by-step checklist.
