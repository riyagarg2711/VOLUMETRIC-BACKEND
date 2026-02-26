package repo

import (
	"database/sql"
	"fmt"

	"volumetric-backend/internal/model"
)

type CoordinateRepo struct {
	db *sql.DB
}

func NewCoordinateRepo(db *sql.DB) *CoordinateRepo {
	return &CoordinateRepo{db: db}
}

// BatchInsertCoordinates — insert many coords fast
func (r *CoordinateRepo) BatchInsertCoordinates(scanID int, coords []model.Coordinate) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("batch insert begin failed: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO coordinates (scan_id, x, y, z, created_at)
		VALUES ($1, $2, $3, $4, now())
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt failed: %w", err)
	}
	defer stmt.Close()

	for _, i := range coords {
		_, err := stmt.Exec(scanID, i.X, i.Y, i.Z)
		if err != nil {
			return fmt.Errorf("insert coord failed: %w", err)
		}
	}

	return tx.Commit()
}

// GetCoordinatesByScanID — fetch all coords for a scan
func (r *CoordinateRepo) GetCoordinatesByScanID(scanID int) ([]model.Coordinate, error) {
	rows, err := r.db.Query(`
		SELECT id, scan_id, x, y, z, created_at
		FROM coordinates
		WHERE scan_id = $1
		ORDER BY created_at ASC  // optional order
	`, scanID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var coords []model.Coordinate
	for rows.Next() {
		var coord model.Coordinate
		err := rows.Scan(&coord.ID, &coord.ScanID, &coord.X, &coord.Y, &coord.Z, &coord.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan row failed: %w", err)
		}
		coords = append(coords, coord)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return coords, nil
}