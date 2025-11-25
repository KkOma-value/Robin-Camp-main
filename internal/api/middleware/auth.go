package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	raterIDKey contextKey = "raterID"
)

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}

// Logger logs HTTP requests.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.statusCode,
				"duration", time.Since(start),
			)
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// BearerAuth validates Authorization: Bearer token.
func BearerAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeError(w, "UNAUTHORIZED", "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, "UNAUTHORIZED", "Invalid Authorization format", http.StatusUnauthorized)
				return
			}

			if parts[1] != token {
				writeError(w, "UNAUTHORIZED", "Invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRaterID enforces X-Rater-Id header and stores it in context.
func RequireRaterID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raterID := strings.TrimSpace(r.Header.Get("X-Rater-Id"))
		if raterID == "" {
			writeError(w, "UNAUTHORIZED", "Missing X-Rater-Id header", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), raterIDKey, raterID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRaterID retrieves the rater ID from request context.
func GetRaterID(ctx context.Context) string {
	if val := ctx.Value(raterIDKey); val != nil {
		return val.(string)
	}
	return ""
}
