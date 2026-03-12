package handler

import (
	"net/http"
	"strconv"
	"volumetric-backend/internal/auth/middleware"
	"volumetric-backend/internal/model"
	"volumetric-backend/internal/repo"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type VolumeHandler struct {
	ScanRepo   *repo.ScanRepo
	EntryRepo  *repo.EntryRepo
	VolumeCalc VolumeCalculator
}

func NewVolumeHandler(scanRepo *repo.ScanRepo, entryRepo *repo.EntryRepo, volumeCalc VolumeCalculator) *VolumeHandler {
	return &VolumeHandler{
		ScanRepo:   scanRepo,
		EntryRepo:  entryRepo,
		VolumeCalc: volumeCalc,
	}
}

// POST /trucks/{vehicle_id}/volume-diff
func (h *VolumeHandler) CalculateDiff(w http.ResponseWriter, r *http.Request) {
	vehicleIDStr := chi.URLParam(r, "vehicle_id")
	vehicleID, err := strconv.Atoi(vehicleIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid vehicle ID"})
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "User not authenticated"})
		return
	}

	entry, err := h.EntryRepo.GetActiveEntryByVehicle(vehicleID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to find entry"})
		return
	}

	if entry == nil || entry.Status != model.EntryStatusBoth {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "No complete pair found for this vehicle"})
		return
	}

	// Both IDs exist → calculate (if not already calculated)
	if entry.VolumeM3 == nil {
		emptyVol, _ := h.VolumeCalc.CalculateVolume(*entry.EmptyScanID, false, claims.UserID)
		filledVol, _ := h.VolumeCalc.CalculateVolume(*entry.FilledScanID, true, claims.UserID)
		diff := filledVol - emptyVol
		entry.VolumeM3 = &diff
		h.EntryRepo.UpdateEntry(entry)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"vehicle_id":     vehicleID,
		"entry_uuid":     entry.EntryUUID.String(),
		"empty_scan_id":  entry.EmptyScanID,
		"filled_scan_id": entry.FilledScanID,
		"volume_m3":      entry.VolumeM3,
		"status":         entry.Status,
	})
}