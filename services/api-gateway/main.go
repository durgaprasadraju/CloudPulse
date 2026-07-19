package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"ride-sharing/shared/tracking"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
)

func main() {
	log.Println("Starting API Gateway")

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)

	mux := http.NewServeMux()

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// Redis (Terraform ElastiCache / local compose) — real-time driver tracking
	var locations *tracking.LocationStore
	if redisURL := env.GetString("REDIS_URL", ""); redisURL != "" {
		locations, err = tracking.NewLocationStore(redisURL)
		if err != nil {
			log.Printf("Redis unavailable, driver location tracking disabled: %v", err)
			locations = nil
		} else {
			defer locations.Close()
			log.Println("Connected to Redis for driver location tracking")
		}
	}

	// Liveness endpoint for ALB target-group health checks
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.Handle("POST /trip/preview", tracing.WrapHandlerFunc(enableCORS(handleTripPreview), "/trip/preview"))
	mux.Handle("POST /trip/start", tracing.WrapHandlerFunc(enableCORS(handleTripStart), "/trip/start"))
	mux.Handle("POST /drivers/register", tracing.WrapHandlerFunc(enableCORS(handleDriverRegister), "/drivers/register"))
	mux.Handle("OPTIONS /drivers/register", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/drivers/register"))
	mux.Handle("POST /drivers/login", tracing.WrapHandlerFunc(enableCORS(handleDriverLogin), "/drivers/login"))
	mux.Handle("OPTIONS /drivers/login", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/drivers/login"))
	mux.Handle("GET /drivers/me", tracing.WrapHandlerFunc(enableCORS(handleDriverMe), "/drivers/me"))
	mux.Handle("OPTIONS /drivers/me", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/drivers/me"))
	mux.Handle("GET /drivers/me/dashboard", tracing.WrapHandlerFunc(enableCORS(handleDriverDashboard), "/drivers/me/dashboard"))
	mux.Handle("OPTIONS /drivers/me/dashboard", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/drivers/me/dashboard"))
	mux.Handle("GET /drivers/me/trips", tracing.WrapHandlerFunc(enableCORS(handleDriverTrips), "/drivers/me/trips"))
	mux.Handle("OPTIONS /drivers/me/trips", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/drivers/me/trips"))
	mux.Handle("POST /trips/{id}/review", tracing.WrapHandlerFunc(enableCORS(handleTripReview), "/trips/{id}/review"))
	mux.Handle("OPTIONS /trips/{id}/review", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/trips/{id}/review"))
	mux.Handle("OPTIONS /trip/preview", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/trip/preview"))
	mux.Handle("OPTIONS /trip/start", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/trip/start"))
	mux.Handle("POST /payment/mock-success", tracing.WrapHandlerFunc(enableCORS(func(w http.ResponseWriter, r *http.Request) {
		handleMockPaymentSuccess(w, r, rabbitmq)
	}), "/payment/mock-success"))
	mux.Handle("OPTIONS /payment/mock-success", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/payment/mock-success"))
	mux.Handle("/ws/drivers", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq, locations)
	}, "/ws/drivers"))
	mux.Handle("/ws/riders", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq, locations)
	}, "/ws/riders"))
	mux.Handle("/webhook/stripe", tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleStripeWebhook(w, r, rabbitmq)
	}, "/webhook/stripe"))
	mux.Handle("POST /webhook/phonepe", tracing.WrapHandlerFunc(enableCORS(func(w http.ResponseWriter, r *http.Request) {
		handlePhonePeWebhook(w, r, rabbitmq)
	}), "/webhook/phonepe"))
	mux.Handle("OPTIONS /webhook/phonepe", tracing.WrapHandlerFunc(enableCORS(func(http.ResponseWriter, *http.Request) {}), "/webhook/phonepe"))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server listening on %s", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error starting the server: %v", err)

	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Could not stop the server gracefully: %v", err)
			server.Close()
		}
	}
}
