package model

import (
	"github.com/google/uuid"
	"time"
)

type ProductStatus string

const (
	ProductStatusActive ProductStatus = "ACTIVE"

	ProductStatusInactive ProductStatus = "INACTIVE"

	ProductStatusArchived ProductStatus = "ARCHIVED"
)

type Product struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	Price       float64       `json:"price"`
	Stock       int           `json:"stock"`
	Category    string        `json:"category"`
	Status      ProductStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	SellerID    uuid.UUID     `json:"seller_id"`
}
