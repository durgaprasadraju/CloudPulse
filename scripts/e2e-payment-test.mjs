// End-to-end check: full trip lifecycle with OTP → rider receives payment session.
// Run: node scripts/e2e-payment-test.mjs
const API = "http://localhost:8080";
const PACKAGE = "sedan";
const EMAIL = `e2e-pay-${Date.now()}@test.local`;

const log = (...a) => console.log(new Date().toISOString().slice(11, 19), ...a);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const waitFor = async (pred, ms = 20000) => {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    if (pred()) return true;
    await sleep(400);
  }
  return false;
};

async function main() {
  let res = await fetch(`${API}/drivers/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email: EMAIL, password: "secret123", name: "E2E Pay Driver",
      phone: "8888888888", packageSlug: PACKAGE, carPlate: "TS09PAY01",
    }),
  });
  const reg = await res.json();
  if (!res.ok) throw new Error(`register failed: ${JSON.stringify(reg)}`);
  log("driver registered:", reg.driver.id);

  const driverWS = new WebSocket(
    `ws://localhost:8080/ws/drivers?token=${encodeURIComponent(reg.token)}&packageSlug=${PACKAGE}`
  );
  let driverProfile = null;
  let offeredTrip = null;
  let otpOk = false;
  driverWS.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    log("driver WS:", msg.type);
    if (msg.type === "driver.cmd.register") driverProfile = msg.data;
    if (msg.type === "driver.cmd.trip_request") offeredTrip = msg.data?.trip ?? msg.data;
    if (msg.type === "trip.event.otp_verified") otpOk = true;
  };
  await new Promise((res2, rej) => { driverWS.onopen = res2; setTimeout(() => rej(new Error("driver WS timeout")), 5000); });
  driverWS.send(JSON.stringify({
    type: "driver.cmd.location",
    data: { location: { latitude: 17.4401, longitude: 78.3915 }, geohash: "tepe", packageSlug: PACKAGE },
  }));

  const riderID = `e2e-pay-rider-${Date.now()}`;
  const riderWS = new WebSocket(`ws://localhost:8080/ws/riders?userID=${riderID}`);
  let paymentSession = null;
  let pickupOTP = null;
  riderWS.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.type !== "driver.cmd.location") log("rider WS:", msg.type);
    if (msg.type === "trip.event.otp_issued") pickupOTP = msg.data?.otp;
    if (msg.type === "payment.event.session_created") paymentSession = msg.data;
  };
  await new Promise((res2, rej) => { riderWS.onopen = res2; setTimeout(() => rej(new Error("rider WS timeout")), 5000); });
  await sleep(1000);

  res = await fetch(`${API}/trip/preview`, {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      userID: riderID,
      pickup: { latitude: 17.4401, longitude: 78.3915 },
      destination: { latitude: 17.4485, longitude: 78.3908 },
    }),
  });
  const preview = await res.json();
  const fare = (preview.data?.rideFares ?? []).find((f) => f.packageSlug === PACKAGE);
  if (!fare) throw new Error("no sedan fare");
  res = await fetch(`${API}/trip/start`, {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rideFareID: fare.id, userID: riderID }),
  });
  if (!res.ok) throw new Error(`start failed: ${await res.text()}`);
  log("trip started");

  if (!(await waitFor(() => offeredTrip && driverProfile))) throw new Error("driver never received trip offer");
  log("driver got offer for trip", offeredTrip.id);

  driverWS.send(JSON.stringify({
    type: "driver.cmd.trip_accept",
    data: { tripID: offeredTrip.id, riderID, driver: driverProfile },
  }));
  if (!(await waitFor(() => pickupOTP))) throw new Error("no OTP");

  const trip = { ...offeredTrip, driver: driverProfile, userID: riderID };
  for (const event of ["trip.event.driver_en_route", "trip.event.driver_arrived"]) {
    driverWS.send(JSON.stringify({
      type: "trip.cmd.status",
      data: { event, riderID, trip: { ...trip, status: event } },
    }));
    await sleep(600);
  }

  driverWS.send(JSON.stringify({
    type: "trip.cmd.verify_otp",
    data: { tripID: offeredTrip.id, otp: pickupOTP, riderID },
  }));
  if (!(await waitFor(() => otpOk))) throw new Error("OTP not verified");

  driverWS.send(JSON.stringify({
    type: "trip.cmd.status",
    data: { event: "trip.event.completed", riderID, trip: { ...trip, status: "completed" } },
  }));

  if (!(await waitFor(() => paymentSession))) throw new Error("no payment session");
  log("SUCCESS: rider received payment session:", JSON.stringify(paymentSession));
  driverWS.close(); riderWS.close();
  process.exit(0);
}

main().catch((e) => { console.error("FAILED:", e.message); process.exit(1); });
