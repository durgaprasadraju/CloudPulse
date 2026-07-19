import { Coordinate, Driver, Route, RouteFare, Trip } from "./types";

export const BackendEndpoints = {
  PREVIEW_TRIP: "/trip/preview",
  START_TRIP: "/trip/start",
  WS_DRIVERS: "/drivers",
  WS_RIDERS: "/riders",
  MOCK_PAYMENT_SUCCESS: "/payment/mock-success",
  TRIP_REVIEW: (tripID: string) => `/trips/${tripID}/review`,
} as const;

export enum TripEvents {
  NoDriversFound = "trip.event.no_drivers_found",
  DriverAssigned = "trip.event.driver_assigned",
  DriverEnRoute = "trip.event.driver_en_route",
  DriverArrived = "trip.event.driver_arrived",
  TripStarted = "trip.event.started",
  Completed = "trip.event.completed",
  Cancelled = "trip.event.cancelled",
  Created = "trip.event.created",
  OTPIssued = "trip.event.otp_issued",
  OTPFailed = "trip.event.otp_failed",
  OTPVerified = "trip.event.otp_verified",
  DriverLocation = "driver.cmd.location",
  RiderLocation = "rider.cmd.location",
  DriverTripRequest = "driver.cmd.trip_request",
  DriverTripAccept = "driver.cmd.trip_accept",
  DriverTripDecline = "driver.cmd.trip_decline",
  DriverRegister = "driver.cmd.register",
  TripStatus = "trip.cmd.status",
  TripCancel = "trip.cmd.cancel",
  TripVerifyOTP = "trip.cmd.verify_otp",
  PaymentSessionCreated = "payment.event.session_created",
}

export type ServerWsMessage =
  | PaymentSessionCreatedRequest
  | DriverAssignedRequest
  | DriverLocationRequest
  | DriverTripRequest
  | DriverRegisterRequest
  | TripCreatedRequest
  | TripLifecycleRequest
  | OTPIssuedRequest
  | NoDriversFoundRequest;

export type ClientWsMessage =
  | DriverResponseToTripResponse
  | DriverLocationUpdate
  | TripStatusUpdate
  | TripCancelMessage;

interface TripCreatedRequest {
  type: TripEvents.Created;
  data: Trip;
}

interface NoDriversFoundRequest {
  type: TripEvents.NoDriversFound;
}

interface DriverRegisterRequest {
  type: TripEvents.DriverRegister;
  data: Driver;
}
interface DriverTripRequest {
  type: TripEvents.DriverTripRequest;
  data: Trip;
}

export interface PaymentEventSessionCreatedData {
  tripID: string;
  sessionID: string;
  amount: number;
  currency: string;
  provider?: string;
  checkoutURL?: string;
}

export interface TripOTPIssuedData {
  tripID: string;
  otp: string;
}

interface PaymentSessionCreatedRequest {
  type: TripEvents.PaymentSessionCreated;
  data: PaymentEventSessionCreatedData;
}

interface OTPIssuedRequest {
  type: TripEvents.OTPIssued;
  data: TripOTPIssuedData;
}

interface DriverAssignedRequest {
  type: TripEvents.DriverAssigned;
  data: Trip;
}

interface TripLifecycleRequest {
  type:
    | TripEvents.DriverEnRoute
    | TripEvents.DriverArrived
    | TripEvents.TripStarted
    | TripEvents.Completed
    | TripEvents.Cancelled;
  data: Trip | { trip: Trip };
}

interface DriverLocationRequest {
  type: TripEvents.DriverLocation;
  data: Driver[];
}

interface DriverResponseToTripResponse {
  type: TripEvents.DriverTripAccept | TripEvents.DriverTripDecline;
  data: {
    tripID: string;
    riderID: string;
    driver: Driver;
  };
}

interface DriverLocationUpdate {
  type: TripEvents.DriverLocation;
  data: {
    location: Coordinate;
    geohash: string;
  };
}

interface TripStatusUpdate {
  type: TripEvents.TripStatus;
  data: {
    event: TripEvents;
    riderID: string;
    trip: Trip;
  };
}

interface TripCancelMessage {
  type: TripEvents.TripCancel;
  data: {
    tripID: string;
  };
}

export interface HTTPTripPreviewResponse {
  route: Route;
  rideFares: RouteFare[];
}

export interface HTTPTripStartRequestPayload {
  rideFareID: string;
  userID: string;
}

export interface HTTPTripPreviewRequestPayload {
  userID: string;
  pickup: Coordinate;
  destination: Coordinate;
}

export function isValidTripEvent(event: string): event is TripEvents {
  return Object.values(TripEvents).includes(event as TripEvents);
}

export function isValidWsMessage(message: ServerWsMessage): message is ServerWsMessage {
  return isValidTripEvent(message.type);
}

export function unwrapTrip(data: Trip | { trip: Trip } | undefined): Trip | null {
  if (!data) return null;
  if ("trip" in data && data.trip) return data.trip;
  return data as Trip;
}

/** OSRM/geojson is [lon, lat]; Leaflet needs [lat, lon]. */
export function toLeafletRoute(coords: number[][]): [number, number][] {
  return coords.map(([a, b]) => {
    // Heuristic: Hyderabad lon ~78, lat ~17 — if first value looks like lon, swap.
    if (Math.abs(a) > 40 && Math.abs(b) < 40) {
      return [b, a];
    }
    return [a, b];
  });
}
