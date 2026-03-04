package model

import (
	"github.com/google/uuid"
	"time"
)

type OrderStatus string

const (
	OrderStatusCreated OrderStatus = "CREATED"

	OrderStatusPaymentPending OrderStatus = "PAYMENT_PENDING"

	OrderStatusPaid OrderStatus = "PAID"

	OrderStatusShipped OrderStatus = "SHIPPED"

	OrderStatusCompleted OrderStatus = "COMPLETED"

	OrderStatusCanceled OrderStatus = "CANCELED"
)

type Order struct {
	ID             uuid.UUID   `json:"id"`
	UserID         uuid.UUID   `json:"user_id"`
	Status         OrderStatus `json:"status"`
	PromoCodeID    *uuid.UUID  `json:"promo_code_id"`
	TotalAmount    float64     `json:"total_amount"`
	DiscountAmount float64     `json:"discount_amount"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID           uuid.UUID `json:"id"`
	OrderID      uuid.UUID `json:"order_id"`
	ProductID    uuid.UUID `json:"product_id"`
	Quantity     int       `json:"quantity"`
	PriceAtOrder float64   `json:"price_at_order"`
}
