import { useState } from "react"
import { Trip } from "../types"
import { TripOverviewCard } from "./TripOverviewCard"
import { Button } from "./ui/button"
import { TripEvents } from "../contracts"
import { CheckCircle2, MapPin, Navigation, Flag } from "lucide-react"

interface DriverTripOverviewProps {
  trip?: Trip | null
  status?: TripEvents | null
  online?: boolean
  otpError?: string | null
  otpPending?: boolean
  travelProgress?: number
  atDestination?: boolean
  simulating?: boolean
  onAcceptTrip?: () => void
  onDeclineTrip?: () => void
  onEnRoute?: () => void
  onArrived?: () => void
  onVerifyOTP?: (otp: string) => void
  onCompleted?: () => void
  onReset?: () => void
}

const statusLabel: Partial<Record<TripEvents, string>> = {
  [TripEvents.DriverTripAccept]: "Accepted — head to pickup",
  [TripEvents.DriverEnRoute]: "En route to pickup",
  [TripEvents.DriverArrived]: "Arrived — enter rider OTP",
  [TripEvents.TripStarted]: "Trip in progress",
  [TripEvents.Completed]: "Completed",
  [TripEvents.Cancelled]: "Cancelled by rider",
}

export const DriverTripOverview = ({
  trip,
  status,
  online,
  otpError,
  otpPending,
  travelProgress = 0,
  atDestination = false,
  simulating = false,
  onAcceptTrip,
  onDeclineTrip,
  onEnRoute,
  onArrived,
  onVerifyOTP,
  onCompleted,
  onReset,
}: DriverTripOverviewProps) => {
  const [otp, setOtp] = useState("")

  if (!online) {
    return (
      <TripOverviewCard
        title="You're offline"
        description="Toggle Go Online in the header to start receiving trip requests."
      />
    )
  }

  if (status === TripEvents.Cancelled) {
    return (
      <TripOverviewCard
        title="Trip cancelled"
        description="The rider cancelled before the trip started."
      >
        <Button variant="outline" className="w-full" onClick={onReset}>
          Ready for next trip
        </Button>
      </TripOverviewCard>
    )
  }

  if (!trip) {
    return (
      <TripOverviewCard
        title="Waiting for a rider…"
        description="Stay online — trip requests for your package will appear here. Tap the map to set your position."
      />
    )
  }

  if (status === TripEvents.DriverTripRequest) {
    return (
      <TripOverviewCard
        title="New trip offer"
        description="A rider needs a ride. Review the route on the map, then accept or decline."
      >
        <div className="flex flex-col gap-3">
          <p className="text-xs font-mono text-muted-foreground truncate">Trip {trip.id}</p>
          <Button className="bg-emerald-700 hover:bg-emerald-800" onClick={onAcceptTrip}>
            Accept trip
          </Button>
          <Button variant="outline" onClick={onDeclineTrip}>
            Decline
          </Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Completed) {
    return (
      <TripOverviewCard
        title="Trip completed"
        description="Nice work — the rider will pay their fare. You're free for the next request."
      >
        <Button variant="outline" className="w-full" onClick={onReset}>
          Ready for next trip
        </Button>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.DriverArrived || status === TripEvents.OTPFailed) {
    return (
      <TripOverviewCard
        title="Enter rider OTP"
        description="Ask the rider for their 6-digit pickup code to start the trip."
      >
        <div className="flex flex-col gap-3">
          <input
            inputMode="numeric"
            maxLength={6}
            className="w-full rounded-md border px-3 py-3 text-center text-2xl tracking-[0.4em] font-mono"
            placeholder="••••••"
            value={otp}
            onChange={(e) => setOtp(e.target.value.replace(/\D/g, "").slice(0, 6))}
          />
          {otpError && <p className="text-sm text-red-600">{otpError}</p>}
          <Button
            className="bg-emerald-700 hover:bg-emerald-800"
            disabled={otp.length !== 6 || otpPending}
            onClick={() => onVerifyOTP?.(otp)}
          >
            {otpPending ? "Verifying…" : "Start trip"}
          </Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.TripStarted) {
    return (
      <TripOverviewCard
        title="Travelling to destination"
        description={
          simulating
            ? "Simulating your route — Complete unlocks at the dropoff."
            : atDestination
              ? "You've reached the destination."
              : "Drive to the dropoff to complete the ride."
        }
      >
        <div className="flex flex-col gap-3">
          <div className="h-2 w-full rounded bg-slate-200 overflow-hidden">
            <div
              className="h-full bg-emerald-600 transition-all"
              style={{ width: `${Math.round(travelProgress * 100)}%` }}
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Progress {Math.round(travelProgress * 100)}%
          </p>
          <Button
            className="bg-emerald-700 hover:bg-emerald-800"
            disabled={!atDestination}
            onClick={onCompleted}
          >
            <Flag className="size-4" />
            Complete trip
          </Button>
          {!atDestination && (
            <p className="text-xs text-amber-700">Complete is enabled once you reach the destination.</p>
          )}
        </div>
      </TripOverviewCard>
    )
  }

  const preStart =
    status === TripEvents.DriverTripAccept || status === TripEvents.DriverEnRoute

  if (preStart) {
    return (
      <TripOverviewCard
        title="Trip in progress"
        description={statusLabel[status!] ?? "Update your trip status as you drive."}
      >
        <div className="flex flex-col gap-2">
          <p className="text-xs font-mono text-muted-foreground truncate mb-2">Trip {trip.id}</p>

          <Button
            variant={status === TripEvents.DriverTripAccept ? "default" : "outline"}
            className={status === TripEvents.DriverTripAccept ? "bg-emerald-700 hover:bg-emerald-800" : ""}
            disabled={status !== TripEvents.DriverTripAccept}
            onClick={onEnRoute}
          >
            <Navigation className="size-4" />
            En route
          </Button>

          <Button
            variant={status === TripEvents.DriverEnRoute ? "default" : "outline"}
            className={status === TripEvents.DriverEnRoute ? "bg-emerald-700 hover:bg-emerald-800" : ""}
            disabled={status !== TripEvents.DriverEnRoute}
            onClick={onArrived}
          >
            <MapPin className="size-4" />
            Arrived
          </Button>

          {status && status !== TripEvents.DriverTripAccept && (
            <div className="flex items-center gap-2 text-sm text-emerald-700 mt-2">
              <CheckCircle2 className="size-4" />
              {statusLabel[status]}
            </div>
          )}
        </div>
      </TripOverviewCard>
    )
  }

  return null
}
