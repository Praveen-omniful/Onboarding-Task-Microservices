package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/omniful/ims-service/internal/models"
	"github.com/omniful/ims-service/internal/repository"
)

type InventoryService interface {
	// UpsertInventory atomically updates or inserts inventory records in batches
	UpsertInventory(ctx context.Context, tenantID uuid.UUID, updates []models.InventoryUpdate) error
	// GetInventory retrieves inventory with filtering and pagination
	GetInventory(ctx context.Context, filter models.InventoryFilter) ([]models.Inventory, int64, error)
	// GetInventoryItem retrieves a single inventory item by hub and SKU codes
	GetInventoryItem(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string) (*models.Inventory, error)
	// ReserveInventory reserves the specified quantity of an inventory item
	ReserveInventory(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, quantity int) error
	// ReleaseInventory releases the specified quantity of a reserved inventory item
	ReleaseInventory(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, quantity int) error
	// FulfillInventory marks the specified quantity as fulfilled and updates available quantity
	FulfillInventory(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, quantity int) error
}

type inventoryService struct {
	repo    repository.InventoryRepository
	hubRepo repository.HubRepository
	skuRepo repository.SKURepository
}

func NewInventoryService(
	repo repository.InventoryRepository,
	hubRepo repository.HubRepository,
	skuRepo repository.SKURepository,
) InventoryService {
	return &inventoryService{
		repo:    repo,
		hubRepo: hubRepo,
		skuRepo: skuRepo,
	}
}

func (s *inventoryService) UpsertInventory(ctx context.Context, tenantID uuid.UUID, updates []models.InventoryUpdate) error {
	// Validate input
	if len(updates) == 0 {
		return fmt.Errorf("no inventory updates provided")
	}

	// Validate each update
	for i, update := range updates {
		if update.HubCode == "" {
			return fmt.Errorf("hub code is required for update at index %d", i)
		}
		if update.SkuCode == "" {
			return fmt.Errorf("SKU code is required for update at index %d", i)
		}
		if update.Quantity < 0 {
			return fmt.Errorf("quantity cannot be negative for update at index %d", i)
		}

		updates[i] = update
	}

	// Process updates in batches
	batchSize := 100
	for i := 0; i < len(updates); i += batchSize {
		end := i + batchSize
		if end > len(updates) {
			end = len(updates)
		}

		batch := updates[i:end]
		if err := s.repo.UpsertInventory(ctx, tenantID, batch); err != nil {
			return fmt.Errorf("failed to upsert inventory batch %d-%d: %w", i, end, err)
		}
	}

	return nil
}

