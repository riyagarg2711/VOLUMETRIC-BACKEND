package model

import (
	"time"

	"github.com/google/uuid"
)

type EntryStatus int

const (
	EntryStatusEmptyOnly  EntryStatus = 0
	EntryStatusFilledOnly EntryStatus = 1
	EntryStatusBoth       EntryStatus = 2
)

type Entry struct {
	ID             int64      `json:"id"`
	EntryUUID      uuid.UUID  `json:"entry_uuid"`
	VehicleID      int        `json:"vehicle_id"`
	EmptyScanID    *int       `json:"empty_scan_id,omitempty"`
	FilledScanID   *int       `json:"filled_scan_id,omitempty"`
	VolumeM3       *float64   `json:"volume_m3,omitempty"`
	Status         EntryStatus `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}