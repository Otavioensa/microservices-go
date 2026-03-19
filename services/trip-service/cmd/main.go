package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"syscall"

	grpcServer "google.golang.org/grpc"
)

var (
	grpcAddr = env.GetString("GRPC_ADDR", ":9093")
)

func main() {
	log.Printf("Starting Trip service at %s", grpcAddr)
	inmemRepo := repository.NewInMemRepository()
	svc := service.NewService(inmemRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize tracing
	tracingCfg := tracing.Config{
		ServiceName:    "trip-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	shtdwn, err := tracing.InitTracer(tracingCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}

	defer shtdwn(ctx)

	go func() {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
		// channel will block until any of the signals is received
		<-signalChannel
		cancel()
	}()

	// first step to start gRPC server by listening on the specified address via TCP
	listener, err := net.Listen("tcp", grpcAddr)

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// rabbitmq conntection
	rabbitmqUri := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rmq, err := messaging.NewRabbitMQ(rabbitmqUri)

	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}

	defer rmq.Close()

	log.Println("Connected to RabbitMQ")

	publisher := events.NewTripEventPublisher(rmq)

	driverConsumer := events.NewDriverConsumer(rmq, svc)
	go driverConsumer.Listen()

	// next, create a new gRPC server instance
	grpcserver := grpcServer.NewServer(tracing.WithTracingInterceptors()...)

	// then register the Trip service gRPC handler to the server
	grpc.NewgRPCHandler(grpcserver, svc, publisher)

	paymentConsumer := events.NewPaymentConsumer(rmq, svc)
	go paymentConsumer.Listen()

	go func() {
		// finally, start serving incoming connections
		if err := grpcserver.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal triggered by the context cancellation (cancel function called)
	<-ctx.Done()
	log.Println("Shutting down the Trip service...")
	// gracefully stop the gRPC server
	grpcserver.GracefulStop()
}
