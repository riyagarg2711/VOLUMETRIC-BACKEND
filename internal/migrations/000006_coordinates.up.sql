CREATE TABLE IF NOT EXISTS coordinates (
    id BIGSERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    x DOUBLE PRECISION NOT NULL,
    y DOUBLE PRECISION NOT NULL,
    z DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for coordinates (critical for performance with lakhs of rows)
CREATE INDEX IF NOT EXISTS idx_coordinates_scan_id ON coordinates(scan_id);
CREATE INDEX IF NOT EXISTS idx_coordinates_created_at ON coordinates(created_at);
