'use client';

import Image from 'next/image';
import { useRiderStreamConnection } from '../hooks/useRiderStreamConnection';
import { MapContainer, Marker, Popup, TileLayer } from 'react-leaflet'
import L from 'leaflet';
import { useMemo, useRef, useState } from 'react';
import { MapClickHandler } from './MapClickHandler';
import { RouteFare, RequestRideProps, TripPreview, HTTPTripStartResponse, Coordinate } from "../types";
import { RoutingControl } from "./RoutingControl";
import { API_URL } from '../constants';
import { DEFAULT_LOCATION } from '../constants/location';
import { RiderTripOverview } from './RiderTripOverview';
import { BackendEndpoints, HTTPTripPreviewRequestPayload, HTTPTripPreviewResponse, HTTPTripStartRequestPayload, TripEvents } from '../contracts';

const userMarker = new L.Icon({
    iconUrl: "data:image/svg+xml;utf8," + encodeURIComponent(
        `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#e11d48" stroke="#fff" stroke-width="1.5"><path d="M12 2C7.6 2 4 5.6 4 10c0 5.5 7.3 11.5 7.6 11.7.2.2.6.2.8 0C12.7 21.5 20 15.5 20 10c0-4.4-3.6-8-8-8zm0 11a3 3 0 1 1 0-6 3 3 0 0 1 0 6z"/></svg>`
    ),
    iconSize: [40, 40],
    iconAnchor: [20, 40],
});

const pickupMarker = new L.Icon({
    iconUrl: "data:image/svg+xml;utf8," + encodeURIComponent(
        `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#2563eb" stroke="#fff" stroke-width="1.5"><circle cx="12" cy="12" r="8"/></svg>`
    ),
    iconSize: [28, 28],
    iconAnchor: [14, 14],
});

const driverMarker = new L.Icon({
    iconUrl: "https://www.svgrepo.com/show/25407/car.svg",
    iconSize: [30, 30],
    iconAnchor: [15, 30],
});

const assignedDriverMarker = new L.Icon({
    iconUrl: "https://www.svgrepo.com/show/375838/car.svg",
    iconSize: [36, 36],
    iconAnchor: [18, 18],
});

function fmtCoord(c: Coordinate) {
    return `${c.latitude.toFixed(4)}, ${c.longitude.toFixed(4)}`;
}

interface RiderMapProps {
    onRouteSelected?: (distance: number) => void;
}

