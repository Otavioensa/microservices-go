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

	// estimate ride fare prices based on the route
	estimatedFares := grpcHandler.service.EstimatePackagesPriceWithRoute(route)

	// store the ride fare estimates for the user
	fares, err := grpcHandler.service.GenerateTripFares(ctx, estimatedFares, req.GetUserID())

	if err != nil {
		fmt.Println("Error generating trip fares:", err)
		return nil, status.Errorf(codes.Internal, "failed to generate trip fares: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRidesFareProtoList(fares),
	}, nil
}

func (grpcHandler *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fmt.Println("Received CreateTrip request")

	rideFare := &domain.RideFareModel{
		UserID: req.UserID,
	}

	trip, err := grpcHandler.service.CreateTrip(ctx, rideFare)
	if err != nil {
		fmt.Println("Error creating trip:", err)
		return nil, status.Errorf(codes.Internal, "failed to create trip: %v", err)
	}

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}
