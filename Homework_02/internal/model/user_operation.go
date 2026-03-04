package model

import (
	"time"

	"github.com/google/uuid"
)

type OperationType string

const (
	OperationCreateOrder OperationType = "CREATE_ORDER"
	OperationUpdateOrder OperationType = "UPDATE_ORDER"
)

type UserOperation struct {
	ID            uuid.UUID     `json:"id"`
	UserID        uuid.UUID     `json:"user_id"`
	OperationType OperationType `json:"operation_type"`
	CreatedAt     time.Time     `json:"created_at"`
}
