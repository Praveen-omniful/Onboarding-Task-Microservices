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
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	commonsHttp "github.com/omniful/go_commons/http"
	logger "github.com/omniful/go_commons/log"
	"github.com/omniful/ims-service/internal/api/handlers"
	"github.com/omniful/ims-service/internal/config"
	"github.com/omniful/ims-service/internal/repository"
	"github.com/omniful/ims-service/internal/service"
	"github.com/omniful/ims-service/pkg/constants"
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
	} else {
		logger.Info("Database initialized successfully")
	}

	// Initialize Redis client
	redisClient := initializeRedis()

	// Initialize repositories in correct order (due to dependencies)
	hubRepo := repository.NewHubRepository(config.DBCluster, redisClient)
	skuRepo := repository.NewSKURepository(config.DBCluster, redisClient)
	inventoryRepo := repository.NewInventoryRepository(config.DBCluster, hubRepo, skuRepo, redisClient)

	// Initialize services
	hubService := service.NewHubService(hubRepo)
	skuService := service.NewSKUService(skuRepo)
	inventoryService := service.NewInventoryService(inventoryRepo, hubRepo, skuRepo)

	// Initialize handlers
	hubHandler := handlers.NewHubHandler(hubService)
	skuHandler := handlers.NewSKUHandler(skuService)
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)

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

	// Register real database-backed routes
	hubHandler.RegisterRoutes(api)
	skuHandler.RegisterRoutes(api)
	inventoryHandler.RegisterRoutes(api)

	// Add validation endpoint for OMS integration
	api.POST("/validate", func(c *gin.Context) {
		var req struct {
			SKU      string    `json:"sku" binding:"required"`
			HubID    string    `json:"hub_id" binding:"required"`
			TenantID uuid.UUID `json:"tenant_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": constants.ErrInvalidRequest})
			return
		}

		// Get tenant ID from header if not in body
		if req.TenantID == uuid.Nil {
			tenantIDStr := c.GetHeader("tenant_id")
			if tenantIDStr == "" {
				c.JSON(400, gin.H{"error": "tenant_id header or field is required"})
				return
			}
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid tenant_id"})
				return
			}
			req.TenantID = tenantID
		}

		// Real validation logic using database
		ctx := c.Request.Context()

		// Validate Hub
		hub, err := hubService.GetHubByCode(ctx, req.TenantID, req.HubID)
		hubValid := err == nil && hub != nil && hub.IsActive

		// Validate SKU
		sku, err := skuService.GetSKUByCode(ctx, req.TenantID, req.SKU)
		skuValid := err == nil && sku != nil && sku.IsActive

		c.JSON(200, gin.H{
			"sku_valid": skuValid,
			"hub_valid": hubValid,
			"valid":     skuValid && hubValid,
			"message":   "Validation completed",
		})
	})

	// Batch validation endpoint
	api.POST("/validate/batch", func(c *gin.Context) {
		var req struct {
			TenantID uuid.UUID `json:"tenant_id"`
			Orders   []struct {
				SKU   string `json:"sku"`
				HubID string `json:"hub_id"`
			} `json:"orders"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": constants.ErrInvalidRequest})
			return
		}

		// Get tenant ID from header if not in body
		if req.TenantID == uuid.Nil {
			tenantIDStr := c.GetHeader("tenant_id")
			if tenantIDStr == "" {
				c.JSON(400, gin.H{"error": "tenant_id header or field is required"})
				return
			}
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid tenant_id"})
				return
			}
			req.TenantID = tenantID
		}

		ctx := c.Request.Context()
		results := make([]gin.H, len(req.Orders))

		for i, order := range req.Orders {
			// Validate Hub
			hub, err := hubService.GetHubByCode(ctx, req.TenantID, order.HubID)
			hubValid := err == nil && hub != nil && hub.IsActive

			// Validate SKU
			sku, err := skuService.GetSKUByCode(ctx, req.TenantID, order.SKU)
			skuValid := err == nil && sku != nil && sku.IsActive

			results[i] = gin.H{
				"sku":       order.SKU,
				"hub_id":    order.HubID,
				"sku_valid": skuValid,
				"hub_valid": hubValid,
				"valid":     skuValid && hubValid,
			}
		}

		c.JSON(200, gin.H{
			"message": "Batch validation completed",
			"results": results,
			"count":   len(results),
		})
	})

	// Basic routes
	server.GET("/health", func(c *gin.Context) {
		redisStatus := "disconnected"
		if redisClient != nil {
			if _, err := redisClient.Ping(context.Background()).Result(); err == nil {
				redisStatus = "connected"
			}
		}

		dbStatus := "disconnected"
		if config.DBCluster != nil {
			dbStatus = "connected"
		}

		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "ims-service",
			"timestamp": time.Now().Format(time.RFC3339),
			"redis":     redisStatus,
			"database":  dbStatus,
			"version":   "1.0.0",
		})
	})

	server.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "IMS Service is running with real PostgreSQL database!",
			"version": "1.0.0",
			"endpoints": []string{
				constants.EndpointHealth,
				constants.EndpointHubs,
				constants.EndpointSKUs,
				constants.EndpointInventory,
				constants.EndpointValidation,
			},
		})
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
	fmt.Println("========================")
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
