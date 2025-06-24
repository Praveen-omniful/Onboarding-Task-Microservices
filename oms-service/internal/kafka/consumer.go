package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/omniful/go_commons/pubsub"
	"github.com/omniful/go_commons/pubsub/interceptor"
	"github.com/omniful/go_commons/sqs"
)

// ConsumerClient represents a Kafka consumer with message handling capabilities
type ConsumerClient struct {
	config                   *ConsumerConfig
	consumer                 sarama.ConsumerGroup
	HandlerMap               map[string]pubsub.IPubSubMessageHandler
	Interceptor              interceptor.Interceptor
	DeadLetterQueuePublisher *sqs.Publisher
	transactionName          string
	mutex                    sync.Mutex
	isRunning                bool
}

// ConsumerConfig holds configuration for Kafka consumer
type ConsumerConfig struct {
	Brokers         []string
	ConsumerGroupID string
	ClientID        string
	Version         string
	RetryInterval   time.Duration
	Topics          []string
	// Authentication
	UseSASL      bool
	SASLUsername string
	SASLPassword string
	// Dead Letter Queue config
	DeadLetterConfig *DeadLetterConfig
}

// DeadLetterConfig holds configuration for dead letter queue
type DeadLetterConfig struct {
	Queue    string
	Account  string
	Endpoint string
	Region   string
	Prefix   string
}

// OrderEventHandler interface for handling order events
type OrderEventHandler interface {
	HandleOrderCreated(ctx context.Context, event *OrderCreatedEvent) error
	HandleOrderUpdated(ctx context.Context, event *OrderCreatedEvent) error
	HandleOrderCancelled(ctx context.Context, event *OrderCreatedEvent) error
}

// DefaultOrderEventHandler provides a default implementation
type DefaultOrderEventHandler struct{}

// HandleOrderCreated handles order created events
func (h *DefaultOrderEventHandler) HandleOrderCreated(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("📥 [KAFKA CONSUMER] Order created event received: OrderID=%s, CustomerID=%s, TotalAmount=%.2f",
		event.OrderID, event.CustomerID, event.TotalAmount)
	return nil
}

// HandleOrderUpdated handles order updated events
func (h *DefaultOrderEventHandler) HandleOrderUpdated(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("📥 [KAFKA CONSUMER] Order updated event received: OrderID=%s, Status=%s",
		event.OrderID, event.Status)
	return nil
}

// HandleOrderCancelled handles order cancelled events
func (h *DefaultOrderEventHandler) HandleOrderCancelled(ctx context.Context, event *OrderCreatedEvent) error {
	log.Printf("📥 [KAFKA CONSUMER] Order cancelled event received: OrderID=%s", event.OrderID)
	return nil
}

// NewConsumer creates a new Kafka consumer client
func NewConsumer(config *ConsumerConfig) *ConsumerClient {
	if config.RetryInterval == 0 {
		config.RetryInterval = time.Second
	}

	consumer := newConsumerGroup(config)

	consumerClient := &ConsumerClient{
		consumer:        consumer,
		config:          config,
		HandlerMap:      make(map[string]pubsub.IPubSubMessageHandler),
		Interceptor:     interceptor.NewRelicInterceptor(),
		transactionName: "kafka-consumer",
		isRunning:       false,
	}

	return consumerClient
}

// NewDefaultConsumer creates a consumer with default configuration for OMS
func NewDefaultConsumer() *ConsumerClient {
	config := &ConsumerConfig{
		Brokers:         []string{"localhost:9092"},
		ConsumerGroupID: "oms-consumer-group",
		ClientID:        "oms-service",
		Version:         "2.8.0",
		RetryInterval:   time.Second,
		Topics:          []string{"order-events"},
		UseSASL:         false,
	}

	return NewConsumer(config)
}

// newConsumerGroup creates a new Sarama consumer group
func newConsumerGroup(config *ConsumerConfig) sarama.ConsumerGroup {
	saramaConfig := sarama.NewConfig()

	// Authentication setup
	if config.UseSASL {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.TLS.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		saramaConfig.Net.SASL.User = config.SASLUsername
		saramaConfig.Net.SASL.Password = config.SASLPassword
	}

	saramaConfig.ClientID = config.ClientID
	saramaConfig.Version = parseKafkaVersion(config.Version)
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Fetch.Min = 50 * 1024
	saramaConfig.Consumer.MaxWaitTime = 400 * time.Millisecond
	saramaConfig.Metadata.RefreshFrequency = 60 * time.Second
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.ConsumerGroupID, saramaConfig)
	if err != nil {
		log.Printf("❌ Failed to create Kafka consumer group: %v", err)
		return nil
	}

	log.Printf("✅ Kafka consumer group created: %s", config.ConsumerGroupID)
	return consumerGroup
}

// parseKafkaVersion parses Kafka version string
func parseKafkaVersion(version string) sarama.KafkaVersion {
	v, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		log.Printf("⚠️ Failed to parse Kafka version %s, using default", version)
		return sarama.V2_8_0_0
	}
	return v
}

// RegisterHandler registers a handler for a topic in a concurrent-safe way
func (c *ConsumerClient) RegisterHandler(topic string, handler pubsub.IPubSubMessageHandler) *ConsumerClient {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.HandlerMap[topic] = handler
	log.Printf("📝 Registered handler for topic: %s", topic)
	return c
}

// RegisterOrderEventHandler registers a handler for order events
func (c *ConsumerClient) RegisterOrderEventHandler(handler OrderEventHandler) *ConsumerClient {
	orderHandler := &OrderMessageHandler{
		EventHandler: handler,
	}
	return c.RegisterHandler("order.created", orderHandler)
}

