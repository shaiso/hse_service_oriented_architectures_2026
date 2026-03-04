package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type logEntry struct {
	RequestID   string      `json:"request_id"`
	Method      string      `json:"method"`
	Endpoint    string      `json:"endpoint"`
	StatusCode  int         `json:"status_code"`
	DurationMs  int64       `json:"duration_ms"`
	UserID      *string     `json:"user_id"`
	Timestamp   string      `json:"timestamp"`
	RequestBody interface{} `json:"request_body,omitempty"`
}

// responseWriter — обёртка для перехвата status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := uuid.New().String()
		w.Header().Set("X-Request-Id", requestID)

		// Читаем body для мутирующих запросов
		var requestBody interface{}
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil && len(bodyBytes) > 0 {
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				var bodyJSON map[string]interface{}
				if json.Unmarshal(bodyBytes, &bodyJSON) == nil {
					maskSensitiveFields(bodyJSON)
					requestBody = bodyJSON
				}
			}
		}

		// Оборачиваем ResponseWriter
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Milliseconds()

		var userID *string
		if uid, ok := GetUserID(r.Context()); ok {
			s := uid.String()
			userID = &s
		}

		entry := logEntry{
			RequestID:   requestID,
			Method:      r.Method,
			Endpoint:    r.URL.Path,
			StatusCode:  wrapped.statusCode,
			DurationMs:  duration,
			UserID:      userID,
			Timestamp:   start.UTC().Format(time.RFC3339),
			RequestBody: requestBody,
		}

		logJSON, err := json.Marshal(entry)
		if err != nil {
			log.Printf("failed to marshal log entry: %v", err)
			return
		}

		log.Println(string(logJSON))
	})
}

// maskSensitiveFields маскирует пароли и другие чувствительные данные.
func maskSensitiveFields(data map[string]interface{}) {
	sensitiveKeys := []string{"password", "token", "secret", "refresh_token"}
	for key, value := range data {
		for _, sensitive := range sensitiveKeys {
			if strings.EqualFold(key, sensitive) {
				data[key] = "***"
				break
			}
		}
		if nested, ok := value.(map[string]interface{}); ok {
			maskSensitiveFields(nested)
		}
	}
}
