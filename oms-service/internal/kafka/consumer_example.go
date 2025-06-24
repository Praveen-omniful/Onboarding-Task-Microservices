package kafka

import (
	"context"
	"log"
	"time"
)

// ExampleOrderEventHandler demonstrates how to implement OrderEventHandler
type ExampleOrderEventHandler struct {
	// You can add dependencies here like database connections, other services, etc.
}

// HandleOrderCreated processes newly created orders
func (h *ExampleOrderEventHandler) HandleOrderCreated(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("🆕 Processing new order: %s", event.OrderID)

	// Example processing logic:
	// 1. Update inventory
	// 2. Send notification emails
	// 3. Update analytics
	// 4. Trigger fulfillment processes

	log.Printf("📊 Order Details: Customer=%s, Total=$%.2f, Items=%d",
		event.CustomerID, event.TotalAmount, len(event.Items))

	// Process each item
	for _, item := range event.Items {
		log.Printf("  📦 Item: %s x%d @ $%.2f (Hub: %s)",
			item.ProductName, item.Quantity, item.UnitPrice, item.HubID)
	}

	return nil
}

// HandleOrderUpdated processes order updates
func (h *ExampleOrderEventHandler) HandleOrderUpdated(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("🔄 Processing order update: %s -> %s", event.OrderID, event.Status)

	// Example processing logic based on status:
	switch event.Status {
	case "processing":
		log.Printf("🏭 Order %s is now being processed", event.OrderID)
		// Trigger warehouse operations
	case "shipped":
		log.Printf("🚚 Order %s has been shipped", event.OrderID)
		// Send tracking information to customer
	case "delivered":
		log.Printf("📦 Order %s has been delivered", event.OrderID)
		// Send delivery confirmation, request review
	}

	return nil
}

// HandleOrderCancelled processes order cancellations
func (h *ExampleOrderEventHandler) HandleOrderCancelled(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("❌ Processing order cancellation: %s", event.OrderID)

	// Example processing logic:
	// 1. Restore inventory
	// 2. Process refunds
	// 3. Send cancellation confirmation
	// 4. Update analytics

	log.Printf("💰 Refunding $%.2f for cancelled order %s", event.TotalAmount, event.OrderID)

	return nil
}

// StartKafkaConsumer demonstrates how to start the Kafka consumer in your application
func StartKafkaConsumer(ctx context.Context) error {
	// Create consumer with default configuration
	consumer := NewDefaultConsumer()

	// Create and register your custom order event handler
	orderHandler := &ExampleOrderEventHandler{}
	consumer.RegisterOrderEventHandler(orderHandler)

	// You can also register custom handlers for other topics
	// consumer.RegisterHandler("user-events", userHandler)
	// consumer.RegisterHandler("inventory-events", inventoryHandler)

	// Start consuming
	err := consumer.Subscribe(ctx)
	if err != nil {
		log.Printf("❌ Failed to start Kafka consumer: %v", err)
		return err
	}

	log.Printf("✅ Kafka consumer started successfully")

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		log.Printf("🔄 Shutting down Kafka consumer...")
		consumer.Close()
	}()

	return nil
}

// TestKafkaConsumer provides a simple test function
func TestKafkaConsumer() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("🧪 Testing Kafka consumer...")

	err := StartKafkaConsumer(ctx)
	if err != nil {
		log.Printf("❌ Kafka consumer test failed: %v", err)
		return
	}

	// Wait for test duration
	<-ctx.Done()
	log.Printf("✅ Kafka consumer test completed")
}
