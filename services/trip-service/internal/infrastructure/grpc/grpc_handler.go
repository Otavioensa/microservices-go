package grpc

import (
	"context"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/types"

	pb "ride-sharing/shared/proto/trip"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer
	service domain.TripService
}

func NewgRPCHandler(server *grpc.Server, service domain.TripService) *gRPCHandler {
	handler := &gRPCHandler{
		service: service,
	}
	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (grpcHandler *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	fmt.Println("Received PreviewTrip request")
	startLocation := &types.Coordinate{
		Latitude:  req.StartLocation.Latitude,
		Longitude: req.StartLocation.Longitude,
	}

	endLocation := &types.Coordinate{
		Latitude:  req.EndLocation.Latitude,
		Longitude: req.EndLocation.Longitude,
	}

	route, err := grpcHandler.service.GetRoute(ctx, startLocation, endLocation)

	if err != nil {
		fmt.Println("Error getting route:", err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}
	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: []*pb.RideFare{},
	}, nil
}
