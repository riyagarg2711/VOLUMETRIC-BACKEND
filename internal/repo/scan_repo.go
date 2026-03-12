package repo

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"volumetric-backend/internal/model"
)

type ScanRepo struct {
	db *sql.DB
}

func NewScanRepo(db *sql.DB) *ScanRepo {
	return &ScanRepo{db: db}
}

// CreateScan inserts a new scan and returns its ID
func (r *ScanRepo) CreateScan(input model.CreateScanRequest, userID uuid.UUID) (int, error) {
	var newID int
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

	err := r.db.QueryRow(
		query,
		uuid.New(),
		input.StationID,
		input.VehicleID,
		input.OperatorID,
		input.IsFilled,
		input.MaterialID,
		userID,
		time.Now().UTC(),
	).Scan(&newID)

	if err != nil {
		return 0, fmt.Errorf("CreateScan failed: %w", err)
	}

	return newID, nil
}

// GetScanByID fetches a scan (for ownership check)
func (r *ScanRepo) GetScanByID(id int) (*model.Scan, error) {
	scan := &model.Scan{}
	err := r.db.QueryRow(`
		SELECT id, scan_uuid, station_id, vehicle_id, operator_id, is_filled, material_id, created_by, created_at
		FROM scans
		WHERE id = $1
	`, id).Scan(
		&scan.ID, &scan.ScanUUID, &scan.StationID, &scan.VehicleID, &scan.OperatorID,
		&scan.IsFilled, &scan.MaterialID, &scan.CreatedBy, &scan.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // not found
	}
	if err != nil {
		return nil, fmt.Errorf("GetScanByID failed: %w", err)
	}
	return scan, nil
}
// GetUserScans returns summary list of scans created by the user
func (r *ScanRepo) GetUserScans(userID uuid.UUID) ([]model.ScanSummary, error) {
    rows, err := r.db.Query(`
        SELECT id, scan_uuid, vehicle_id, is_filled, material_id, created_at
        FROM scans
        WHERE created_by = $1
        ORDER BY created_at DESC
        LIMIT 100  -- temporary limit — add pagination later
    `, userID)
    if err != nil {
        return nil, fmt.Errorf("query user scans failed: %w", err)
    }
    defer rows.Close()

    var scans []model.ScanSummary
    for rows.Next() {
        var s model.ScanSummary
        err := rows.Scan(&s.ID, &s.ScanUUID, &s.VehicleID, &s.IsFilled, &s.MaterialID, &s.CreatedAt)
        if err != nil {
            return nil, fmt.Errorf("scan row failed: %w", err)
        }
        scans = append(scans, s)
    }

    return scans, rows.Err()
}

