package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"ride-sharing/shared/tracking"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9092"

func main() {
	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
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

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	svc := NewService()

	// PostgreSQL (Terraform RDS / local compose) — persistent driver registry
	if databaseURL := env.GetString("DATABASE_URL", ""); databaseURL != "" {
		store, err := NewDriverStore(databaseURL)
		if err != nil {
			log.Printf("PostgreSQL unavailable, using in-memory drivers only: %v", err)
		} else {
			defer store.Close()
			svc.AttachStore(store)
			log.Println("Connected to PostgreSQL for driver persistence")
		}
	}

	// Redis (Terraform ElastiCache / local compose) — real-time location tracking
	if redisURL := env.GetString("REDIS_URL", ""); redisURL != "" {
		locations, err := tracking.NewLocationStore(redisURL)
		if err != nil {
			log.Printf("Redis unavailable, location tracking disabled: %v", err)
		} else {
			defer locations.Close()
			svc.AttachLocationStore(locations)
			log.Println("Connected to Redis for driver location tracking")
		}
	}

	autoAccept := env.GetBool("LOCAL_AUTO_ACCEPT", false)
	if env.GetBool("LOCAL_SEED_DRIVERS", false) {
		svc.SeedLocalDrivers([]string{"sedan", "suv", "van", "luxury"})
		log.Println("Seeded local drivers for sedan/suv/van/luxury")
	}

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// Starting the gRPC server
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	NewGrpcHandler(grpcServer, svc)

	consumer := NewTripConsumer(rabbitmq, svc, autoAccept)
	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to the message: %v", err)
		}
	}()

	log.Printf("Starting gRPC server Driver service on port %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	<-ctx.Done()
	log.Println("Shutting down the server...")
	grpcServer.GracefulStop()
}
