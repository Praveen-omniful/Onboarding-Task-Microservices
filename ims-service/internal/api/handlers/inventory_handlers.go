package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omniful/ims-service/internal/models"
	"github.com/omniful/ims-service/internal/service"
)

type InventoryHandler struct {
	service service.InventoryService
}

func NewInventoryHandler(service service.InventoryService) *InventoryHandler {
	return &InventoryHandler{service: service}
}

func (h *InventoryHandler) RegisterRoutes(r *gin.RouterGroup) {
	inv := r.Group("/inventory")
	{
		inv.POST("/", h.UpsertInventory)
		inv.GET("/", h.GetInventory)
		inv.GET("/:hubCode/:skuCode", h.GetInventoryItem)
		inv.POST("/reserve", h.ReserveInventory)
		inv.POST("/release", h.ReleaseInventory)
		inv.POST("/fulfill", h.FulfillInventory)
	}
}

type UpsertInventoryRequest struct {
	Updates []models.InventoryUpdate `json:"updates"`
}

// UpsertInventory handles inventory updates
// @Summary Update or insert inventory
// @Description Update existing inventory or insert new records if they don't exist
// @Tags inventory
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param request body UpsertInventoryRequest true "Inventory updates"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /inventory [post]
func (h *InventoryHandler) UpsertInventory(c *gin.Context) {
	// Get tenant ID from header
	tenantIDStr := c.GetHeader("X-Tenant-ID")
	if tenantIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant ID"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	// Parse request body
	var req UpsertInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate updates
	if len(req.Updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updates provided"})
		return
	}

	// Process updates
	if err := h.service.UpsertInventory(c.Request.Context(), tenantID, req.Updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "inventory updated successfully",
	})
}

// GetInventory retrieves inventory based on filters
// @Summary Get inventory
// @Description Get inventory with optional filtering by hub, seller, and SKU codes
// @Tags inventory
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param hub_code query string false "Filter by hub code"
// @Param seller_id query string false "Filter by seller ID"
// @Param sku_codes query string false "Comma-separated list of SKU codes to filter by"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Number of items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /inventory [get]
func (h *InventoryHandler) GetInventory(c *gin.Context) {
	// Get tenant ID from header
	tenantIDStr := c.GetHeader("X-Tenant-ID")
	if tenantIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing tenant ID"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
		return
	}

	// Parse query parameters
	query := c.Request.URL.Query()

	// Parse pagination parameters
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	switch {
	case pageSize > 100:
		pageSize = 100
	case pageSize <= 0:
		pageSize = 20
	}

	// Parse SKU codes
	var skuCodes []string
	if skuCodesStr := query.Get("sku_codes"); skuCodesStr != "" {
		// Simple split by comma, in production you might want more robust parsing
		for _, code := range strings.Split(skuCodesStr, ",") {
			if code != "" {
				skuCodes = append(skuCodes, strings.TrimSpace(code))
			}
		}
	}

	// Build filter
	filter := models.InventoryFilter{
		TenantID: tenantID.String(),
		HubCode:  query.Get("hub_code"),
		SellerID: query.Get("seller_id"),
		SkuCodes: skuCodes,
		Page:     page,
		PageSize: pageSize,
	}

	// Get inventory
	inventories, total, err := h.service.GetInventory(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"data": inventories,
		"pagination": map[string]interface{}{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pages":     (int(total) + pageSize - 1) / pageSize,
		},
	}

	// Return response
	c.JSON(http.StatusOK, response)
}

// GetInventoryItem retrieves a single inventory item by hub and SKU codes
// @Summary Get inventory item
// @Description Get a specific inventory item by hub code and SKU code
// @Tags inventory
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param hubCode path string true "Hub code"
// @Param skuCode path string true "SKU code"
// @Success 200 {object} models.Inventory
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /inventory/{hubCode}/{skuCode} [get]
func (h *InventoryHandler) GetInventoryItem(c *gin.Context) {
	// Get tenant ID from header
	tenantID, err := getTenantID(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get path parameters
	hubCode := c.Param("hubCode")
	skuCode := c.Param("skuCode")

	// Get inventory item
	inventory, err := h.service.GetInventoryItem(c.Request.Context(), tenantID, hubCode, skuCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, inventory)
}

// ReserveInventoryRequest represents the request body for reserving inventory
type ReserveInventoryRequest struct {
	HubCode  string `json:"hub_code"`
	SkuCode  string `json:"sku_code"`
	Quantity int    `json:"quantity"`
}

// ReserveInventory reserves inventory for an order
// @Summary Reserve inventory
// @Description Reserve a specific quantity of inventory for an order
// @Tags inventory
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param request body ReserveInventoryRequest true "Reservation details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /inventory/reserve [post]
func (h *InventoryHandler) ReserveInventory(c *gin.Context) {
	// Get tenant ID from header
	tenantID, err := getTenantID(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse request body
	var req ReserveInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate request
	if req.HubCode == "" || req.SkuCode == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub_code, sku_code, and positive quantity are required"})
		return
	}

	// Reserve inventory
	err = h.service.ReserveInventory(c.Request.Context(), tenantID, req.HubCode, req.SkuCode, req.Quantity)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "insufficient available quantity") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Successfully reserved %d of %s at hub %s", req.Quantity, req.SkuCode, req.HubCode),
	})
}

// ReleaseInventoryRequest represents the request body for releasing inventory
type ReleaseInventoryRequest struct {
	HubCode  string `json:"hub_code"`
	SkuCode  string `json:"sku_code"`
	Quantity int    `json:"quantity"`
}

// ReleaseInventory releases previously reserved inventory
// @Summary Release inventory
// @Description Release previously reserved inventory
// @Tags inventory
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param request body ReleaseInventoryRequest true "Release details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /inventory/release [post]
func (h *InventoryHandler) ReleaseInventory(c *gin.Context) {
	// Get tenant ID from header
	tenantID, err := getTenantID(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse request body
	var req ReleaseInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate request
	if req.HubCode == "" || req.SkuCode == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub_code, sku_code, and positive quantity are required"})
		return
	}

	// Release inventory
	err = h.service.ReleaseInventory(c.Request.Context(), tenantID, req.HubCode, req.SkuCode, req.Quantity)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "insufficient reserved quantity") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Successfully released %d of %s at hub %s", req.Quantity, req.SkuCode, req.HubCode),
	})
}

// FulfillInventoryRequest represents the request body for fulfilling inventory
type FulfillInventoryRequest struct {
	HubCode  string `json:"hub_code"`
	SkuCode  string `json:"sku_code"`
	Quantity int    `json:"quantity"`
}

// FulfillInventory marks inventory as fulfilled
// @Summary Fulfill inventory
// @Description Mark reserved inventory as fulfilled and update quantities
// @Tags inventory
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param request body FulfillInventoryRequest true "Fulfillment details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /inventory/fulfill [post]
func (h *InventoryHandler) FulfillInventory(c *gin.Context) {
	// Get tenant ID from header
	tenantID, err := getTenantID(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse request body
	var req FulfillInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Validate request
	if req.HubCode == "" || req.SkuCode == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hub_code, sku_code, and positive quantity are required"})
		return
	}

	// Fulfill inventory
	err = h.service.FulfillInventory(c.Request.Context(), tenantID, req.HubCode, req.SkuCode, req.Quantity)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "insufficient reserved quantity") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Successfully fulfilled %d of %s at hub %s", req.Quantity, req.SkuCode, req.HubCode),
	})
}

// Helper function to get tenant ID from request header
func getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		return uuid.Nil, fmt.Errorf("missing tenant ID")
	}
	return uuid.Parse(tenantIDStr)
}
