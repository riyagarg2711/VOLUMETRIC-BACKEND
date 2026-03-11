package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/render"

	"volumetric-backend/internal/auth/jwt"
	"volumetric-backend/internal/auth/models"

	"volumetric-backend/internal/auth/repo"
)

// context key to store claims
type contextKey string

const ClaimsKey contextKey = "claims"



func AuthMiddleware(repo *repo.AuthRepo) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            log.Printf("AuthMiddleware called for path: %s", r.URL.Path)

            var tokenStr string
            var deviceID string

            // 1. First try cookies (preferred for browser/QT clients)
            accessCookie, errCookie := r.Cookie("access_token")
            deviceCookie, errDevice := r.Cookie("device_id")
			log.Println(errDevice)

            if errCookie == nil && deviceCookie == nil {
                // Cookies exist → use them
                tokenStr = accessCookie.Value
                deviceID = deviceCookie.Value
                log.Printf("Using cookies - access_token & device_id found")
            } else {
                // 2. Fallback to headers (for curl/Postman/manual testing)
                authHeader := r.Header.Get("Authorization")
                if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
                    tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
                }
                deviceID = r.Header.Get("X-Device-ID")
                log.Printf("Using headers - Authorization & X-Device-ID")
            }

            // 3. If neither cookies nor headers have token → reject
            if tokenStr == "" {
                log.Printf("No access token found (neither cookie nor header)")
                render.Status(r, http.StatusUnauthorized)
                render.JSON(w, r, map[string]string{"error": "Missing token"})
                return
            }

            // 4. If no device ID → reject (required for session check)
            if deviceID == "" {
                log.Printf("No device ID found")
                render.Status(r, http.StatusBadRequest)
                render.JSON(w, r, map[string]string{"error": "Device ID required"})
                return
            }

            // 5. Validate JWT (same as before)
            claims, err := jwt.ValidateAccessToken(tokenStr)
            if err != nil {
                log.Printf("Token invalid: %v", err)
                render.Status(r, http.StatusUnauthorized)
                render.JSON(w, r, map[string]string{"error": "Invalid or expired token"})
                return
            }

            // 6. Skip session check for logout (no device check needed)
            if r.URL.Path == "/auth/logout" {
                ctx := context.WithValue(r.Context(), ClaimsKey, claims)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }

            // 7. Full session validation (using device_id)
            session, err := repo.GetSessionByUserAndDevice(claims.UserID, deviceID)
            if err != nil || session == nil || !session.IsValid {
                log.Printf("Session invalid for user %s device %s: %v", claims.UserID, deviceID, err)
                render.Status(r, http.StatusUnauthorized)
                render.JSON(w, r, map[string]string{"error": "Session invalid or logged out"})
                return
            }

            // 8. Store claims in context
            ctx := context.WithValue(r.Context(), ClaimsKey, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
// Helper to get claims in handlers
func GetClaims(r *http.Request) (*models.Claims, bool) {
	claims, ok := r.Context().Value(ClaimsKey).(*models.Claims)
	return claims, ok
}

// (5) This file creates an authentication middleware.

// Middleware = A function that runs before your handler

// It checks if the user is authorized

// This middleware :-

// Checks if token exists

// Validates JWT

// Checks device ID

// Checks session in DB

// Stores user info

// Allows request if everything is valid