package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeBooking = "booking"
	QueueAuditLog   = "audit_log"
	QueueNotify     = "notify"

	EventBookingConfirmed = "booking.confirmed"
	EventBookingTimeout   = "booking.timeout"
	EventSeatReleased     = "seat.released"
)

// Publisher sends events to the booking exchange.
type Publisher struct {
	ch *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	// Durable topic exchange — survives broker restart
	if err := ch.ExchangeDeclare(ExchangeBooking, "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &Publisher{ch: ch}, nil
}

// Publish serialises payload to JSON and routes it via the booking exchange.
func (p *Publisher) Publish(ctx context.Context, routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, ExchangeBooking, routingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survive broker restart
			Body:         body,
		},
	)
}

// Consumer reads events from a named queue bound to the booking exchange.
type Consumer struct {
	ch *amqp.Channel
}

func NewConsumer(conn *amqp.Connection) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return &Consumer{ch: ch}, nil
}

// Consume binds a queue to a routing-key pattern and runs handler in a goroutine.
// Uses manual ack so messages are not lost if the worker panics.
func (c *Consumer) Consume(queue, routingKey string, handler func(body []byte)) error {
	if _, err := c.ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := c.ch.QueueBind(queue, routingKey, ExchangeBooking, false, nil); err != nil {
		return err
	}
	msgs, err := c.ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range msgs {
			handler(d.Body)
			if err := d.Ack(false); err != nil {
				log.Printf("[rabbitmq] ack error: %v", err)
			}
		}
	}()
	return nil
}