import { RouteFare, TripPreview, Driver } from "../types"
import { DriverList } from "./DriversList"
import { Card } from "./ui/card"
import { Button } from "./ui/button"
import { convertMetersToKilometers, convertSecondsToMinutes } from "../utils/math"
import { Skeleton } from "./ui/skeleton"
import { TripOverviewCard } from "./TripOverviewCard"
import { DriverCard } from "./DriverCard"
import { TripEvents, PaymentEventSessionCreatedData } from "../contracts"
import { MockPaymentModal } from "./MockPaymentModal"
import { TripOtpCard } from "./TripOtpCard"
import { PhonePePaymentButton } from "./PhonePePaymentButton"
import { StripePaymentButton } from "./StripePaymentButton"
import { TripFeedbackForm } from "./TripFeedbackForm"

interface TripOverviewProps {
  trip: TripPreview | null;
  status: TripEvents | null;
  assignedDriver?: Driver | null;
  paymentSession?: PaymentEventSessionCreatedData | null;
  pickupOTP?: string | null;
  pickupLabel?: string;
  dropoffLabel?: string;
  userID?: string;
  awaitingFeedback?: boolean;
  onPackageSelect: (carPackage: RouteFare) => void;
  onCancel: () => void;
  onPaymentDone?: () => void;
  onFeedbackDone?: () => void;
}

const canCancelBeforeStart = (status: TripEvents | null) =>
  status === TripEvents.Created ||
  status === TripEvents.DriverAssigned ||
  status === TripEvents.DriverEnRoute ||
  status === TripEvents.DriverArrived ||
  Boolean(status === null);

export const RiderTripOverview = ({
  trip,
  status,
  assignedDriver,
  paymentSession,
  pickupOTP,
  pickupLabel,
  dropoffLabel,
  userID,
  awaitingFeedback,
  onPackageSelect,
  onCancel,
  onPaymentDone,
  onFeedbackDone,
}: TripOverviewProps) => {
  if (!trip) {
    return (
      <TripOverviewCard
        title="Where to?"
        description="Tap the map to set your dropoff. Drag or long-press isn't required — one click chooses destination."
      >
        {(pickupLabel || dropoffLabel) && (
          <div className="text-sm text-gray-600 space-y-1 mt-2">
            {pickupLabel && <p><span className="font-medium">Pickup:</span> {pickupLabel}</p>}
            {dropoffLabel && <p><span className="font-medium">Dropoff:</span> {dropoffLabel}</p>}
          </div>
        )}
      </TripOverviewCard>
    )
  }

  if (awaitingFeedback && trip.tripID) {
    return (
      <TripOverviewCard
        title="How was your ride?"
        description="Your payment went through — leave a quick rating for your driver."
      >
        <TripFeedbackForm
          tripID={trip.tripID}
          userID={userID ?? ""}
          onSubmitted={() => {}}
          onSkip={() => onFeedbackDone?.()}
        />
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.PaymentSessionCreated && paymentSession) {
    const isLocalMock =
      paymentSession.sessionID?.startsWith("pp_test_local_") ||
      paymentSession.sessionID?.startsWith("cs_test_local_")
    const isPhonePe =
      paymentSession.provider === "phonepe" ||
      Boolean(paymentSession.checkoutURL) ||
      paymentSession.sessionID?.startsWith("pp_")
    return (
      <TripOverviewCard
        title="Trip complete — pay your fare"
        description="Review the fare and complete checkout"
      >
        <div className="flex flex-col gap-4">
          <DriverCard driver={assignedDriver} />
          {isLocalMock ? (
            <MockPaymentModal
              paymentSession={paymentSession}
              driverName={assignedDriver?.name}
              distanceMeters={trip.distance}
              durationSeconds={trip.duration}
              driverID={assignedDriver?.id}
              userID={userID}
              onPaid={() => onPaymentDone?.()}
            />
          ) : isPhonePe ? (
            <PhonePePaymentButton paymentSession={paymentSession} />
          ) : (
            <StripePaymentButton paymentSession={paymentSession} />
          )}
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.NoDriversFound) {
    return (
      <TripOverviewCard
        title="No drivers nearby"
        description="Try another package or wait a moment and request again."
      >
        <Button variant="outline" className="w-full" onClick={onCancel}>
          Back
        </Button>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Completed && !paymentSession) {
    return (
      <TripOverviewCard
        title="You arrived!"
        description="Preparing your receipt…"
      >
        <DriverCard driver={assignedDriver} />
        <Skeleton className="h-10 w-full mt-3" />
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Cancelled) {
    return (
      <TripOverviewCard
        title="Trip cancelled"
        description="Your trip was cancelled. You can request a new ride anytime."
      >
        <Button variant="outline" className="w-full" onClick={onCancel}>
          Book another ride
        </Button>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.TripStarted) {
    return (
      <TripOverviewCard
        title="On the way"
        description="Enjoy the ride — your driver is heading to the destination."
      >
        <DriverCard driver={assignedDriver} />
        {trip.duration != null && (
          <p className="text-sm text-gray-600 mt-2">
            ETA {convertSecondsToMinutes(trip.duration)} · {convertMetersToKilometers(trip.distance ?? 0)}
          </p>
        )}
      </TripOverviewCard>
    )
  }

  if (
    status === TripEvents.DriverAssigned ||
    status === TripEvents.DriverEnRoute ||
    status === TripEvents.DriverArrived
  ) {
    const title =
      status === TripEvents.DriverAssigned
        ? "Driver assigned"
        : status === TripEvents.DriverEnRoute
          ? "Driver is coming"
          : "Driver has arrived"
    const description =
      status === TripEvents.DriverArrived
        ? "Share the OTP below so your driver can start the trip."
        : "Share the OTP with your driver when they arrive."

    return (
      <TripOverviewCard title={title} description={description}>
        <div className="flex flex-col gap-3">
          {pickupOTP && <TripOtpCard otp={pickupOTP} driverName={assignedDriver?.name} />}
          <DriverCard driver={assignedDriver} />
          {canCancelBeforeStart(status) && (
            <Button variant="destructive" className="w-full" onClick={onCancel}>
              Cancel trip
            </Button>
          )}
        </div>
      </TripOverviewCard>
    )
  }

  if (status === TripEvents.Created || Boolean(trip.tripID)) {
    return (
      <TripOverviewCard
        title="Finding your driver"
        description="Matching you with a nearby driver…"
      >
        <div className="flex flex-col space-y-3 justify-center items-center mb-4">
          <Skeleton className="h-[125px] w-[250px] rounded-xl" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-[250px]" />
            <Skeleton className="h-4 w-[200px]" />
          </div>
        </div>

        <div className="flex flex-col items-center justify-center gap-2">
          {trip?.duration != null && (
            <h3 className="text-sm font-medium text-gray-700 mb-2">
              Trip ~{convertSecondsToMinutes(trip.duration)} ({convertMetersToKilometers(trip.distance ?? 0)})
            </h3>
          )}

          <Button variant="destructive" className="w-full" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </TripOverviewCard>
    )
  }

  if (trip.rideFares && trip.rideFares.length >= 0 && !trip.tripID) {
    return (
      <DriverList
        trip={trip}
        onPackageSelect={onPackageSelect}
        onCancel={onCancel}
      />
    )
  }

  return (
    <Card className="w-full md:max-w-[500px] z-[9999] flex-[0.3]">
      No trip ride fares, please refresh the page
    </Card>
  )
}
