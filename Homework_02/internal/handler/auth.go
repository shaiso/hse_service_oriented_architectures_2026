package handler

import (
	"context"
	"errors"

	"github.com/shaiso/marketplace/generated"
	"github.com/shaiso/marketplace/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) PostAuthRegister(ctx context.Context, request generated.PostAuthRegisterRequestObject) (generated.PostAuthRegisterResponseObject, error) {
	result, err := h.authService.RegisterRaw(ctx, request.Body.Username, request.Body.Password, string(request.Body.Role))
	if err != nil {
		var validationErr *service.ValidationError
		if errors.As(err, &validationErr) {
			details := mapToDetails(validationErr.Fields)
			return generated.PostAuthRegister400JSONResponse{
				ErrorCode: service.ErrCodeValidationError,
				Message:   "Validation failed",
				Details:   &details,
			}, nil
		}
		var bizErr *service.BusinessError
		if errors.As(err, &bizErr) {
			if bizErr.Status == 409 {
				return generated.PostAuthRegister409JSONResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}, nil
			}
		}
		return generated.PostAuthRegister500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: err.Error()}, nil
	}

	return generated.PostAuthRegister201JSONResponse{
		Id:        result.ID,
		Username:  result.Username,
		Role:      generated.UserRole(result.Role),
		CreatedAt: result.CreatedAt,
	}, nil
}

func (h *AuthHandler) PostAuthLogin(ctx context.Context, request generated.PostAuthLoginRequestObject) (generated.PostAuthLoginResponseObject, error) {
	result, err := h.authService.LoginRaw(ctx, request.Body.Username, request.Body.Password)
	if err != nil {
		var bizErr *service.BusinessError
		if errors.As(err, &bizErr) {
			if bizErr.Status == 401 {
				return generated.PostAuthLogin401JSONResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}, nil
			}
		}
		return generated.PostAuthLogin500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: err.Error()}, nil
	}

	return generated.PostAuthLogin200JSONResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *AuthHandler) PostAuthRefresh(ctx context.Context, request generated.PostAuthRefreshRequestObject) (generated.PostAuthRefreshResponseObject, error) {
	result, err := h.authService.RefreshRaw(ctx, request.Body.RefreshToken)
	if err != nil {
		var bizErr *service.BusinessError
		if errors.As(err, &bizErr) {
			if bizErr.Status == 401 {
				return generated.PostAuthRefresh401JSONResponse{ErrorCode: bizErr.Code, Message: bizErr.Message}, nil
			}
		}
		return generated.PostAuthRefresh500JSONResponse{ErrorCode: "INTERNAL_ERROR", Message: err.Error()}, nil
	}

	return generated.PostAuthRefresh200JSONResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}
