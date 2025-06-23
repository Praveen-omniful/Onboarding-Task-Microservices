package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	commonsHttp "github.com/omniful/go_commons/http"
	logger "github.com/omniful/go_commons/log"
	"github.com/omniful/ims-service/internal/config"
)

// Global variables
var (
	dbRedisClient *redis.Client
)

func main() {
	_ = godotenv.Load()

	// Initialize config
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config: " + err.Error())
		os.Exit(1)
	}

	// Initialize database cluster
	if err := config.InitDB(cfg); err != nil {
		logger.Error("Failed to initialize database: " + err.Error())
		os.Exit(1)
	}

	// Initialize Redis client
	redisClient := initializeRedis()

	// Set global variables
	dbRedisClient = redisClient

	// Initialize server with custom timeouts
	server := commonsHttp.InitializeServer(
		":8081",        // listen address
		10*time.Second, // read timeout
		10*time.Second, // write timeout
		70*time.Second, // idle timeout
		false,          // TLS disabled (set to true if needed)
	)

	// Create a router group for API v1
	api := server.Group("/api/v1")

	// Set up routes
	setupRoutes(api)

	// Basic routes
	server.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	server.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "IMS Service is running!"})
	})

	// Print all registered routes
	printRoutes(server.Engine)

	// Start the server in a goroutine
	go func() {
		if err := server.StartServer("ims-service"); err != nil {
			logger.Error("Failed to start server: " + err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: ", err)
		os.Exit(1)
	}

	logger.Info("Server exiting")
}

func printRoutes(router *gin.Engine) {
	fmt.Println("\n=== Registered Routes ===")
	for _, route := range router.Routes() {
		fmt.Printf("%s\t%s\t-> %s\n", route.Method, route.Path, route.Handler)
	}
	fmt.Println("========================\n")
}

func initializeRedis() *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDRESS")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test the connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		logger.Error("Failed to connect to Redis: " + err.Error())
		return nil
	}

	logger.Info("Successfully connected to Redis")
	return client
}
