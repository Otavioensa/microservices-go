package main

import (
	"context"
	"log"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
)

type grpcHandler struct {
	pb.UnimplementedDriverServiceServer
	service *Service
}

func NewGRPCHandler(server *grpc.Server, service *Service) *grpcHandler {
	handler := &grpcHandler{
		service: service,
	}

	pb.RegisterDriverServiceServer(server, handler)
	return handler
}

func (handler *grpcHandler) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	log.Println("RegisterDriver gRPC method called with DriverID:", req.GetDriverID(), "PackageSlug:", req.GetPackageSlug())
	driver, err := handler.service.RegisterDriver(req.GetDriverID(), req.GetPackageSlug())

	if err != nil {
		log.Fatal("Error registering driver: ", err)
		return nil, err
	}

	return &pb.RegisterDriverResponse{Driver: driver}, nil
}

func (handler *grpcHandler) UnregisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	handler.service.UnregisterDriver(req.GetDriverID())
	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id: req.GetDriverID(),
		},
	}, nil
}
