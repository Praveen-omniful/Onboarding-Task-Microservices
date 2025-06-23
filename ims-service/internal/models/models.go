package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type Tenant struct {
	BaseModel
	Name        string `gorm:"not null;size:255" json:"name"`
	Code        string `gorm:"uniqueIndex;not null;size:100" json:"code"`
	Description string `gorm:"size:1000" json:"description"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`
}

type Hub struct {
	BaseModel
	TenantID    uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Name        string    `gorm:"not null;size:255" json:"name"`
	Code        string    `gorm:"uniqueIndex:idx_hub_code_tenant;not null;size:100" json:"code"`
	Description string    `gorm:"size:1000" json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	Address     string    `gorm:"size:500" json:"address"`
	City        string    `gorm:"size:100" json:"city"`
	State       string    `gorm:"size:100" json:"state"`
	Country     string    `gorm:"size:100" json:"country"`
	PostalCode  string    `gorm:"size:20" json:"postal_code"`
}

type Seller struct {
	BaseModel
	TenantID uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Name     string    `gorm:"not null;size:255" json:"name"`
	Code     string    `gorm:"uniqueIndex:idx_seller_code_tenant;not null;size:100" json:"code"`
	Email    string    `gorm:"size:255" json:"email"`
	Phone    string    `gorm:"size:50" json:"phone"`
	IsActive bool      `gorm:"default:true" json:"is_active"`
}

type SKU struct {
	BaseModel
	TenantID      uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	SellerID      uuid.UUID `gorm:"type:uuid;not null;index" json:"seller_id"`
	Name          string    `gorm:"not null;size:255" json:"name"`
	Code          string    `gorm:"uniqueIndex:idx_sku_code_tenant;not null;size:100" json:"code"`
	Description   string    `gorm:"size:1000" json:"description"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	Barcode       string    `gorm:"size:100" json:"barcode"`
	Seller        Seller    `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Weight        float64   `gorm:"default:0" json:"weight,omitempty"`
	WeightUnit    string    `gorm:"size:20" json:"weight_unit,omitempty"`
	Length        float64   `gorm:"default:0" json:"length,omitempty"`
	Width         float64   `gorm:"default:0" json:"width,omitempty"`
	Height        float64   `gorm:"default:0" json:"height,omitempty"`
	DimensionUnit string    `gorm:"size:20" json:"dimension_unit,omitempty"`
}

type Inventory struct {
	BaseModel
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	HubID     uuid.UUID `gorm:"type:uuid;not null;index" json:"hub_id"`
	SkuID     uuid.UUID `gorm:"type:uuid;not null;index" json:"sku_id"`
	Quantity  int       `gorm:"not null;default:0" json:"quantity"`
	Available int       `gorm:"not null;default:0" json:"available"`
	Reserved  int       `gorm:"not null;default:0" json:"reserved"`
	InTransit int       `gorm:"not null;default:0" json:"in_transit"`
	SKU       SKU       `gorm:"foreignKey:SkuID" json:"sku,omitempty"`
	Hub       Hub       `gorm:"foreignKey:HubID" json:"hub,omitempty"`
}

// InventoryUpdate represents a single inventory update operation
type InventoryUpdate struct {
	HubCode  string `json:"hub_code" validate:"required"`
	SkuCode  string `json:"sku_code" validate:"required"`
	Quantity int    `json:"quantity"`
}

// InventoryFilter represents the filter criteria for inventory queries
type SKUFilter struct {
	TenantID uuid.UUID
	SellerID uuid.UUID
	IsActive *bool
	Page     int
	PageSize int
}

type InventoryFilter struct {
	TenantID string
	HubCode  string
	SellerID string
	SkuCodes []string
	Page     int
	PageSize int
}
