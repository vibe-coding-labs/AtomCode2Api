package auth

import (
	"log"
	"net/http"
	"strings"
)

// JWTMiddleware wraps a handler with JWT validation for dashboard routes.
// Requests to /api/* require a valid JWT token (via cookie or Authorization header).
// Public routes (/, /login, /health, /v1/*) are passed through.
func JWTMiddleware(jwtManager *JWTManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public routes
		path := r.URL.Path
		if path == "/" || path == "/login" || strings.HasPrefix(path, "/health") ||
			strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/api/login") {
			next.ServeHTTP(w, r)
			return
		}

		// Only protect /api/* routes
		if strings.HasPrefix(path, "/api/") {
			token := ""
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				token = auth[7:]
			}
			if token == "" {
				if c, err := r.Cookie("token"); err == nil {
					token = c.Value
				}
			}
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := jwtManager.ValidateToken(token)
			if err != nil {
				log.Printf("auth: invalid JWT token: %v", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			r.Header.Set("X-User-ID", claims.UserID)
		}

		next.ServeHTTP(w, r)
	})
}