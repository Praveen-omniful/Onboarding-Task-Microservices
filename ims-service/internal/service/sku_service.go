package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/omniful/ims-service/internal/models"
	"github.com/omniful/ims-service/internal/repository"
)

type SKUService interface {
	CreateSKU(ctx context.Context, sku *models.SKU) error
	GetSKU(ctx context.Context, id uuid.UUID) (*models.SKU, error)
	GetSKUByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.SKU, error)
	ListSKUs(ctx context.Context, filter models.SKUFilter) ([]models.SKU, int64, error)
	UpdateSKU(ctx context.Context, sku *models.SKU) error
	DeleteSKU(ctx context.Context, id uuid.UUID) error
}

type skuService struct {
	skuRepo repository.SKURepository
}

func NewSKUService(skuRepo repository.SKURepository) SKUService {
	return &skuService{
		skuRepo: skuRepo,
	}
}

func (s *skuService) CreateSKU(ctx context.Context, sku *models.SKU) error {
	// Validate input
	if sku.TenantID == uuid.Nil {
		return fmt.Errorf("tenant ID is required")
	}
	if sku.SellerID == uuid.Nil {
		return fmt.Errorf("seller ID is required")
	}
	if sku.Code == "" {
		return fmt.Errorf("SKU code is required")
	}
	if sku.Name == "" {
		return fmt.Errorf("SKU name is required")
	}

	// Create SKU
	return s.skuRepo.Create(ctx, sku)
}

func (s *skuService) GetSKU(ctx context.Context, id uuid.UUID) (*models.SKU, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("SKU ID is required")
	}

	return s.skuRepo.GetByID(ctx, id)
}

func (s *skuService) GetSKUByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.SKU, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant ID is required")
	}
	if code == "" {
		return nil, fmt.Errorf("SKU code is required")
	}

	return s.skuRepo.GetByCode(ctx, tenantID, code)
}

func (s *skuService) ListSKUs(ctx context.Context, filter models.SKUFilter) ([]models.SKU, int64, error) {
	if filter.TenantID == uuid.Nil {
		return nil, 0, fmt.Errorf("tenant ID is required")
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	return s.skuRepo.List(ctx, filter)
}

func (s *skuService) UpdateSKU(ctx context.Context, sku *models.SKU) error {
	if sku.ID == uuid.Nil {
		return fmt.Errorf("SKU ID is required")
	}
	if sku.TenantID == uuid.Nil {
		return fmt.Errorf("tenant ID is required")
	}
	if sku.SellerID == uuid.Nil {
		return fmt.Errorf("seller ID is required")
	}
	if sku.Code == "" {
		return fmt.Errorf("SKU code is required")
	}
	if sku.Name == "" {
		return fmt.Errorf("SKU name is required")
	}

	// Update SKU
	return s.skuRepo.Update(ctx, sku)
}

func (s *skuService) DeleteSKU(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("SKU ID is required")
	}

	return s.skuRepo.Delete(ctx, id)
}
