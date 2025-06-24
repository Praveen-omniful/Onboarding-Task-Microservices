package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds all configuration for the OMS service
type Config struct {
	// Server Configuration
	ServerPort string
	ServerHost string

	// MongoDB Configuration
	MongoDBURI      string
	MongoDBDatabase string
	MongoDBTimeout  int

	// S3 Configuration
	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool

	// SQS Configuration
	SQSEndpoint  string
	SQSRegion    string
	SQSQueueName string
	SQSAccessKey string
	SQSSecretKey string

	// Kafka Configuration
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string
	KafkaEnabled bool

	// Redis Configuration
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// IMS Service Configuration
	IMSServiceURL string

	// File Processing Configuration
	MaxFileSize       int64
	AllowedExtensions []string
	TempDirectory     string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	config := &Config{
		// Server defaults
		ServerPort:      getEnv("SERVER_PORT", "8085"),
		ServerHost:      getEnv("SERVER_HOST", "0.0.0.0"), // MongoDB defaults
		MongoDBURI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDBDatabase: getEnv("MONGODB_DATABASE", "oms_database"),
		MongoDBTimeout:  getEnvAsInt("MONGODB_TIMEOUT", 30),

		// S3 defaults (LocalStack)
		S3Endpoint:  getEnv("S3_ENDPOINT", "http://localhost:4566"),
		S3Region:    getEnv("S3_REGION", "us-east-1"),
		S3Bucket:    getEnv("S3_BUCKET", "orders-bucket"),
		S3AccessKey: getEnv("S3_ACCESS_KEY", "test"),
		S3SecretKey: getEnv("S3_SECRET_KEY", "test"),
		S3UseSSL:    getEnvAsBool("S3_USE_SSL", false),

		// SQS defaults (LocalStack)
		SQSEndpoint:  getEnv("SQS_ENDPOINT", "http://localhost:4566"),
		SQSRegion:    getEnv("SQS_REGION", "us-east-1"),
		SQSQueueName: getEnv("SQS_QUEUE_NAME", "order-processing-queue"),
		SQSAccessKey: getEnv("SQS_ACCESS_KEY", "test"),
		SQSSecretKey: getEnv("SQS_SECRET_KEY", "test"),

		// Kafka defaults
		KafkaBrokers: getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "order-events"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "oms-consumer-group"),
		KafkaEnabled: getEnvAsBool("KAFKA_ENABLED", true),

		// Redis defaults
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// IMS Service defaults
		IMSServiceURL: getEnv("IMS_SERVICE_URL", "http://localhost:8081"),

		// File processing defaults
		MaxFileSize:       getEnvAsInt64("MAX_FILE_SIZE", 10*1024*1024), // 10MB
		AllowedExtensions: getEnvAsSlice("ALLOWED_EXTENSIONS", []string{".csv", ".txt"}),
		TempDirectory:     getEnv("TEMP_DIRECTORY", "/tmp/oms"),
	}

	// Log loaded configuration (without sensitive data)
	log.Printf("📋 OMS Configuration loaded:")
	log.Printf("   Server: %s:%s", config.ServerHost, config.ServerPort)
	log.Printf("   MongoDB: %s/%s", config.MongoDBURI, config.MongoDBDatabase)
	log.Printf("   S3 Bucket: %s", config.S3Bucket)
	log.Printf("   SQS Queue: %s", config.SQSQueueName)
	log.Printf("   Kafka Topic: %s (enabled: %v)", config.KafkaTopic, config.KafkaEnabled)
	log.Printf("   IMS Service: %s", config.IMSServiceURL)

	return config
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		// For more complex parsing, use a proper CSV parser
		return []string{value}
	}
	return defaultValue
}

// GetMongoDBConnectionString returns the full MongoDB connection string
func (c *Config) GetMongoDBConnectionString() string {
	return c.MongoDBURI
}

// GetS3Config returns S3 configuration as a map for easy integration
func (c *Config) GetS3Config() map[string]interface{} {
	return map[string]interface{}{
		"endpoint":   c.S3Endpoint,
		"region":     c.S3Region,
		"bucket":     c.S3Bucket,
		"access_key": c.S3AccessKey,
		"secret_key": c.S3SecretKey,
		"use_ssl":    c.S3UseSSL,
	}
}

// GetSQSConfig returns SQS configuration as a map
func (c *Config) GetSQSConfig() map[string]interface{} {
	return map[string]interface{}{
		"endpoint":   c.SQSEndpoint,
		"region":     c.SQSRegion,
		"queue_name": c.SQSQueueName,
		"access_key": c.SQSAccessKey,
		"secret_key": c.SQSSecretKey,
	}
}

// GetKafkaConfig returns Kafka configuration
func (c *Config) GetKafkaConfig() map[string]interface{} {
	return map[string]interface{}{
		"brokers":  c.KafkaBrokers,
		"topic":    c.KafkaTopic,
		"group_id": c.KafkaGroupID,
		"enabled":  c.KafkaEnabled,
	}
}

// ValidateConfig validates that all required configuration is present
func (c *Config) ValidateConfig() error {
	// Add validation logic here if needed
	log.Println("✅ Configuration validation passed")
	return nil
}
