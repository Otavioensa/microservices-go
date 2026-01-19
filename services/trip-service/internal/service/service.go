package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type service struct {
	repo domain.TripRepository
}

func NewService(repo domain.TripRepository) *service {
	return &service{repo: repo}
}

func (ts *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	trip := domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID, // need to get from auth context or elsewhere
		Status:   "pending",
		RideFare: fare,
	}

	created, err := ts.repo.CreateTrip(ctx, &trip)

	return created, err
}

func (ts *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*types.OSRMAPIResponse, error) {
	url := fmt.Sprintf("http://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson", pickup.Longitude, pickup.Latitude, destination.Longitude, destination.Latitude)

	response, err := http.Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var routeResponse types.OSRMAPIResponse

	if err := json.Unmarshal(body, &routeResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal route response: %w", err)
	}

	return &routeResponse, nil
}
