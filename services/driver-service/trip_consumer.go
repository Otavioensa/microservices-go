package main

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  *Service
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripConsumer {
	return &tripConsumer{rabbitmq: rabbitmq, service: service}
}

func (tc *tripConsumer) Listen() error {
	return tc.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		log.Printf("Driver received a message")

		var tripEvent contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
			log.Println("Failed to unmarshal message: ", err)
			return err
		}

		var payload messaging.TripEventData

		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			log.Println("Failed to unmarshal payload: ", err)
			return err
		}

		log.Printf("Trip Event payload: %+v", payload)

		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return tc.findAndNotifyDrivers(context.Background(), payload)
		}

		log.Printf("unknown routing key/event %s", msg.RoutingKey)

		return nil
	})
}

func (tc *tripConsumer) findAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	suitableDriverIds := tc.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug)

	log.Printf("Found suitable drivers %v", suitableDriverIds)

	if len(suitableDriverIds) == 0 {
		log.Printf("No suitable drivers found for trip %s", payload.Trip.Id)

		// notify rider that no drivers are available
		if err := tc.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("Failed to publish no drivers found message: %v", err)
			return err
		}

		return nil
	}
	suitableDriverId := suitableDriverIds[0]

	marshaledData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal trip event data: %v", err)
		return err
	}

	// notify the suitable driver about a potential trip
	if err := tc.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverId,
		Data:    marshaledData,
	}); err != nil {
		log.Printf("Failed to publish driver assigned message: %v", err)
		return err
	}
	return nil
}
