package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"volumetric-backend/internal/auth/middleware"
	"volumetric-backend/internal/model"

	"volumetric-backend/internal/repo"

	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type ScanHandler struct {
	Repo *repo.ScanRepo  
}

func NewScanHandler(repo *repo.ScanRepo) *ScanHandler {
	return &ScanHandler{Repo: repo}
}

func (h *ScanHandler) CreateScan(w http.ResponseWriter, r *http.Request) {
	var input model.CreateScanRequest

	if err := render.DecodeJSON(r.Body, &input); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid json format"})
		return
	}

	if input.VehicleID <= 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "vehicle_id is required and must be positive"})
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "User not authenticated"})
		return
	}

	userID := claims.UserID

	log.Printf("Creating scan for user: %s", userID.String())

	// Use repo instead of direct DB
	newID, err := h.Repo.CreateScan(input, userID)
	if err != nil {
		fmt.Printf("SCAN INSERT ERROR: %v\n", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "could not create scan"})
		return
	}

	response := map[string]interface{}{
		"id":         newID,
		"scan_uuid":  uuid.New().String(), // or fetch from repo if needed
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"is_filled":  input.IsFilled,
		"created_by": userID.String(),
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}
// GET /scans — list current user's scans
func (h *ScanHandler) ListUserScans(w http.ResponseWriter, r *http.Request) {
    claims, ok := middleware.GetClaims(r)
    if !ok {
        render.Status(r, http.StatusUnauthorized)
        render.JSON(w, r, map[string]string{"error": "User not authenticated"})
        return
    }

    start := time.Now()
    scans, err := h.Repo.GetUserScans(claims.UserID)
    duration := time.Since(start)

    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to fetch scans"})
        return
    }

    log.Printf("User %s fetched %d scans in %v", claims.UserID, len(scans), duration)

    render.Status(r, http.StatusOK)
    render.JSON(w, r, map[string]interface{}{
        "scans": scans,
        "count": len(scans),
        "time_taken": duration.String(),
    })
}