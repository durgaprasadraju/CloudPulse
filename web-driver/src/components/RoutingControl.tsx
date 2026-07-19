import { Polyline } from "react-leaflet";

export function RoutingControl({ route }: {
    route: [number, number][]
}) {
    if (!route) return null

    return <Polyline positions={route} color="#059669" weight={5} />
}
