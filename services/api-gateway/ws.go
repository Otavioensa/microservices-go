package main

import (
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/util"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket connection
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	defer connection.Close()

	userID := r.URL.Query().Get("userID")

	if userID == "" {
		log.Printf("Missing userID in query parameters")
		return
	}

	log.Printf("Rider connected: %s", userID)

	for {
		_, message, err := connection.ReadMessage()

		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Printf("Received message from rider %s: %s", userID, message)
	}

}

// handleDriversWebSocket handles WebSocket connections for drivers.
// It upgrades the HTTP connection to a WebSocket, registers the driver,
// and listens for incoming messages.
func handleDriversWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	defer connection.Close()

	userID := r.URL.Query().Get("userID")

	if userID == "" {
		log.Printf("Missing userID in query parameters")
		return
	}

	log.Printf("Driver connected: %s", userID)

	packageSlug := r.URL.Query().Get("packageSlug")

	if packageSlug == "" {
		log.Printf("Missing packageSlug in query parameters")
		return
	}

	type Driver struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		ProfilePictureURL string `json:"profilePicture"`
		CarPlate          string `json:"carPlate"`
		PackageSlug       string `json:"packageSlug"`
	}

	msg := contracts.WSMessage{
		Type: "driver.cmd.register",
		Data: Driver{
			ID:                userID,
			Name:              "John Doe",
			ProfilePictureURL: util.GetRandomAvatar(1),
			CarPlate:          "XYZ-1234",
			PackageSlug:       packageSlug,
		},
	}

	if err := connection.WriteJSON(msg); err != nil {
		log.Printf("Error sending registration message: %v", err)
		return
	}

	for {
		_, message, err := connection.ReadMessage()

		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Printf("Received message from driver %s: %s", userID, message)
	}
}
