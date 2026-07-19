import { Trip } from "../types"
import { TripOverviewCard } from "./TripOverviewCard"
import { Button } from "./ui/button"
import { TripEvents } from "../contracts"
import { Skeleton } from "./ui/skeleton"

interface DriverTripOverviewProps {
  trip?: Trip | null,
  status?: TripEvents | null,
  simulating?: boolean,
  onAcceptTrip?: () => void,
  onDeclineTrip?: () => void
}

export const DriverTripOverview = ({
  trip,
  status,
  simulating,
  onAcceptTrip,
  onDeclineTrip,
}: DriverTripOverviewProps) => {
  if (!trip) {
    return (
      <TripOverviewCard
        title="Waiting for a rider…"
        description="Stay online — trip requests for your package will appear here. Click the map to set your position."
      />
    )
  }

  if (status === TripEvents.DriverTripRequest) {
    return (
      <TripOverviewCard
        title="New trip request"
        description="A rider needs a ride. Check the route, then accept or decline."
      >
        <div className="flex flex-col gap-2">
          <p className="text-sm text-gray-600">Trip ID: {trip.id}</p>
          <Button onClick={onAcceptTrip}>Accept trip</Button>
          <Button variant="outline" onClick={onDeclineTrip}>Decline</Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (simulating || status === TripEvents.DriverTripAccept || status === TripEvents.DriverEnRoute || status === TripEvents.DriverArrived || status === TripEvents.TripStarted) {
    return (
      <TripOverviewCard
        title={simulating ? "Simulating trip…" : "Trip in progress"}
        description="Driving to pickup, then to the destination. The rider map updates live."
      >
        <div className="space-y-3">
          <Skeleton className="h-3 w-full" />
          <Skeleton className="h-3 w-4/5" />
          <p className="text-sm text-gray-500">Trip {trip.id}</p>
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Completed) {
    return (
      <TripOverviewCard
        title="Trip completed"
        description="Nice work — the rider will pay and you are free for the next request."
      />
    )
  }

  return null
}
