# Local End-to-End Simulation & Test Guide

Full ride simulation on Docker Compose with **OTP start**, **cancel-before-start**, **travel simulation**, **PhonePe (mock-first) payment**, **rider feedback**, and **driver dashboard**.

---

## 1. Prerequisites

- Docker + Docker Compose
- Ports free: `3000`, `3001`, `8080`, `5672`, `15672`, `27017`, `16686`
- `.env` based on `.env.example` with two-app flags:

```env
LOCAL_SEED_DRIVERS=false
LOCAL_AUTO_ACCEPT=false
LOCAL_SIMULATE_TRIPS=false
TRIP_OFFER_TIMEOUT_SEC=30
JWT_SECRET=cloudpulse-dev-jwt-secret-change-me
DRIVER_SERVICE_HTTP_URL=http://driver-service:8085
TRIP_SERVICE_HTTP_URL=http://trip-service:8086
PAYMENT_PROVIDER=phonepe
# Leave PhonePe merchant credentials empty for local mock (pp_test_local_*)
PHONEPE_MERCHANT_ID=
PHONEPE_SALT_KEY=
```

---

## 2. Start the system

```bash
cd /path/to/CloudPulse
cp -n .env.example .env
docker compose up --build -d
docker compose ps
```

| Check | URL / command |
|-------|----------------|
| Rider UI | http://localhost:3000 |
| Driver UI | http://localhost:3001 |
| Driver dashboard | http://localhost:3001/dashboard |
| API health | `curl -s http://localhost:8080/health` → `ok` |
| RabbitMQ | http://localhost:15672 (guest/guest) |

---

## 3. Simulation — Two-app OTP happy path

1. **Driver** at http://localhost:3001 → Register (choose package e.g. Sedan) → **Go Online**
2. **Rider** at http://localhost:3000 → set dropoff near Hyderabad → choose the **same** package
3. Driver receives offer → **Accept** → **En route** → **Arrived**
4. Rider sees **6-digit OTP** → shares with driver
5. Driver enters OTP → trip starts → map simulates travel → **Complete** unlocks at destination
6. Rider sees **PhonePe mock** checkout → **Pay** → rating/feedback form → submit
7. Driver opens **Dashboard** → sees trip, bonus points (= rating), and review

### Cancel path

After accept (before OTP start), rider taps **Cancel trip**. Both apps reset; no payment session is created.

### Automated checks

```bash
node scripts/e2e-otp-trip-test.mjs           # OTP wrong/right + cancel + payment
node scripts/e2e-payment-test.mjs            # complete → payment session
node scripts/e2e-dashboard-review-test.mjs   # PhonePe mock pay → review → dashboard
node scripts/e2e-driver-offer-test.mjs       # offer delivery only
```

Go unit tests:

```bash
docker run --rm -v "$PWD":/app -w /app -e JWT_SECRET=test -e GOFLAGS=-buildvcs=false \
  golang:1.23-bookworm go test ./shared/otp/ ./services/trip-service/internal/service/ \
  ./services/payment-service/internal/infrastructure/phonepe/
```

---

## 4. Pass criteria

- [ ] Driver receives offer only when online and package matches
- [ ] Rider receives OTP after accept
- [ ] Wrong OTP rejected; correct OTP starts trip
- [ ] Rider can cancel until OTP verified; cannot cancel after start
- [ ] Driver Complete disabled until near destination / sim finished
- [ ] PhonePe mock session (`pp_test_local_*`) after complete; mock pay marks trip paid
- [ ] Rider can submit one review (1–5); bonus points = rating
- [ ] Driver dashboard shows trip count, bonus points, average rating, reviews
- [ ] Cancel path creates no payment session

---

## 5. Under the hood

```text
POST /trip/start              → trip.event.created
driver-service                → driver.cmd.trip_request (offer)
driver accept                 → accepted + IssueOTP → trip.event.otp_issued (rider)
driver en_route / arrived     → CAS status updates
driver trip.cmd.verify_otp    → arrived + hash match → in_progress + trip.event.started
driver complete               → completed → payment.cmd.create_session
payment-service               → payment.event.session_created (PhonePe or pp_test_local_*)
POST /payment/mock-success    → payment.event.success → payed
POST /trips/{id}/review       → reviews + driver bonus_points
GET /drivers/me/dashboard     → trips, points, reviews (JWT)
rider trip.cmd.cancel         → cancelled (pre-start only) + driver released
```
