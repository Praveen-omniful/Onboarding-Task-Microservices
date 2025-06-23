package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/omniful/go_commons/db/sql/postgres"
	"github.com/omniful/ims-service/internal/models"
	"gorm.io/gorm"
)

type InventoryRepository interface {
	// UpsertInventory atomically updates or inserts inventory records
	UpsertInventory(ctx context.Context, tenantID uuid.UUID, updates []models.InventoryUpdate) error
	// GetInventory retrieves inventory with filtering and pagination
	GetInventory(ctx context.Context, filter models.InventoryFilter) ([]models.Inventory, int64, error)
	// GetInventoryItem retrieves a single inventory item by hub and SKU codes
	GetInventoryItem(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string) (*models.Inventory, error)
	// UpdateAvailableQuantity updates the available quantity of an inventory item
	UpdateAvailableQuantity(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, delta int) error
	// UpdateReservedQuantity updates the reserved quantity of an inventory item
	UpdateReservedQuantity(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, delta int) error
	// UpdateInTransitQuantity updates the in-transit quantity of an inventory item
	UpdateInTransitQuantity(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, delta int) error
	// GetInventoryWithLock gets an inventory item with a row lock for update
	GetInventoryWithLock(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string) (*models.Inventory, error)
}

type inventoryRepository struct {
	dbCluster *postgres.DbCluster
	hubRepo   HubRepository
	skuRepo   SKURepository
	redis     *redis.Client
}

func NewInventoryRepository(dbCluster *postgres.DbCluster, hubRepo HubRepository, skuRepo SKURepository, redis *redis.Client) InventoryRepository {
	return &inventoryRepository{
		dbCluster: dbCluster,
		hubRepo:   hubRepo,
		skuRepo:   skuRepo,
		redis:     redis,
	}
}

func (r *inventoryRepository) UpsertInventory(ctx context.Context, tenantID uuid.UUID, updates []models.InventoryUpdate) error {
	db := r.dbCluster.GetMasterDB(ctx)
	tx := db.Begin().WithContext(ctx)
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Process each update
	for _, update := range updates {
		// Get hub and SKU using repositories (with caching)
		hub, err := r.hubRepo.GetByCode(ctx, tenantID, update.HubCode)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("invalid hub code %s: %w", update.HubCode, err)
		}

		sku, err := r.skuRepo.GetByCode(ctx, tenantID, update.SkuCode)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("invalid SKU code %s: %w", update.SkuCode, err)
		}

		// Try to update existing record
		result := tx.Model(&models.Inventory{}).
			Where("tenant_id = ? AND hub_id = ? AND sku_id = ?", tenantID, hub.ID, sku.ID).
			Update("quantity", update.Quantity)

		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update inventory: %w", result.Error)
		}

		// If no rows were updated, insert new record
		if result.RowsAffected == 0 {
			inv := models.Inventory{
				TenantID: tenantID,
				HubID:    hub.ID,
				SkuID:    sku.ID,
				Quantity: update.Quantity,
			}
			if err := tx.Create(&inv).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to insert inventory: %w", err)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *inventoryRepository) GetInventory(ctx context.Context, filter models.InventoryFilter) ([]models.Inventory, int64, error) {
	var (
		inventories []models.Inventory
		total       int64
	)

	// Try to get from cache if no filters except pagination
	if filter.HubCode == "" && filter.SellerID == "" && len(filter.SkuCodes) == 0 {
		cacheKey := fmt.Sprintf("inventory:tenant:%s:page:%d:size:%d", filter.TenantID, filter.Page, filter.PageSize)
		if err := r.getFromCache(ctx, cacheKey, &inventories); err == nil {
			// Get total count from cache or database
			countKey := fmt.Sprintf("inventory:tenant:%s:count", filter.TenantID)
			if err := r.redis.Get(ctx, countKey).Scan(&total); err != nil {
				// If count not in cache, get it from database
				db := r.dbCluster.GetMasterDB(ctx)
				if err := db.Model(&models.Inventory{}).
					Where("tenant_id = ?", filter.TenantID).
					Count(&total).Error; err != nil {
					return nil, 0, fmt.Errorf("failed to count inventory: %w", err)
				}
				// Cache the count
				r.redis.Set(ctx, countKey, total, time.Hour*24)
			}
			return inventories, total, nil
		}
	}

	// Build base query
	db := r.dbCluster.GetMasterDB(ctx)
	query := db.Model(&models.Inventory{}).Where("inventories.tenant_id = ?", filter.TenantID)

	// Apply filters
	if filter.HubCode != "" {
		tenantUUID, err := uuid.Parse(filter.TenantID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid tenant ID: %v", err)
		}
		hub, err := r.hubRepo.GetByCode(ctx, tenantUUID, filter.HubCode)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid hub code: %w", err)
		}
		query = query.Where("hub_id = ?", hub.ID)
	}

	if filter.SellerID != "" {
		sellerID, err := uuid.Parse(filter.SellerID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid seller ID: %v", err)
		}
		query = query.Joins("JOIN skus ON skus.id = inventories.sku_id AND skus.seller_id = ?", sellerID)
	}

	if len(filter.SkuCodes) > 0 {
		query = query.Joins("JOIN skus ON skus.id = inventories.sku_id AND skus.code IN ?", filter.SkuCodes)
	}

	// Count total matching records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count inventory: %w", err)
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Preload related data
	query = query.Preload("SKU").Preload("Hub")

	// Execute query
	if err := query.Find(&inventories).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch inventory: %w", err)
	}

	// Cache the results if no filters applied
	if filter.HubCode == "" && filter.SellerID == "" && len(filter.SkuCodes) == 0 {
		cacheKey := fmt.Sprintf("inventory:tenant:%s:page:%d:size:%d", filter.TenantID, filter.Page, filter.PageSize)
		r.cacheSet(ctx, cacheKey, inventories, time.Minute*5)
		// Update count cache
		countKey := fmt.Sprintf("inventory:tenant:%s:count", filter.TenantID)
		r.redis.Set(ctx, countKey, total, time.Hour*24)
	}

	return inventories, total, nil
}

