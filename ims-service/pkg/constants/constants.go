package constants

const (
	// Context keys
	ContextKeyRequestID = "request_id"
	ContextKeyUserID    = "user_id"
	ContextKeyTenantID  = "tenant_id"

	// API Response Messages
	MsgHubCreated    = "Hub created successfully"
	MsgHubUpdated    = "Hub updated successfully"
	MsgHubDeleted    = "Hub deleted successfully"
	MsgHubRetrieved  = "Hub retrieved successfully"
	MsgHubsRetrieved = "Hubs retrieved successfully"

	MsgSKUCreated    = "SKU created successfully"
	MsgSKUUpdated    = "SKU updated successfully"
	MsgSKUDeleted    = "SKU deleted successfully"
	MsgSKURetrieved  = "SKU retrieved successfully"
	MsgSKUsRetrieved = "SKUs retrieved successfully"

	MsgInventoryCreated   = "Inventory created successfully"
	MsgInventoryUpdated   = "Inventory updated successfully"
	MsgInventoryRetrieved = "Inventory retrieved successfully"

	// Error Messages
	ErrInvalidRequest     = "Invalid request data"
	ErrHubNotFound        = "Hub not found"
	ErrSKUNotFound        = "SKU not found"
	ErrInventoryNotFound  = "Inventory not found"
	ErrDatabaseConnection = "Database connection error"
	ErrRedisConnection    = "Redis connection error"
	ErrInternalServer     = "Internal server error"

	// Cache keys
	CacheKeyHubPrefix       = "hub:"
	CacheKeySKUPrefix       = "sku:"
	CacheKeyTenantHubs      = "tenant:hubs:%s"
	CacheKeyHubInventory    = "hub:inventory:%s"
	CacheKeyInventoryPrefix = "inventory:"
	CacheKeyHubList         = "hubs:list"
	CacheKeySKUList         = "skus:list"

	// Cache TTLs (in seconds)
	CacheTTLHubInfo    = 3600 // 1 hour
	CacheTTLSKUInfo    = 3600 // 1 hour
	CacheTTLInventory  = 300  // 5 minutes
	CacheTTLShortLived = 60   // 1 minute
	CacheTTLShort      = 300  // 5 minutes
	CacheTTLMedium     = 1800 // 30 minutes
	CacheTTLLong       = 3600 // 1 hour

	// Status Constants
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusDeleted  = "deleted"

	// Default Values
	DefaultInventoryQuantity = 0
	DefaultPageSize          = 20
	MaxPageSize              = 100
	DefaultPage              = 1

	// API Endpoints
	EndpointHealth     = "/health"
	EndpointHubs       = "/api/v1/hubs"
	EndpointSKUs       = "/api/v1/skus"
	EndpointInventory  = "/api/v1/inventory"
	EndpointValidation = "/api/v1/validate"

	// Validation
	MaxNameLength        = 255
	MaxDescriptionLength = 1000
	MaxCodeLength        = 100

	// Additional Error messages
	ErrRecordNotFound   = "record not found"
	ErrDuplicateEntry   = "duplicate entry"
	ErrUnauthorized     = "unauthorized"
	ErrForbidden        = "forbidden"
	ErrValidationFailed = "validation failed"
)
