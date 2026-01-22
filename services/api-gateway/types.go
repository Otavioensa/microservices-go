package main

import (
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"
)

type previewTripRequest struct {
	UserID      string           `json:"userID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

func (ptrq *previewTripRequest) ToProto() *pb.PreviewTripRequest {
	return &pb.PreviewTripRequest{
		UserID: ptrq.UserID,
		StartLocation: &pb.Coordinate{
			Latitude:  ptrq.Pickup.Latitude,
			Longitude: ptrq.Pickup.Longitude,
		},
		EndLocation: &pb.Coordinate{
			Latitude:  ptrq.Destination.Latitude,
			Longitude: ptrq.Destination.Longitude,
		},
	}
}
