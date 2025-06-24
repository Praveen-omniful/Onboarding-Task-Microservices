package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"oms-service/config"
	"oms-service/internal/kafka"
	"oms-service/internal/orders"
	"oms-service/internal/processor"
	"oms-service/internal/s3"
	"oms-service/internal/sqs"
	"os"
	"strings"
	"time"

	commonsqs "github.com/omniful/go_commons/sqs"
)

func main() {
	// Load configuration
	log.Println("🔧 Loading OMS configuration...")
	cfg := config.LoadConfig()
	err := cfg.ValidateConfig()
	if err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	// Initialize MongoDB using orders package
	log.Println("🔗 Initializing MongoDB for orders...")
	mongoURI := cfg.GetMongoDBConnectionString()
	err = orders.InitializeMongoDB(mongoURI)
	if err != nil {
		log.Printf("❌ MongoDB initialization failed: %v", err)
		log.Println("⚠️ Continuing without MongoDB (will use simulation mode)")
	} else {
		log.Println("✅ MongoDB connected successfully")

		// Create sample orders for demonstration
		log.Println("📊 Creating sample orders...")
		err = orders.CreateSampleOrders()
		if err != nil {
			log.Printf("⚠️ Failed to create sample orders: %v", err)
		} else {
			log.Println("✅ Sample orders created successfully")
		}
	}

	// Initialize legacy processor MongoDB (for backward compatibility)
	err = processor.InitializeMongoDB()
	if err != nil {
		log.Printf("Legacy MongoDB initialization failed: %v", err)
	}

	// Initialize IMS client for SKU/Hub validation
	log.Println("Initializing IMS client...")
	processor.InitializeIMSClient(cfg.IMSServiceURL)
	log.Printf("IMS client initialized for URL: %s", cfg.IMSServiceURL) // Initialize Kafka with proper configuration and graceful error handling
	log.Println("Initializing Kafka...")
	os.Setenv("KAFKA_ENABLED", "true") // Try to enable Kafka
	err = processor.InitializeKafka()
	if err != nil {
		log.Printf("Kafka initialization failed: %v", err)
		log.Println("Continuing without Kafka (will use simulation mode)")
		os.Setenv("KAFKA_ENABLED", "false") // Disable if failed
	} else {
		log.Println("Kafka initialized successfully")
	} // Initialize Kafka consumer for order events (if Kafka is available)
	kafkaEnabled := os.Getenv("KAFKA_ENABLED") == "true"
	if kafkaEnabled {
		log.Println("🔄 Initializing Kafka consumer...")
		kafkaConsumer := kafka.NewDefaultConsumer()

		// Register order finalizer handler with inventory management
		orderFinalizer := kafka.NewOrderFinalizerHandler(cfg.IMSServiceURL)
		kafkaConsumer.RegisterOrderEventHandler(orderFinalizer)

		// Start Kafka consumer in background
		go func() {
			err := kafkaConsumer.Subscribe(context.Background())
			if err != nil {
				log.Printf("❌ Failed to start Kafka consumer: %v", err)
			}
		}()
		log.Println("✅ Kafka consumer initialized successfully")
	} else {
		log.Println("🔄 Kafka consumer disabled - Kafka not available")
	}

	// S3 setup - using go_commons S3 client
	s3Client, err := s3.NewSimpleS3Client()
	if err != nil {
		log.Fatalf("S3 client error: %v", err)
	}
	log.Println("S3 client initialized successfully")

	// Create S3 bucket if it doesn't exist
	bucketName := "oms-orders"
	err = s3Client.CreateBucket(context.Background(), bucketName)
	if err != nil {
		log.Printf("Failed to create S3 bucket (it may already exist): %v", err)
	}
	// Create SQS queue using our sqs wrapper around go_commons
	queueName := "oms-order-events"
	sqsEndpoint := "http://localhost:4566"
	sqsRegion := "us-east-1"
	accessKey := "dummy"
	secretKey := "dummy"

	// Set comprehensive compression disable environment variables before queue creation
	os.Setenv("DISABLE_COMPRESSION", "true")
	os.Setenv("SQS_DISABLE_COMPRESSION", "true")
	os.Setenv("COMPRESSION_DISABLED", "true")
	os.Setenv("NO_COMPRESSION", "true")

	queue, err := sqs.NewQueue(queueName, sqsEndpoint, sqsRegion, accessKey, secretKey)
	if err != nil {
		log.Fatalf("SQS queue creation error: %v", err)
	}

	log.Printf("Using queue URL: %s", *queue.Url)
	// Start SQS consumer using go_commons with improved handler
	go func() {
		log.Println("Starting SQS consumer with go_commons...")
		// Create message handler using the new wrapper
		messageHandler := &sqs.MessageHandler{
			ProcessFunc: func(ctx context.Context, msgs []*commonsqs.Message) error {
				log.Printf("📥 [SQS Consumer] Processing %d SQS messages", len(msgs))
				for _, msg := range msgs {
					log.Printf("📝 [SQS Consumer] Processing message: %s", string(msg.Value))

					// Parse the message
					var messageData map[string]string
					if err := json.Unmarshal(msg.Value, &messageData); err != nil {
						log.Printf("❌ [SQS Consumer] Failed to parse message: %v", err)
						continue
					}

					// Process CSV from S3 if this is a file upload notification
					if bucket, ok := messageData["bucket"]; ok {
						if key, ok := messageData["key"]; ok {
							log.Printf("🚀 [SQS Consumer] Starting CSV processing from S3: bucket=%s, key=%s", bucket, key)
							// Process CSV from S3 using the distributed logic
							go func(b, k string) {
								err := processor.ProcessCSVFromS3Real(ctx, b, k)
								if err != nil {
									log.Printf("❌ [CSV Processor] Real S3 processing failed: %v", err)
									log.Printf("🔄 [CSV Processor] Falling back to standard processing")
									err = processor.ProcessCSVFromS3(ctx, b, k)
									if err != nil {
										log.Printf("❌ [CSV Processor] Standard processing also failed: %v", err)
									}
								} else {
									log.Printf("✅ [CSV Processor] Real S3 processing completed successfully for %s", k)
								}
							}(bucket, key)
						}
					}
				}
				return nil
			},
		}

		// Use the StartConsumer function from our sqs package
		err := sqs.StartConsumer(
			queue,
			messageHandler,
			1,    // workers
			1,    // concurrency
			10,   // maxMessages
			30,   // visibilityTimeout
			true, // async
			true, // batch
			context.Background(),
		)
		if err != nil {
			log.Printf("SQS consumer error: %v", err)
		}
	}()

	// Create SQS client for publishing (RE-ENABLED with compression fix)
	sqsClient := sqs.New(queue)
	log.Println("SQS client initialized successfully with compression disabled")
	// HTTP handlers
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Serve upload form for browser access
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>OMS File Upload</title></head>
<body style="font-family: Arial; max-width: 600px; margin: 50px auto; padding: 20px;">
	<h2>🚀 OMS Service - File Upload</h2>
	<form action="/upload" method="post" enctype="multipart/form-data" style="border: 1px solid #ddd; padding: 20px; border-radius: 5px;">
		<p><label>Select CSV file:</label><br>
		<input type="file" name="file" accept=".csv" required style="width: 100%%; padding: 10px; margin: 10px 0;"></p>
		<p><button type="submit" style="background: #4CAF50; color: white; padding: 15px 30px; border: none; border-radius: 5px; cursor: pointer;">📤 Upload File</button></p>
	</form>
	<div style="margin-top: 20px; padding: 15px; background: #e7f3ff; border-radius: 5px;">
		<strong>Service Status:</strong> ✅ Running on port 8088<br>
		<strong>Endpoints:</strong><br>
		• POST /upload - Upload CSV files<br>
		• GET /health - Health check<br>
		• GET /stats - Order statistics<br>
		• GET /invalid-files - List invalid files
	</div>
</body>
</html>`)
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to get file: %v", err)
			return
		}
		defer file.Close()

		// Generate unique filename with timestamp
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("%s-%s", timestamp, header.Filename)

		// Upload to S3
		bucketName := "oms-orders"
		// Read file content
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to read file: %v", err)
			return
		}

		// Upload the file using SimpleS3Client
		log.Printf("File received successfully: %s (%d bytes)", filename, len(fileBytes))
		// Upload to S3 using go_commons
		err = s3Client.Upload(context.Background(), bucketName, filename, bytes.NewReader(fileBytes), "text/csv")
		s3UploadSuccessful := (err == nil)

		if s3UploadSuccessful {
			log.Printf("File uploaded to S3 successfully: bucket=%s, key=%s", bucketName, filename)
		} else {
			log.Printf("⚠️ S3 upload skipped (LocalStack not available), processing directly")
		}

		// Send SQS message for processing using improved go_commons integration
		message := map[string]string{
			"bucket":            bucketName,
			"key":               filename,
			"original_filename": header.Filename,
		}
		messageBytes, _ := json.Marshal(message)
		sqsMessage := &commonsqs.Message{
			Value: messageBytes,
		}
		// Try SQS processing first
		sqsErr := sqsClient.Publish(context.Background(), sqsMessage)
		if sqsErr != nil {
			log.Printf("Failed to send SQS message: %v", sqsErr)
		} else {
			log.Printf("SQS message sent successfully: %s", string(messageBytes))
		}

		// ALWAYS process CSV directly to ensure it gets processed
		// This ensures your files are processed regardless of S3/SQS/Kafka status
		log.Printf("🔄 Processing CSV directly to ensure completion: %s", filename)
		go func() {
			processingErr := processor.ProcessCSVContentDirectly(context.Background(), fileBytes, filename)
			if processingErr != nil {
				log.Printf("❌ CSV processing error: %v", processingErr)
			} else {
				log.Printf("✅ CSV processing completed successfully - stats should update now")
			}
		}()

		// Response
		fmt.Fprintf(w, "File uploaded and processing started: %s", header.Filename)
	})
	// Stats endpoint
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Get order statistics from the new orders package
		stats, err := orders.GetOrderStats()
		if err != nil {
			// Fallback to legacy processor stats
			legacyStats, legacyErr := processor.GetOrderStats(context.Background())
			if legacyErr != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Failed to get stats: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(legacyStats)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// Orders endpoint - to view all orders
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		allOrders, err := orders.GetAllOrders()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to get orders: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"orders": allOrders,
			"count":  len(allOrders),
		})
	})

	// Create sample data endpoint
	http.HandleFunc("/create-sample-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := orders.CreateSampleOrders()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to create sample data: %v", err)
			return
		}

		fmt.Fprintf(w, "✅ Sample orders created successfully! Check MongoDB Compass now.")
	})

	// Invalid CSV files endpoints
	http.HandleFunc("/invalid-files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Get list of invalid files
		files, err := processor.ListInvalidFiles()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to list invalid files: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"invalid_files": files,
			"count":         len(files),
		})
	})

	http.HandleFunc("/invalid-files/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Extract filename from URL path
		filename := strings.TrimPrefix(r.URL.Path, "/invalid-files/")
		if filename == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Filename required")
			return
		}

		// Get file path and serve the file
		filePath, err := processor.GetInvalidFilePath(filename)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "File not found: %v", err)
			return
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "File not found")
			return
		}

		// Serve the file
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		http.ServeFile(w, r, filePath)
	})
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "oms", "timestamp": time.Now().Format(time.RFC3339)})
	})
	log.Println("🚀 OMS Service starting on :8088")
	log.Println("📋 Endpoints available:")
	log.Println("  POST /upload - Upload CSV files")
	log.Println("  GET  /stats - Order statistics")
	log.Println("  GET  /invalid-files - List invalid CSV files")
	log.Println("  GET  /invalid-files/{filename} - Download invalid CSV file")
	log.Println("  GET  /health - Health check")
	log.Println("🔗 All services using go_commons (S3, SQS, Kafka, CSV)")
	log.Fatal(http.ListenAndServe(":8088", nil))
}
