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
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ) *tripConsumer {
	return &tripConsumer{rabbitmq: rabbitmq}
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

		return nil
	})
}
