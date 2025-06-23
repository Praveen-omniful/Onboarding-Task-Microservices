package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omniful/ims-service/internal/api/handlers"
	"github.com/omniful/ims-service/internal/config"
	"github.com/omniful/ims-service/internal/repository"
	"github.com/omniful/ims-service/internal/service"
)

// setupRoutes configures all the routes for the application
func setupRoutes(router *gin.RouterGroup) {
	// Initialize repositories
	hubRepo := repository.NewHubRepository(config.DBCluster, dbRedisClient)
	skuRepo := repository.NewSKURepository(config.DBCluster, dbRedisClient)
	inventoryRepo := repository.NewInventoryRepository(config.DBCluster, hubRepo, skuRepo, dbRedisClient)

	// Initialize services
	hubService := service.NewHubService(hubRepo)
	skuService := service.NewSKUService(skuRepo)
	inventoryService := service.NewInventoryService(inventoryRepo, hubRepo, skuRepo)

	// Initialize handlers
	hubHandler := handlers.NewHubHandler(hubService)
	skuHandler := handlers.NewSKUHandler(skuService)
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Register routes
	hubHandler.RegisterRoutes(router)
	skuHandler.RegisterRoutes(router)
	inventoryHandler.RegisterRoutes(router)
}
