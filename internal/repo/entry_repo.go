package repo

import (
	"database/sql"
	"fmt"

	"volumetric-backend/internal/model"
)

type EntryRepo struct {
	db *sql.DB
}

func NewEntryRepo(db *sql.DB) *EntryRepo {
	return &EntryRepo{db: db}
}

// CreateEntry creates a new entry (first scan of pair)
func (r *EntryRepo) CreateEntry(entry *model.Entry) error {
	_, err := r.db.Exec(`
		INSERT INTO entries (
			vehicle_id, empty_scan_id, filled_scan_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, entry.VehicleID, entry.EmptyScanID, entry.FilledScanID, entry.Status)
	return err
}

// GetActiveEntryByVehicle finds the current incomplete pair for a vehicle
func (r *EntryRepo) GetActiveEntryByVehicle(vehicleID int) (*model.Entry, error) {
	entry := &model.Entry{}
	err := r.db.QueryRow(`
		SELECT id, entry_uuid, vehicle_id, empty_scan_id, filled_scan_id, volume_m3, status, created_at, updated_at
		FROM entries
		WHERE vehicle_id = $1 AND status < 2
		ORDER BY created_at DESC
		LIMIT 1
	`, vehicleID).Scan(
		&entry.ID, &entry.EntryUUID, &entry.VehicleID,
		&entry.EmptyScanID, &entry.FilledScanID, &entry.VolumeM3,
		&entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active entry failed: %w", err)
	}
	return entry, nil
}

// UpdateEntry updates the entry (adds missing scan_id or volume)
func (r *EntryRepo) UpdateEntry(entry *model.Entry) error {
	_, err := r.db.Exec(`
		UPDATE entries
		SET empty_scan_id = $1,
		    filled_scan_id = $2,
		    volume_m3 = $3,
		    status = $4,
		    updated_at = NOW()
		WHERE id = $5
	`, entry.EmptyScanID, entry.FilledScanID, entry.VolumeM3, entry.Status, entry.ID)
	return err
}