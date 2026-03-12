CREATE TABLE IF NOT EXISTS entries (
    id SERIAL PRIMARY KEY,
    entry_uuid UUID UNIQUE DEFAULT gen_random_uuid(),
    vehicle_id TEXT NOT NULL,
    empty_scan_id INTEGER REFERENCES scans(id) ON DELETE SET NULL,
    filled_scan_id INTEGER REFERENCES scans(id) ON DELETE SET NULL,
    volume DOUBLE PRECISION,
    status INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entries_vehicle_id ON entries(vehicle_id);
CREATE INDEX idx_entries_status ON entries(status);
CREATE INDEX idx_entries_empty_scan_id ON entries(empty_scan_id);
CREATE INDEX idx_entries_filled_scan_id ON entries(filled_scan_id);