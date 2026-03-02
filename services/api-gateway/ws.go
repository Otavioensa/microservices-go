package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	grpcclients "ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"
)

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rmq *messaging.RabbitMQ) {
	// Upgrade initial GET request to a websocket connection
	connection, err := connManager.Upgrade(w, r)
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

	connManager.Add(userID, connection)
	defer connManager.Remove(userID)

	// Initialize queue consumer
	queues := []string{
		messaging.NotifyRiderNoDriversFoundQueue,
		messaging.NotifyRiderDriverAssignedQueue,
	}

	for _, queue := range queues {
		consumer := messaging.NewQueueConsumer(rmq, connManager, queue)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue %s: %v", queue, err)
		}
	}

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
func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rmq *messaging.RabbitMQ) {
	// Upgrade initial GET request to a websocket
	connection, err := connManager.Upgrade(w, r)
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

	connManager.Add(userID, connection)

	packageSlug := r.URL.Query().Get("packageSlug")

	if packageSlug == "" {
		log.Printf("Missing packageSlug in query parameters")
		return
	}

	driverParams := &pb.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	}

	driverService, err := grpcclients.NewDriverServiceClient()

	if err != nil {
		log.Fatal("Could not connect to driver service gRPC:", err)
		http.Error(w, "Failed to connect to driver service", http.StatusBadRequest)
	}

	defer driverService.Close()

	// Closing connections
	defer func() {
		connManager.Remove(userID)

		driverService.Client.UnregisterDriver(r.Context(), &pb.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})

		driverService.Close()

		log.Println("Driver unregistered: ", userID)
	}()

	driverResponse, err := driverService.Client.RegisterDriver(r.Context(), driverParams)

	if err != nil {
		log.Fatal("Error calling RegisterDriver gRPC method:", err)
		http.Error(w, "Failed to register driver", http.StatusBadRequest)
	}

	msg := contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverResponse.Driver,
	}

	if err := connManager.SendMessage(userID, msg); err != nil {
		log.Printf("Error sending registration message: %v", err)
		return
	}

	// Initialize queue consumer
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	for _, queue := range queues {
		consumer := messaging.NewQueueConsumer(rmq, connManager, queue)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue %s: %v", queue, err)
		}
	}

	for {
		_, message, err := connection.ReadMessage()

		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		type driverMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		var driverMsg driverMessage

		if err := json.Unmarshal(message, &driverMsg); err != nil {
			log.Printf("Failed to unmarshal driver message: %v", err)
			continue
		}

		log.Printf("Received message from driver %s: %s", userID, message)

		// handle different types of messages from the driver if needed
		switch driverMsg.Type {
		case contracts.DriverCmdLocation:
			// handle driver location update in the future if needed
			continue
		case contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline:
			// forward the message to rabbitmq so it can be processed
			fmt.Println("sending driver data ", driverMsg.Data)
			if err := rmq.PublishMessage(r.Context(), driverMsg.Type, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    driverMsg.Data,
			}); err != nil {
				log.Printf("Failed to publish driver command message: %v", err)
			}
		default:
			log.Printf("Unknown message type received from driver %s: %s", userID, driverMsg.Type)
		}
	}
}
