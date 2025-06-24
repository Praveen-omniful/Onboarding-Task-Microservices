package sqs

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/omniful/go_commons/compression"
	commonsqs "github.com/omniful/go_commons/sqs"
)

// Client wraps SQS functionality with compression workaround
type Client struct {
	Queue     *commonsqs.Queue
	QueueURL  string
	sqsClient *sqs.SQS
}

// New creates a new SQS client that bypasses go_commons Publisher to avoid compression issues
func New(queue *commonsqs.Queue) *Client {
	// Set environment variables for LocalStack
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_ENDPOINT", "http://localhost:4566")
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	// Set LocalStack endpoint for local testing
	os.Setenv("LOCAL_SQS_ENDPOINT", "http://localhost:4566")

	// Disable compression completely to avoid nil pointer issues
	os.Setenv("DISABLE_COMPRESSION", "true")
	os.Setenv("SQS_DISABLE_COMPRESSION", "true")
	queueURL := ""
	if queue != nil && queue.Url != nil {
		queueURL = *queue.Url
	}

	// Create AWS SQS client for direct messaging (bypassing go_commons Publisher)
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String("http://localhost:4566"),
		Credentials: credentials.NewStaticCredentials("test", "test", ""),
	})
	if err != nil {
		log.Printf("Failed to create AWS session: %v", err)
	}

	sqsClient := sqs.New(sess)

	return &Client{
		Queue:     queue,
		QueueURL:  queueURL,
		sqsClient: sqsClient,
	}
}

// Publish publishes a message to SQS using direct AWS SDK to avoid compression issues
func (c *Client) Publish(ctx context.Context, msg *commonsqs.Message) error {
	if c.sqsClient == nil {
		log.Printf("❌ [SQS] No SQS client available, logging message: %s", string(msg.Value))
		return fmt.Errorf("SQS client not initialized")
	}

	// Send message directly to SQS using AWS SDK
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.QueueURL),
		MessageBody: aws.String(string(msg.Value)),
	}

	// Add message attributes if any
	if msg.Attributes != nil && len(msg.Attributes) > 0 {
		attributes := make(map[string]*sqs.MessageAttributeValue)
		for key, value := range msg.Attributes {
			attributes[key] = &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(fmt.Sprintf("%v", value)),
			}
		}
		input.MessageAttributes = attributes
	}

	result, err := c.sqsClient.SendMessage(input)
	if err != nil {
		log.Printf("❌ [SQS] Failed to send message to queue %s: %v", c.QueueURL, err)
		return fmt.Errorf("failed to send SQS message: %w", err)
	}

	log.Printf("✅ [SQS] Message sent successfully to queue %s, MessageId: %s", c.QueueURL, *result.MessageId)
	log.Printf("📤 [SQS] Message content: %s", string(msg.Value))

	return nil
}

// BatchPublish publishes multiple messages to SQS using direct AWS SDK
func (c *Client) BatchPublish(ctx context.Context, msgs []*commonsqs.Message) error {
	if c.sqsClient == nil {
		log.Printf("❌ [SQS] No SQS client available, logging %d messages", len(msgs))
		return fmt.Errorf("SQS client not initialized")
	}

	if len(msgs) == 0 {
		return nil
	}

	// Send messages individually for simplicity (could be optimized with batch API)
	successCount := 0
	for i, msg := range msgs {
		err := c.Publish(ctx, msg)
		if err != nil {
			log.Printf("❌ [SQS] Failed to send message %d/%d: %v", i+1, len(msgs), err)
		} else {
			successCount++
		}
	}

	log.Printf("✅ [SQS] Batch publish completed: %d/%d messages sent successfully", successCount, len(msgs))

	if successCount == 0 {
		return fmt.Errorf("failed to send any messages in batch")
	}

	return nil
}

// NewQueue creates an SQS queue using go_commons with proper compression configuration
func NewQueue(queueName, endpoint, region, accessKey, secretKey string) (*commonsqs.Queue, error) {
	// Set environment variables for go_commons to use LocalStack
	os.Setenv("AWS_REGION", region)
	os.Setenv("AWS_ENDPOINT", endpoint)
	os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	os.Setenv("LOCAL_SQS_ENDPOINT", endpoint)

	// Create AWS session for direct queue creation
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	sqsClient := sqs.New(sess)

	// Try to create the queue directly using AWS SDK
	createInput := &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	}

	createResult, err := sqsClient.CreateQueue(createInput)
	if err != nil {
		log.Printf("⚠️ Failed to create queue %s (may already exist): %v", queueName, err)
	} else {
		log.Printf("✅ Queue created successfully: %s", *createResult.QueueUrl)
	}

	// Get queue URL
	getUrlInput := &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}

	urlResult, err := sqsClient.GetQueueUrl(getUrlInput)
	if err != nil {
		// Fallback to manual URL construction
		queueURL := fmt.Sprintf("%s/000000000000/%s", endpoint, queueName)
		log.Printf("⚠️ Failed to get queue URL, using fallback: %s", queueURL)

		queue := &commonsqs.Queue{
			Url:  &queueURL,
			Name: queueName,
			Config: &commonsqs.Config{
				Account:     "000000000000",
				Endpoint:    endpoint,
				Region:      region,
				Compression: compression.None,
			},
		}
		return queue, nil
	}

	// Create proper SQS configuration with compression explicitly disabled
	config := &commonsqs.Config{
		Account:     "000000000000", // LocalStack account ID
		Endpoint:    endpoint,
		Region:      region,
		Compression: compression.None, // Explicitly disable compression
	}

	queue := &commonsqs.Queue{
		Url:    urlResult.QueueUrl,
		Name:   queueName,
		Config: config,
	}

	log.Printf("✅ Queue ready with URL: %s", *queue.Url)
	return queue, nil
}

// MessageHandler implements the go_commons SQS message handler interface
type MessageHandler struct {
	ProcessFunc func(ctx context.Context, msgs []*commonsqs.Message) error
}

// Process implements the ISqsMessageHandler interface
func (h *MessageHandler) Process(ctx context.Context, msgs *[]commonsqs.Message) error {
	// Convert pointer to slice to slice of pointers for easier handling
	messages := make([]*commonsqs.Message, len(*msgs))
	for i := range *msgs {
		messages[i] = &(*msgs)[i]
	}

	return h.ProcessFunc(ctx, messages)
}

// StartConsumer starts an SQS consumer using go_commons
func StartConsumer(queue *commonsqs.Queue, handler commonsqs.ISqsMessageHandler, workers, concurrency, maxMessages, visibilityTimeout int, async, batch bool, ctx context.Context) error {
	consumer, err := commonsqs.NewConsumer(
		queue,
		uint64(workers),
		uint64(concurrency),
		handler,
		int64(maxMessages),
		int64(visibilityTimeout),
		async,
		batch,
	)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	go consumer.Start(ctx)
	return nil
}
