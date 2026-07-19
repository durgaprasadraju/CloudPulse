import { useEffect, useRef, useState } from "react";
import { WEBSOCKET_URL } from "../constants";
import { Trip, Driver, Coordinate } from "../types";
import {
  ServerWsMessage,
  TripEvents,
  isValidWsMessage,
  isValidTripEvent,
  ClientWsMessage,
  BackendEndpoints,
  unwrapTrip,
  TripOTPResultData,
} from "../contracts";
import { isNearDestination, tripRouteLatLng } from "../lib/geo";

interface UseDriverStreamConnectionProps {
  location: Coordinate;
  geohash: string;
  token: string | null;
  packageSlug: string;
  online: boolean;
  onLocationSimulated?: (loc: Coordinate) => void;
}

export const useDriverStreamConnection = ({
  location,
  geohash,
  token,
  packageSlug,
  online,
  onLocationSimulated,
}: UseDriverStreamConnectionProps) => {
  const [requestedTrip, setRequestedTrip] = useState<Trip | null>(null);
  const [tripStatus, setTripStatus] = useState<TripEvents | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [otpError, setOtpError] = useState<string | null>(null);
  const [otpPending, setOtpPending] = useState(false);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [driver, setDriver] = useState<Driver | null>(null);
  const [connected, setConnected] = useState(false);
  const [travelProgress, setTravelProgress] = useState(0);
  const [atDestination, setAtDestination] = useState(false);
  const [simulating, setSimulating] = useState(false);

  const locationRef = useRef(location);
  locationRef.current = location;
  const onLocationSimulatedRef = useRef(onLocationSimulated);
  onLocationSimulatedRef.current = onLocationSimulated;
  const simAbortRef = useRef<AbortController | null>(null);
  const requestedTripRef = useRef<Trip | null>(null);
  requestedTripRef.current = requestedTrip;

  useEffect(() => {
    if (!token || !online) {
      setConnected(false);
      setWs(null);
      return;
    }

    const url = `${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?token=${encodeURIComponent(token)}&packageSlug=${encodeURIComponent(packageSlug)}`;
    const websocket = new WebSocket(url);
    setWs(websocket);

    websocket.onopen = () => {
      setConnected(true);
      setError(null);
      if (locationRef.current) {
        websocket.send(
          JSON.stringify({
            type: TripEvents.DriverLocation,
            data: {
              location: locationRef.current,
              geohash,
              packageSlug,
            },
          })
        );
      }
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        setError(
          `Unknown message type "${(message as { type?: string })?.type}", allowed types are: ${Object.values(TripEvents).join(", ")}`
        );
        return;
      }

      switch (message.type) {
        case TripEvents.DriverTripRequest: {
          const trip = unwrapTrip(message.data as Trip | { trip: Trip });
          setRequestedTrip(trip);
          setTripStatus(TripEvents.DriverTripRequest);
          setOtpError(null);
          setTravelProgress(0);
          setAtDestination(false);
          break;
        }
        case TripEvents.DriverRegister:
          setDriver(message.data);
          break;
        case TripEvents.OTPFailed: {
          const data = message.data as TripOTPResultData;
          setOtpPending(false);
          setOtpError(data.message || "Invalid OTP");
          break;
        }
        case TripEvents.OTPVerified: {
          setOtpPending(false);
          setOtpError(null);
          setTripStatus(TripEvents.TripStarted);
          break;
        }
        case TripEvents.TripStarted: {
          setOtpPending(false);
          setOtpError(null);
          setTripStatus(TripEvents.TripStarted);
          break;
        }
        case TripEvents.Cancelled: {
          simAbortRef.current?.abort();
          setSimulating(false);
          setRequestedTrip(null);
          setTripStatus(TripEvents.Cancelled);
          setOtpError(null);
          setTravelProgress(0);
          setAtDestination(false);
          break;
        }
      }

      if (
        isValidTripEvent(message.type) &&
        message.type !== TripEvents.DriverTripRequest &&
        message.type !== TripEvents.OTPFailed &&
        message.type !== TripEvents.OTPVerified
      ) {
        if (message.type !== TripEvents.Cancelled) {
          setTripStatus(message.type);
        }
      }
    };

    websocket.onclose = () => {
      setConnected(false);
    };

    websocket.onerror = () => {
      setError("WebSocket error occurred");
    };

    return () => {
      simAbortRef.current?.abort();
      if (websocket.readyState === WebSocket.OPEN || websocket.readyState === WebSocket.CONNECTING) {
        websocket.close();
      }
      setConnected(false);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, online]);

  useEffect(() => {
    if (ws?.readyState === WebSocket.OPEN && online && !simulating) {
      ws.send(
        JSON.stringify({
          type: TripEvents.DriverLocation,
          data: { location, geohash, packageSlug },
        })
      );
    }
  }, [ws, location, geohash, packageSlug, online, simulating]);

  const sendMessage = (message: ClientWsMessage) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      setError("WebSocket is not connected");
    }
  };

  const publishStatus = (event: TripEvents, trip: Trip) => {
    sendMessage({
      type: TripEvents.TripStatus,
      data: {
        event,
        riderID: trip.userID,
        trip: { ...trip, status: event },
      },
    });
    setTripStatus(event);
  };

  const acceptTrip = () => {
    if (!requestedTrip?.id || !driver) {
      setError("No trip ID found or driver is not registered yet");
      return;
    }

    sendMessage({
      type: TripEvents.DriverTripAccept,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver,
      },
    });
    setTripStatus(TripEvents.DriverTripAccept);
  };

  const declineTrip = () => {
    if (!requestedTrip?.id || !driver) {
      setError("No trip ID found or driver is not registered yet");
      return;
    }

    sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver,
      },
    });
    setTripStatus(TripEvents.DriverTripDecline);
    setRequestedTrip(null);
  };

  const advanceTripStatus = (event: TripEvents) => {
    if (!requestedTrip || !driver) return;
    publishStatus(event, { ...requestedTrip, driver });
  };

  const verifyOTP = (otp: string) => {
    if (!requestedTrip?.id) {
      setOtpError("No active trip");
      return;
    }
    setOtpPending(true);
    setOtpError(null);
    sendMessage({
      type: TripEvents.TripVerifyOTP,
      data: {
        tripID: requestedTrip.id,
        otp: otp.trim(),
        riderID: requestedTrip.userID,
      },
    });
  };

  // After OTP verified / trip started — simulate travel along the route.
  useEffect(() => {
    if (tripStatus !== TripEvents.TripStarted || !requestedTrip || simulating) {
      return;
    }

    const route = tripRouteLatLng(requestedTrip);
    if (route.length < 2) {
      setAtDestination(true);
      setTravelProgress(1);
      return;
    }

    const abort = new AbortController();
    simAbortRef.current = abort;
    setSimulating(true);
    setAtDestination(false);
    setTravelProgress(0);

    const run = async () => {
      const steps = Math.min(24, Math.max(8, route.length));
      for (let i = 1; i <= steps; i++) {
        if (abort.signal.aborted) return;
        const idx = Math.round(((route.length - 1) * i) / steps);
        const [lat, lng] = route[Math.min(idx, route.length - 1)];
        const next = { latitude: lat, longitude: lng };
        onLocationSimulatedRef.current?.(next);
        if (ws?.readyState === WebSocket.OPEN) {
          ws.send(
            JSON.stringify({
              type: TripEvents.DriverLocation,
              data: { location: next, geohash, packageSlug },
            })
          );
        }
        setTravelProgress(i / steps);
        await new Promise((r) => setTimeout(r, 650));
      }
      if (!abort.signal.aborted) {
        const last = route[route.length - 1];
        const dest = { latitude: last[0], longitude: last[1] };
        setAtDestination(true);
        setTravelProgress(1);
        setSimulating(false);
        onLocationSimulatedRef.current?.(dest);
      }
    };

    void run();

    return () => {
      abort.abort();
      setSimulating(false);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tripStatus, requestedTrip?.id]);

  // Also allow Complete if driver manually near destination.
  useEffect(() => {
    if (tripStatus !== TripEvents.TripStarted || !requestedTrip) return;
    const route = tripRouteLatLng(requestedTrip);
    if (!route.length) return;
    const last = route[route.length - 1];
    if (isNearDestination(location, { latitude: last[0], longitude: last[1] })) {
      setAtDestination(true);
    }
  }, [location, tripStatus, requestedTrip]);

  const completeTrip = () => {
    if (!requestedTrip || !driver || !atDestination) return;
    publishStatus(TripEvents.Completed, { ...requestedTrip, driver });
    setSimulating(false);
  };

  const resetTripStatus = () => {
    simAbortRef.current?.abort();
    setTripStatus(null);
    setRequestedTrip(null);
    setOtpError(null);
    setTravelProgress(0);
    setAtDestination(false);
    setSimulating(false);
  };

  return {
    error,
    otpError,
    otpPending,
    tripStatus,
    driver,
    requestedTrip,
    connected,
    travelProgress,
    atDestination,
    simulating,
    resetTripStatus,
    sendMessage,
    setTripStatus,
    acceptTrip,
    declineTrip,
    advanceTripStatus,
    verifyOTP,
    completeTrip,
  };
};
