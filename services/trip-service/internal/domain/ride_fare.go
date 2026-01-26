package domain

import (
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// models
type RideFareModel struct {
	ID                primitive.ObjectID
	UserID            string
	PackageSlug       string // ex: van, luxury, etc...
	TotalPriceInCents float64
}

func (rf *RideFareModel) ToProto() *pb.RideFare {
	return &pb.RideFare{
		Id:                rf.ID.Hex(),
		UserID:            rf.UserID,
		PackageSlug:       rf.PackageSlug,
		TotalPriceInCents: rf.TotalPriceInCents,
	}
}

func ToRidesFareProtoList(fare []*RideFareModel) []*pb.RideFare {
	var protoFares []*pb.RideFare
	for _, fare := range fare {
		protoFares = append(protoFares, fare.ToProto())
	}
	return protoFares
}
