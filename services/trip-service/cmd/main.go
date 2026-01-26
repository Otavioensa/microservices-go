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
	log.Printf("Starting Trip service at %s", grpcAddr)
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

	// first step to start gRPC server by listening on the specified address via TCP
	listener, err := net.Listen("tcp", grpcAddr)

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// next, create a new gRPC server instance
	grpcserver := grpcServer.NewServer()

	// then register the Trip service gRPC handler to the server
	grpc.NewgRPCHandler(grpcserver, svc)

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
