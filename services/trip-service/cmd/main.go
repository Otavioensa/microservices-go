package main

import (
	"context"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"time"
)

func main() {
	ctx := context.Background()
	inmemRepo := repository.NewInMemRepository()
	svc := service.NewService(inmemRepo)

	fare := &domain.RideFareModel{
		UserID: "12345",
	}

	trip, err := svc.CreateTrip(ctx, fare)
	if err != nil {
		fmt.Println("Error creating trip:", err)
	}

	fmt.Println("Trip created successfully:", trip)

	// temporary keeps the program alive
	for {
		time.Sleep(time.Second)
	}
}
