package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/httpapi"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/auth"
	"ride-sharing/shared/db"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9093"

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "trip-service",
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

	mongoClient, err := db.NewMongoClient(ctx, db.NewMongoDefaultConfig())
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB, err: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	mongoDb := db.GetDatabase(mongoClient, db.NewMongoDefaultConfig())
	if err := repository.EnsureReviewIndexes(ctx, mongoDb); err != nil {
		log.Printf("warning: review indexes: %v", err)
	}

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	mongoDBRepo := repository.NewMongoRepository(mongoDb)

	var tripSvc domain.TripService
	if dbURL := env.GetString("DATABASE_URL", ""); dbURL != "" {
		store, err := auth.NewAccountStore(dbURL)
		if err != nil {
			log.Printf("warning: account store unavailable (bonus points disabled): %v", err)
			tripSvc = service.NewService(mongoDBRepo)
		} else {
			defer store.Close()
			tripSvc = service.NewServiceWithBonus(mongoDBRepo, &service.AccountBonusAdapter{Store: store})
			log.Println("Connected to Postgres for driver bonus points")
		}
	} else {
		tripSvc = service.NewService(mongoDBRepo)
	}

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

	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	publisher := events.NewTripEventPublisher(rabbitmq)

	driverConsumer := events.NewDriverConsumer(rabbitmq, tripSvc)
	go driverConsumer.Listen()

	paymentConsumer := events.NewPaymentConsumer(rabbitmq, tripSvc)
	go paymentConsumer.Listen()

	lifecycleConsumer := events.NewLifecycleConsumer(rabbitmq, tripSvc)
	go lifecycleConsumer.Listen()

	otpConsumer := events.NewOTPConsumer(rabbitmq, tripSvc)
	go otpConsumer.Listen()

	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	grpc.NewGRPCHandler(grpcServer, tripSvc, publisher)

	log.Printf("Starting gRPC server Trip service on port %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	httpAddr := env.GetString("HTTP_ADDR", ":8086")
	mux := http.NewServeMux()
	httpapi.NewHandler(tripSvc).Mount(mux)
	httpServer := &http.Server{Addr: httpAddr, Handler: mux}
	go func() {
		log.Printf("Starting trip-service HTTP on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("trip HTTP error: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down the server...")
	_ = httpServer.Shutdown(context.Background())
	grpcServer.GracefulStop()
}
