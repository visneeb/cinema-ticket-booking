package service

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"cinema-ticket-booking/internal/model"
	wshub "cinema-ticket-booking/internal/websocket"
	"cinema-ticket-booking/pkg/rabbitmq"
)

// StartAuditConsumer listens for booking.timeout events from RabbitMQ,
// writes BOOKING_TIMEOUT audit logs, and broadcasts AVAILABLE via WebSocket.
func (s *BookingService) StartAuditConsumer(conn *amqp.Connection) error {
	consumer, err := rabbitmq.NewConsumer(conn)
	if err != nil {
		return err
	}
	return consumer.Consume("audit_timeout", rabbitmq.EventBookingTimeout, func(body []byte) {
		var ev struct {
			UserID     string `json:"user_id"`
			ShowtimeID string `json:"showtime_id"`
			SeatID     string `json:"seat_id"`
		}
		if err := json.Unmarshal(body, &ev); err != nil {
			log.Printf("[AuditConsumer] unmarshal error: %v", err)
			return
		}
		ctx := context.Background()
		s.locks.DeleteStatus(ctx, ev.ShowtimeID, ev.SeatID)
		s.bookings.SetSeatStatus(ctx, ev.SeatID, model.SeatAvailable)
		s.Hub.BroadcastSeatEvent(wshub.SeatEvent{SeatID: ev.SeatID, ShowtimeID: ev.ShowtimeID, Status: model.SeatAvailable})
		s.bookings.InsertAuditLog(ctx, model.AuditBookingTimeout, ev.UserID, ev.ShowtimeID, ev.SeatID, "lock expired without booking")
		log.Printf("[AuditConsumer] timeout seat=%s showtime=%s", ev.SeatID, ev.ShowtimeID)
	})
}

// StartNotificationConsumer listens for booking.confirmed events and sends a mock email.
// Replace the log.Printf with a real SMTP / SendGrid call when ready.
func (s *BookingService) StartNotificationConsumer(conn *amqp.Connection) error {
	consumer, err := rabbitmq.NewConsumer(conn)
	if err != nil {
		return err
	}
	return consumer.Consume(rabbitmq.QueueNotify, rabbitmq.EventBookingConfirmed, func(body []byte) {
		var b model.Booking
		if err := json.Unmarshal(body, &b); err != nil {
			log.Printf("[Notify] unmarshal error: %v", err)
			return
		}
		// Mock email
		log.Printf("[Notify] EMAIL → user_id=%s seat=%s showtime=%s status=%s",
			b.UserID, b.SeatID, b.ShowtimeID, b.Status)
	})
}

// StartAuditLogConsumer listens for booking.confirmed events and writes an async audit log.
func (s *BookingService) StartAuditLogConsumer(conn *amqp.Connection) error {
	consumer, err := rabbitmq.NewConsumer(conn)
	if err != nil {
		return err
	}
	return consumer.Consume(rabbitmq.QueueAuditLog, rabbitmq.EventBookingConfirmed, func(body []byte) {
		var b model.Booking
		if err := json.Unmarshal(body, &b); err != nil {
			log.Printf("[AuditLog] unmarshal error: %v", err)
			return
		}
		s.bookings.InsertAuditLog(
			context.Background(),
			model.AuditBookingSuccess,
			b.UserID, b.ShowtimeID, b.SeatID,
			"async: booking confirmed",
		)
	})
}