// UnRegisterHandler removes a handler for a topic in a concurrent-safe way
func (c *ConsumerClient) UnRegisterHandler(topic string) *ConsumerClient {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.HandlerMap, topic)
	log.Printf("🗑️ Unregistered handler for topic: %s", topic)
	return c
}

// SetInterceptor sets a custom interceptor
func (c *ConsumerClient) SetInterceptor(interceptor interceptor.Interceptor) *ConsumerClient {
	if interceptor != nil {
		c.Interceptor = interceptor
	}
	return c
}

// Subscribe starts consuming messages from registered topics
func (c *ConsumerClient) Subscribe(ctx context.Context) error {
	if c.consumer == nil {
		return fmt.Errorf("consumer group not initialized")
	}

	c.isRunning = true

	// Start error handler goroutine
	go c.handleErrors()

	// Main consumption loop
	go func() {
		for c.isRunning {
			if len(c.HandlerMap) == 0 {
				log.Printf("⚠️ No handlers registered, waiting...")
				time.Sleep(5 * time.Second)
				continue
			}

			consumerHandler := &ConsumerGroupHandler{
				HandlerMap:      c.HandlerMap,
				Interceptor:     c.Interceptor,
				Context:         ctx,
				TransactionName: c.transactionName,
				RetryInterval:   c.config.RetryInterval,
			}

			topics := make([]string, 0, len(c.HandlerMap))
			for topic := range c.HandlerMap {
				topics = append(topics, topic)
			}

			log.Printf("🔄 Starting to consume from topics: %v", topics)
			err := c.consumer.Consume(ctx, topics, consumerHandler)
			if err != nil {
				log.Printf("❌ Kafka consume error: %v", err)
				if ctx.Err() != nil {
					// Context cancelled, exit gracefully
					break
				}
			}

			// Brief pause before reconnecting
			time.Sleep(c.config.RetryInterval)
		}
	}()

	log.Printf("✅ Kafka consumer started successfully")
	return nil
}

// handleErrors handles consumer errors
func (c *ConsumerClient) handleErrors() {
	if c.consumer == nil {
		return
	}

	for err := range c.consumer.Errors() {
		if consumerErr, ok := err.(*sarama.ConsumerError); ok {
			log.Printf("❌ Kafka consumer error: Topic=%s, Partition=%d, Error=%v",
				consumerErr.Topic, consumerErr.Partition, consumerErr.Err)
		} else {
			log.Printf("❌ Kafka unknown error: %v", err)
		}
	}
}

// Close stops the consumer gracefully
func (c *ConsumerClient) Close() error {
	c.isRunning = false

	if c.consumer != nil {
		log.Printf("🔄 Closing Kafka consumer...")
		err := c.consumer.Close()
		if err != nil {
			log.Printf("❌ Error closing Kafka consumer: %v", err)
			return err
		}
		log.Printf("✅ Kafka consumer closed successfully")
	}
	return nil
}

// IsRunning returns whether the consumer is currently running
func (c *ConsumerClient) IsRunning() bool {
	return c.isRunning
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
type ConsumerGroupHandler struct {
	HandlerMap      map[string]pubsub.IPubSubMessageHandler
	Interceptor     interceptor.Interceptor
	Context         context.Context
	TransactionName string
	RetryInterval   time.Duration
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Printf("🔄 Kafka consumer group session started")
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Printf("🔄 Kafka consumer group session ended")
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			h.processMessage(session, message)

		case <-h.Context.Done():
			return nil
		}
	}
}

// processMessage processes a single Kafka message
func (h *ConsumerGroupHandler) processMessage(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	topic := message.Topic

	handler, exists := h.HandlerMap[topic]
	if !exists {
		log.Printf("⚠️ No handler found for topic: %s", topic)
		session.MarkMessage(message, "")
		return
	}
	// Convert Sarama message to pubsub message
	pubsubMessage := &pubsub.Message{
		Topic: message.Topic,
		Key:   string(message.Key), // Convert []byte to string
		Value: message.Value,
		Headers: map[string]string{
			"partition": fmt.Sprintf("%d", message.Partition),
			"offset":    fmt.Sprintf("%d", message.Offset),
			"timestamp": message.Timestamp.Format(time.RFC3339),
		},
	}
	// Process message with handler
	err := handler.Process(h.Context, pubsubMessage)
	if err != nil {
		log.Printf("❌ Failed to process message from topic %s: %v", topic, err)
		// Could implement retry logic here
		return
	}

	// Mark message as processed
	session.MarkMessage(message, "")
	log.Printf("✅ Message processed successfully from topic: %s", topic)
}

// OrderMessageHandler handles order-specific messages
type OrderMessageHandler struct {
	EventHandler OrderEventHandler
}

// Process implements pubsub.IPubSubMessageHandler
func (h *OrderMessageHandler) Process(ctx context.Context, message *pubsub.Message) error {
	var event OrderCreatedEvent
	err := json.Unmarshal(message.Value, &event)
	if err != nil {
		log.Printf("❌ Failed to unmarshal order event: %v", err)
		return err
	}

	// Route based on event type (could be extended with event type field)
	switch event.Status {
	case "created", "pending", "on_hold":
		err = h.EventHandler.HandleOrderCreated(ctx, &event)
	case "updated", "processing", "shipped", "new_order":
		err = h.EventHandler.HandleOrderUpdated(ctx, &event)
	case "cancelled":
		err = h.EventHandler.HandleOrderCancelled(ctx, &event)
	default:
		err = h.EventHandler.HandleOrderCreated(ctx, &event) // Default to created
	}

	if err != nil {
		log.Printf("❌ Failed to handle order event %s: %v", event.OrderID, err)
		return err
	}

	return nil
}
