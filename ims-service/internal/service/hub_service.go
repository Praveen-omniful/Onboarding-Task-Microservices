package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/omniful/ims-service/internal/models"
	"github.com/omniful/ims-service/internal/repository"
)

type HubService interface {
	CreateHub(ctx context.Context, hub *models.Hub) error
	GetHub(ctx context.Context, id uuid.UUID) (*models.Hub, error)
	GetHubByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.Hub, error)
	ListHubs(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Hub, int64, error)
	UpdateHub(ctx context.Context, hub *models.Hub) error
	DeleteHub(ctx context.Context, id uuid.UUID) error
}

type hubService struct {
	hubRepo repository.HubRepository
}

func NewHubService(hubRepo repository.HubRepository) HubService {
	return &hubService{
		hubRepo: hubRepo,
	}
}

func (s *hubService) CreateHub(ctx context.Context, hub *models.Hub) error {
	// Validate input
	if hub.TenantID == uuid.Nil {
		return fmt.Errorf("tenant ID is required")
	}
	if hub.Code == "" {
		return fmt.Errorf("hub code is required")
	}
	if hub.Name == "" {
		return fmt.Errorf("hub name is required")
	}

	// Create hub
	return s.hubRepo.Create(ctx, hub)
}

func (s *hubService) GetHub(ctx context.Context, id uuid.UUID) (*models.Hub, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("hub ID is required")
	}

	return s.hubRepo.GetByID(ctx, id)
}

func (s *hubService) GetHubByCode(ctx context.Context, tenantID uuid.UUID, code string) (*models.Hub, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant ID is required")
	}
	if code == "" {
		return nil, fmt.Errorf("hub code is required")
	}

	return s.hubRepo.GetByCode(ctx, tenantID, code)
}

func (s *hubService) ListHubs(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]models.Hub, int64, error) {
	if tenantID == uuid.Nil {
		return nil, 0, fmt.Errorf("tenant ID is required")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return s.hubRepo.List(ctx, tenantID, page, pageSize)
}

func (s *hubService) UpdateHub(ctx context.Context, hub *models.Hub) error {
	if hub.ID == uuid.Nil {
		return fmt.Errorf("hub ID is required")
	}
	if hub.TenantID == uuid.Nil {
		return fmt.Errorf("tenant ID is required")
	}
	if hub.Code == "" {
		return fmt.Errorf("hub code is required")
	}
	if hub.Name == "" {
		return fmt.Errorf("hub name is required")
	}

	// Update hub
	return s.hubRepo.Update(ctx, hub)
}

func (s *hubService) DeleteHub(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("hub ID is required")
	}

	return s.hubRepo.Delete(ctx, id)
}
