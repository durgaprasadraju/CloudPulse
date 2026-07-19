import { Coordinate, Driver, Route, RouteFare, Trip } from "./types";

// These are the endpoints the API Gateway must have for the frontend to work correctly
export enum BackendEndpoints {
  PREVIEW_TRIP = "/trip/preview",
  START_TRIP = "/trip/start",
  WS_DRIVERS = "/drivers",
  WS_RIDERS = "/riders",
}

export enum TripEvents {
  NoDriversFound = "trip.event.no_drivers_found",
  DriverAssigned = "trip.event.driver_assigned",
  DriverEnRoute = "trip.event.driver_en_route",
  DriverArrived = "trip.event.driver_arrived",
  TripStarted = "trip.event.started",
  Completed = "trip.event.completed",
  Cancelled = "trip.event.cancelled",
  Created = "trip.event.created",
  DriverLocation = "driver.cmd.location",
  RiderLocation = "rider.cmd.location",
  DriverTripRequest = "driver.cmd.trip_request",
  DriverTripAccept = "driver.cmd.trip_accept",
  DriverTripDecline = "driver.cmd.trip_decline",
  DriverRegister = "driver.cmd.register",
  TripStatus = "trip.cmd.status",
  TripCancel = "trip.cmd.cancel",
  PaymentSessionCreated = "payment.event.session_created",
}

// Messages sent from the server to the client via the websocket
export type ServerWsMessage =
  | PaymentSessionCreatedRequest
  | DriverAssignedRequest
  | DriverLocationRequest
  | DriverTripRequest
  | DriverRegisterRequest
  | TripCreatedRequest
  | TripLifecycleRequest
  | NoDriversFoundRequest;

// Messages sent from the client to the server via the websocket
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
}

interface PaymentSessionCreatedRequest {
  type: TripEvents.PaymentSessionCreated;
  data: PaymentEventSessionCreatedData;
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
