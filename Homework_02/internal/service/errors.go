package service

import "fmt"

const (
	ErrCodeProductNotFound         = "PRODUCT_NOT_FOUND"
	ErrCodeProductInactive         = "PRODUCT_INACTIVE"
	ErrCodeOrderNotFound           = "ORDER_NOT_FOUND"
	ErrCodeOrderLimitExceeded      = "ORDER_LIMIT_EXCEEDED"
	ErrCodeOrderHasActive          = "ORDER_HAS_ACTIVE"
	ErrCodeInvalidStateTransition  = "INVALID_STATE_TRANSITION"
	ErrCodeInsufficientStock       = "INSUFFICIENT_STOCK"
	ErrCodePromoCodeInvalid        = "PROMO_CODE_INVALID"
	ErrCodePromoCodeMinAmount      = "PROMO_CODE_MIN_AMOUNT"
	ErrCodeOrderOwnershipViolation = "ORDER_OWNERSHIP_VIOLATION"
	ErrCodeValidationError         = "VALIDATION_ERROR"
)

type BusinessError struct {
	Code    string
	Message string
	Status  int
	Details map[string]interface{}
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
