package domain

import (
	"context"
	"ride-sharing/shared/types"

	tripTypes "ride-sharing/services/trip-service/pkg/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// model and its repository and service interfaces

type TripModel struct {
	ID       primitive.ObjectID
	UserID   string
	Status   string
	RideFare *RideFareModel
}

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)
	SaveRideFares(ctx context.Context, fare *RideFareModel) error
}

type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)
	GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OSRMAPIResponse, error)
	EstimatePackagesPriceWithRoute(route *tripTypes.OSRMAPIResponse) []*RideFareModel
	GenerateTripFares(ctx context.Context, fares []*RideFareModel, userID string) ([]*RideFareModel, error)
}
