package types

import "time"

// PaymentStatus represents the current status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSuccess   PaymentStatus = "success"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"

	ProviderPhonePe = "phonepe"
	ProviderStripe  = "stripe"
	ProviderMock    = "mock"
)

// PaymentSession is the provider-agnostic checkout session returned to callers.
type PaymentSession struct {
	SessionID   string `json:"sessionID"`
	Provider    string `json:"provider"`
	CheckoutURL string `json:"checkoutURL,omitempty"`
}

// Payment represents a payment transaction
type Payment struct {
	ID         string        `json:"id"`
	TripID     string        `json:"trip_id"`
	UserID     string        `json:"user_id"`
	Amount     int64         `json:"amount"`   // Amount in paise / cents
	Currency   string        `json:"currency"` // e.g., "inr"
	Status     PaymentStatus `json:"status"`
	SessionID  string        `json:"session_id"`
	Provider   string        `json:"provider"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// PaymentIntent represents the intent to collect a payment
type PaymentIntent struct {
	ID          string    `json:"id"`
	TripID      string    `json:"trip_id"`
	UserID      string    `json:"user_id"`
	DriverID    string    `json:"driver_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	SessionID   string    `json:"session_id"`
	Provider    string    `json:"provider"`
	CheckoutURL string    `json:"checkout_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// PaymentConfig holds shared redirect URLs and optional Stripe keys.
type PaymentConfig struct {
	StripeSecretKey     string `json:"stripeSecretKey"`
	StripeWebhookSecret string `json:"stripeWebhookSecret"`
	Currency            string `json:"currency"`
	SuccessURL          string `json:"successURL"`
	CancelURL           string `json:"cancelURL"`
}

// PhonePeConfig holds PhonePe merchant credentials.
type PhonePeConfig struct {
	MerchantID  string
	SaltKey     string
	SaltIndex   string
	Env         string // UAT or PROD
	CallbackURL string
	SuccessURL  string
	CancelURL   string
}
