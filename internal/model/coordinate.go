package model

import "time"

type Coordinate struct {
	ID        uint      `sql:"id"`
	ScanID    int       `sql:"scan_id"`
	X         float64   `sql:"x"`
	Y         float64   `sql:"y"`
	Z         float64   `sql:"z"`
	CreatedAt time.Time `sql:"created_at"`
}