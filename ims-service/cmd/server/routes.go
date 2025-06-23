package main

import (
	"github.com/gin-gonic/gin"
	logger "github.com/omniful/go_commons/log"
	"github.com/omniful/ims-service/internal/api/handlers"
	"github.com/omniful/ims-service/internal/config"
	"github.com/omniful/ims-service/internal/repository"
	"github.com/omniful/ims-service/internal/service"
)

// setupRoutes configures all the routes for the application
func setupRoutes(router *gin.RouterGroup) {
	logger.Info("Setting up routes...")

	// Try to initialize repositories - if this fails, we'll see it in logs
	if config.DBCluster == nil {
		logger.Error("DBCluster is nil - database not initialized properly")
		return
	}

	if dbRedisClient == nil {
		logger.Error("Redis client is nil")
		return
	}

	logger.Info("Database and Redis clients are available")

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

	// Register routes
	logger.Info("Registering hub routes...")
	hubHandler.RegisterRoutes(router)
	logger.Info("Registering SKU routes...")
	skuHandler.RegisterRoutes(router)
	logger.Info("Registering inventory routes...")
	inventoryHandler.RegisterRoutes(router)

	logger.Info("All routes registered successfully")
}
