package router

import (
	"volumetric-backend/internal/handler"

	"github.com/go-chi/chi/v5"
)

func Setup(h *handler.CoordinateHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Post("/coordinates", h.CreateCoordinate)

	return r
}
