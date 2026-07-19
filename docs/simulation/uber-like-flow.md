# Uber-like product simulation

How CloudPulse mimics a real Uber booking experience with a **rider app** and a **driver app**.

## User journey (two-app OTP flow)

```text
Rider (:3000)                         Driver (:3001)
──────────────                        ──────────────
Open map / set dropoff                Register / login / Go Online
Choose package (sedan/suv/…)          Receive trip offer alarm
Request ride                          Accept trip
See driver assigned + 6-digit OTP     Tap En route → Arrived
Share OTP with driver                 Enter rider OTP → trip starts
Cancel allowed until OTP verified     Auto-simulate travel to dropoff
See live driver progress              Complete only at destination
Pay via mock / Stripe checkout        Ready for next trip
```

## Modes

| Mode | Flags | Apps |
|------|-------|------|
| **Two-sided live (recommended)** | `LOCAL_SEED_DRIVERS=false`, `LOCAL_AUTO_ACCEPT=false`, `LOCAL_SIMULATE_TRIPS=false` | Rider `:3000` + Driver `:3001` |
| **Seeded demo** | seeds + auto-accept + simulate true | Rider only — simulator still waits for OTP (use driver app or disable OTP gate for demos) |

## Lifecycle + OTP

| Status | Trigger |
|--------|---------|
| `pending` | Rider starts trip |
| `accepted` | Driver accepts (+ hashed OTP stored, plaintext to rider only) |
| `en_route` / `arrived` | Driver status buttons |
| `in_progress` | **OTP verified** (`trip.cmd.verify_otp`) while `arrived` |
| `completed` | Driver completes at destination → payment session |
| `payed` | Mock `/payment/mock-success` or Stripe webhook |
| `cancelled` | Rider cancel **before** OTP start only |

## Events

| Event | Who |
|-------|-----|
| `trip.event.otp_issued` | Rider (contains plaintext OTP) |
| `trip.cmd.verify_otp` | Driver → trip-service |
| `trip.event.otp_failed` / `otp_verified` | Driver |
| `trip.event.started` | Rider (after OTP) |
| `trip.cmd.cancel` | Rider (pre-start) |
| `payment.event.session_created` | Rider checkout |
| `payment.event.success` | Marks trip `payed` |

## Quick start

```bash
docker compose up --build -d
# Rider:  http://localhost:3000
# Driver: http://localhost:3001  (register → Go Online)
# E2E:    node scripts/e2e-otp-trip-test.mjs
```

Payment: local mock sessions (`cs_test_local_*`) use the in-app card form and `POST /payment/mock-success`. Real Stripe Checkout is used when valid Stripe keys are configured.
