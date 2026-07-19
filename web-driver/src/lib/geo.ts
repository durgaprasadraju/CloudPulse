/** Haversine distance in meters between two lat/lng points. */
export function distanceMeters(
  a: { latitude: number; longitude: number },
  b: { latitude: number; longitude: number }
): number {
  const R = 6371000;
  const toRad = (d: number) => (d * Math.PI) / 180;
  const dLat = toRad(b.latitude - a.latitude);
  const dLon = toRad(b.longitude - a.longitude);
  const lat1 = toRad(a.latitude);
  const lat2 = toRad(b.latitude);
  const h =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLon / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(h));
}

export function isNearDestination(
  current: { latitude: number; longitude: number },
  destination: { latitude: number; longitude: number },
  thresholdMeters = 80
): boolean {
  return distanceMeters(current, destination) <= thresholdMeters;
}

/** Extract Leaflet-friendly [lat,lng] route points from a trip route. */
export function tripRouteLatLng(trip: {
  route?: {
    geometry?: { coordinates?: { latitude: number; longitude: number }[] }[];
  };
}): [number, number][] {
  const coords = trip.route?.geometry?.[0]?.coordinates;
  if (!coords?.length) return [];
  return coords.map((c) => [c.latitude, c.longitude]);
}
