package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omniful/ims-service/internal/models"
	"github.com/omniful/ims-service/internal/service"
)

type SKUHandler struct {
	service service.SKUService
}

func NewSKUHandler(service service.SKUService) *SKUHandler {
	return &SKUHandler{service: service}
}

func (h *SKUHandler) RegisterRoutes(r *gin.RouterGroup) {
	skus := r.Group("/skus")
	skus.POST("/", h.CreateSKU)
	skus.GET("/", h.ListSKUs)
	skus.GET("/:id", h.GetSKU)
	skus.GET("/code/:code", h.GetSKUByCode)
	skus.PUT("/:id", h.UpdateSKU)
	skus.DELETE("/:id", h.DeleteSKU)
}

// CreateSKURequest represents the request body for creating a SKU
type CreateSKURequest struct {
	Code          string  `json:"code" validate:"required"`
	Name          string  `json:"name" validate:"required"`
	Description   string  `json:"description,omitempty"`
	SellerID      string  `json:"seller_id" validate:"required"`
	IsActive      bool    `json:"is_active"`
	Weight        float64 `json:"weight,omitempty"`
	WeightUnit    string  `json:"weight_unit,omitempty"`
	Length        float64 `json:"length,omitempty"`
	Width         float64 `json:"width,omitempty"`
	Height        float64 `json:"height,omitempty"`
	DimensionUnit string  `json:"dimension_unit,omitempty"`
}

// CreateSKU creates a new SKU
// @Summary Create a new SKU
// @Description Create a new SKU with the provided details
// @Tags skus
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param request body CreateSKURequest true "SKU details"
// @Success 201 {object} models.SKU
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /skus [post]
func (h *SKUHandler) CreateSKU(c *gin.Context) {
	tenantIDStr := c.GetHeader("tenant_id")
	if tenantIDStr == "" {
		c.JSON(400, gin.H{"error": "tenant_id header is required"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid tenant_id"})
		return
	}

	var req CreateSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	sellerID, err := uuid.Parse(req.SellerID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid seller ID"})
		return
	}

	sku := &models.SKU{
		TenantID:      tenantID,
		SellerID:      sellerID,
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		IsActive:      req.IsActive,
		Weight:        req.Weight,
		WeightUnit:    req.WeightUnit,
		Length:        req.Length,
		Width:         req.Width,
		Height:        req.Height,
		DimensionUnit: req.DimensionUnit,
	}

	if err := h.service.CreateSKU(c.Request.Context(), sku); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, sku)
}

// ListSKUs lists all SKUs for a tenant with pagination and filtering
// @Summary List all SKUs
// @Description Get a paginated and filtered list of SKUs for the tenant
// @Tags skus
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param seller_id query string false "Filter by seller ID"
// @Param is_active query boolean false "Filter by active status"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /skus [get]
func (h *SKUHandler) ListSKUs(c *gin.Context) {
	// Get tenant ID from header
	tenantID, err := getTenantID(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse pagination parameters
	page, pageSize := getPaginationParams(c)

	// Parse filters
	filter := models.SKUFilter{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
	}

	// Parse seller ID filter
	if sellerIDStr := c.Query("seller_id"); sellerIDStr != "" {
		sellerID, err := uuid.Parse(sellerIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid seller ID"})
			return
		}
		filter.SellerID = sellerID
	}

	// Parse is_active filter
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid is_active value"})
			return
		}
		filter.IsActive = &isActive
	}

	// List SKUs
	skus, total, err := h.service.ListSKUs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"data": skus,
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

// GetSKU gets a SKU by ID
// @Summary Get a SKU by ID
// @Description Get a SKU by its ID
// @Tags skus
// @Accept json
// @Produce json
// @Param id path string true "SKU ID"
// @Success 200 {object} models.SKU
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /skus/{id} [get]
func (h *SKUHandler) GetSKU(c *gin.Context) {
	// Get SKU ID from URL
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SKU ID"})
		return
	}

	// Get SKU
	sku, err := h.service.GetSKU(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Return SKU
	c.JSON(http.StatusOK, sku)
}

// GetSKUByCode gets a SKU by code
// @Summary Get a SKU by code
// @Description Get a SKU by its code
// @Tags skus
// @Accept json
// @Produce json
// @Param code path string true "SKU Code"
// @Success 200 {object} models.SKU
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /skus/code/{code} [get]
func (h *SKUHandler) GetSKUByCode(c *gin.Context) {
	// Get tenant ID from header
	tenantID, err := getTenantID(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get SKU code from URL
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SKU code is required"})
		return
	}

	// Get SKU by code
	sku, err := h.service.GetSKUByCode(c.Request.Context(), tenantID, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Return SKU
	c.JSON(http.StatusOK, sku)
}

// UpdateSKURequest represents the request body for updating a SKU
type UpdateSKURequest struct {
	Code          *string  `json:"code,omitempty"`
	Name          *string  `json:"name,omitempty"`
	Description   *string  `json:"description,omitempty"`
	SellerID      *string  `json:"seller_id,omitempty"`
	IsActive      *bool    `json:"is_active,omitempty"`
	Weight        *float64 `json:"weight,omitempty"`
	WeightUnit    *string  `json:"weight_unit,omitempty"`
	Length        *float64 `json:"length,omitempty"`
	Width         *float64 `json:"width,omitempty"`
	Height        *float64 `json:"height,omitempty"`
	DimensionUnit *string  `json:"dimension_unit,omitempty"`
}

// UpdateSKU updates a SKU
// @Summary Update a SKU
// @Description Update an existing SKU
// @Tags skus
// @Accept json
// @Produce json
// @Param id path string true "SKU ID"
// @Param request body UpdateSKURequest true "SKU details"
// @Success 200 {object} models.SKU
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /skus/{id} [put]
func (h *SKUHandler) UpdateSKU(c *gin.Context) {
	// Get SKU ID from URL
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SKU ID"})
		return
	}

	// Get existing SKU
	sku, err := h.service.GetSKU(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Parse request body
	var req UpdateSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Update fields if provided
	if req.Code != nil {
		sku.Code = *req.Code
	}
	if req.Name != nil {
		sku.Name = *req.Name
	}
	if req.Description != nil {
		sku.Description = *req.Description
	}
	if req.SellerID != nil {
		sellerID, err := uuid.Parse(*req.SellerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid seller ID"})
			return
		}
		sku.SellerID = sellerID
	}
	if req.IsActive != nil {
		sku.IsActive = *req.IsActive
	}
	if req.Weight != nil {
		sku.Weight = *req.Weight
	}
	if req.WeightUnit != nil {
		sku.WeightUnit = *req.WeightUnit
	}
	if req.Length != nil {
		sku.Length = *req.Length
	}
	if req.Width != nil {
		sku.Width = *req.Width
	}
	if req.Height != nil {
		sku.Height = *req.Height
	}
	if req.DimensionUnit != nil {
		sku.DimensionUnit = *req.DimensionUnit
	}

	// Update SKU
	if err := h.service.UpdateSKU(c.Request.Context(), sku); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return updated SKU
	c.JSON(http.StatusOK, sku)
}

// DeleteSKU deletes a SKU
// @Summary Delete a SKU
// @Description Delete a SKU by its ID
// @Tags skus
// @Accept json
// @Produce json
// @Param id path string true "SKU ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /skus/{id} [delete]
func (h *SKUHandler) DeleteSKU(c *gin.Context) {
	// Get SKU ID from URL
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid SKU ID"})
		return
	}

	// Delete SKU
	if err := h.service.DeleteSKU(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return no content
	c.Status(http.StatusNoContent)
}

func getPaginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
