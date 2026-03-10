package handler

import (
	"net/http"
	"strconv"
	"volumetric-backend/internal/auth/middleware"
	"volumetric-backend/internal/repo"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)


type VolumeHandler struct {
    ScanRepo   *repo.ScanRepo
    CoordRepo  *repo.CoordinateRepo
    Calculator VolumeCalculator  
}

func NewVolumeHandler(scanRepo *repo.ScanRepo, coordRepo *repo.CoordinateRepo, calculator VolumeCalculator) *VolumeHandler {
    return &VolumeHandler{ScanRepo: scanRepo, CoordRepo: coordRepo, Calculator: calculator}
}

// POST /scans/{id}/volume — calculate for single scan
func (h *VolumeHandler) CalculateSingleVolume(w http.ResponseWriter, r *http.Request) {
    scanIDStr := chi.URLParam(r, "id")
    scanID, err := strconv.Atoi(scanIDStr)
    if err != nil {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, map[string]string{"error": "Invalid scan ID"})
        return
    }

    claims, ok := middleware.GetClaims(r)
    if !ok {
        render.Status(r, http.StatusUnauthorized)
        render.JSON(w, r, map[string]string{"error": "User not authenticated"})
        return
    }

    scan, err := h.ScanRepo.GetScanByID(scanID)
    if err != nil || scan == nil || scan.CreatedBy != claims.UserID {
        render.Status(r, http.StatusForbidden)
        render.JSON(w, r, map[string]string{"error": "You don't own this scan"})
        return
    }

    volume, err := h.Calculator.CalculateVolume(scanID, scan.IsFilled, claims.UserID)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Volume calculation failed"})
        return
    }

    // Store in DB
    err = h.ScanRepo.UpdateScanVolume(scanID, volume, claims.UserID)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to save volume"})
        return
    }

    render.Status(r, http.StatusOK)
    render.JSON(w, r, map[string]interface{}{
        "scan_id": scanID,
        "is_filled": scan.IsFilled,
        "volume_m3": volume,
    })
}

// POST /trucks/{vehicle_id}/volume-diff — pair and calculate diff
func (h *VolumeHandler) CalculateTruckVolumeDiff(w http.ResponseWriter, r *http.Request) {
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

    // Get latest empty + filled for vehicle
    emptyScan, filledScan, err := h.ScanRepo.GetLatestEmptyFilledPair(vehicleID, claims.UserID)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to find scan pair"})
        return
    }

    if emptyScan == nil {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, map[string]string{"error": "No empty scan found for this truck"})
        return
    }

    if filledScan == nil {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, map[string]string{"error": "No filled scan found for this truck"})
        return
    }

    // Calculate
    emptyVol, err := h.Calculator.CalculateVolume(emptyScan.ID, false, claims.UserID)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Empty volume failed"})
        return
    }

    filledVol, err := h.Calculator.CalculateVolume(filledScan.ID, true, claims.UserID)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Filled volume failed"})
        return
    }

    diffVol := filledVol - emptyVol

    // Store volumes
    err = h.ScanRepo.UpdateScanVolume(emptyScan.ID, emptyVol, claims.UserID)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to save empty volume"})
        return
    }

    err = h.ScanRepo.UpdateScanVolume(filledScan.ID, filledVol, claims.UserID)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to save filled volume"})
        return
    }

    render.Status(r, http.StatusOK)
    render.JSON(w, r, map[string]interface{}{
        "vehicle_id": vehicleID,
        "empty_scan": map[string]interface{}{
            "id": emptyScan.ID,
            "volume_m3": emptyVol,
        },
        "filled_scan": map[string]interface{}{
            "id": filledScan.ID,
            "volume_m3": filledVol,
        },
        "diff_volume_m3": diffVol,  // material volume
    })
}