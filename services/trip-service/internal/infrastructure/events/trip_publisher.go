package events

import (
	"context"
	"ride-sharing/shared/messaging"
)

type TripEventPublisher struct {
	rabbitmq *messaging.RabbitMQ
}

func NewTripEventPublisher(rabbitmq *messaging.RabbitMQ) *TripEventPublisher {
	return &TripEventPublisher{
		rabbitmq: rabbitmq,
	}
}

func (tevp *TripEventPublisher) PublishTripCreate(ctx context.Context) error {
	tevp.rabbitmq.PublishMessage(ctx, "hello", "Trip created event payload")
	return nil
}
