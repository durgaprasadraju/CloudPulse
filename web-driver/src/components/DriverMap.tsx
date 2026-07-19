"use client"

import { useDriverStreamConnection } from "../hooks/useDriverStreamConnection"
import { MapContainer, Marker, Popup, TileLayer } from "react-leaflet"
import L from "leaflet"
import { MapClickHandler } from "./MapClickHandler"
import { useMemo, useState } from "react"
import { useRef } from "react"
import { Coordinate } from "../types"
import { DriverTripOverview } from "./DriverTripOverview"
import * as Geohash from "ngeohash"
import { RoutingControl } from "./RoutingControl"
import { TripEvents } from "../contracts"
import { DEFAULT_LOCATION } from "../constants/location"
import { useAuth } from "../context/AuthContext"
import { DriverHeader } from "./DriverHeader"

const START_LOCATION: Coordinate = {
  latitude: DEFAULT_LOCATION.latitude,
  longitude: DEFAULT_LOCATION.longitude,
}

const driverMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/25407/car.svg",
  iconSize: [30, 30],
  iconAnchor: [15, 30],
})

const startLocationMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/535711/user.svg",
  iconSize: [30, 40],
  iconAnchor: [20, 40],
})

const destinationMarker = new L.Icon({
  iconUrl:
    "data:image/svg+xml;utf8," +
    encodeURIComponent(
      `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#e11d48" stroke="#fff" stroke-width="1.5"><path d="M12 2C7.6 2 4 5.6 4 10c0 5.5 7.3 11.5 7.6 11.7.2.2.6.2.8 0C12.7 21.5 20 15.5 20 10c0-4.4-3.6-8-8-8zm0 11a3 3 0 1 1 0-6 3 3 0 0 1 0 6z"/></svg>`
    ),
  iconSize: [40, 40],
  iconAnchor: [20, 40],
})

export const DriverMap = () => {
  const { token, profile } = useAuth()
  const mapRef = useRef<L.Map>(null)
  const [driverLocation, setDriverLocation] = useState<Coordinate>(START_LOCATION)
  const [online, setOnline] = useState(false)

  const driverGeohash = useMemo(
    () => Geohash.encode(driverLocation.latitude, driverLocation.longitude, 7),
    [driverLocation.latitude, driverLocation.longitude]
  )

  const packageSlug = profile?.packageSlug ?? "sedan"

  const {
    error,
    otpError,
    otpPending,
    tripStatus,
    requestedTrip,
    connected,
    travelProgress,
    atDestination,
    simulating,
    resetTripStatus,
    acceptTrip,
    declineTrip,
    advanceTripStatus,
    verifyOTP,
    completeTrip,
  } = useDriverStreamConnection({
    location: driverLocation,
    geohash: driverGeohash,
    token,
    packageSlug,
    online,
    onLocationSimulated: setDriverLocation,
  })

  const handleMapClick = (e: L.LeafletMouseEvent) => {
    if (simulating || tripStatus === TripEvents.TripStarted) {
      return
    }
    setDriverLocation({
      latitude: e.latlng.lat,
      longitude: e.latlng.lng,
    })
  }

  const parsedRoute = useMemo(
    () =>
      requestedTrip?.route?.geometry[0]?.coordinates.map(
        (coord) => [coord.latitude, coord.longitude] as [number, number]
      ),
    [requestedTrip]
  )

  const destination = useMemo(
    () =>
      requestedTrip?.route?.geometry[0]?.coordinates[
        requestedTrip.route.geometry[0].coordinates.length - 1
      ],
    [requestedTrip]
  )

  const startLocation = useMemo(
    () => requestedTrip?.route?.geometry[0]?.coordinates[0],
    [requestedTrip]
  )

  return (
    <div className="flex flex-col h-screen bg-zinc-100">
      <DriverHeader
        online={online}
        connected={connected}
        onToggleOnline={() => setOnline((v) => !v)}
        showOnlineControls
      />

      <div className="relative flex flex-col md:flex-row flex-1 min-h-0">
        <div className="flex-1 relative min-h-[45vh] md:min-h-0">
          <MapContainer
            center={[driverLocation.latitude, driverLocation.longitude]}
            zoom={13}
            style={{ height: "100%", width: "100%" }}
            ref={mapRef}
          >
            <TileLayer
              url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
              attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a> contributors &copy; <a href='https://carto.com/'>CARTO</a>"
            />

            <Marker
              position={[driverLocation.latitude, driverLocation.longitude]}
              icon={driverMarker}
            >
              <Popup>
                You (driver)
                <br />
                {simulating ? "Simulating trip…" : "Tap map to reposition"}
              </Popup>
            </Marker>

            {startLocation && (
              <Marker
                position={[startLocation.latitude, startLocation.longitude]}
                icon={startLocationMarker}
              >
                <Popup>Rider pickup</Popup>
              </Marker>
            )}

            {destination && (
              <Marker
                position={[destination.latitude, destination.longitude]}
                icon={destinationMarker}
              >
                <Popup>Dropoff</Popup>
              </Marker>
            )}

            {parsedRoute && <RoutingControl route={parsedRoute} />}

            <MapClickHandler onClick={handleMapClick} />
          </MapContainer>
        </div>

        <div className="flex flex-col md:w-[380px] bg-zinc-50 border-t md:border-t-0 md:border-l border-zinc-200">
          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {error && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {error}
              </div>
            )}
            <DriverTripOverview
              trip={requestedTrip}
              status={tripStatus}
              online={online}
              otpError={otpError}
              otpPending={otpPending}
              travelProgress={travelProgress}
              atDestination={atDestination}
              simulating={simulating}
              onAcceptTrip={acceptTrip}
              onDeclineTrip={declineTrip}
              onEnRoute={() => advanceTripStatus(TripEvents.DriverEnRoute)}
              onArrived={() => advanceTripStatus(TripEvents.DriverArrived)}
              onVerifyOTP={verifyOTP}
              onCompleted={completeTrip}
              onReset={resetTripStatus}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
