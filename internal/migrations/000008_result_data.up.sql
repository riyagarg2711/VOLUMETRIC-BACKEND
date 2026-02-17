CREATE TABLE IF NOT EXISTS result_data (
    id SERIAL PRIMARY KEY,
    vehicle_id INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    empty_scan_id INTEGER REFERENCES scans(id) ON DELETE SET NULL,
    filled_scan_id INTEGER REFERENCES scans(id) ON DELETE SET NULL,
    material_id INTEGER REFERENCES material_types(id) ON DELETE SET NULL,
    result_volume_m3 NUMERIC(15,6),
    result_weight_kg NUMERIC(15,4),
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'calculated', 'approved', 'flagged')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    calculated_at TIMESTAMP WITH TIME ZONE,
    calculated_by INTEGER REFERENCES users(id)
);

-- Unique constraint: one result per vehicle + material + day
-- The cast (created_at::date) is IMMUTABLE and correct
CREATE UNIQUE INDEX IF NOT EXISTS uniq_result_per_vehicle_material_day 
ON result_data (
    vehicle_id, 
    material_id, 
    ((created_at AT TIME ZONE 'UTC')::date) -- Fixed: Added explicit timezone
);

-- Additional Indexes
CREATE INDEX IF NOT EXISTS idx_scans_vehicle_id ON scans(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_scans_station_id ON scans(station_id);
CREATE INDEX IF NOT EXISTS idx_scans_is_filled ON scans(is_filled);
CREATE INDEX IF NOT EXISTS idx_scans_created_at ON scans(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_result_data_vehicle ON result_data(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_result_data_status ON result_data(status);;
CREATE INDEX IF NOT EXISTS idx_result_data_vehicle ON result_data(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_result_data_status ON result_data(status);
