import { useEffect, useRef, useState } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { Trip, Driver, Coordinate } from '../types';
import {
  PaymentEventSessionCreatedData,
  TripEvents,
  ServerWsMessage,
  isValidWsMessage,
  BackendEndpoints,
  unwrapTrip,
} from '../contracts';

export function useRiderStreamConnection(location: Coordinate, userID: string) {
  const [drivers, setDrivers] = useState<Driver[]>([]);
  const [tripStatus, setTripStatus] = useState<TripEvents | null>(null);
  const [paymentSession, setPaymentSession] = useState<PaymentEventSessionCreatedData | null>(null);
  const [assignedDriver, setAssignedDriver] = useState<Driver | null>(null);
  const [activeTrip, setActiveTrip] = useState<Trip | null>(null);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!userID) return;

    const ws = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_RIDERS}?userID=${userID}`);
    wsRef.current = ws;

    ws.onopen = () => {
      if (location) {
        ws.send(JSON.stringify({
          type: TripEvents.RiderLocation,
          data: { location },
        }));
      }
    };

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        setError(`Unknown message type "${(message as { type?: string })?.type}", allowed types are: ${Object.values(TripEvents).join(', ')}`);
        return;
      }

      switch (message.type) {
        case TripEvents.DriverLocation:
          setDrivers(Array.isArray(message.data) ? message.data : []);
          break;
        case TripEvents.PaymentSessionCreated:
          setPaymentSession(message.data);
          setTripStatus(message.type);
          break;
        case TripEvents.DriverAssigned: {
          const trip = unwrapTrip(message.data);
          if (trip?.driver) {
            setAssignedDriver({
              id: trip.driver.id,
              name: trip.driver.name,
              profilePicture: trip.driver.profilePicture,
              carPlate: trip.driver.carPlate,
              geohash: (trip.driver as Driver).geohash ?? '',
              location: (trip.driver as Driver).location ?? { latitude: 0, longitude: 0 },
            });
          }
          setActiveTrip(trip);
          setTripStatus(message.type);
          break;
        }
        case TripEvents.DriverEnRoute:
        case TripEvents.DriverArrived:
        case TripEvents.TripStarted:
        case TripEvents.Completed:
        case TripEvents.Cancelled: {
          const trip = unwrapTrip(message.data as Trip | { trip: Trip });
          if (trip) {
            setActiveTrip(trip);
            if (trip.driver) {
              setAssignedDriver({
                id: trip.driver.id,
                name: trip.driver.name,
                profilePicture: trip.driver.profilePicture,
                carPlate: trip.driver.carPlate,
                geohash: (trip.driver as Driver).geohash ?? '',
                location: (trip.driver as Driver).location ?? { latitude: 0, longitude: 0 },
              });
            }
          }
          setTripStatus(message.type);
          break;
        }
        case TripEvents.Created:
          setTripStatus(message.type);
          break;
        case TripEvents.NoDriversFound:
          setTripStatus(message.type);
          break;
      }
    };

    ws.onclose = () => {
      console.log('WebSocket closed');
    };

    ws.onerror = (event) => {
      setError('WebSocket error occurred');
      console.error('WebSocket error:', event);
    };

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userID]);

  // Keep sending rider location so nearby-driver queries stay centered
  useEffect(() => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN || !location) return;
    ws.send(JSON.stringify({
      type: TripEvents.RiderLocation,
      data: { location },
    }));
  }, [location]);

  const cancelTrip = (tripID?: string) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: TripEvents.TripCancel,
        data: { tripID: tripID ?? '' },
      }));
    }
  };

  const resetTripStatus = () => {
    setTripStatus(null);
    setPaymentSession(null);
    setAssignedDriver(null);
    setActiveTrip(null);
  };

  return {
    drivers,
    assignedDriver,
    activeTrip,
    error,
    tripStatus,
    paymentSession,
    resetTripStatus,
    cancelTrip,
  };
}
