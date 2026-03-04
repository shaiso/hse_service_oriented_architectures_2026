package handler

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/middleware"
	"github.com/shaiso/marketplace/internal/model"
	"github.com/shaiso/marketplace/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) PostOrders(ctx context.Context, request generated.PostOrdersRequestObject) (generated.PostOrdersResponseObject, error) {
	userID, role := getUserFromContext(ctx)
	if userID == uuid.Nil {
		return generated.PostOrders500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: "user not found in context"}, nil
	}

	// ADMIN может создавать заказы от своего имени
	_ = role

	result, err := h.orderService.CreateOrder(ctx, userID, *request.Body)
	if err != nil {
		return handleOrderCreateError(err)
	}
	return generated.PostOrders201JSONResponse(*result), nil
}

func (h *OrderHandler) GetOrdersId(ctx context.Context, request generated.GetOrdersIdRequestObject) (generated.GetOrdersIdResponseObject, error) {
	userID, role := getUserFromContext(ctx)

	// ADMIN может видеть любые заказы — передаём специальный uuid.Nil
	requestUserID := userID
	if role == model.RoleAdmin {
		requestUserID = uuid.Nil
	}

	result, err := h.orderService.GetOrder(ctx, requestUserID, request.Id)
	if err != nil {
		var bizErr *service.BusinessError
		if errors.As(err, &bizErr) {
			switch bizErr.Status {
			case 403:
				return generated.GetOrdersId403JSONResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}, nil
			case 404:
				return generated.GetOrdersId404JSONResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}, nil
			}
		}
		return generated.GetOrdersId500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: err.Error()}, nil
	}
	return generated.GetOrdersId200JSONResponse(*result), nil
}

func (h *OrderHandler) PutOrdersId(ctx context.Context, request generated.PutOrdersIdRequestObject) (generated.PutOrdersIdResponseObject, error) {
	userID, role := getUserFromContext(ctx)

	requestUserID := userID
	if role == model.RoleAdmin {
		requestUserID = uuid.Nil
	}

	result, err := h.orderService.UpdateOrder(ctx, requestUserID, request.Id, *request.Body)
	if err != nil {
		return handleOrderUpdateError(err)
	}
	return generated.PutOrdersId200JSONResponse(*result), nil
}

func (h *OrderHandler) PostOrdersIdCancel(ctx context.Context, request generated.PostOrdersIdCancelRequestObject) (generated.PostOrdersIdCancelResponseObject, error) {
	userID, role := getUserFromContext(ctx)

	requestUserID := userID
	if role == model.RoleAdmin {
		requestUserID = uuid.Nil
	}

	result, err := h.orderService.CancelOrder(ctx, requestUserID, request.Id)
	if err != nil {
		var bizErr *service.BusinessError
		if errors.As(err, &bizErr) {
			resp := generated.ErrorResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}
			if bizErr.Details != nil {
				details := toInterfaceMap(bizErr.Details)
				resp.Details = &details
			}
			switch bizErr.Status {
			case 403:
				return generated.PostOrdersIdCancel403JSONResponse(resp), nil
			case 404:
				return generated.PostOrdersIdCancel404JSONResponse(resp), nil
			case 409:
				return generated.PostOrdersIdCancel409JSONResponse(resp), nil
			}
		}
		return generated.PostOrdersIdCancel500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: err.Error()}, nil
	}
	return generated.PostOrdersIdCancel200JSONResponse(*result), nil
}

func handleOrderCreateError(err error) (generated.PostOrdersResponseObject, error) {
	var validationErr *service.ValidationError
	if errors.As(err, &validationErr) {
		details := mapToDetails(validationErr.Fields)
		return generated.PostOrders400JSONResponse{
			ErrorCode: service.ErrCodeValidationError,
			Message:   "Validation failed",
			Details:   &details,
		}, nil
	}

	var bizErr *service.BusinessError
	if errors.As(err, &bizErr) {
		resp := generated.ErrorResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}
		if bizErr.Details != nil {
			details := toInterfaceMap(bizErr.Details)
			resp.Details = &details
		}
		switch bizErr.Status {
		case 409:
			return generated.PostOrders409JSONResponse(resp), nil
		case 422:
			return generated.PostOrders422JSONResponse(resp), nil
		case 429:
			return generated.PostOrders429JSONResponse(resp), nil
		}
	}

	return generated.PostOrders500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: err.Error()}, nil
}

func handleOrderUpdateError(err error) (generated.PutOrdersIdResponseObject, error) {
	var validationErr *service.ValidationError
	if errors.As(err, &validationErr) {
		details := mapToDetails(validationErr.Fields)
		return generated.PutOrdersId400JSONResponse{
			ErrorCode: service.ErrCodeValidationError,
			Message:   "Validation failed",
			Details:   &details,
		}, nil
	}

	var bizErr *service.BusinessError
	if errors.As(err, &bizErr) {
		resp := generated.ErrorResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}
		if bizErr.Details != nil {
			details := toInterfaceMap(bizErr.Details)
			resp.Details = &details
		}
		switch bizErr.Status {
		case 403:
			return generated.PutOrdersId403JSONResponse(resp), nil
		case 404:
			return generated.PutOrdersId404JSONResponse(resp), nil
		case 409:
			return generated.PutOrdersId409JSONResponse(resp), nil
		case 422:
			return generated.PutOrdersId422JSONResponse(resp), nil
		case 429:
			return generated.PutOrdersId429JSONResponse(resp), nil
		}
	}

	return generated.PutOrdersId500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: err.Error()}, nil
}

func toInterfaceMap(m map[string]interface{}) map[string]interface{} {
	return m
}

func getUserFromContext(ctx context.Context) (uuid.UUID, model.UserRole) {
	userID, _ := middleware.GetUserID(ctx)
	role, _ := middleware.GetRole(ctx)
	return userID, role
}