export default function RiderMap({ onRouteSelected }: RiderMapProps) {
    const [trip, setTrip] = useState<TripPreview | null>(null)
    const [destination, setDestination] = useState<[number, number] | null>(null)
    const [pickup, setPickup] = useState<Coordinate>({
        latitude: DEFAULT_LOCATION.latitude,
        longitude: DEFAULT_LOCATION.longitude,
    });
    const [settingPickup, setSettingPickup] = useState(false);
    const [paid, setPaid] = useState(false);
    const [awaitingFeedback, setAwaitingFeedback] = useState(false);
    const [feedbackTripID, setFeedbackTripID] = useState<string | null>(null);
    const mapRef = useRef<L.Map>(null)
    const userID = useMemo(() => crypto.randomUUID(), [])
    const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    const {
        drivers,
        error,
        tripStatus,
        assignedDriver,
        paymentSession,
        pickupOTP,
        resetTripStatus,
        cancelTrip,
    } = useRiderStreamConnection(pickup, userID);

    const handleMapClick = async (e: L.LeafletMouseEvent) => {
        if (trip?.tripID || paid) {
            return
        }

        // First mode: reposition pickup
        if (settingPickup) {
            setPickup({ latitude: e.latlng.lat, longitude: e.latlng.lng });
            setSettingPickup(false);
            setTrip(null);
            setDestination(null);
            return;
        }

        if (debounceTimeoutRef.current) {
            clearTimeout(debounceTimeoutRef.current);
        }

        debounceTimeoutRef.current = setTimeout(async () => {
            setDestination([e.latlng.lat, e.latlng.lng])

            try {
                const data = await requestRidePreview({
                    pickup: [pickup.latitude, pickup.longitude],
                    destination: [e.latlng.lat, e.latlng.lng],
                })

                const parsedRoute = data.route.geometry[0].coordinates
                    .map((coord) => [coord.latitude, coord.longitude] as [number, number])

                setTrip({
                    tripID: "",
                    route: parsedRoute,
                    rideFares: data.rideFares,
                    distance: data.route.distance,
                    duration: data.route.duration,
                })

                onRouteSelected?.(data.route.distance)
            } catch (err) {
                console.error(err)
                alert("Failed to preview route. Is the API running?")
            }
        }, 400);
    }

    const requestRidePreview = async (props: RequestRideProps): Promise<HTTPTripPreviewResponse> => {
        const { pickup: p, destination: d } = props
        const payload = {
            userID: userID,
            pickup: {
                latitude: p[0],
                longitude: p[1],
            },
            destination: {
                latitude: d[0],
                longitude: d[1],
            },
        } as HTTPTripPreviewRequestPayload

        const response = await fetch(`${API_URL}${BackendEndpoints.PREVIEW_TRIP}`, {
            method: 'POST',
            body: JSON.stringify(payload),
        })
        const { data } = await response.json() as { data: HTTPTripPreviewResponse }
        return data
    }

    const handleStartTrip = async (fare: RouteFare) => {
        const payload = {
            rideFareID: fare.id,
            userID: userID,
        } as HTTPTripStartRequestPayload

        if (!fare.id) {
            alert("No Fare ID in the payload")
            return
        }

        const response = await fetch(`${API_URL}${BackendEndpoints.START_TRIP}`, {
            method: 'POST',
            body: JSON.stringify(payload),
        })
        const data = await response.json() as HTTPTripStartResponse

        if (response.ok && trip) {
            setTrip((prev) => ({
                ...prev,
                tripID: data.tripID,
            } as TripPreview))
        }

        return data
    }

    const handleCancelTrip = () => {
        if (trip?.tripID && tripStatus !== TripEvents.Cancelled) {
            cancelTrip(trip.tripID)
            // Keep UI until cancelled event arrives (or user books again from cancelled screen).
            if (
                tripStatus === TripEvents.DriverAssigned ||
                tripStatus === TripEvents.DriverEnRoute ||
                tripStatus === TripEvents.DriverArrived ||
                tripStatus === TripEvents.Created
            ) {
                return
            }
        }
        setTrip(null)
        setDestination(null)
        setPaid(false)
        setAwaitingFeedback(false)
        setFeedbackTripID(null)
        resetTripStatus()
    }

    const handlePaymentDone = () => {
        setPaid(true)
        setAwaitingFeedback(true)
        setFeedbackTripID(trip?.tripID ?? paymentSession?.tripID ?? null)
        resetTripStatus()
    }

    const handleFeedbackDone = () => {
        setAwaitingFeedback(false)
        setFeedbackTripID(null)
        setTrip(null)
        setDestination(null)
        setPaid(false)
    }

    // Prefer live assigned-driver marker from nearby stream when IDs match
    const liveAssigned = assignedDriver
        ? drivers.find((d) => d.id === assignedDriver.id) ?? assignedDriver
        : null;

    if (error) {
        return <div className="p-4 text-red-600">Error: {error}</div>
    }

    if (paid) {
        return (
            <div className="flex h-screen items-center justify-center bg-slate-50">
                <div className="rounded-xl bg-white p-8 shadow-lg max-w-md text-center space-y-4">
                    <h1 className="text-2xl font-bold">Thanks for riding!</h1>
                    <p className="text-gray-600">Payment completed. Your receipt is confirmed.</p>
                    <button
                        className="rounded-md bg-black text-white px-4 py-2"
                        onClick={() => setPaid(false)}
                    >
                        Book another trip
                    </button>
                </div>
            </div>
        )
    }

    return (
        // 100dvh keeps the bottom panel visible above mobile browser chrome (100vh overflows)
        <div className="relative flex flex-col md:flex-row h-screen" style={{ height: "100dvh" }}>
            <div className={`${destination || trip ? 'flex-[0.6] min-h-0' : 'flex-1'} relative`}>
                <div className="absolute top-3 left-3 z-[1000] flex gap-2">
                    <button
                        className={`rounded-full px-3 py-1.5 text-sm shadow ${settingPickup ? 'bg-blue-600 text-white' : 'bg-white'}`}
                        onClick={() => setSettingPickup((v) => !v)}
                        disabled={Boolean(trip?.tripID)}
                    >
                        {settingPickup ? 'Tap map for pickup…' : 'Move pickup'}
                    </button>
                </div>
                <MapContainer
                    center={[pickup.latitude, pickup.longitude]}
                    zoom={13}
                    style={{ height: '100%', width: '100%' }}
                    ref={mapRef}
                >
                    <TileLayer
                        url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
                        attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a> contributors &copy; <a href='https://carto.com/'>CARTO</a>"
                    />
                    <Marker position={[pickup.latitude, pickup.longitude]} icon={pickupMarker}>
                        <Popup>Pickup</Popup>
                    </Marker>

                    {/* Nearby available drivers */}
                    {drivers
                        ?.filter((d) => !assignedDriver || d.id !== assignedDriver.id)
                        .map((driver) => (
                        <Marker
                            key={driver?.id}
                            position={[driver?.location?.latitude, driver?.location?.longitude]}
                            icon={driverMarker}
                        >
                            <Popup>
                                {driver?.name || 'Driver'}
                                <br />
                                {driver?.carPlate}
                                {driver?.profilePicture ? (
                                    <>
                                        <br />
                                        <Image
                                            src={driver.profilePicture}
                                            alt={`${driver?.name}'s profile`}
                                            width={80}
                                            height={80}
                                        />
                                    </>
                                ) : null}
                            </Popup>
                        </Marker>
                    ))}

                    {liveAssigned?.location && (
                        <Marker
                            position={[liveAssigned.location.latitude, liveAssigned.location.longitude]}
                            icon={assignedDriverMarker}
                        >
                            <Popup>Your driver: {liveAssigned.name}</Popup>
                        </Marker>
                    )}

                    {destination && (
                        <Marker position={destination} icon={userMarker}>
                            <Popup>Dropoff</Popup>
                        </Marker>
                    )}

                    {trip && <RoutingControl route={trip.route} />}
                    <MapClickHandler onClick={handleMapClick} />
                </MapContainer>
            </div>

            <div className="flex-[0.4] min-h-0 overflow-y-auto">
                <RiderTripOverview
                    trip={
                        awaitingFeedback && feedbackTripID
                            ? { ...(trip ?? { tripID: feedbackTripID, route: [], rideFares: [], distance: 0, duration: 0 }), tripID: feedbackTripID }
                            : trip
                    }
                    assignedDriver={assignedDriver}
                    status={paid && !awaitingFeedback ? TripEvents.Completed : tripStatus}
                    paymentSession={paymentSession}
                    pickupOTP={pickupOTP}
                    pickupLabel={fmtCoord(pickup)}
                    dropoffLabel={destination ? `${destination[0].toFixed(4)}, ${destination[1].toFixed(4)}` : undefined}
                    userID={userID}
                    awaitingFeedback={awaitingFeedback}
                    onPackageSelect={handleStartTrip}
                    onCancel={handleCancelTrip}
                    onPaymentDone={handlePaymentDone}
                    onFeedbackDone={handleFeedbackDone}
                />
            </div>
        </div>
    )
}
