package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shaiso/marketplace/internal/model"
	"github.com/shaiso/marketplace/internal/service"
)

type contextKey string

const (
	ContextUserID contextKey = "user_id"
	ContextRole   contextKey = "role"
)

// Auth — middleware для проверки JWT токена.
// publicPaths — пути, которые не требуют авторизации.
func Auth(authService *service.AuthService, publicPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем, является ли путь публичным
			for _, path := range publicPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Извлекаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "TOKEN_INVALID", "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeAuthError(w, "TOKEN_INVALID", "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			claims, err := authService.ParseToken(parts[1])
			if err != nil {
				if strings.Contains(err.Error(), "expired") {
					writeAuthError(w, "TOKEN_EXPIRED", "access token expired", http.StatusUnauthorized)
				} else {
					writeAuthError(w, "TOKEN_INVALID", "invalid access token", http.StatusUnauthorized)
				}
				return
			}

			// Кладём user_id и role в контекст
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID извлекает user_id из контекста.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(ContextUserID).(uuid.UUID)
	return userID, ok
}

// GetRole извлекает роль из контекста.
func GetRole(ctx context.Context) (model.UserRole, bool) {
	role, ok := ctx.Value(ContextRole).(model.UserRole)
	return role, ok
}

func writeAuthError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error_code": code,
		"message":    message,
	})
}
