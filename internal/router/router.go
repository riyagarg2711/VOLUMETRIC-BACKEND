package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"volumetric-backend/internal/app"
)

func Setup(a *app.App) *chi.Mux {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// ── Public routes ──
	r.Group(func(r chi.Router) {
		r.Post("/auth/otp/send", a.AuthHandler.SendOTP)
		r.Post("/auth/otp/verify", a.AuthHandler.VerifyOTP)
		r.Post("/auth/refresh", a.AuthHandler.RefreshToken)
	})

	// ── Protected routes ──
	r.Group(func(r chi.Router) {
		r.Use(a.AuthMiddleware)

		r.Post("/auth/logout", a.AuthHandler.Logout)
		r.Post("/scans", a.ScanHandler.CreateScan)
		r.Get("/scans", a.ScanHandler.ListUserScans)
		r.Post("/scans/coordinates/bulk", a.CoordHandler.GetCoordinatesBulk)
		r.Post("/scans/{id}/coordinates", a.CoordHandler.UploadCoordinates)
		r.Get("/scans/{id}/coordinates", a.CoordHandler.GetCoordinates)
		r.Post("/trucks/{vehicle_id}/volume-diff", a.VolumeHandler.CalculateDiff)
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
