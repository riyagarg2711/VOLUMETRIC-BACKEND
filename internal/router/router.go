package router

import (
	"volumetric-backend/internal/auth/handler"

"volumetric-backend/internal/auth/repo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"                  // Chi's middlewares
	"github.com/go-chi/render"
	domain "volumetric-backend/internal/handler"
	authmw "volumetric-backend/internal/auth/middleware"
)

func Setup(
	scanHandler *domain.ScanHandler,    
	authHandler *handler.AuthHandler, 
	authRepo *repo.AuthRepo,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middlewares (Chi's)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// ── Public routes ──
	r.Group(func(r chi.Router) {
		r.Post("/auth/otp/send", authHandler.SendOTP)
		r.Post("/auth/otp/verify", authHandler.VerifyOTP)
		 r.Post("/auth/refresh", authHandler.RefreshToken)
		 
	})

	// ── Protected routes ──
	r.Group(func(r chi.Router) {
		r.Use(authmw.AuthMiddleware(authRepo)) 

		// Protected endpoints
		r.Post("/scans", scanHandler.CreateScan)
		r.Post("/auth/logout", authHandler.Logout)
		// r.Get("/scans/{id}", scanHandler.GetScan)
		// r.Post("/scans/{id}/coordinates", coordHandler.Create)
	})

	return r
}

// (4) This file defines all API routes (URLs) and connects them to handlers.

// It decides:
// Which URL
// Uses which middleware
// Calls which function

// This file :-

// Creates router

// Adds global middleware

// Defines public routes

// Defines protected routes

// Returns router