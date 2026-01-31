package grpcclients

import (
	"log"
	pb "ride-sharing/shared/proto/driver"

	"ride-sharing/shared/env"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type driverServiceClient struct {
	Client pb.DriverServiceClient
	conn   *grpc.ClientConn
}

func (tsc *driverServiceClient) Close() {
	if tsc.conn != nil {
		if err := tsc.conn.Close(); err != nil {
			log.Println("Error closing driver service gRPC connection:", err)
			return
		}
	}
}

func NewDriverServiceClient() (*driverServiceClient, error) {
	driverServiceURL := env.GetString("DRIVER_SERVICE_URL", "driver-service:9092")

	// create gRPC connection with the driver service
	// using insecure credentials for simplicity; TODO: in production, use TLS
	conn, err := grpc.NewClient(driverServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	// instantiate DRIVER service client
	client := pb.NewDriverServiceClient(conn)

	return &driverServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}