func (r *inventoryRepository) GetInventoryItem(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string) (*models.Inventory, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("inventory:tenant:%s:hub:%s:sku:%s", tenantID, hubCode, skuCode)
	var inventory models.Inventory
	if err := r.getFromCache(ctx, cacheKey, &inventory); err == nil {
		return &inventory, nil
	}

	// Get from database
	hub, err := r.hubRepo.GetByCode(ctx, tenantID, hubCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get hub: %w", err)
	}

	sku, err := r.skuRepo.GetByCode(ctx, tenantID, skuCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get SKU: %w", err)
	}

	var inv models.Inventory
	err = r.dbCluster.GetMasterDB(ctx).
		Where("tenant_id = ? AND hub_id = ? AND sku_id = ?", tenantID, hub.ID, sku.ID).
		First(&inv).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return zero inventory if not found
			return &models.Inventory{
				TenantID:  tenantID,
				HubID:     hub.ID,
				SkuID:     sku.ID,
				Quantity:  0,
				Available: 0,
				Reserved:  0,
				InTransit: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Cache the result
	r.cacheSet(ctx, cacheKey, inv, time.Minute*5)

	return &inv, nil
}

func (r *inventoryRepository) UpdateAvailableQuantity(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, delta int) error {
	// Get inventory item
	inv, err := r.GetInventoryItem(ctx, tenantID, hubCode, skuCode)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Update available quantity atomically
	result := r.dbCluster.GetMasterDB(ctx).Model(&models.Inventory{}).
		Where("tenant_id = ? AND hub_id = ? AND sku_id = ?",
			tenantID, inv.HubID, inv.SkuID).
		Updates(map[string]interface{}{
			"available": gorm.Expr("available + ?", delta),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update available quantity: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("inventory record is out of date, please refresh and try again")
	}

	// Invalidate cache
	r.invalidateCache(ctx, tenantID, hubCode, skuCode)

	return nil
}

func (r *inventoryRepository) UpdateReservedQuantity(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, delta int) error {
	// Get inventory item
	inv, err := r.GetInventoryItem(ctx, tenantID, hubCode, skuCode)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Update reserved quantity atomically
	result := r.dbCluster.GetMasterDB(ctx).Model(&models.Inventory{}).
		Where("tenant_id = ? AND hub_id = ? AND sku_id = ?",
			tenantID, inv.HubID, inv.SkuID).
		Updates(map[string]interface{}{
			"reserved": gorm.Expr("reserved + ?", delta),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update reserved quantity: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("inventory record is out of date, please refresh and try again")
	}

	// Invalidate cache
	r.invalidateCache(ctx, tenantID, hubCode, skuCode)

	return nil
}

// Helper methods for caching
func (r *inventoryRepository) getFromCache(ctx context.Context, key string, dest interface{}) error {
	val, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (r *inventoryRepository) cacheSet(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.redis.Set(ctx, key, val, ttl).Err()
}

func (r *inventoryRepository) invalidateCache(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string) {
	// Invalidate specific item cache
	itemKey := fmt.Sprintf("inventory:tenant:%s:hub:%s:sku:%s", tenantID, hubCode, skuCode)
	r.redis.Del(ctx, itemKey)

	// Invalidate list caches (simplified - in production you might want to be more granular)
	pattern := fmt.Sprintf("inventory:tenant:%s:*", tenantID)
	keys, _ := r.redis.Keys(ctx, pattern).Result()
	if len(keys) > 0 {
		r.redis.Del(ctx, keys...)
	}
}

func (r *inventoryRepository) UpdateInTransitQuantity(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, delta int) error {
	// Get inventory item
	inv, err := r.GetInventoryItem(ctx, tenantID, hubCode, skuCode)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Update in-transit quantity atomically
	result := r.dbCluster.GetMasterDB(ctx).Model(&models.Inventory{}).
		Where("tenant_id = ? AND hub_id = ? AND sku_id = ?",
			tenantID, inv.HubID, inv.SkuID).
		Updates(map[string]interface{}{
			"in_transit": gorm.Expr("in_transit + ?", delta),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update in-transit quantity: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("inventory record is out of date, please refresh and try again")
	}

	// Invalidate cache
	r.invalidateCache(ctx, tenantID, hubCode, skuCode)

	return nil
}

func (r *inventoryRepository) GetInventoryWithLock(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string) (*models.Inventory, error) {
	// Get hub and SKU
	hub, err := r.hubRepo.GetByCode(ctx, tenantID, hubCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get hub: %w", err)
	}

	sku, err := r.skuRepo.GetByCode(ctx, tenantID, skuCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get SKU: %w", err)
	}

	var inv models.Inventory
	err = r.dbCluster.GetMasterDB(ctx).
		Set("gorm:query_option", "FOR UPDATE").
		Where("tenant_id = ? AND hub_id = ? AND sku_id = ?", tenantID, hub.ID, sku.ID).
		First(&inv).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return zero inventory if not found
			return &models.Inventory{
				TenantID:  tenantID,
				HubID:     hub.ID,
				SkuID:     sku.ID,
				Quantity:  0,
				Available: 0,
				Reserved:  0,
				InTransit: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get inventory with lock: %w", err)
	}

	return &inv, nil
}