func (s *inventoryService) GetInventory(ctx context.Context, filter models.InventoryFilter) ([]models.Inventory, int64, error) {
	// Set default pagination values
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	// Convert filter.TenantID from string to uuid.UUID
	tenantUUID, err := uuid.Parse(filter.TenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid tenant ID: %v", err)
	}

	// Get inventory from repository
	inventories, total, err := s.repo.GetInventory(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// If specific SKUs were requested but not all were found, create zero-quantity entries for missing SKUs
	if len(filter.SkuCodes) > 0 {
		// Get unique requested SKU codes
		skuSet := make(map[string]bool)
		for _, code := range filter.SkuCodes {
			skuSet[code] = true
		}

		// Remove found SKUs from the set
		for _, inv := range inventories {
			if inv.SKU.ID != uuid.Nil {
				delete(skuSet, inv.SKU.Code)
			}
		}

		// If we have remaining SKUs that weren't found, create zero-quantity entries
		if len(skuSet) > 0 {
			// Get hub ID if filtering by hub
			var hubID *uuid.UUID
			if filter.HubCode != "" {
				hub, err := s.hubRepo.GetByCode(ctx, tenantUUID, filter.HubCode)
				if err == nil {
					hubID = &hub.ID
				}
			}

			// Get SKU details for missing SKUs
			for skuCode := range skuSet {
				sku, err := s.skuRepo.GetByCode(ctx, tenantUUID, skuCode)
				if err != nil {
					continue // Skip if SKU not found
				}

				// Create zero-quantity inventory
				zeroInv := models.Inventory{
					TenantID:  tenantUUID,
					HubID:     uuid.Nil, // or *hubID if not nil
					SkuID:     sku.ID,
					SKU:       *sku,
					Quantity:  0,
					Available: 0,
					Reserved:  0,
					InTransit: 0,
				}

				// If we have a hub filter, set the hub details
				if hubID != nil && filter.HubCode != "" {
					hub, _ := s.hubRepo.GetByCode(ctx, tenantUUID, filter.HubCode)
					zeroInv.Hub = *hub
				}

				inventories = append(inventories, zeroInv)
			}
		}
	}

	return inventories, total, nil
}

func (s *inventoryService) GetInventoryItem(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string) (*models.Inventory, error) {
	// Validate inputs
	if hubCode == "" {
		return nil, errors.New("hub code is required")
	}
	if skuCode == "" {
		return nil, errors.New("SKU code is required")
	}

	// Get the inventory item
	inventory, err := s.repo.GetInventoryItem(ctx, tenantID, hubCode, skuCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory item: %w", err)
	}

	// If inventory is nil, return a zero-quantity inventory
	if inventory == nil {
		hub, err := s.hubRepo.GetByCode(ctx, tenantID, hubCode)
		if err != nil {
			return nil, fmt.Errorf("invalid hub code: %w", err)
		}

		sku, err := s.skuRepo.GetByCode(ctx, tenantID, skuCode)
		if err != nil {
			return nil, fmt.Errorf("invalid SKU code: %w", err)
		}

		inventory = &models.Inventory{
			TenantID:  tenantID,
			HubID:     hub.ID,
			SkuID:     sku.ID,
			Hub:       *hub,
			SKU:       *sku,
			Quantity:  0,
			Available: 0,
			Reserved:  0,
			InTransit: 0,
		}
	}

	return inventory, nil
}

func (s *inventoryService) ReserveInventory(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, quantity int) error {
	// Validate inputs
	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	// Get current inventory
	inventory, err := s.GetInventoryItem(ctx, tenantID, hubCode, skuCode)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Check if there's enough available quantity
	if inventory.Available < quantity {
		return fmt.Errorf("insufficient available quantity. available: %d, requested: %d",
			inventory.Available, quantity)
	}

	// Update reserved quantity
	if err := s.repo.UpdateReservedQuantity(ctx, tenantID, hubCode, skuCode, quantity); err != nil {
		return fmt.Errorf("failed to reserve inventory: %w", err)
	}

	// Update available quantity
	if err := s.repo.UpdateAvailableQuantity(ctx, tenantID, hubCode, skuCode, -quantity); err != nil {
		// Try to rollback the reservation if available quantity update fails
		_ = s.repo.UpdateReservedQuantity(ctx, tenantID, hubCode, skuCode, -quantity)
		return fmt.Errorf("failed to update available quantity: %w", err)
	}

	return nil
}

func (s *inventoryService) ReleaseInventory(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, quantity int) error {
	// Validate inputs
	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	// Get current inventory
	inventory, err := s.GetInventoryItem(ctx, tenantID, hubCode, skuCode)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Check if there's enough reserved quantity
	if inventory.Reserved < quantity {
		return fmt.Errorf("insufficient reserved quantity. reserved: %d, requested: %d",
			inventory.Reserved, quantity)
	}

	// Update reserved quantity
	if err := s.repo.UpdateReservedQuantity(ctx, tenantID, hubCode, skuCode, -quantity); err != nil {
		return fmt.Errorf("failed to release inventory: %w", err)
	}

	// Update available quantity
	if err := s.repo.UpdateAvailableQuantity(ctx, tenantID, hubCode, skuCode, quantity); err != nil {
		// Try to rollback the release if available quantity update fails
		_ = s.repo.UpdateReservedQuantity(ctx, tenantID, hubCode, skuCode, quantity)
		return fmt.Errorf("failed to update available quantity: %w", err)
	}

	return nil
}

func (s *inventoryService) FulfillInventory(ctx context.Context, tenantID uuid.UUID, hubCode, skuCode string, quantity int) error {
	// Validate inputs
	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}

	// Get current inventory
	inventory, err := s.GetInventoryItem(ctx, tenantID, hubCode, skuCode)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Check if there's enough reserved quantity
	if inventory.Reserved < quantity {
		return fmt.Errorf("insufficient reserved quantity. reserved: %d, requested: %d",
			inventory.Reserved, quantity)
	}

	// Update reserved quantity
	if err := s.repo.UpdateReservedQuantity(ctx, tenantID, hubCode, skuCode, -quantity); err != nil {
		return fmt.Errorf("failed to update reserved quantity: %w", err)
	}

	// Update total quantity
	if err := s.repo.UpsertInventory(ctx, tenantID, []models.InventoryUpdate{{
		HubCode:  hubCode,
		SkuCode:  skuCode,
		Quantity: inventory.Quantity - quantity,
	}}); err != nil {
		// Try to rollback the reserved quantity update
		_ = s.repo.UpdateReservedQuantity(ctx, tenantID, hubCode, skuCode, quantity)
		return fmt.Errorf("failed to update total quantity: %w", err)
	}

	return nil
}
