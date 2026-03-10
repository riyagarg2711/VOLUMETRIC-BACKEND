package handler

import (
    
    "math/rand"
    "time"

    "github.com/google/uuid"
)

// VolumeCalculator interface — easy to swap mock with real service
type VolumeCalculator interface {
    CalculateVolume(scanID int, isFilled bool, userID uuid.UUID) (float64, error)
    CalculateVolumeDiff(emptyScanID, filledScanID int, userID uuid.UUID) (float64, error)
}

// MockVolumeCalculator — random volumes for testing
type MockVolumeCalculator struct{}

func NewMockVolumeCalculator() VolumeCalculator {
    return &MockVolumeCalculator{}
}

func (m *MockVolumeCalculator) CalculateVolume(scanID int, isFilled bool, userID uuid.UUID) (float64, error) {
    // Seed random with scan ID for reproducible results
    rand.Seed(time.Now().Unix() + int64(scanID))

    if isFilled {
        // Filled: 20–80 m³ (random)
        return rand.Float64()*60 + 20, nil
    } else {
        // Empty: 10–50 m³ (random)
        return rand.Float64()*40 + 10, nil
    }
}

func (m *MockVolumeCalculator) CalculateVolumeDiff(emptyScanID, filledScanID int, userID uuid.UUID) (float64, error) {
    emptyVol, err := m.CalculateVolume(emptyScanID, false, userID)
    if err != nil {
        return 0, err
    }

    filledVol, err := m.CalculateVolume(filledScanID, true, userID)
    if err != nil {
        return 0, err
    }

    return filledVol - emptyVol, nil
}