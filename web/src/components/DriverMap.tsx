"use client"

import { useDriverStreamConnection } from "../hooks/useDriverStreamConnection"
import { MapContainer, Marker, Popup, TileLayer } from 'react-leaflet'
import L from 'leaflet';
import { MapClickHandler } from './MapClickHandler';
import { useMemo, useState } from "react";
import { useRef } from "react";
import { CarPackageSlug, Coordinate } from "../types";
import { DriverTripOverview } from "./DriverTripOverview";
import * as Geohash from 'ngeohash';
import { RoutingControl } from "./RoutingControl";
import { DriverCard } from "./DriverCard";
import { TripEvents } from "../contracts";

const START_LOCATION: Coordinate = {
  latitude: 37.7749,
  longitude: -122.4194,
}

const driverMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/25407/car.svg",
  iconSize: [30, 30],
  iconAnchor: [15, 30],
});

const startLocationMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/535711/user.svg",
  iconSize: [30, 40],
  iconAnchor: [20, 40],
});

const destinationMarker = new L.Icon({
  iconUrl: "data:image/svg+xml;utf8," + encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#e11d48" stroke="#fff" stroke-width="1.5"><path d="M12 2C7.6 2 4 5.6 4 10c0 5.5 7.3 11.5 7.6 11.7.2.2.6.2.8 0C12.7 21.5 20 15.5 20 10c0-4.4-3.6-8-8-8zm0 11a3 3 0 1 1 0-6 3 3 0 0 1 0 6z"/></svg>`
  ),
  iconSize: [40, 40],
  iconAnchor: [20, 40],
});

export const DriverMap = ({ packageSlug }: { packageSlug: CarPackageSlug }) => {
  const mapRef = useRef<L.Map>(null)
  const userID = useMemo(() => crypto.randomUUID(), [])
  const [driverLocation, setDriverLocation] = useState<Coordinate>(START_LOCATION)
  const [autoSimulate, setAutoSimulate] = useState(true)

  const driverGeohash = useMemo(() =>
    Geohash.encode(driverLocation?.latitude, driverLocation?.longitude, 7)
    , [driverLocation?.latitude, driverLocation?.longitude]);

  const {
    error,
    driver,
    tripStatus,
    requestedTrip,
    sendMessage,
    setTripStatus,
    resetTripStatus,
    acceptAndMaybeSimulate,
    simulating,
  } = useDriverStreamConnection({
    location: driverLocation,
    geohash: driverGeohash,
    userID,
    packageSlug,
    autoSimulate,
  })

  const handleMapClick = (e: L.LeafletMouseEvent) => {
    if (simulating) return;
    setDriverLocation({
      latitude: e.latlng.lat,
      longitude: e.latlng.lng
    })
  }

  const handleAcceptTrip = () => {
    acceptAndMaybeSimulate()
  }

  const handleDeclineTrip = () => {
    if (!requestedTrip || !requestedTrip.id || !driver) {
      alert("No trip ID found or driver is not set")
      return
    }

    sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver,
      }
    })
    setTripStatus(TripEvents.DriverTripDecline)
    resetTripStatus()
  }

  const parsedRoute = useMemo(() =>
    requestedTrip?.route?.geometry[0]?.coordinates
      .map((coord) => [coord?.longitude, coord?.latitude] as [number, number])
    , [requestedTrip])

  const destination = useMemo(() =>
    requestedTrip?.route?.geometry[0]?.coordinates[requestedTrip?.route?.geometry[0]?.coordinates?.length - 1]
    , [requestedTrip])

  const startLocation = useMemo(() =>
    requestedTrip?.route?.geometry[0]?.coordinates[0]
    , [requestedTrip])

  if (error) {
    return <div className="p-4 text-red-600">Error: {error}</div>
  }

  return (
    <div className="relative flex flex-col md:flex-row h-screen">
      <div className="flex-1 relative">
        <div className="absolute top-3 left-3 z-[1000]">
          <label className="flex items-center gap-2 rounded-full bg-white px-3 py-1.5 text-sm shadow cursor-pointer">
            <input
              type="checkbox"
              checked={autoSimulate}
              onChange={(e) => setAutoSimulate(e.target.checked)}
            />
            Auto-simulate trip after accept
          </label>
        </div>
        <MapContainer
          center={[driverLocation.latitude, driverLocation.longitude]}
          zoom={13}
          style={{ height: '100%', width: '100%' }}
          ref={mapRef}
        >
          <TileLayer
            url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
            attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a> contributors &copy; <a href='https://carto.com/'>CARTO</a>"
          />

          <Marker
            key={userID}
            position={[driverLocation.latitude, driverLocation.longitude]}
            icon={driverMarker}
          >
            <Popup>
              You (driver)
              <br />
              {simulating ? 'Simulating trip…' : 'Click map to reposition'}
            </Popup>
          </Marker>

          {startLocation && (
            <Marker position={[startLocation.latitude, startLocation.longitude]} icon={startLocationMarker}>
              <Popup>Rider pickup</Popup>
            </Marker>
          )}

          {destination && (
            <Marker position={[destination.latitude, destination.longitude]} icon={destinationMarker}>
              <Popup>Dropoff</Popup>
            </Marker>
          )}

          {parsedRoute && (
            <RoutingControl route={parsedRoute} />
          )}

          <MapClickHandler onClick={handleMapClick} />
        </MapContainer>
      </div>

      <div className="flex flex-col md:w-[400px] bg-white border-t md:border-t-0 md:border-l">
        <div className="p-4 border-b">
          <DriverCard driver={driver} packageSlug={packageSlug} />
        </div>
        <div className="flex-1 overflow-y-auto">
          <DriverTripOverview
            trip={requestedTrip}
            status={tripStatus}
            simulating={simulating}
            onAcceptTrip={handleAcceptTrip}
            onDeclineTrip={handleDeclineTrip}
          />
        </div>
      </div>
    </div>
  )
}
