ALTER TABLE scans ADD COLUMN IF NOT EXISTS volume_m3 NUMERIC(10,2) DEFAULT NULL;
ALTER TABLE scans ADD COLUMN IF NOT EXISTS volume_calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_scans_volume_calculated_at ON scans(volume_calculated_at DESC);
CREATE INDEX IF NOT EXISTS idx_scans_vehicle_filled_created ON scans(vehicle_id, is_filled, created_at DESC);