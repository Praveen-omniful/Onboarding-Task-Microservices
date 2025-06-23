package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/omniful/go_commons/db/sql/postgres"
	"github.com/omniful/ims-service/internal/models"
)

type SKURepository interface {
	Create(ctx context.Context, sku *models.SKU) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SKU, error)
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.SKU, error)
	List(ctx context.Context, filter models.SKUFilter) ([]models.SKU, int64, error)
	Update(ctx context.Context, sku *models.SKU) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type skuRepository struct {
	dbCluster *postgres.DbCluster
	redis     *redis.Client
}

func NewSKURepository(dbCluster *postgres.DbCluster, redis *redis.Client) SKURepository {
	return &skuRepository{
		dbCluster: dbCluster,
		redis:     redis,
	}
}

func (r *skuRepository) Create(ctx context.Context, sku *models.SKU) error {
	db := r.dbCluster.GetMasterDB(ctx)
	// Check if SKU with same code already exists for this tenant
	existing := &models.SKU{}
	if err := db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", sku.TenantID, sku.Code).
		First(existing).Error; err == nil {
		return fmt.Errorf("SKU with code %s already exists", sku.Code)
	} else if err.Error() != "record not found" {
		return fmt.Errorf("failed to check SKU existence: %w", err)
	}

	// Check if seller exists
	seller := &models.Seller{BaseModel: models.BaseModel{ID: sku.SellerID}}
	if err := db.WithContext(ctx).First(seller).Error; err != nil {
		if err.Error() == "record not found" {
			return fmt.Errorf("seller not found with id: %s", sku.SellerID)
		}
		return fmt.Errorf("failed to get seller: %w", err)
	}

	// Create SKU
	if err := db.WithContext(ctx).Create(sku).Error; err != nil {
		return fmt.Errorf("failed to create SKU: %w", err)
	}

	// Cache the SKU
	r.cacheSKU(sku)

	return nil
}

func (r *skuRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SKU, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("sku:%s", id.String())
	var sku models.SKU
	if err := r.redis.Get(ctx, cacheKey).Scan(&sku); err == nil {
		return &sku, nil
	}

	db := r.dbCluster.GetMasterDB(ctx)
	// Get from database
	sku = models.SKU{BaseModel: models.BaseModel{ID: id}}
	if err := db.WithContext(ctx).Preload("Seller").First(&sku).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("SKU not found with id: %s", id)
		}
		return nil, fmt.Errorf("failed to get SKU: %w", err)
	}

	// Cache the SKU
	r.cacheSKU(&sku)

	return &sku, nil
}

func (r *skuRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.SKU, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("sku:code:%s:%s", tenantID, code)
	var sku models.SKU
	if err := r.redis.Get(ctx, cacheKey).Scan(&sku); err == nil {
		return &sku, nil
	}

	db := r.dbCluster.GetMasterDB(ctx)
	// Get from database
	sku = models.SKU{}
	if err := db.WithContext(ctx).
		Where("tenant_id = ? AND code = ?", tenantID, code).
		Preload("Seller").
		First(&sku).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, fmt.Errorf("SKU not found with code: %s", code)
		}
		return nil, fmt.Errorf("failed to get SKU: %w", err)
	}

	// Cache the SKU
	r.cacheSKU(&sku)

	return &sku, nil
}

func (r *skuRepository) List(ctx context.Context, filter models.SKUFilter) ([]models.SKU, int64, error) {
	var skus []models.SKU
	var count int64

	db := r.dbCluster.GetMasterDB(ctx)
	// Build query
	query := db.WithContext(ctx).Model(&models.SKU{}).
		Where("tenant_id = ?", filter.TenantID)

	// Apply filters
	if filter.SellerID != uuid.Nil {
		query = query.Where("seller_id = ?", filter.SellerID)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	// Get total count
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count SKUs: %w", err)
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.
		Preload("Seller").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&skus).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list SKUs: %w", err)
	}

	return skus, count, nil
}

func (r *skuRepository) Update(ctx context.Context, sku *models.SKU) error {
	db := r.dbCluster.GetMasterDB(ctx)
	// Check if SKU exists
	existing := &models.SKU{}
	if err := db.WithContext(ctx).First(existing, sku.ID).Error; err != nil {
		if err.Error() == "record not found" {
			return fmt.Errorf("SKU not found with id: %s", sku.ID)
		}
		return fmt.Errorf("failed to get SKU: %w", err)
	}

	// Check if code is being changed and if the new code already exists
	if existing.Code != sku.Code {
		codeExists := &models.SKU{}
		if err := db.WithContext(ctx).
			Where("tenant_id = ? AND code = ? AND id != ?", sku.TenantID, sku.Code, sku.ID).
			First(codeExists).Error; err == nil {
			return fmt.Errorf("SKU with code %s already exists", sku.Code)
		} else if err.Error() != "record not found" {
			return fmt.Errorf("failed to check SKU code uniqueness: %w", err)
		}
	}

	// Check if seller exists if it's being updated
	if sku.SellerID != existing.SellerID {
		seller := &models.Seller{BaseModel: models.BaseModel{ID: sku.SellerID}}
		if err := db.WithContext(ctx).First(seller).Error; err != nil {
			if err.Error() == "record not found" {
				return fmt.Errorf("seller not found with id: %s", sku.SellerID)
			}
			return fmt.Errorf("failed to get seller: %w", err)
		}
	}

	// Update SKU
	if err := db.WithContext(ctx).Save(sku).Error; err != nil {
		return fmt.Errorf("failed to update SKU: %w", err)
	}

	// Update cache
	r.cacheSKU(sku)

	return nil
}

func (r *skuRepository) Delete(ctx context.Context, id uuid.UUID) error {
	db := r.dbCluster.GetMasterDB(ctx)
	// Check if SKU exists
	sku := &models.SKU{BaseModel: models.BaseModel{ID: id}}
	if err := db.WithContext(ctx).First(sku).Error; err != nil {
		if err.Error() == "record not found" {
			return fmt.Errorf("SKU not found with id: %s", id)
		}
		return fmt.Errorf("failed to get SKU: %w", err)
	}

	// Delete SKU
	if err := db.WithContext(ctx).Delete(sku).Error; err != nil {
		return fmt.Errorf("failed to delete SKU: %w", err)
	}

	// Delete from cache
	r.deleteSKUFromCache(sku.ID, sku.TenantID, sku.Code)

	return nil
}

func (r *skuRepository) cacheSKU(sku *models.SKU) {
	if sku == nil {
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("sku:%s", sku.ID.String())
	codeCacheKey := fmt.Sprintf("sku:code:%s:%s", sku.TenantID, sku.Code)

	// Cache for 1 hour
	r.redis.Set(ctx, cacheKey, sku, time.Hour)
	r.redis.Set(ctx, codeCacheKey, sku, time.Hour)
}

func (r *skuRepository) deleteSKUFromCache(id uuid.UUID, tenantID uuid.UUID, code string) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("sku:%s", id.String())
	codeCacheKey := fmt.Sprintf("sku:code:%s:%s", tenantID, code)

	r.redis.Del(ctx, cacheKey, codeCacheKey)
}
