package repository

import (
	"context"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
)

type inMemRrpository struct {
	trips     map[string]*domain.TripModel
	rideFares map[string]*domain.RideFareModel
}

func NewInMemRepository() *inMemRrpository {
	return &inMemRrpository{
		trips:     make(map[string]*domain.TripModel),
		rideFares: make(map[string]*domain.RideFareModel),
	}
}

func (r *inMemRrpository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	r.trips[trip.ID.Hex()] = trip
	return trip, nil
}

func (r *inMemRrpository) SaveRideFares(ctx context.Context, fare *domain.RideFareModel) error {
	r.rideFares[fare.ID.Hex()] = fare
	return nil
}

func (r *inMemRrpository) GetRideFareByID(ctx context.Context, fareID string) (*domain.RideFareModel, error) {
	if fare, exists := r.rideFares[fareID]; exists {
		return fare, nil
	}
	return nil, fmt.Errorf("ride fare with ID %s not found", fareID)
}

func (r *inMemRrpository) GetTripByID(ctx context.Context, tripID string) (*domain.TripModel, error) {
	trip, ok := r.trips[tripID]
	if !ok {
		return nil, fmt.Errorf("trip with ID %s not found", tripID)
	}
	return trip, nil
}

func (r *inMemRrpository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	trip, ok := r.trips[tripID]
	if !ok {
		return fmt.Errorf("trip with ID %s not found", tripID)
	}
	trip.Status = status

	if driver != nil {
		trip.Driver = &pb.TripDriver{
			Id:             driver.Id,
			Name:           driver.Name,
			CarPlate:       driver.CarPlate,
			ProfilePicture: driver.ProfilePicture,
		}

	}
	return nil
}
