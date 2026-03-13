package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	grpcclients "ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
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

func handleTripStart(w http.ResponseWriter, r *http.Request) {
	log.Println("Endpoint hit: trip/start")

	// get parameters and parse from request body
	var requestBody startTripRequest

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		log.Println("Error parsing JSON data:", err)
		http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// call rpc method
	grpcClient, err := grpcclients.NewTripServiceClient()

	if err != nil {
		log.Fatal("Could not connect to trip service gRPC:", err)
		http.Error(w, "Failed to connect to trip service", http.StatusBadRequest)
		return
	}

	defer grpcClient.Close()

	createTripResponse, err := grpcClient.Client.CreateTrip(r.Context(), requestBody.ToProto())

	if err != nil {
		log.Fatal("Error calling CreateTrip gRPC method: ", err)
		http.Error(w, "Failed to create trip", http.StatusBadRequest)
		return
	}

	// respond with created trip details
	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: createTripResponse})
}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request, rmq *messaging.RabbitMQ) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading request body:", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Println("Endpoint hit: webhook/stripe success")

	stripeWebhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")

	event, err := webhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		stripeWebhookKey,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		log.Println("Error verifying webhook signature:", err)
		http.Error(w, "Failed to construct Stripe event", http.StatusBadRequest)
		return
	}

	log.Printf("Received Stripe event %v", event)

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			log.Println("Error parsing webhook data:", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		payload := messaging.PaymentStatusUpdateData{
			TripID:   session.Metadata["trip_id"],
			UserID:   session.Metadata["user_id"],
			DriverID: session.Metadata["driver_id"],
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Println("Error marshaling payload:", err)
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			return
		}

		message := contracts.AmqpMessage{
			OwnerID: session.Metadata["user_id"],
			Data:    payloadBytes,
		}

		if err := rmq.PublishMessage(
			r.Context(),
			contracts.PaymentEventSuccess,
			message,
		); err != nil {
			log.Println("Error publishing message to RabbitMQ:", err)
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, contracts.APIResponse{Data: "Webhook received"})
}
