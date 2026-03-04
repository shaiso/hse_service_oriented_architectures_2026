package handler

import (
	"context"
	"errors"

	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/service"
)

type PromoCodeHandler struct {
	promoService *service.PromoCodeService
}

func NewPromoCodeHandler(promoService *service.PromoCodeService) *PromoCodeHandler {
	return &PromoCodeHandler{promoService: promoService}
}

func (h *PromoCodeHandler) PostPromoCodes(ctx context.Context, request generated.PostPromoCodesRequestObject) (generated.PostPromoCodesResponseObject, error) {
	result, err := h.promoService.Create(ctx, *request.Body)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			details := mapToDetails(validationErr.Fields)
			return generated.PostPromoCodes400JSONResponse{
				ErrorCode: service.ErrCodeValidationError,
				Message:   "Validation failed",
				Details:   &details,
			}, nil
		}
		return generated.PostPromoCodes500JSONResponse{
			ErrorCode: "INTERNAL_ERROR",
			Message:   err.Error(),
		}, nil
	}
	return generated.PostPromoCodes201JSONResponse(*result), nil
}
