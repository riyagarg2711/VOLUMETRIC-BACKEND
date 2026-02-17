CREATE TABLE IF NOT EXISTS volume_calculation_data (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    volume_m3 NUMERIC(15,6),
    weight_kg NUMERIC(15,4),
    is_filled BOOLEAN NOT NULL,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scan_id)  
);