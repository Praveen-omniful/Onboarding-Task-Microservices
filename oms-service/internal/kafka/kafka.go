package kafka

// This file contains event types and interfaces for Kafka integration.
// The actual producer implementation is in producer.go using go_commons.

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// OrderCreatedEvent represents the event emitted when an order is created
type OrderCreatedEvent struct {
	OrderID     string      `json:"order_id"`
	CustomerID  string      `json:"customer_id"`
	TotalAmount float64     `json:"total_amount"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	Items       []OrderItem `json:"items"`
}

type OrderItem struct {
	SKU         string  `json:"sku"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	HubID       string  `json:"hub_id"`
}

// EventPublisher interface for order events
// EventPublisher interface for order events
type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, event *OrderCreatedEvent) error
	Close() error
}

// SimulatedEventPublisher provides a working implementation for your completed project
type SimulatedEventPublisher struct{}

// PublishOrderCreated simulates publishing an order created event
func (s *SimulatedEventPublisher) PublishOrderCreated(ctx context.Context, event *OrderCreatedEvent) error {
	eventJSON, _ := json.Marshal(event)
	log.Printf("📤 [KAFKA EVENT] Order created: %s", string(eventJSON))
	return nil
}

// Close is a no-op for simulation
func (s *SimulatedEventPublisher) Close() error {
	return nil
}
