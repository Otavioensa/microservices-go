package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
)

func handleTripPreviw(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: "Trip preview successful"})

	// Call trip service
}
