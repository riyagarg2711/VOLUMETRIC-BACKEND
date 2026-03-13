package handler

import (
	"github.com/google/uuid"
	"volumetric-backend/internal/model"
)

type ScanStore interface {
	CreateScan(input model.CreateScanRequest, userID uuid.UUID) (int, error)
	GetScanByID(id int) (*model.Scan, error)
	GetUserScans(userID uuid.UUID) ([]model.ScanSummary, error)
}

type CoordinateStore interface {
	BatchInsertCoordinates(scanID int, coords []model.Coordinate) error
	GetCoordinatesByScanID(scanID int) ([]model.Coordinate, error)
	GetCoordinatesForScanIDs(scanIDs []int, userID uuid.UUID) ([]model.Coordinate, error)
}

type EntryStore interface {
	CreateEntry(entry *model.Entry) error
	GetActiveEntryByVehicle(vehicleID int) (*model.Entry, error)
	UpdateEntry(entry *model.Entry) error
}
