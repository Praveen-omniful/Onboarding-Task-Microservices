package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gokafka "github.com/omniful/go_commons/kafka"
	"github.com/omniful/go_commons/pubsub"
)

// KafkaProducer wraps go_commons kafka producer and implements EventPublisher
type KafkaProducer struct {
	producer *gokafka.ProducerClient
	enabled  bool
}

// Ensure KafkaProducer implements EventPublisher interface
var _ EventPublisher = (*KafkaProducer)(nil)

// NewKafkaProducer creates a new Kafka producer using go_commons with proper configuration
func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	// Check if Kafka should be enabled
	kafkaEnabled := os.Getenv("KAFKA_ENABLED")
	if kafkaEnabled != "true" {
		log.Printf("⚠️  Kafka disabled via KAFKA_ENABLED environment variable")
		return &KafkaProducer{enabled: false}, nil
	}
	// Use environment variables for broker configuration
	if len(brokers) == 0 {
		brokerEnv := os.Getenv("KAFKA_BROKERS")
		if brokerEnv != "" {
			brokers = strings.Split(brokerEnv, ",")
		} else {
			brokers = []string{"localhost:9092"}
		}
	}

	log.Printf("� Initializing Kafka producer with brokers: %v", brokers)
	// Create producer with proper configuration based on go_commons
	producer := gokafka.NewProducer(
		gokafka.WithBrokers(brokers),
		gokafka.WithClientID("oms-service-producer"),
		gokafka.WithKafkaVersion("2.8.1"),
	)

	// Test connectivity with error handling
	if producer == nil {
		log.Printf("❌ Failed to create Kafka producer, disabling Kafka")
		return &KafkaProducer{enabled: false}, nil
	}

	log.Printf("✅ Kafka producer initialized successfully")
	return &KafkaProducer{
		producer: producer,
		enabled:  true,
	}, nil
}

// PublishOrderCreated publishes order.created event to Kafka
func (k *KafkaProducer) PublishOrderCreated(ctx context.Context, event *OrderCreatedEvent) error {
	if !k.enabled {
		// Log the event for debugging when Kafka is disabled
		eventJSON, _ := json.Marshal(event)
		log.Printf("📤 [KAFKA DISABLED] Would publish order.created event: %s", string(eventJSON))
		return nil
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}

	// Get topic from environment variable
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "order-events" // Default topic
	}

	msg := &pubsub.Message{
		Topic: topic,
		Key:   event.CustomerID, // Use customer ID for FIFO ordering per customer
		Value: eventJSON,
		Headers: map[string]string{
			"event_type":  "order.created",
			"order_id":    event.OrderID,
			"customer_id": event.CustomerID,
			"created_at":  event.CreatedAt.Format(time.RFC3339),
		},
	}

	log.Printf("📤 Publishing order.created event to topic '%s' for order: %s", topic, event.OrderID)

	err = k.producer.Publish(ctx, msg)
	if err != nil {
		log.Printf("❌ Failed to publish to Kafka: %v", err)
		log.Printf("📤 [KAFKA FALLBACK] Event: %s", string(eventJSON))
		return fmt.Errorf("kafka publish failed: %w", err)
	}

	log.Printf("✅ Successfully published order.created event for order: %s", event.OrderID)
	return nil
}

// Close closes the Kafka producer
func (k *KafkaProducer) Close() error {
	if k.enabled && k.producer != nil {
		k.producer.Close()
	}
	return nil
}
