// End-to-end OTP trip lifecycle: offer → accept → OTP → start → complete → payment
// Also verifies wrong OTP rejection and cancel-before-start.
// Run: node scripts/e2e-otp-trip-test.mjs
const API = "http://localhost:8080";
const PACKAGE = "sedan";

const log = (...a) => console.log(new Date().toISOString().slice(11, 19), ...a);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function registerDriver(email) {
  const res = await fetch(`${API}/drivers/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email, password: "secret123", name: "E2E OTP Driver",
      phone: "7777777777", packageSlug: PACKAGE, carPlate: "TS09OTP01",
    }),
  });
  const body = await res.json();
  if (!res.ok) throw new Error(`register failed: ${JSON.stringify(body)}`);
  return body;
}

async function startTrip(riderID) {
  let res = await fetch(`${API}/trip/preview`, {
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
}

async function waitFor(pred, ms = 20000, step = 400) {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    if (pred()) return true;
    await sleep(step);
  }
  return false;
}

async function happyPath() {
  log("--- happy path ---");
  const reg = await registerDriver(`e2e-otp-${Date.now()}@test.local`);
  log("driver", reg.driver.id);

  let offeredTrip = null;
  let driverProfile = null;
  let otpResult = null;
  const driverWS = new WebSocket(
    `ws://localhost:8080/ws/drivers?token=${encodeURIComponent(reg.token)}&packageSlug=${PACKAGE}`
  );
  driverWS.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    log("driver WS:", msg.type);
    if (msg.type === "driver.cmd.register") driverProfile = msg.data;
    if (msg.type === "driver.cmd.trip_request") offeredTrip = msg.data?.trip ?? msg.data;
    if (msg.type === "trip.event.otp_failed" || msg.type === "trip.event.otp_verified") otpResult = msg;
  };
  await new Promise((res, rej) => { driverWS.onopen = res; setTimeout(() => rej(new Error("driver WS timeout")), 5000); });
  driverWS.send(JSON.stringify({
    type: "driver.cmd.location",
    data: { location: { latitude: 17.4401, longitude: 78.3915 }, geohash: "tepe", packageSlug: PACKAGE },
  }));

  const riderID = `e2e-otp-rider-${Date.now()}`;
  let pickupOTP = null;
  let paymentSession = null;
  const riderEvents = [];
  const riderWS = new WebSocket(`ws://localhost:8080/ws/riders?userID=${riderID}`);
  riderWS.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.type === "driver.cmd.location") return;
    riderEvents.push(msg.type);
    log("rider WS:", msg.type);
    if (msg.type === "trip.event.otp_issued") pickupOTP = msg.data?.otp;
    if (msg.type === "payment.event.session_created") paymentSession = msg.data;
  };
  await new Promise((res, rej) => { riderWS.onopen = res; setTimeout(() => rej(new Error("rider WS timeout")), 5000); });
  await sleep(800);

  await startTrip(riderID);
  if (!(await waitFor(() => offeredTrip))) throw new Error("no offer");
  if (!driverProfile) throw new Error("no driver profile");

  driverWS.send(JSON.stringify({
    type: "driver.cmd.trip_accept",
    data: { tripID: offeredTrip.id, riderID, driver: driverProfile },
  }));
  if (!(await waitFor(() => pickupOTP))) throw new Error(`no OTP. events=${riderEvents.join(",")}`);
  log("OTP issued:", pickupOTP);

  // walk to arrived
  const trip = { ...offeredTrip, driver: driverProfile, userID: riderID };
  for (const event of ["trip.event.driver_en_route", "trip.event.driver_arrived"]) {
    driverWS.send(JSON.stringify({
      type: "trip.cmd.status",
      data: { event, riderID, trip: { ...trip, status: event } },
    }));
    await sleep(600);
  }

  // wrong OTP
  otpResult = null;
  driverWS.send(JSON.stringify({
    type: "trip.cmd.verify_otp",
    data: { tripID: offeredTrip.id, otp: "000000", riderID },
  }));
  if (!(await waitFor(() => otpResult?.type === "trip.event.otp_failed"))) {
    throw new Error("expected OTP failure");
  }
  log("wrong OTP rejected");

  // correct OTP
  otpResult = null;
  driverWS.send(JSON.stringify({
    type: "trip.cmd.verify_otp",
    data: { tripID: offeredTrip.id, otp: pickupOTP, riderID },
  }));
  if (!(await waitFor(() => otpResult?.type === "trip.event.otp_verified" || riderEvents.includes("trip.event.started")))) {
    throw new Error("expected OTP verified / started");
  }
  log("OTP verified, trip started");

  await sleep(500);
  driverWS.send(JSON.stringify({
    type: "trip.cmd.status",
    data: { event: "trip.event.completed", riderID, trip: { ...trip, status: "completed" } },
  }));

  if (!(await waitFor(() => paymentSession))) {
    throw new Error(`no payment. events=${riderEvents.join(",")}`);
  }
  log("SUCCESS happy path payment:", paymentSession.sessionID);

  // mock pay
  const payRes = await fetch(`${API}/payment/mock-success`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      tripID: paymentSession.tripID,
      sessionID: paymentSession.sessionID,
      userID: riderID,
      driverID: reg.driver.id,
    }),
  });
  if (!payRes.ok) throw new Error(`mock pay failed: ${await payRes.text()}`);
  log("mock payment recorded");

  driverWS.close(); riderWS.close();
}

