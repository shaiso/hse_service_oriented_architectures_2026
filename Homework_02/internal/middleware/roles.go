package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/shaiso/marketplace/internal/model"
)

// RoleCheck проверяет роли для конкретных эндпоинтов согласно матрице доступа.
func RoleCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, hasRole := GetRole(r.Context())
		if !hasRole {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path
		method := r.Method

		if strings.HasPrefix(path, "/products") && (method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete) {
			if role != model.RoleSeller && role != model.RoleAdmin {
				writeAccessDenied(w)
				return
			}
		}

		if path == "/orders" && method == http.MethodPost {
			if role == model.RoleSeller {
				writeAccessDenied(w)
				return
			}
		}

		if strings.HasPrefix(path, "/orders/") {
			if role == model.RoleSeller {
				writeAccessDenied(w)
				return
			}
		}

		if strings.HasPrefix(path, "/promo-codes") && method == http.MethodPost {
			if role == model.RoleUser {
				writeAccessDenied(w)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func writeAccessDenied(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{
		"error_code": "ACCESS_DENIED",
		"message":    "insufficient permissions",
	})
}
