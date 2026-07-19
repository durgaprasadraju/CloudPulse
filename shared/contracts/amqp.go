package contracts

// AmqpMessage is the message structure for AMQP.
type AmqpMessage struct {
	OwnerID string `json:"ownerId"`
	Data    []byte `json:"data"`
}

// Routing keys - using consistent event/command patterns
const (
	// Trip events (trip.event.*)
	TripEventCreated             = "trip.event.created"
	TripEventDriverAssigned      = "trip.event.driver_assigned"
	TripEventNoDriversFound      = "trip.event.no_drivers_found"
	TripEventDriverNotInterested = "trip.event.driver_not_interested"
	TripEventDriverEnRoute       = "trip.event.driver_en_route"
	TripEventDriverArrived       = "trip.event.driver_arrived"
	TripEventStarted             = "trip.event.started"
	TripEventCompleted           = "trip.event.completed"
	TripEventCancelled           = "trip.event.cancelled"
	TripEventOTPIssued           = "trip.event.otp_issued"
	TripEventOTPFailed           = "trip.event.otp_failed"
	TripEventOTPVerified         = "trip.event.otp_verified"

	// Driver commands (driver.cmd.*)
	DriverCmdTripRequest = "driver.cmd.trip_request"
	DriverCmdTripAccept  = "driver.cmd.trip_accept"
	DriverCmdTripDecline = "driver.cmd.trip_decline"
	DriverCmdLocation    = "driver.cmd.location"
	DriverCmdRegister    = "driver.cmd.register"
	RiderCmdLocation     = "rider.cmd.location"
	TripCmdCancel        = "trip.cmd.cancel"
	TripCmdStatus        = "trip.cmd.status"
	TripCmdVerifyOTP     = "trip.cmd.verify_otp"

	// Payment events (payment.event.*)
	PaymentEventSessionCreated = "payment.event.session_created"
	PaymentEventSuccess        = "payment.event.success"
	PaymentEventFailed         = "payment.event.failed"
	PaymentEventCancelled      = "payment.event.cancelled"

	// Payment commands (payment.cmd.*)
	PaymentCmdCreateSession = "payment.cmd.create_session"
)
