# Local End-to-End Simulation & Test Guide

This guide walks through a **full ride simulation** on the Docker Compose stack: preview → match → payment → success.

---

## 1. Prerequisites

- Docker + Docker Compose
- Ports free: `3000`, `8080`, `5672`, `15672`, `27017`, `16686`
- Repo root with `.env` (copy from `.env.example` if needed)

Confirm local prototype flags (default in compose):

```env
LOCAL_SEED_DRIVERS=true
LOCAL_AUTO_ACCEPT=true
LOCAL_SIMULATE_TRIPS=true
STRIPE_SECRET_KEY=sk_test_replace_me
```

With these, **one rider browser tab** is enough (no live driver required). You should see nearby cars on the map, then a full Uber-like lifecycle before payment.

---

## 2. Start the system

```bash
cd /path/to/CloudPulse
cp -n .env.example .env
docker compose up --build -d
docker compose ps
```

### Expected health

| Check | Command / URL | Expect |
|-------|---------------|--------|
| UI | http://localhost:3000 | HTTP 200, role picker |
| API up | `curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/` | any HTTP response (often 404) |
| RabbitMQ | http://localhost:15672 | guest / guest |
| Jaeger | http://localhost:16686 | UI loads |
| Driver seeds | `docker compose logs driver-service \| grep Seeded` | sedan/suv/van/luxury |
| Payment mock | `docker compose logs payment-service \| grep mock` | local processor |

---

## 3. Simulation A — Rider-only happy path (auto-accept)

### Steps

1. Open http://localhost:3000  
2. Click **I Need a Ride**  
3. Confirm **nearby cars** are visible and moving on the map  
4. (Optional) Click **Move pickup**, then tap a new pickup point  
5. Tap a **dropoff** near San Francisco  
6. Choose a fare package (e.g. Sedan)  
7. Watch status progress:
   - Finding your driver  
   - Driver assigned / Driver is coming (assigned car moves toward you)  
   - Driver has arrived  
   - On the way (car follows the route)  
   - Trip complete → mock payment form  
8. Enter any card details → **Pay** → thanks / receipt screen  

### What happened under the hood

```text
WS nearby drivers   → Redis GEO → rider map markers
POST /trip/preview  → trip-service + OSRM + Mongo ride_fares
POST /trip/start    → Mongo trip + trip.event.created
driver-service      → match local-driver-* → auto-accept
                    → trip_simulator animates locations + lifecycle events
trip-service        → persist en_route/arrived/in_progress/completed
                    → payment.cmd.create_session (after completed)
payment-service     → cs_test_local_* + payment.event.session_created
Rider UI            → MockPaymentModal (no real Stripe charge)
```

### Log verification

```bash
docker compose logs --tail=50 trip-service | grep -E 'Publishing|OSRM|lifecycle'
docker compose logs --tail=50 driver-service | grep -E 'Found suitable|Auto-accepting|simulation'
docker compose logs --tail=50 payment-service | grep -E 'Payment session|local payment'
docker compose logs --tail=30 api-gateway | grep -E 'nearby|WebSocket'
```

**Pass criteria**

- [ ] Nearby drivers visible before booking  
- [ ] Preview returns multiple fares  
- [ ] Full lifecycle to completed without a second tab  
- [ ] Mock payment panel after trip completes  
- [ ] Receipt / thanks screen after Pay  

---

## 4. Simulation B — Live driver (two browsers)

Turn off auto-accept to exercise the real accept UX (keep simulation on the driver UI):

```bash
# In .env
LOCAL_AUTO_ACCEPT=false
# Keep LOCAL_SEED_DRIVERS=true OR false — if true, seeds still exist;
# live drivers are preferred when registered.
# Driver tab "Auto-simulate trip after accept" should stay ON.

docker compose up -d --force-recreate driver-service
```

### Steps

1. **Tab 1 — Driver:** open http://localhost:3000 → Driver → choose **Sedan** → leave open  
2. Confirm gateway log: driver registered / WS connected  
3. **Tab 2 — Rider:** Rider → destination → select **Sedan**  
4. Tab 1 shows trip request → **Accept** (auto-simulate drives the trip)  
5. Tab 2 shows en route → arrived → in trip → payment  

**Pass criteria**

- [ ] Driver receives `driver.cmd.trip_request`  
- [ ] Accept advances rider through lifecycle  
- [ ] Decline rematches (`trip.event.driver_not_interested`)  

---

## 5. Simulation C — No drivers

```bash
# .env
LOCAL_SEED_DRIVERS=false
LOCAL_AUTO_ACCEPT=false
docker compose up -d --force-recreate driver-service
```

Do **not** open a Driver tab. Start a rider trip.

**Expect:** UI **No drivers nearby**; logs `Found suitable drivers 0` and `trip.event.no_drivers_found`.

Restore seeds afterward:

```bash
LOCAL_SEED_DRIVERS=true
LOCAL_AUTO_ACCEPT=true
LOCAL_SIMULATE_TRIPS=true
docker compose up -d --force-recreate driver-service
```

---

## 6. Simulation D — Uber-like two-tab checklist

See also [uber-like-flow.md](./uber-like-flow.md).

| Step | Rider | Driver |
|------|-------|--------|
| 1 | Open map, see fleet | Online as Sedan |
| 2 | Set dropoff, request Sedan | Receive request |
| 3 | See "Finding…" then assigned | Accept (auto-sim ON) |
| 4 | Watch car approach | Panel shows simulating |
| 5 | Arrived → in trip | Driving animation |
| 6 | Mock pay + thanks | Trip completed |

---

## 7. API-level smoke tests (optional)

```bash
# Preview (adjust coordinates as needed)
curl -s http://localhost:8080/trip/preview \
  -H 'Content-Type: application/json' \
  -d '{
    "userID":"test-user-1",
    "pickup":{"latitude":37.7749,"longitude":-122.4194},
    "destination":{"latitude":37.7849,"longitude":-122.4094}
  }' | jq .
```

Use a returned `rideFareID` for start:

```bash
curl -s http://localhost:8080/trip/start \
  -H 'Content-Type: application/json' \
  -d '{"userID":"test-user-1","rideFareID":"<ID_FROM_PREVIEW>"}' | jq .
```

Watch RabbitMQ management UI → Queues for message rates.

---

## 8. Observability during tests

| Tool | Use |
|------|-----|
| Jaeger http://localhost:16686 | Search service `api-gateway`, `trip-service`, etc. |
| RabbitMQ http://localhost:15672 | Confirm queue bindings & rates |
| `docker compose logs -f` | Live correlation |

---

## 9. Tear down

```bash
docker compose down          # keep volumes
docker compose down -v       # wipe mongo/redis/postgres data
```

---

## 10. CI / production simulation note

GitOps does **not** replace this local sim. After images ship to Docker Hub, validate the same user journey against:

- `https://cloudpulse.live` (UI)
- `https://api.cloudpulse.live` (API)

using ArgoCD-synced Helm values (`gitops/cloudpulse/values-production.yaml`).
