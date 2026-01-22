package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	grpcclients "ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	var requestBody previewTripRequest

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
		return
	}

	if requestBody.UserID == "" {
		http.Error(w, "Missing userID in request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Println("Endpoint hit: trip/preview success")

	// there are tradeoffs to consider when stablishing a connection per request vs connecting it once
	// when conneting it once, our whole service might get impacted if the trip service goes down
	// while connecting per request adds some overhead to each request
	// if we expect a high volume of requests, we might consider using a connection pool or keep-alive connections
	// and monitor the trip service health to re-establish connections when needed
	tripService, err := grpcclients.NewTripServiceClient()

	if err != nil {
		log.Fatal("Could not connect to trip service gRPC:", err)
		http.Error(w, "Failed to connect to trip service", http.StatusBadRequest)
	}

	defer tripService.Close()

	previewTripResponse, err := tripService.Client.PreviewTrip(r.Context(), requestBody.ToProto())

	if err != nil {
		fmt.Printf("Error calling PreviewTrip gRPC method: %v\n", err)
		http.Error(w, "Failed to obtain trip preview", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: previewTripResponse})
}
