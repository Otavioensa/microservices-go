package domain

import (
	"context"
	"ride-sharing/shared/types"

	tripTypes "ride-sharing/services/trip-service/pkg/types"

	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// model and its repository and service interfaces

type TripModel struct {
	ID       primitive.ObjectID
	UserID   string
	Status   string
	RideFare *RideFareModel
	Driver   *pb.TripDriver
}

func (t *TripModel) ToProto() *pb.Trip {
	return &pb.Trip{
		Id:           t.ID.Hex(),
		UserID:       t.UserID,
		SelectedFare: t.RideFare.ToProto(),
		Status:       t.Status,
		Driver:       t.Driver,
		Route:        t.RideFare.Route.ToProto(),
	}
}

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)
	SaveRideFares(ctx context.Context, fare *RideFareModel) error
	GetRideFareByID(ctx context.Context, fareID string) (*RideFareModel, error)
	GetTripByID(ctx context.Context, tripID string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error
}

type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)
	GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OSRMAPIResponse, error)
	EstimatePackagesPriceWithRoute(route *tripTypes.OSRMAPIResponse) []*RideFareModel
	GenerateTripFares(
		ctx context.Context,
		fares []*RideFareModel,
		userID string,
		route *tripTypes.OSRMAPIResponse,
	) ([]*RideFareModel, error)
	GetAndValidateFare(ctx context.Context, fareID, userID string) (*RideFareModel, error)
	GetTripByID(ctx context.Context, tripID string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error
}
