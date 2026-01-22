package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/env"
	"syscall"

	grpcServer "google.golang.org/grpc"
)

var (
	grpcAddr = env.GetString("GRPC_ADDR", ":9093")
)

func main() {
	log.Println("Starting Trip service at %s", grpcAddr)
	inmemRepo := repository.NewInMemRepository()
	svc := service.NewService(inmemRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
		// channel will block until any of the signals is received
		<-signalChannel
		cancel()
	}()

	lis, err := net.Listen("tcp", grpcAddr)

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcserver := grpcServer.NewServer()

	grpc.NewgRPCHandler(grpcserver, svc)

	log.Printf("Trip service is running on port %s", lis.Addr().String())

	go func() {
		if err := grpcserver.Serve(lis); err != nil {
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
