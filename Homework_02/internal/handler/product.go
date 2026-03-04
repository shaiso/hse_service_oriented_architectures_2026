package handler

import (
	"context"
	"errors"

	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/middleware"
	"github.com/shaiso/marketplace/internal/model"
	"github.com/shaiso/marketplace/internal/repo"
	"github.com/shaiso/marketplace/internal/service"
)

type ProductHandler struct {
	service *service.ProductService
}

func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) GetProductsId(ctx context.Context, request generated.GetProductsIdRequestObject) (generated.GetProductsIdResponseObject, error) {
	result, err := h.service.GetByID(ctx, request.Id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return generated.GetProductsId404JSONResponse{
				ErrorCode: "PRODUCT_NOT_FOUND",
				Message:   "Product not found",
			}, nil
		}
		return generated.GetProductsId500JSONResponse{
			ErrorCode: "INTERNAL_ERROR",
			Message:   err.Error(),
		}, nil
	}
	return generated.GetProductsId200JSONResponse(*result), nil
}

func (h *ProductHandler) PostProducts(ctx context.Context, request generated.PostProductsRequestObject) (generated.PostProductsResponseObject, error) {
	// SELLER устанавливает seller_id = свой user_id
	userID, _ := middleware.GetUserID(ctx)

	result, err := h.service.CreateWithSeller(ctx, *request.Body, userID)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			details := mapToDetails(validationErr.Fields)
			return generated.PostProducts400JSONResponse{
				ErrorCode: "VALIDATION_ERROR",
				Message:   "Validation failed",
				Details:   &details,
			}, nil
		}
		return generated.PostProducts500JSONResponse{
			ErrorCode: "INTERNAL_ERROR",
			Message:   err.Error(),
		}, nil
	}
	return generated.PostProducts201JSONResponse(*result), nil
}

func (h *ProductHandler) DeleteProductsId(ctx context.Context, request generated.DeleteProductsIdRequestObject) (generated.DeleteProductsIdResponseObject, error) {
	userID, role := middleware.GetUserID(ctx)
	userRole, _ := middleware.GetRole(ctx)
	_ = role

	// SELLER может удалять только свои товары
	if userRole == model.RoleSeller {
		if err := h.service.CheckOwnership(ctx, request.Id, userID); err != nil {
			return generated.DeleteProductsId404JSONResponse{
				ErrorCode: "ACCESS_DENIED",
				Message:   "you can only manage your own products",
			}, nil
		}
	}

	result, err := h.service.Delete(ctx, request.Id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return generated.DeleteProductsId404JSONResponse{
				ErrorCode: "PRODUCT_NOT_FOUND",
				Message:   "Product not found",
			}, nil
		}
		return generated.DeleteProductsId500JSONResponse{
			ErrorCode: "INTERNAL_ERROR",
			Message:   err.Error(),
		}, nil
	}
	return generated.DeleteProductsId200JSONResponse(*result), nil
}

func (h *ProductHandler) GetProducts(ctx context.Context, request generated.GetProductsRequestObject) (generated.GetProductsResponseObject, error) {
	result, err := h.service.List(ctx, request.Params)
	if err != nil {
		return generated.GetProducts500JSONResponse{
			ErrorCode: "INTERNAL_ERROR",
			Message:   err.Error(),
		}, nil
	}
	return generated.GetProducts200JSONResponse(*result), nil
}

func (h *ProductHandler) PutProductsId(ctx context.Context, request generated.PutProductsIdRequestObject) (generated.PutProductsIdResponseObject, error) {
	userID, _ := middleware.GetUserID(ctx)
	userRole, _ := middleware.GetRole(ctx)

	// SELLER может обновлять только свои товары
	if userRole == model.RoleSeller {
		if err := h.service.CheckOwnership(ctx, request.Id, userID); err != nil {
			return generated.PutProductsId404JSONResponse{
				ErrorCode: "ACCESS_DENIED",
				Message:   "you can only manage your own products",
			}, nil
		}
	}

	result, err := h.service.Update(ctx, request.Id, *request.Body)
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			details := mapToDetails(validationErr.Fields)
			return generated.PutProductsId400JSONResponse{
				ErrorCode: "VALIDATION_ERROR",
				Message:   "Validation failed",
				Details:   &details,
			}, nil
		}
		if errors.Is(err, repo.ErrNotFound) {
			return generated.PutProductsId404JSONResponse{
				ErrorCode: "PRODUCT_NOT_FOUND",
				Message:   "Product not found",
			}, nil
		}
		return generated.PutProductsId500JSONResponse{
			ErrorCode: "INTERNAL_ERROR",
			Message:   err.Error(),
		}, nil
	}
	return generated.PutProductsId200JSONResponse(*result), nil
}

func mapToDetails(fields map[string]string) map[string]interface{} {
	details := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		details[k] = v
	}
	return details
}
