package messaging

import (
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue        = "find_available_drivers"
	DriverCmdTripRequestQueue        = "driver_cmd_trip_request"
	DriverTripResponseQueue          = "driver_trip_response"
	NotifyDriverNoDriversFoundQueue  = "notify_driver_no_drivers_found"
	NotifyDriverAssignQueue          = "notify_driver_assign"
	NotifyTripLifecycleQueue         = "notify_trip_lifecycle"
	TripLifecycleUpdateQueue         = "trip_lifecycle_update"
	TripOTPVerifyQueue               = "trip_otp_verify"
	NotifyDriverControlQueue         = "notify_driver_control"
	DriverSimControlQueue            = "driver_sim_control"
	PaymentTripResponseQueue         = "payment_trip_response"
	NotifyPaymentSessionCreatedQueue = "notify_payment_session_created"
	NotifyPaymentSuccessQueue        = "payment_success"
	DeadLetterQueue                  = "dead_letter_queue"
)

type TripEventData struct {
	Trip *pb.Trip `json:"trip"`
}

type DriverTripResponseData struct {
	Driver  *pbd.Driver `json:"driver"`
	TripID  string      `json:"tripID"`
	RiderID string      `json:"riderID"`
}

// TripOTPIssuedData is delivered only to the rider who owns the trip.
type TripOTPIssuedData struct {
	TripID string `json:"tripID"`
	OTP    string `json:"otp"`
}

// TripVerifyOTPData is sent by the driver to start the trip after arrival.
type TripVerifyOTPData struct {
	TripID   string `json:"tripID"`
	OTP      string `json:"otp"`
	RiderID  string `json:"riderID"`
	DriverID string `json:"driverID"`
}

// TripOTPResultData is delivered to the assigned driver.
type TripOTPResultData struct {
	TripID  string `json:"tripID"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// TripCancelData is sent by the rider before the trip starts.
type TripCancelData struct {
	TripID string `json:"tripID"`
	UserID string `json:"userID"`
}

type PaymentEventSessionCreatedData struct {
	TripID      string  `json:"tripID"`
	SessionID   string  `json:"sessionID"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Provider    string  `json:"provider,omitempty"`
	CheckoutURL string  `json:"checkoutURL,omitempty"`
}

type PaymentTripResponseData struct {
	TripID   string  `json:"tripID"`
	UserID   string  `json:"userID"`
	DriverID string  `json:"driverID"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentStatusUpdateData struct {
	TripID   string `json:"tripID"`
	UserID   string `json:"userID"`
	DriverID string `json:"driverID"`
}
