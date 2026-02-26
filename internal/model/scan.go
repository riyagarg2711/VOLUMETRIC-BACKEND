package model

import (
	"time"

	"github.com/google/uuid"
)

type Scan struct {
	ID         int       `json:"id"`
	ScanUUID   uuid.UUID `json:"scan_uuid"`
	StationID  *int      `json:"station_id,omitempty"`
	VehicleID  int       `json:"vehicle_id"`
	OperatorID *int      `json:"operator_id,omitempty"`
	IsFilled   bool      `json:"is_filled"`
	MaterialID *int      `json:"material_id,omitempty"`
	CreatedBy  uuid.UUID `sql:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateScanRequest is the JSON payload for POST /scans
type CreateScanRequest struct {
	StationID  *int `json:"station_id,omitempty"`
	VehicleID  int  `json:"vehicle_id"`
	OperatorID *int `json:"operator_id,omitempty"`
	IsFilled   bool `json:"is_filled"`
	MaterialID *int `json:"material_id,omitempty"`
}

type ScanSummary struct {
    ID           int       `json:"id"`
    ScanUUID     uuid.UUID `json:"scan_uuid"`
    VehicleID    int       `json:"vehicle_id"`
    IsFilled     bool      `json:"is_filled"`
    MaterialID   *int      `json:"material_id,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
}