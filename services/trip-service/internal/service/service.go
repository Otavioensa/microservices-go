package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
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

func (ts *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OSRMAPIResponse, error) {
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

	var routeResponse tripTypes.OSRMAPIResponse

	if err := json.Unmarshal(body, &routeResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal route response: %w", err)
	}

	return &routeResponse, nil
}

func (ts *service) EstimatePackagesPriceWithRoute(route *tripTypes.OSRMAPIResponse) []*domain.RideFareModel {
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, fare := range baseFares {
		estimatedFares[i] = estimateFareRoute(fare, route)
	}

	return estimatedFares
}

func (ts *service) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string) ([]*domain.RideFareModel, error) {
	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, f := range rideFares {
		ID := primitive.NewObjectID()
		fare := &domain.RideFareModel{
			UserID:            userID,
			ID:                ID,
			TotalPriceInCents: f.TotalPriceInCents,
			PackageSlug:       f.PackageSlug,
		}

		if err := ts.repo.SaveRideFares(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save trip fare: %w", err)
		}

		fares[i] = fare
	}

	return fares, nil
}

func estimateFareRoute(fare *domain.RideFareModel, route *tripTypes.OSRMAPIResponse) *domain.RideFareModel {
	// distance, time and care price
	pricingConfig := tripTypes.DefaultPricingConfig()
	carPackagePrice := fare.TotalPriceInCents
	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	// distance fare
	distanceFare := distanceKm * pricingConfig.PricePerDistance

	// time fare
	timeFare := durationInMinutes * pricingConfig.PricePerMinute

	// total price
	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		TotalPriceInCents: totalPrice,
		PackageSlug:       fare.PackageSlug,
	}
}

func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 200,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 350,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 400,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 1000,
		},
	}
}
