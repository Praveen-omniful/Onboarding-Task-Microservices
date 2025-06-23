package constants

const (
	// Context keys
	ContextKeyRequestID = "request_id"
	ContextKeyUserID    = "user_id"
	ContextKeyTenantID  = "tenant_id"

	// Cache keys
	CacheKeyHubPrefix    = "hub:"
	CacheKeySKUPrefix    = "sku:"
	CacheKeyTenantHubs   = "tenant:hubs:%s"
	CacheKeyHubInventory = "hub:inventory:%s"

	// Cache TTLs (in seconds)
	CacheTTLHubInfo    = 3600    // 1 hour
	CacheTTLSKUInfo    = 3600    // 1 hour
	CacheTTLInventory  = 300     // 5 minutes
	CacheTTLShortLived = 60      // 1 minute

	// Pagination defaults
	DefaultPageSize = 20
	MaxPageSize     = 100

	// Validation
	MaxNameLength     = 255
	MaxDescriptionLength = 1000
	MaxCodeLength     = 100

	// Error messages
	ErrInvalidRequest     = "invalid request"
	ErrRecordNotFound     = "record not found"
	ErrDuplicateEntry     = "duplicate entry"
	ErrUnauthorized      = "unauthorized"
	ErrForbidden         = "forbidden"
	ErrInternalServer    = "internal server error"
	ErrValidationFailed  = "validation failed"
)
