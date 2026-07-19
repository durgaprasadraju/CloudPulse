package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

var tracer = tracing.GetTracer("api-gateway")

func handleTripStart(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripStart")
	defer span.End()

	var reqBody startTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// Why we need to create a new client for each connection:
	// because if a service is down, we don't want to block the whole application
	// so we create a new client for each connection
	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Don't forget to close the client to avoid resource leaks!
	defer tripService.Close()

	trip, err := tripService.Client.CreateTrip(ctx, reqBody.toProto())
	if err != nil {
		log.Printf("Failed to start a trip: %v", err)
		http.Error(w, "Failed to start trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: trip}

	writeJSON(w, http.StatusCreated, response)
}

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripPreview")
	defer span.End()

	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// validation
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	// Why we need to create a new client for each connection:
	// because if a service is down, we don't want to block the whole application
	// so we create a new client for each connection
	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// Don't forget to close the client to avoid resource leaks!
	defer tripService.Close()

	tripPreview, err := tripService.Client.PreviewTrip(ctx, reqBody.toProto())
	if err != nil {
		log.Printf("Failed to preview a trip: %v", err)
		http.Error(w, "Failed to preview trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: tripPreview}

	writeJSON(w, http.StatusCreated, response)
}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	ctx, span := tracer.Start(r.Context(), "handleStripeWebhook")
	defer span.End()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	webhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")
	if webhookKey == "" {
		log.Printf("Webhook key is required")
		return
	}

	event, err := webhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		webhookKey,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	log.Printf("Received Stripe event: %v", event)

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		payload := messaging.PaymentStatusUpdateData{
			TripID:   session.Metadata["trip_id"],
			UserID:   session.Metadata["user_id"],
			DriverID: session.Metadata["driver_id"],
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshalling payload: %v", err)
			http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
			return
		}

		message := contracts.AmqpMessage{
			OwnerID: session.Metadata["user_id"],
			Data:    payloadBytes,
		}

		if err := rb.PublishMessage(
			ctx,
			contracts.PaymentEventSuccess,
			message,
		); err != nil {
			log.Printf("Error publishing payment event: %v", err)
			http.Error(w, "Failed to publish payment event", http.StatusInternalServerError)
			return
		}
	}
}

// handleMockPaymentSuccess marks a local mock checkout as paid (no Stripe webhook).
func handleMockPaymentSuccess(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	ctx := r.Context()
	var body struct {
		TripID   string `json:"tripID"`
		UserID   string `json:"userID"`
		DriverID string `json:"driverID"`
		SessionID string `json:"sessionID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if body.TripID == "" {
		http.Error(w, "tripID required", http.StatusBadRequest)
		return
	}
	if body.SessionID != "" && !isLocalMockSession(body.SessionID) {
		http.Error(w, "only local mock sessions allowed", http.StatusBadRequest)
		return
	}

	payloadBytes, err := json.Marshal(messaging.PaymentStatusUpdateData{
		TripID:   body.TripID,
		UserID:   body.UserID,
		DriverID: body.DriverID,
	})
	if err != nil {
		http.Error(w, "marshal failed", http.StatusInternalServerError)
		return
	}
	if err := rb.PublishMessage(ctx, contracts.PaymentEventSuccess, contracts.AmqpMessage{
		OwnerID: body.UserID,
		Data:    payloadBytes,
	}); err != nil {
		http.Error(w, "publish failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: map[string]string{"status": "ok"}})
}

func isLocalMockSession(sessionID string) bool {
	return strings.HasPrefix(sessionID, "cs_test_local_") || strings.HasPrefix(sessionID, "pp_test_local_")
}

// handlePhonePeWebhook accepts PhonePe server callbacks and marks the trip paid.
func handlePhonePeWebhook(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	ctx := r.Context()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	saltKey := env.GetString("PHONEPE_SALT_KEY", "")
	saltIndex := env.GetString("PHONEPE_SALT_INDEX", "1")
	xVerify := r.Header.Get("X-VERIFY")

	var envelope struct {
		Response string `json:"response"`
	}
	_ = json.Unmarshal(body, &envelope)
	b64 := envelope.Response
	if b64 == "" {
		// Some PhonePe payloads send the base64 blob as the raw body string.
		b64 = strings.TrimSpace(string(body))
	}

	if saltKey != "" && xVerify != "" && b64 != "" {
		expectedSum := sha256.Sum256([]byte(b64 + saltKey))
		expected := hex.EncodeToString(expectedSum[:]) + "###" + saltIndex
		if !strings.EqualFold(expected, xVerify) {
			http.Error(w, "invalid checksum", http.StatusUnauthorized)
			return
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Fall back to treating body as JSON payload directly.
		decoded = body
	}

	var callback struct {
		Code string `json:"code"`
		Data struct {
			MerchantTransactionID string `json:"merchantTransactionId"`
			State                 string `json:"state"`
			Amount                int64  `json:"amount"`
			MetaInfo              struct {
				UDF1 string `json:"udf1"`
				UDF2 string `json:"udf2"`
				UDF3 string `json:"udf3"`
			} `json:"metaInfo"`
		} `json:"data"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(decoded, &callback); err != nil {
		log.Printf("phonepe webhook decode: %v", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if !(callback.Success || callback.Code == "PAYMENT_SUCCESS" || strings.EqualFold(callback.Data.State, "COMPLETED")) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	tripID := callback.Data.MetaInfo.UDF1
	userID := callback.Data.MetaInfo.UDF2
	driverID := callback.Data.MetaInfo.UDF3
	if tripID == "" {
		http.Error(w, "trip id missing", http.StatusBadRequest)
		return
	}

	payloadBytes, err := json.Marshal(messaging.PaymentStatusUpdateData{
		TripID:   tripID,
		UserID:   userID,
		DriverID: driverID,
	})
	if err != nil {
		http.Error(w, "marshal failed", http.StatusInternalServerError)
		return
	}
	if err := rb.PublishMessage(ctx, contracts.PaymentEventSuccess, contracts.AmqpMessage{
		OwnerID: userID,
		Data:    payloadBytes,
	}); err != nil {
		http.Error(w, "publish failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
