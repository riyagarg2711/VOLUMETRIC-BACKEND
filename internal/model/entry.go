package model

import (
	"github.com/google/uuid"
	"time"
)

type EntryStatus int

const (
	EmptyScanDone  = 0
	FilledScanDone = 1
	BothScanDone   = 2
)

type Entry struct {
	ID           int         `json:"id"`
	EntryUUID    uuid.UUID   `json:"entry_uuid"`
	VehicleID    string      `json:"vehicle_id"`
	EmptyScanID  *int        `json:"empty_scan_id,omitempty"`
	FilledScanID *int        `json:"filled_scan_id,omitempty"`
	Volume       *float64    `json:"volume,omitempty"`
	Status       EntryStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}
