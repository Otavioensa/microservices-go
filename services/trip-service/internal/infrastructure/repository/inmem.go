package repository

import (
	"context"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
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
