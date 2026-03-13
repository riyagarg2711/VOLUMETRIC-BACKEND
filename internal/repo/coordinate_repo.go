package repo

import (
	"database/sql"
	"fmt"
	"strings"

	"volumetric-backend/internal/model"

	"github.com/google/uuid"
)

type CoordinateRepo struct {
	db *sql.DB // database conn
}

func NewCoordinateRepo(db *sql.DB) *CoordinateRepo {
	return &CoordinateRepo{db: db}
}

// METHOD BatchInsertCoordinates — insert many coords fast
func (r *CoordinateRepo) BatchInsertCoordinates(scanID int, coords []model.Coordinate) error {
	tx, err := r.db.Begin() //transaction begin
	if err != nil {
		return fmt.Errorf("batch insert begin failed: %w", err)
	}
	defer tx.Rollback() // defer means execute at function end
	// prepared SQL statement
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
		ORDER BY created_at ASC  
	`, scanID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var coords []model.Coordinate    // Create empty slice to store results
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

// GetCoordinatesForScanIDs fetches coordinates for multiple scan IDs. Only returns rows for scans that belong to the given user
func (r *CoordinateRepo) GetCoordinatesForScanIDs(scanIDs []int, userID uuid.UUID) ([]model.Coordinate, error) {
	if len(scanIDs) == 0 {
		return []model.Coordinate{}, nil
	}

	placeholders := make([]string, len(scanIDs))
	args := make([]interface{}, len(scanIDs)+1)
	args[0] = userID

	for i, id := range scanIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
        SELECT c.id, c.scan_id, c.x, c.y, c.z, c.created_at
        FROM coordinates c
        JOIN scans s ON c.scan_id = s.id
        WHERE c.scan_id IN (%s) AND s.created_by = $1
        ORDER BY c.scan_id, c.created_at
    `, strings.Join(placeholders, ","))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var coords []model.Coordinate
	for rows.Next() {
		var c model.Coordinate
		err := rows.Scan(&c.ID, &c.ScanID, &c.X, &c.Y, &c.Z, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		coords = append(coords, c)
	}

	return coords, rows.Err()
}
