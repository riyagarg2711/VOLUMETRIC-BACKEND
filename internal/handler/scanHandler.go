package handler

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"volumetric-backend/internal/auth/middleware"

	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type ScanHandler struct {
	DB *sql.DB
}

type CreateScanRequest struct {
	StationID  *int `json:"station_id,omitempty"`
	VehicleID  int  `json:"vehicle_id"`
	OperatorID *int `json:"operator_id,omitempty"`
	IsFilled   bool `json:"is_filled"`
	MaterialID *int `json:"material_id,omitempty"`
}

func (h *ScanHandler) CreateScan(w http.ResponseWriter, r *http.Request) {
	var input CreateScanRequest

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

	userID := claims.UserID  // uuid.UUID from token
	
	log.Printf("Creating scan for user: %s", userID.String())


	scanUUID := uuid.New()
	now := time.Now().UTC()

	query := `
		INSERT INTO scans (
			scan_uuid,
			station_id,
			vehicle_id,
			operator_id,
			is_filled,
			material_id,
			created_by,     
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var newID int
	err := h.DB.QueryRow(
		query,
		scanUUID,
		input.StationID,
		input.VehicleID,
		input.OperatorID,
		input.IsFilled,
		input.MaterialID,
		userID,    
		now,
	).Scan(&newID)

	if err != nil {
		fmt.Printf("SCAN INSERT ERROR: %v\n", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error": "could not create scan",
		})
		return
	}

	response := map[string]interface{}{
		"id":         newID,
		"scan_uuid":  scanUUID.String(),
		"created_at": now.Format(time.RFC3339),
		"is_filled":  input.IsFilled,
		"created_by": userID.String(),  
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}