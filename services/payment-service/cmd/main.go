package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"ride-sharing/services/payment-service/internal/domain"
	"ride-sharing/services/payment-service/internal/events"
	"ride-sharing/services/payment-service/internal/infrastructure/phonepe"
	"ride-sharing/services/payment-service/internal/infrastructure/stripe"
	"ride-sharing/services/payment-service/internal/service"
	"ride-sharing/services/payment-service/pkg/types"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
)

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "payment-service",
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

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	appURL := env.GetString("APP_URL", "http://localhost:3000")
	successURL := env.GetString("PHONEPE_SUCCESS_URL", env.GetString("STRIPE_SUCCESS_URL", appURL+"?payment=success"))
	cancelURL := env.GetString("PHONEPE_CANCEL_URL", env.GetString("STRIPE_CANCEL_URL", appURL+"?payment=cancel"))
	provider := strings.ToLower(env.GetString("PAYMENT_PROVIDER", "phonepe"))

	var paymentProcessor domain.PaymentProcessor

	switch provider {
	case "stripe":
		stripeCfg := &types.PaymentConfig{
			StripeSecretKey: env.GetString("STRIPE_SECRET_KEY", ""),
			SuccessURL:      successURL,
			CancelURL:       cancelURL,
		}
		if stripe.IsLocalStripeKey(stripeCfg.StripeSecretKey) {
			paymentProcessor = stripe.NewLocalClient(stripeCfg)
		} else {
			paymentProcessor = stripe.NewStripeClient(stripeCfg)
		}
	default:
		phoneCfg := &types.PhonePeConfig{
			MerchantID:  env.GetString("PHONEPE_MERCHANT_ID", ""),
			SaltKey:     env.GetString("PHONEPE_SALT_KEY", ""),
			SaltIndex:   env.GetString("PHONEPE_SALT_INDEX", "1"),
			Env:         env.GetString("PHONEPE_ENV", "UAT"),
			CallbackURL: env.GetString("PHONEPE_CALLBACK_URL", "http://localhost:8080/webhook/phonepe"),
			SuccessURL:  successURL,
			CancelURL:   cancelURL,
		}
		if phonepe.IsConfigured(phoneCfg) {
			paymentProcessor = phonepe.NewClient(phoneCfg)
		} else {
			paymentProcessor = phonepe.NewLocalClient(phoneCfg)
		}
	}

	svc := service.NewPaymentService(paymentProcessor)

	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	tripConsumer := events.NewTripConsumer(rabbitmq, svc)
	go tripConsumer.Listen()

	<-ctx.Done()
	log.Println("Shutting down payment service...")
}
