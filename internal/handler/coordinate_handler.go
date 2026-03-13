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
	EntryRepo  *repo.EntryRepo     
    VolumeCalc VolumeCalculator   
}

// constructor
func NewCoordinateHandler(
    coordRepo *repo.CoordinateRepo,
    scanRepo *repo.ScanRepo,
    entryRepo *repo.EntryRepo,
    volumeCalc VolumeCalculator,
) *CoordinateHandler {
    return &CoordinateHandler{
        Repo:       coordRepo,
        ScanRepo:   scanRepo,
        EntryRepo:  entryRepo,
        VolumeCalc: volumeCalc,
    }
}


// POST /scans/{id}/coordinates — upload CNS file + create/update entry
func (h *CoordinateHandler) UploadCoordinates(w http.ResponseWriter, r *http.Request) {
	scanIDStr := chi.URLParam(r, "id")
	scanID, err := strconv.Atoi(scanIDStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid scan ID"})
		return
	}

	// Ownership check
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

	// Parse & Store Coordinates
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Failed to parse file"})
		return
	}

	file, _, err := r.FormFile("cns_file")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Missing CNS file"})
		return
	}
	defer file.Close()

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
			render.JSON(w, r, map[string]string{"error": fmt.Sprintf("Parse error at line %d", lineNumber)})
			return
		}
		if len(line) < 3 {
			continue
		}

		x, _ := strconv.ParseFloat(line[0], 64)
		y, _ := strconv.ParseFloat(line[1], 64)
		z, _ := strconv.ParseFloat(line[2], 64)

		coords = append(coords, model.Coordinate{X: x, Y: y, Z: z})
	}

	err = h.Repo.BatchInsertCoordinates(scanID, coords) // save coords to db
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to store coordinates"})
		return
	}

	// === Entry Table Logic ===
	log.Printf("Starting entry logic for scan %d (vehicle %d, is_filled=%v)", scanID, scan.VehicleID, scan.IsFilled)

	entry, err := h.EntryRepo.GetActiveEntryByVehicle(scan.VehicleID)
	if err != nil {
		log.Printf("Entry lookup failed for vehicle %d: %v", scan.VehicleID, err)
		// Continue — entry is optional
	} else if entry != nil {
		log.Printf("Found existing entry %d (status %d, empty=%v, filled=%v)", 
			entry.ID, entry.Status, entry.EmptyScanID, entry.FilledScanID)
	}

	if entry == nil {
		// No active entry → create new one
		newEntry := &model.Entry{
			VehicleID: scan.VehicleID,
		}
		if scan.IsFilled {
			newEntry.FilledScanID = &scanID
			newEntry.Status = 1 // filled only
		} else {
			newEntry.EmptyScanID = &scanID
			newEntry.Status = 0 // empty only
		}

		err = h.EntryRepo.CreateEntry(newEntry)
		if err != nil {
			log.Printf("Create entry failed: %v", err)
		} else {
			log.Printf("Created new entry for vehicle %d (status %d)", scan.VehicleID, newEntry.Status)
		}
	} else {
		// Update existing entry
		updated := false
		if scan.IsFilled && entry.FilledScanID == nil {
			entry.FilledScanID = &scanID
			updated = true
		} else if !scan.IsFilled && entry.EmptyScanID == nil {
			entry.EmptyScanID = &scanID
			updated = true
		}

		if updated {
			if entry.EmptyScanID != nil && entry.FilledScanID != nil {
				emptyVol, err := h.VolumeCalc.CalculateVolume(*entry.EmptyScanID, false, claims.UserID)
				if err != nil {
					log.Printf("Empty volume calc failed: %v", err)
				}
				filledVol, err := h.VolumeCalc.CalculateVolume(*entry.FilledScanID, true, claims.UserID)
				if err != nil {
					log.Printf("Filled volume calc failed: %v", err)
				}
				diff := filledVol - emptyVol
				entry.VolumeM3 = &diff
				entry.Status = 2 // both done
			}
			err = h.EntryRepo.UpdateEntry(entry)
			if err != nil {
				log.Printf("Update entry failed: %v", err)
			} else {
				log.Printf("Updated entry %d (status %d, volume=%v)", entry.ID, entry.Status, entry.VolumeM3)
			}
		}
	}

	end := time.Since(start)
	log.Printf("Upload for scan %d: %d coords in %v", scanID, len(coords), end)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"message":    "Coordinates stored",
		"scan_id":    scanID,
		"count":      len(coords),
		"time_taken": end.String(),
	})
}


// GET /scans/{id}/coordinates — fetch coords
func (h *CoordinateHandler) GetCoordinates(w http.ResponseWriter, r *http.Request) {
	log.Println("GetCoordinates started for path:", r.URL.Path)
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
		log.Println("GetCoordinates: claims missing")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "User not authenticated"})
		return
	}
	log.Printf("GetCoordinates: user ID from token = %s", claims.UserID.String())

	scan, err := h.ScanRepo.GetScanByID(scanID)
	if err != nil || scan == nil || scan.CreatedBy != claims.UserID {
		log.Printf("GetCoordinates: invalid scan ID %q: %v", scanIDStr, err)
		render.Status(r, http.StatusForbidden)
		render.JSON(w, r, map[string]string{"error": "You don't own this scan"})
		return
	}

	// Time measurement
	start := time.Now()

	coords, err := h.Repo.GetCoordinatesByScanID(scanID)
	if err != nil {
		log.Printf("ERROR in GetCoordinatesByScanID for scan %d: %v", scanID, err)
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