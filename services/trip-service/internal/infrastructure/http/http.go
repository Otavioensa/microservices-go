package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/types"
)

type previewTripRequest struct {
	UserID      string           `json:"userID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

type HttpHandler struct {
	Service domain.TripService
}

func (hh *HttpHandler) HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	var requestBody previewTripRequest

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	route, err := hh.Service.GetRoute(ctx, &requestBody.Pickup, &requestBody.Destination)

	if err != nil {
		fmt.Println("Error getting route:", err)
	}

	defer r.Body.Close()

	writeJSON(w, http.StatusOK, route)

}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
