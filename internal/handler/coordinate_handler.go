package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"volumetric-backend/internal/auth/middleware"
	"volumetric-backend/internal/model"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type CoordinateHandler struct {
	Repo       CoordinateStore
	ScanRepo   ScanStore
	EntryRepo  EntryStore
	VolumeCalc VolumeCalculator
}

func NewCoordinateHandler(
	coordRepo CoordinateStore,
	scanRepo ScanStore,
	entryRepo EntryStore,
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

	// === Parse & Store Coordinates ===
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

	err = h.Repo.BatchInsertCoordinates(scanID, coords)
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
		"scan_id":     scanID,
		"coordinates": coords,
		"count":       len(coords),
		"time_taken":  end.String(),
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

// POST /scans/upload — single API: create scan + upload coords + update entry

// 1) Parse request (file + data)

// 2) Validate input

// 3) Authenticate user

// 4) Create scan record

// 5) Parse CSV → coordinates

// 6) Store coordinates (batch)

// 7) Handle entry lifecycle:

// 8) create OR update

// 9) compute volume if both scans exist

// 10) Return response

func (h *CoordinateHandler) UploadFullScan(w http.ResponseWriter, r *http.Request) {
	// Parse Multipart Form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max memory limit
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Failed to parse form"})
		return
	}

	// Get fields
	vehicleIDStr := r.FormValue("vehicle_id") // FormValue works for both form-data & URL params.
	isFilledStr := r.FormValue("is_filled")
	file, _, err := r.FormFile("cns_file") // FormFile is required for file upload

	if vehicleIDStr == "" || file == nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "vehicle_id and cns_file required"})
		return
	}

	vehicleID, err := strconv.Atoi(vehicleIDStr) // Converts string → integer
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid vehicle_id"})
		return
	}

	isFilled := isFilledStr == "true" || isFilledStr == "1" // Converts string → boolean

	// Auth check
	claims, ok := middleware.GetClaims(r)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "User not authenticated"})
		return
	}

	start := time.Now() // Measure API performance

	// Step 1: Create Scan
	scanUUID := uuid.New()
	query := `
		INSERT INTO scans (scan_uuid, vehicle_id, is_filled, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var scanID int
	err = h.ScanRepo.GetDB().QueryRow(query, scanUUID, vehicleID, isFilled, claims.UserID, time.Now().UTC()).Scan(&scanID)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create scan"})
		return
	}

	// Step 2: Parse & Store Coordinates
	reader := csv.NewReader(file)
	var coords []model.Coordinate

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(line) < 3 {
			continue
		}

		x, _ := strconv.ParseFloat(line[0], 64) // Convert CSV strings → float64.
		y, _ := strconv.ParseFloat(line[1], 64)
		z, _ := strconv.ParseFloat(line[2], 64)

		coords = append(coords, model.Coordinate{X: x, Y: y, Z: z})
	}

	err = h.Repo.BatchInsertCoordinates(scanID, coords) // Inserts all coordinates in one go.
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to store coordinates"})
		return
	}

	// Step 3: Entry logic
	log.Printf("Starting entry logic for scan %d (vehicle %d, is_filled=%v)", scanID, vehicleID, isFilled)

	entry, err := h.EntryRepo.GetActiveEntryByVehicle(vehicleID) // Fetch existing active entry
	if err != nil {
		log.Printf("Entry lookup failed for vehicle %d: %v", vehicleID, err)

	} else if entry != nil {
		log.Printf("Found existing entry %d (status %d, empty=%v, filled=%v)",
			entry.ID, entry.Status, entry.EmptyScanID, entry.FilledScanID)
	}

	if entry == nil {
		// First scan for this vehicle
		newEntry := &model.Entry{
			VehicleID: vehicleID,
		}
		if isFilled {
			newEntry.FilledScanID = &scanID
			newEntry.Status = 1 // filled only
		} else {
			newEntry.EmptyScanID = &scanID
			newEntry.Status = 0 // empty only
		}
		if err := h.EntryRepo.CreateEntry(newEntry); err != nil {
			log.Printf("Create entry failed: %v", err)
		} else {
			log.Printf("Created new entry for vehicle %d (status %d)", vehicleID, newEntry.Status)
		}
	} else {
		// Update existing entry
		updated := false
		if isFilled && entry.FilledScanID == nil {
			entry.FilledScanID = &scanID
			updated = true
		} else if !isFilled && entry.EmptyScanID == nil {
			entry.EmptyScanID = &scanID
			updated = true
		}

		if updated {
			if entry.EmptyScanID != nil && entry.FilledScanID != nil { // Both scans available → compute volume.
				emptyVol, err := h.VolumeCalc.CalculateVolume(*entry.EmptyScanID, false, claims.UserID)
				if err != nil {
					log.Printf("Empty volume calc failed: %v", err)
					emptyVol = 0
				}
				filledVol, err := h.VolumeCalc.CalculateVolume(*entry.FilledScanID, true, claims.UserID)
				if err != nil {
					log.Printf("Filled volume calc failed: %v", err)
					filledVol = 0
				}
				diff := filledVol - emptyVol
				if diff < 0 {
					log.Printf("Warning: negative volume diff (%.2f) for vehicle %d (empty=%d, filled=%d) - clamping to 0",
						diff, vehicleID, *entry.EmptyScanID, *entry.FilledScanID)
					diff = 0 // enforce non-negative
				}
				entry.VolumeM3 = &diff
				entry.Status = 2 // both done
				h.EntryRepo.UpdateEntry(entry)
			}
			if err := h.EntryRepo.UpdateEntry(entry); err != nil {
				log.Printf("Update entry failed: %v", err)
			} else {
				log.Printf("Updated entry %d (status %d, volume=%v)", entry.ID, entry.Status, entry.VolumeM3)
			}
		}
	}

	duration := time.Since(start) // Performance monitoring

	render.Status(r, http.StatusOK)
	// Returns API response
	render.JSON(w, r, map[string]interface{}{
		"message":      "Scan, coordinates, and entry processed",
		"scan_id":      scanID,
		"vehicle_id":   vehicleID,
		"is_filled":    isFilled,
		"coords_count": len(coords),
		"time_taken":   duration.String(),
	})
}

// GET /entries — list all entry records
func (h *CoordinateHandler) ListEntries(w http.ResponseWriter, r *http.Request) {
    // Optional: auth check (keep if you want to restrict who sees entries)
    // claims, ok := middleware.GetClaims(r)
    // if !ok {
    //     render.Status(r, http.StatusUnauthorized)
    //     render.JSON(w, r, map[string]string{"error": "User not authenticated"})
    //     return
    // }

    entries, err := h.EntryRepo.ListEntries()
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to fetch entries"})
        return
    }

    render.Status(r, http.StatusOK)
    render.JSON(w, r, map[string]interface{}{
        "entries": entries,
        "count":   len(entries),
    })
}

// GET /entries/{id} — get one entry by integer ID
func (h *CoordinateHandler) GetEntryByID(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        render.Status(r, http.StatusBadRequest)
        render.JSON(w, r, map[string]string{"error": "Invalid entry ID"})
        return
    }

    entry, err := h.EntryRepo.GetEntryByID(id)
    if err != nil {
        render.Status(r, http.StatusInternalServerError)
        render.JSON(w, r, map[string]string{"error": "Failed to fetch entry"})
        return
    }
    if entry == nil {
        render.Status(r, http.StatusNotFound)
        render.JSON(w, r, map[string]string{"error": "Entry not found"})
        return
    }

    render.Status(r, http.StatusOK)
    render.JSON(w, r, entry)
}