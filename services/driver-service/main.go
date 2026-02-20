package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"

	grpcServer "google.golang.org/grpc"
)

var (
	grpcAddr = env.GetString("GRPC_ADDR", ":9092")
)

func main() {
	log.Printf("Starting Driver service at %s", grpcAddr)

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

	service := NewService()

	// rabbitmq conntection
	rabbitmqUri := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	conn, err := messaging.NewRabbitMQ(rabbitmqUri)

	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}

	defer conn.Close()

	// next, create a new gRPC server instance
	grpcserver := grpcServer.NewServer()

	// then register the Driver service gRPC handler to the server
	NewGRPCHandler(grpcserver, service)

	consumer := NewTripConsumer(conn)

	go func() {
		// finally, start serving incoming connections
		if err := grpcserver.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
			cancel()
		}
	}()

	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("failed to listen to the message: %v", err)
		}
	}()

	// wait for the shutdown signal triggered by the context cancellation (cancel function called)
	<-ctx.Done()
	log.Println("Shutting down the Driver service...")
	// gracefully stop the gRPC server
	grpcserver.GracefulStop()
}