async function cancelPath() {
  log("--- cancel before start ---");
  const reg = await registerDriver(`e2e-cancel-${Date.now()}@test.local`);
  let offeredTrip = null;
  let driverProfile = null;
  let driverCancelled = false;
  const driverWS = new WebSocket(
    `ws://localhost:8080/ws/drivers?token=${encodeURIComponent(reg.token)}&packageSlug=${PACKAGE}`
  );
  driverWS.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.type === "driver.cmd.register") driverProfile = msg.data;
    if (msg.type === "driver.cmd.trip_request") offeredTrip = msg.data?.trip ?? msg.data;
    if (msg.type === "trip.event.cancelled") driverCancelled = true;
  };
  await new Promise((res, rej) => { driverWS.onopen = res; setTimeout(() => rej(new Error("driver WS timeout")), 5000); });
  driverWS.send(JSON.stringify({
    type: "driver.cmd.location",
    data: { location: { latitude: 17.4401, longitude: 78.3915 }, geohash: "tepe", packageSlug: PACKAGE },
  }));

  const riderID = `e2e-cancel-rider-${Date.now()}`;
  let riderCancelled = false;
  let paymentSession = null;
  const riderWS = new WebSocket(`ws://localhost:8080/ws/riders?userID=${riderID}`);
  riderWS.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    if (msg.type === "trip.event.cancelled") riderCancelled = true;
    if (msg.type === "payment.event.session_created") paymentSession = msg.data;
  };
  await new Promise((res, rej) => { riderWS.onopen = res; setTimeout(() => rej(new Error("rider WS timeout")), 5000); });
  await sleep(800);

  await startTrip(riderID);
  if (!(await waitFor(() => offeredTrip && driverProfile))) throw new Error("no offer");

  driverWS.send(JSON.stringify({
    type: "driver.cmd.trip_accept",
    data: { tripID: offeredTrip.id, riderID, driver: driverProfile },
  }));
  await sleep(1200);

  riderWS.send(JSON.stringify({
    type: "trip.cmd.cancel",
    data: { tripID: offeredTrip.id },
  }));

  if (!(await waitFor(() => riderCancelled))) throw new Error("rider did not get cancelled");
  await sleep(800);
  if (paymentSession) throw new Error("payment must not be created after cancel");
  log("SUCCESS cancel path (driverCancelled=", driverCancelled, ")");

  driverWS.close(); riderWS.close();
}

async function main() {
  await happyPath();
  await cancelPath();
  log("ALL OTP E2E CHECKS PASSED");
  process.exit(0);
}

main().catch((e) => { console.error("FAILED:", e.message); process.exit(1); });
