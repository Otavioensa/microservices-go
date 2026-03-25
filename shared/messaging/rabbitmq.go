package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/retry"
	"ride-sharing/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange              = "trip"
	DeadLetterExchange        = "dlx"
	NotifyPaymentSuccessQueue = "payment_success"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

type MessageHandler func(context.Context, amqp.Delivery) error

func NewRabbitMQ(uri string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)

	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	ch, err := conn.Channel()

	if err != nil {
		log.Fatalf("failed to open a channel: %v", err)
		conn.Close()
		return nil, err
	}

	rmq := &RabbitMQ{conn: conn, Channel: ch}

	if err := rmq.setupExchangesAndQueues(); err != nil {
		rmq.Close()
		return nil, err
	}

	return rmq, nil
}

func (rmq *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	err := rmq.Channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)

	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := rmq.Channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)

	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := tracing.TracedConsumer(msg, func(ctx context.Context, d amqp.Delivery) error {
				log.Printf("Received a message: %s", msg.Body)

				retryConfig := retry.DefaultConfig()
				err := retry.WithBackoff(ctx, retryConfig, func() error {
					return handler(ctx, d)
				})

				if err != nil {
					log.Println("Failed to process message after retries: ", err)

					headers := amqp.Table{}

					if d.Headers != nil {
						headers = d.Headers
					}

					headers["x-death-reason"] = err.Error()
					headers["x-origin-exchange"] = d.Exchange
					headers["x-origin-routing-key"] = d.RoutingKey
					headers["x-retry-count"] = retryConfig.MaxRetries

					d.Headers = headers

					_ = d.Reject(false)
					return err
				}

				msg.Ack(false)
				return nil
			}); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}()

	return nil
}

func (rmq *RabbitMQ) PublishMessage(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {
	log.Printf("Publishing message with routing key: %s", routingKey)

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         jsonMessage,
	}

	return tracing.TracedPublisher(ctx, routingKey, TripExchange, msg, rmq.publish)
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}

	if r.Channel != nil {
		r.Channel.Close()
	}
}

func (rmq *RabbitMQ) publish(ctx context.Context, routingKey, exchange string, message amqp.Publishing) error {
	fmt.Printf("Publishing message to exchange %s with routing key %s: %s\n", exchange, routingKey, string(message.Body))
	return rmq.Channel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		message,
	)
}

func (r *RabbitMQ) setupExchangesAndQueues() error {
	// First setup the DLQ exchange and queue
	if err := r.setupDeadLetterExchange(); err != nil {
		return err
	}

	err := r.Channel.ExchangeDeclare(
		TripExchange, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", TripExchange, err)
	}

	if err := r.declareAndBindQueue(
		FindAvailableDriversQueue,
		[]string{contracts.TripEventCreated, contracts.TripEventDriverNotInterested},
		TripExchange,
	); err != nil {
		return fmt.Errorf("failed to declare and bind queues: %w", err)
	}

	if err := r.declareAndBindQueue(
		DriverCmdTripRequestQueue,
		[]string{contracts.DriverCmdTripRequest},
		TripExchange,
	); err != nil {
		return fmt.Errorf("failed to declare and bind queues: %w", err)
	}

	if err := r.declareAndBindQueue(
		DriverTripResponseQueue,
		[]string{contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline},
		TripExchange,
	); err != nil {
		return fmt.Errorf("failed to declare and bind queues: %w", err)
	}

	if err := r.declareAndBindQueue(
		NotifyRiderNoDriversFoundQueue,
		[]string{contracts.TripEventNoDriversFound},
		TripExchange,
	); err != nil {
		return fmt.Errorf("failed to declare and bind queues: %w", err)
	}

	if err := r.declareAndBindQueue(
		NotifyRiderDriverAssignedQueue,
		[]string{contracts.TripEventDriverAssigned},
		TripExchange,
	); err != nil {
		return fmt.Errorf("failed to declare and bind queues: %w", err)
	}

	if err := r.declareAndBindQueue(
		PaymentTripResponseQueue,
		[]string{contracts.PaymentCmdCreateSession},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyPaymentSessionCreatedQueue,
		[]string{contracts.PaymentEventSessionCreated},
		TripExchange,
	); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyPaymentSuccessQueue,
		[]string{contracts.PaymentEventSuccess},
		TripExchange,
	); err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) setupDeadLetterExchange() error {
	// Declare the dead letter exchange
	err := r.Channel.ExchangeDeclare(
		DeadLetterExchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %v", err)
	}

	// Declare the dead letter queue
	q, err := r.Channel.QueueDeclare(
		DeadLetterQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter queue: %v", err)
	}

	// Bind the queue to the exchange with a wildcard routing key
	err = r.Channel.QueueBind(
		q.Name,
		"#", // wildcard routing key to catch all messages
		DeadLetterExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind dead letter queue: %v", err)
	}

	return nil
}

func (r *RabbitMQ) declareAndBindQueue(queueName string, routingKeys []string, exchange string) error {
	// Add dead letter configuration
	args := amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	}

	queue, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		args,      // arguments
	)

	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	for _, routingKey := range routingKeys {
		err = r.Channel.QueueBind(
			queue.Name,   // queue name
			routingKey,   // routing key
			TripExchange, // exchange
			false,
			nil,
		)

		if err != nil {
			return fmt.Errorf("failed to bind a queue: %w", err)
		}
	}

	return nil
}
