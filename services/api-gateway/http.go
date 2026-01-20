package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	var resData any

	jsonBody, _ := json.Marshal(requestBody)
	reader := bytes.NewReader(jsonBody)

	// Call trip service
	res, err := http.Post("http://trip-service:8083/preview", "application/json", reader)

	if err != nil {
		http.Error(w, "Failed to connect to trip service", http.StatusBadRequest)
		return
	}

	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&resData); err != nil {
		http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
		return
	}

	fmt.Println(resData)

	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: resData})
}
