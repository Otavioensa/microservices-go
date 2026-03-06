package events

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type driverConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewDriverConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *driverConsumer {
	return &driverConsumer{rabbitmq: rabbitmq, service: service}
}

func (dc *driverConsumer) Listen() error {
	return dc.rabbitmq.ConsumeMessages(messaging.DriverTripResponseQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		log.Printf("Received a reply from driver on the trip service with routing key %s", msg.RoutingKey)

		var driverEvent contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &driverEvent); err != nil {
			log.Println("Failed to unmarshal message: ", err)
			return err
		}

		var payload messaging.DriverTripResponseData

		if err := json.Unmarshal(driverEvent.Data, &payload); err != nil {
			log.Println("Failed to unmarshal payload: ", err)
			return err
		}

		log.Printf("Driver Trip Response payload: %+v", payload)

		switch msg.RoutingKey {
		case contracts.DriverCmdTripAccept:
			if err := dc.handleDriverTripAccept(context.Background(), payload); err != nil {
				log.Println("Failed to handle driver trip accept: ", err)
				return err
			}

		case contracts.DriverCmdTripDecline:
			log.Println("Declined")
			if err := dc.handleTripDeclined(ctx, payload.TripID, payload.RiderID); err != nil {
				log.Printf("Failed to handle the trip decline: %v", err)
				return err
			}
			return nil
		}

		log.Printf("unknown routing key/event %s", msg.RoutingKey)

		return nil
	})
}

func (dc *driverConsumer) handleDriverTripAccept(ctx context.Context, payload messaging.DriverTripResponseData) error {
	// validate that the trip exists
	trip, err := dc.service.GetTripByID(ctx, payload.TripID)
	if err != nil {
		log.Printf("Failed to get trip by ID: %v", err)
		return err
	}

	if trip == nil {
		log.Printf("Trip with ID %s not found", payload.TripID)
		return nil
	}

	// update the trip

	if err := dc.service.UpdateTrip(ctx, payload.TripID, "accepted", payload.Driver); err != nil {
		log.Printf("Failed to update trip: %v", err)
		return err
	}

	updatedTrip, err := dc.service.GetTripByID(ctx, payload.TripID)
	if err != nil {
		return err
	}

	mashalledTrip, err := json.Marshal(updatedTrip)
	if err != nil {
		log.Printf("Failed to marshal updated trip: %v", err)
		return err
	}

	// publish event to rabbitmq that the trip has been accepted by the driver/assigned to the rider
	if err := dc.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: updatedTrip.UserID,
		Data:    mashalledTrip,
	}); err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}

	return nil
}

func (dc *driverConsumer) handleTripDeclined(ctx context.Context, tripID, riderID string) error {
	// When a driver declines, we should try to find another driver

	trip, err := dc.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	newPayload := messaging.TripEventData{
		Trip: trip.ToProto(),
	}

	marshalledPayload, err := json.Marshal(newPayload)
	if err != nil {
		return err
	}

	if err := dc.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverNotInterested,
		contracts.AmqpMessage{
			OwnerID: riderID,
			Data:    marshalledPayload,
		},
	); err != nil {
		return err
	}

	return nil
}
