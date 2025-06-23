package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omniful/ims-service/internal/models"
	"github.com/omniful/ims-service/internal/service"
)

type HubHandler struct {
	service service.HubService
}

func NewHubHandler(service service.HubService) *HubHandler {
	return &HubHandler{service: service}
}

func (h *HubHandler) RegisterRoutes(r *gin.RouterGroup) {
	hubs := r.Group("/hubs")
	hubs.POST("/", h.CreateHub)
	hubs.GET("/", h.ListHubs)
	hubs.GET("/:id", h.GetHub)
	hubs.PUT("/:id", h.UpdateHub)
	hubs.DELETE("/:id", h.DeleteHub)
}

// CreateHubRequest represents the request body for creating a hub
type CreateHubRequest struct {
	Code        string `json:"code" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	Address     string `json:"address,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Country     string `json:"country,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
}

// CreateHub creates a new hub
// @Summary Create a new hub
// @Description Create a new hub with the provided details
// @Tags hubs
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param request body CreateHubRequest true "Hub details"
// @Success 201 {object} models.Hub
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /hubs [post]
func (h *HubHandler) CreateHub(c *gin.Context) {
	// Get tenant ID from header
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

	// Parse request body
	var req CreateHubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	// Create hub model
	hub := &models.Hub{
		TenantID:    tenantID,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		Address:     req.Address,
		City:        req.City,
		State:       req.State,
		Country:     req.Country,
		PostalCode:  req.PostalCode,
	}

	// Create hub
	if err := h.service.CreateHub(c.Request.Context(), hub); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, hub)
}

// ListHubs lists all hubs for a tenant with pagination
// @Summary List all hubs
// @Description Get a paginated list of hubs for the tenant
// @Tags hubs
// @Accept json
// @Produce json
// @Param tenant_id header string true "Tenant ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /hubs [get]
func (h *HubHandler) ListHubs(c *gin.Context) {
	// Get tenant ID from header
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

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// List hubs
	hubs, total, err := h.service.ListHubs(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"data": hubs,
		"pagination": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pages":     (int(total) + pageSize - 1) / pageSize,
		},
	}
	c.JSON(200, response)
}

// GetHub gets a hub by ID
// @Summary Get a hub by ID
// @Description Get a hub by its ID
// @Tags hubs
// @Accept json
// @Produce json
// @Param id path string true "Hub ID"
// @Success 200 {object} models.Hub
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /hubs/{id} [get]
func (h *HubHandler) GetHub(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid hub ID"})
		return
	}

	hub, err := h.service.GetHub(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, hub)
}

// UpdateHubRequest represents the request body for updating a hub
type UpdateHubRequest struct {
	Code        *string `json:"code,omitempty"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
	Address     *string `json:"address,omitempty"`
	City        *string `json:"city,omitempty"`
	State       *string `json:"state,omitempty"`
	Country     *string `json:"country,omitempty"`
	PostalCode  *string `json:"postal_code,omitempty"`
}

// UpdateHub updates a hub
// @Summary Update a hub
// @Description Update an existing hub
// @Tags hubs
// @Accept json
// @Produce json
// @Param id path string true "Hub ID"
// @Param request body UpdateHubRequest true "Hub details"
// @Success 200 {object} models.Hub
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /hubs/{id} [put]
func (h *HubHandler) UpdateHub(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid hub ID"})
		return
	}

	hub, err := h.service.GetHub(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	var req UpdateHubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	if req.Code != nil {
		hub.Code = *req.Code
	}
	if req.Name != nil {
		hub.Name = *req.Name
	}
	if req.Description != nil {
		hub.Description = *req.Description
	}
	if req.IsActive != nil {
		hub.IsActive = *req.IsActive
	}
	if req.Address != nil {
		hub.Address = *req.Address
	}
	if req.City != nil {
		hub.City = *req.City
	}
	if req.State != nil {
		hub.State = *req.State
	}
	if req.Country != nil {
		hub.Country = *req.Country
	}
	if req.PostalCode != nil {
		hub.PostalCode = *req.PostalCode
	}

	if err := h.service.UpdateHub(c.Request.Context(), hub); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, hub)
}

// DeleteHub deletes a hub
// @Summary Delete a hub
// @Description Delete a hub by its ID
// @Tags hubs
// @Accept json
// @Produce json
// @Param id path string true "Hub ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /hubs/{id} [delete]
func (h *HubHandler) DeleteHub(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid hub ID"})
		return
	}

	if err := h.service.DeleteHub(c.Request.Context(), id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Status(204)
}
