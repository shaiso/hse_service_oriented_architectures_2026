package model

import (
	"github.com/google/uuid"
	"time"
)

type DiscountType string

const (
	DiscountPercentage DiscountType = "PERCENTAGE"

	DiscountFixedAmount DiscountType = "FIXED_AMOUNT"
)

type PromoCode struct {
	ID             uuid.UUID    `json:"id"`
	Code           string       `json:"code"`
	DiscountType   DiscountType `json:"discount_type"`
	DiscountValue  float64      `json:"discount_value"`
	MinOrderAmount float64      `json:"min_order_amount"`
	MaxUses        int          `json:"max_uses"`
	CurrentUses    int          `json:"current_uses"`
	ValidFrom      time.Time    `json:"valid_from"`
	ValidUntil     time.Time    `json:"valid_until"`
	Active         bool         `json:"active"`
}
