// End-to-end check: driver registers + connects WS, rider requests a trip,
// driver must receive driver.cmd.trip_request. Run: node scripts/e2e-driver-offer-test.mjs
const API = "http://localhost:8080";
const EMAIL = `e2e-driver-${Date.now()}@test.local`;
const PACKAGE = "sedan";

const log = (...a) => console.log(new Date().toISOString().slice(11, 19), ...a);

async function main() {
  // 1. Register driver account
  let res = await fetch(`${API}/drivers/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email: EMAIL,
      password: "secret123",
      name: "E2E Driver",
      phone: "9999999999",
      packageSlug: PACKAGE,
      carPlate: "TS09E2E01",
    }),
  });
  const reg = await res.json();
  if (!res.ok) throw new Error(`register failed: ${res.status} ${JSON.stringify(reg)}`);
  log("driver registered:", reg.driver.id);

  // 2. Connect driver WebSocket
  const ws = new WebSocket(
    `ws://localhost:8080/ws/drivers?token=${encodeURIComponent(reg.token)}&packageSlug=${PACKAGE}`
  );
  const gotOffer = new Promise((resolve, reject) => {
    const t = setTimeout(() => reject(new Error("TIMEOUT: no trip offer within 25s")), 25000);
    ws.onmessage = (ev) => {
      const msg = JSON.parse(ev.data);
      log("driver WS received:", msg.type);
      if (msg.type === "driver.cmd.trip_request") {
        clearTimeout(t);
        resolve(msg);
      }
    };
    ws.onerror = (e) => { clearTimeout(t); reject(new Error("WS error")); };
    ws.onclose = (e) => log("driver WS closed", e.code, e.reason || "");
  });
  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    setTimeout(() => reject(new Error("WS connect timeout")), 5000);
  });
  log("driver WS connected");
  ws.send(JSON.stringify({
    type: "driver.cmd.location",
    data: { location: { latitude: 17.4401, longitude: 78.3915 }, geohash: "tepe", packageSlug: PACKAGE },
  }));
  await new Promise((r) => setTimeout(r, 1500));

  // 3. Rider previews and starts a trip with matching package
  const riderID = `e2e-rider-${Date.now()}`;
  res = await fetch(`${API}/trip/preview`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      userID: riderID,
      pickup: { latitude: 17.4401, longitude: 78.3915 },
      destination: { latitude: 17.4485, longitude: 78.3908 },
    }),
  });
  const preview = await res.json();
  if (!res.ok) throw new Error(`preview failed: ${res.status} ${JSON.stringify(preview)}`);
  const fares = preview.data?.rideFares ?? preview.rideFares ?? [];
  const fare = fares.find((f) => f.packageSlug === PACKAGE);
  if (!fare) throw new Error(`no ${PACKAGE} fare in preview: ${JSON.stringify(fares).slice(0, 300)}`);
  log("got fare:", fare.id, fare.packageSlug);

  res = await fetch(`${API}/trip/start`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rideFareID: fare.id, userID: riderID }),
  });
  const started = await res.json();
  if (!res.ok) throw new Error(`start failed: ${res.status} ${JSON.stringify(started)}`);
  log("trip started:", started.data?.id ?? started.id ?? "(id unknown)");

  // 4. Driver must receive the offer
  const offer = await gotOffer;
  const trip = offer.data?.trip ?? offer.data;
  log("SUCCESS: driver received trip offer for trip", trip?.id ?? "(unknown)");
  ws.close();
  process.exit(0);
}

main().catch((e) => {
  console.error("FAILED:", e.message);
  process.exit(1);
});
