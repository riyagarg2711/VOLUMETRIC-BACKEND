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

// AuthMiddleware validates JWT and checks session validity
func AuthMiddleware(repo *repo.AuthRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "Missing token"})
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "Invalid token format"})
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := jwt.ValidateAccessToken(tokenStr)
			if err != nil {
				log.Printf("AuthMiddleware: token invalid: %v", err)
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "Invalid or expired token"})
				return
			}

			// Skip session check for logout endpoint
			if r.URL.Path == "/auth/logout" {
				ctx := context.WithValue(r.Context(), ClaimsKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Require X-Device-ID for all other protected routes
			deviceID := r.Header.Get("X-Device-ID")
			if deviceID == "" {
				log.Printf("AuthMiddleware: missing X-Device-ID for path %s", r.URL.Path)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]string{"error": "X-Device-ID header required"})
				return
			}

			// Full session check for protected routes (scans, etc.)
			session, err := repo.GetSessionByUserAndDevice(claims.UserID, deviceID)
			if err != nil || session == nil || !session.IsValid {
				log.Printf("AuthMiddleware: session invalid for user %s device %s", claims.UserID, deviceID)
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "Session invalid or logged out"})
				return
			}

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