package grpcclients

import (
	"log"
	pb "ride-sharing/shared/proto/trip"

	"ride-sharing/shared/env"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type tripServiceClient struct {
	Client pb.TripServiceClient
	conn   *grpc.ClientConn
}

func (tsc *tripServiceClient) Close() {
	if tsc.conn != nil {
		if err := tsc.conn.Close(); err != nil {
			log.Println("Error closing trip service gRPC connection:", err)
			return
		}
	}
}

func NewTripServiceClient() (*tripServiceClient, error) {
	tripServiceURL := env.GetString("TRIP_SERVICE_URL", "trip-service:9093")

	// create gRPC connection with the trip service
	// using insecure credentials for simplicity; in production, use TLS
	conn, err := grpc.NewClient(tripServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	// instantiate trip service client
	client := pb.NewTripServiceClient(conn)

	return &tripServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}
