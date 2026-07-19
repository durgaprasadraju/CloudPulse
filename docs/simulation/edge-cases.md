# Edge Cases, Failures & Test Matrix

Companion to [local-e2e-test.md](./local-e2e-test.md). Use this as a checklist when validating releases.

---

## 1. Functional matrix

| ID | Scenario | Setup | Expected |
|----|----------|-------|----------|
| F1 | Happy path auto-accept | Seeds + auto-accept on | Rider reaches local Pay → success |
| F2 | Live driver accept | Auto-accept off + driver tab | Trip request → accept → payment |
| F3 | Live driver decline | Same | Rematch or eventually no drivers |
| F4 | No drivers | Seeds off, no driver tab | `no_drivers_found` UI |
| F5 | Package mismatch | Driver = sedan, rider = luxury (no luxury seed/driver) | No match / rematch |
| F6 | Preview only | Stop before start | Fares shown; no RabbitMQ trip.created |
| F7 | Payment mock | Placeholder Stripe key | `cs_test_local_*`, button says “(local)” |
| F8 | Payment real Stripe | Valid `sk_test_…` + publishable key | Redirect to Stripe Checkout |
| F9 | WS reconnect rider | Refresh mid-match | New `userID` = new session (known limitation) |
| F10 | Driver disconnect | Close driver tab | Unregister; may cause no drivers for new trips |

---

## 2. Infrastructure failures

| ID | Failure | Symptom | Mitigation |
|----|---------|---------|------------|
| I1 | RabbitMQ down | Services crash/restart loops | Compose healthcheck; fix URI |
| I2 | Mongo down | trip-service fatal / create fails | Wait for healthy mongo |
| I3 | OSRM unreachable | Preview errors | Check `OSRM_API`, network |
| I4 | Jaeger down | Usually still starts (exporter buffers) | Optional for local |
| I5 | Wrong `HTTP_ADDR` | UI cannot call API | Align compose port 8080 |
| I6 | Stale `NEXT_PUBLIC_*` | Browser hits wrong host | Rebuild web image |

---

## 3. Consistency notes

- **User IDs** are random per page load — payment/WS routing is per-session, not long-lived accounts.
- **Driver memory** is lost on driver-service restart (seeds re-applied if flag set).
- **Fare documents** in Mongo from previews can accumulate; safe to wipe volume between demos.
- Declared payment events `failed` / `cancelled` are **not** fully wired — do not expect UI for them yet.

---

## 4. Suggested automated checks (future)

| Layer | Idea |
|-------|------|
| Unit | trip fare estimation; driver `FindAvailableDrivers` |
| Contract | proto compatibility CI |
| Integration | Testcontainers: RabbitMQ + Mongo + trip/driver consumers |
| E2E | Playwright: rider flow against compose |

None of the above are mandatory for the current prototype; use the manual simulations first.

---

## 5. Quick debug commands

```bash
# Who is registered? (only via logs today)
docker compose logs driver-service | grep -E 'Seeded|Auto-accept|Found suitable'

# Trip events
docker compose logs trip-service | grep Publishing

# RabbitMQ queue depths (needs rabbitmqadmin or UI)
open http://localhost:15672/#/queues

# Reset app data
docker compose restart trip-service driver-service payment-service api-gateway
# or
docker compose down -v && docker compose up --build -d
```

---

## 6. Architecture references

- [System overview](../architecture/system-overview.md)
- [Data & event flow](../architecture/data-flow.md)
- Per-service docs under [`docs/services/`](../services/)
