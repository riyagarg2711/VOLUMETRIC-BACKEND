package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"volumetric-backend/internal/auth/middleware"
	"volumetric-backend/internal/model"
	"volumetric-backend/internal/repo"
)

type CoordinateHandler struct {
	Repo *repo.CoordinateRepo
	ScanRepo *repo.ScanRepo  
}

func NewCoordinateHandler(repo *repo.CoordinateRepo, scanRepo *repo.ScanRepo) *CoordinateHandler {
	return &CoordinateHandler{Repo: repo, ScanRepo: scanRepo}
}

// POST /scans/{id}/coordinates — upload CNS file
func (h *CoordinateHandler) UploadCoordinates(w http.ResponseWriter, r *http.Request) {
	scanIDStr := chi.URLParam(r, "id")
	scanID, err := strconv.Atoi(scanIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid scan ID"})
		return
	}

	// Check ownership
	claims, ok := middleware.GetClaims(r)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "User not authenticated"})
		return
	}

	// Assume GetScanByID in ScanRepo — check created_by
	scan, err := h.ScanRepo.GetScanByID(scanID)
	if err != nil || scan == nil || scan.CreatedBy != claims.UserID {
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, map[string]string{"error": "You don't own this scan"})
		return
	}

	// Parse file
	err = r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Failed to parse file"})
		return
	}

	file, _, err := r.FormFile("cns_file") // input name = "cns_file"
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Missing CNS file"})
		return
	}
	defer file.Close()

	// Time measurement
	start := time.Now()

	reader := csv.NewReader(file)
	var coords []model.Coordinate
	lineNumber := 0

	for {
		line, err := reader.Read()
		lineNumber++
		if err == io.EOF {
			break
		}
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": fmt.Sprintf("Parse error at line %d: %v", lineNumber, err)})
			return
		}

		if len(line) < 3 {
			continue // skip invalid
		}

		x, err := strconv.ParseFloat(line[0], 64)
		if err != nil {
			continue // skip bad data
		}
		y, err := strconv.ParseFloat(line[1], 64)
		if err != nil {
			continue
		}
		z, err := strconv.ParseFloat(line[2], 64)
		if err != nil {
			continue
		}

		coords = append(coords, model.Coordinate{
			X: x,
			Y: y,
			Z: z,
		})
	}

	// Store
	err = h.Repo.BatchInsertCoordinates(scanID, coords)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to store coordinates"})
		return
	}

	end := time.Since(start)
	log.Printf("Upload for scan %d: %d coords in %v", scanID, len(coords), end)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"message": "Coordinates stored",
		"scan_id": scanID,
		"count": len(coords),
		"time_taken": end.String(),
	})
}

// GET /scans/{id}/coordinates — fetch coords
func (h *CoordinateHandler) GetCoordinates(w http.ResponseWriter, r *http.Request) {
	scanIDStr := chi.URLParam(r, "id")
	scanID, err := strconv.Atoi(scanIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid scan ID"})
		return
	}

	// Ownership check (same as upload)
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

	// Time measurement
	start := time.Now()

	coords, err := h.Repo.GetCoordinatesByScanID(scanID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to fetch coordinates"})
		return
	}

	end := time.Since(start)
	log.Printf("Fetch for scan %d: %d coords in %v", scanID, len(coords), end)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"scan_id": scanID,
		"coordinates": coords,
		"count": len(coords),
		"time_taken": end.String(),
	})
}
type BulkScanIDs struct {
    ScanIDs []int `json:"scan_ids"`
}

// POST /scans/coordinates/bulk
func (h *CoordinateHandler) GetCoordinatesBulk(w http.ResponseWriter, r *http.Request) {
    claims, ok := middleware.GetClaims(r)
    if !ok {
        render.Status(r, http.StatusUnauthorized)
        render.JSON(w, r, map[string]string{"error": "User not authenticated"})
        return
    }

    var req BulkScanIDs
    if err := render.DecodeJSON(r.Body, &req); err != nil {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, map[string]string{"error": "Invalid JSON"})
        return
    }

    if len(req.ScanIDs) == 0 {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, map[string]string{"error": "No scan IDs provided"})
        return
    }

    // Optional: limit to reasonable number
    if len(req.ScanIDs) > 500 {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, map[string]string{"error": "Maximum 500 scan IDs per request"})
        return
    }

    start := time.Now()
    coords, err := h.Repo.GetCoordinatesForScanIDs(req.ScanIDs, claims.UserID)
    duration := time.Since(start)

    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to fetch coordinates"})
        return
    }

    log.Printf("Bulk fetch: %d scans, %d coords in %v", len(req.ScanIDs), len(coords), duration)

    render.Status(r, http.StatusOK)
    render.JSON(w, r, map[string]interface{}{
        "coordinates": coords,
        "count":       len(coords),
        "time_taken":  duration.String(),
    })
}