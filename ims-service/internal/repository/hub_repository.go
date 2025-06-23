package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/omniful/go_commons/db/sql/postgres"
	"github.com/omniful/ims-service/internal/models"
	"gorm.io/gorm"
)

type HubRepository interface {
	Create(ctx context.Context, hub *models.Hub) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Hub, error)
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.Hub, error)
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Hub, int64, error)
	Update(ctx context.Context, hub *models.Hub) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type hubRepository struct {
	dbCluster *postgres.DbCluster
	redis     *redis.Client
}

func NewHubRepository(dbCluster *postgres.DbCluster, redis *redis.Client) HubRepository {
	return &hubRepository{
		dbCluster: dbCluster,
		redis:     redis,
	}
}

func (r *hubRepository) Create(ctx context.Context, hub *models.Hub) error {
	// Check if hub with same code already exists for this tenant
	existing := &models.Hub{}
	db := r.dbCluster.GetMasterDB(ctx)
	if err := db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", hub.TenantID, hub.Code).
		First(existing).Error; err == nil {
		return fmt.Errorf("hub with code %s already exists", hub.Code)
	} else if err.Error() != "record not found" {
		return fmt.Errorf("failed to check hub existence: %w", err)
	}

	// Create hub
	if err := db.WithContext(ctx).Create(hub).Error; err != nil {
		return fmt.Errorf("failed to create hub: %w", err)
	}

	// Cache the hub
	r.cacheHub(hub)

	return nil
}

func (r *hubRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Hub, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("hub:%s", id.String())
	var hub models.Hub
	if err := r.redis.Get(ctx, cacheKey).Scan(&hub); err == nil {
		return &hub, nil
	}

	// Get from database
	hub = models.Hub{BaseModel: models.BaseModel{ID: id}}
	db := r.dbCluster.GetMasterDB(ctx)
	if err := db.WithContext(ctx).First(&hub).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("hub not found with id: %s", id)
		}
		return nil, fmt.Errorf("failed to get hub: %w", err)
	}

	// Cache the hub
	r.cacheHub(&hub)

	return &hub, nil
}

func (r *hubRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.Hub, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("hub:code:%s:%s", tenantID, code)
	var hub models.Hub
	if err := r.redis.Get(ctx, cacheKey).Scan(&hub); err == nil {
		return &hub, nil
	}

	// Get from database
	db := r.dbCluster.GetMasterDB(ctx)
	hub = models.Hub{}
	if err := db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, code).
		First(&hub).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("hub not found with code: %s", code)
		}
		return nil, fmt.Errorf("failed to get hub: %w", err)
	}

	// Cache the hub
	r.cacheHub(&hub)

	return &hub, nil
}

func (r *hubRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Hub, int64, error) {
	var hubs []models.Hub
	var count int64
	db := r.dbCluster.GetMasterDB(ctx)
	// Get total count
	if err := db.WithContext(ctx).Model(&models.Hub{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count hubs: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Offset(offset).
		Limit(pageSize).
		Find(&hubs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list hubs: %w", err)
	}

	return hubs, count, nil
}

func (r *hubRepository) Update(ctx context.Context, hub *models.Hub) error {
	// Check if hub exists
	existing := &models.Hub{}
	if err := r.dbCluster.GetMasterDB(ctx).WithContext(ctx).First(existing, hub.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("hub not found with id: %s", hub.ID)
		}
		return fmt.Errorf("failed to get hub: %w", err)
	}

	// Check if code is being changed and if the new code already exists
	if existing.Code != hub.Code {
		codeExists := &models.Hub{}
		if err := r.dbCluster.GetMasterDB(ctx).WithContext(ctx).
			Where("tenant_id = ? AND code = ? AND id != ?", hub.TenantID, hub.Code, hub.ID).
			First(codeExists).Error; err == nil {
			return fmt.Errorf("hub with code %s already exists", hub.Code)
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to check hub code uniqueness: %w", err)
		}
	}

	// Update hub
	if err := r.dbCluster.GetMasterDB(ctx).WithContext(ctx).Save(hub).Error; err != nil {
		return fmt.Errorf("failed to update hub: %w", err)
	}

	// Update cache
	r.cacheHub(hub)

	return nil
}

func (r *hubRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if hub exists
	hub := &models.Hub{BaseModel: models.BaseModel{ID: id}}
	if err := r.dbCluster.GetMasterDB(ctx).WithContext(ctx).First(hub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("hub not found with id: %s", id)
		}
		return fmt.Errorf("failed to get hub: %w", err)
	}

	// Delete hub
	if err := r.dbCluster.GetMasterDB(ctx).WithContext(ctx).Delete(hub).Error; err != nil {
		return fmt.Errorf("failed to delete hub: %w", err)
	}

	// Delete from cache
	r.deleteHubFromCache(hub.ID, hub.TenantID, hub.Code)

	return nil
}

func (r *hubRepository) cacheHub(hub *models.Hub) {
	if hub == nil {
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("hub:%s", hub.ID.String())
	codeCacheKey := fmt.Sprintf("hub:code:%s:%s", hub.TenantID, hub.Code)

	// Cache for 1 hour
	r.redis.Set(ctx, cacheKey, hub, time.Hour)
	r.redis.Set(ctx, codeCacheKey, hub, time.Hour)
}

func (r *hubRepository) deleteHubFromCache(id uuid.UUID, tenantID uuid.UUID, code string) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("hub:%s", id.String())
	codeCacheKey := fmt.Sprintf("hub:code:%s:%s", tenantID, code)

	r.redis.Del(ctx, cacheKey, codeCacheKey)
}
