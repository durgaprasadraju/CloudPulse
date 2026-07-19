import { useEffect, useRef, useState } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { Trip, Driver, CarPackageSlug, Coordinate } from '../types';
import {
  ServerWsMessage,
  TripEvents,
  isValidWsMessage,
  isValidTripEvent,
  ClientWsMessage,
  BackendEndpoints,
  unwrapTrip,
} from '../contracts';

interface useDriverConnectionProps {
  location: Coordinate;
  geohash: string;
  userID: string;
  packageSlug: CarPackageSlug;
  autoSimulate?: boolean;
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function lerp(a: number, b: number, t: number) {
  return a + (b - a) * t;
}

export const useDriverStreamConnection = ({
  location,
  geohash,
  userID,
  packageSlug,
  autoSimulate = true,
}: useDriverConnectionProps) => {
  const [requestedTrip, setRequestedTrip] = useState<Trip | null>(null)
  const [tripStatus, setTripStatus] = useState<TripEvents | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [driver, setDriver] = useState<Driver | null>(null);
  const [simulating, setSimulating] = useState(false);
  const locationRef = useRef(location);
  locationRef.current = location;

  useEffect(() => {
    if (!userID) return;

    const websocket = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?userID=${userID}&packageSlug=${packageSlug}`);
    setWs(websocket);

    websocket.onopen = () => {
      if (location) {
        websocket.send(JSON.stringify({
          type: TripEvents.DriverLocation,
          data: {
            location,
            geohash,
          }
        }));
      }
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        setError(`Unknown message type "${(message as { type?: string })?.type}", allowed types are: ${Object.values(TripEvents).join(', ')}`);
        return;
      }

      switch (message.type) {
        case TripEvents.DriverTripRequest: {
          const trip = unwrapTrip(message.data as Trip | { trip: Trip });
          setRequestedTrip(trip);
          setTripStatus(TripEvents.DriverTripRequest);
          break;
        }
        case TripEvents.DriverRegister:
          setDriver(message.data);
          break;
      }

      if (isValidTripEvent(message.type) && message.type !== TripEvents.DriverTripRequest) {
        setTripStatus(message.type);
      }
    };

    websocket.onclose = () => {
      console.log('WebSocket closed');
    };

    websocket.onerror = (event) => {
      setError('WebSocket error occurred');
      console.error('WebSocket error:', event);
    };

    return () => {
      if (websocket.readyState === WebSocket.OPEN) {
        websocket.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userID]);

  // Push location whenever the driver moves on the map
  useEffect(() => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: TripEvents.DriverLocation,
        data: { location, geohash },
      }));
    }
  }, [ws, location, geohash]);

  const sendMessage = (message: ClientWsMessage) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      setError('WebSocket is not connected');
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
  };

  const sendLocation = (lat: number, lon: number) => {
    sendMessage({
      type: TripEvents.DriverLocation,
      data: {
        location: { latitude: lat, longitude: lon },
        geohash,
      },
    });
  };

  const runTripSimulation = async (trip: Trip, currentDriver: Driver) => {
    if (!trip?.route?.geometry?.[0]?.coordinates?.length) return;
    setSimulating(true);

    const coords = trip.route.geometry[0].coordinates;
    const pickup = coords[0];
    const start = locationRef.current;

    publishStatus(TripEvents.DriverEnRoute, { ...trip, driver: currentDriver });

    // Approach pickup
    for (let i = 1; i <= 8; i++) {
      const t = i / 8;
      const lat = lerp(start.latitude, pickup.latitude, t);
      const lon = lerp(start.longitude, pickup.longitude, t);
      sendLocation(lat, lon);
      await sleep(600);
    }

    publishStatus(TripEvents.DriverArrived, { ...trip, driver: currentDriver });
    await sleep(2000);
    publishStatus(TripEvents.TripStarted, { ...trip, driver: currentDriver });

    // Follow route to destination
    const steps = 12;
    for (let i = 1; i <= steps; i++) {
      const idx = Math.round((coords.length - 1) * (i / steps));
      const c = coords[Math.min(idx, coords.length - 1)];
      sendLocation(c.latitude, c.longitude);
      await sleep(600);
    }

    publishStatus(TripEvents.Completed, { ...trip, driver: currentDriver });
    setSimulating(false);
    setTripStatus(TripEvents.Completed);
  };

  const acceptAndMaybeSimulate = () => {
    if (!requestedTrip || !requestedTrip.id || !driver) {
      alert("No trip ID found or driver is not set");
      return;
    }

    sendMessage({
      type: TripEvents.DriverTripAccept,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver,
      }
    });

    setTripStatus(TripEvents.DriverTripAccept);

    if (autoSimulate) {
      void runTripSimulation(requestedTrip, driver);
    }
  };

  const resetTripStatus = () => {
    setTripStatus(null);
    setRequestedTrip(null);
    setSimulating(false);
  }

  return {
    error,
    tripStatus,
    driver,
    requestedTrip,
    resetTripStatus,
    sendMessage,
    setTripStatus,
    acceptAndMaybeSimulate,
    simulating,
  };
}
