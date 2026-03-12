CREATE TABLE IF NOT EXISTS entries (
    id BIGSERIAL PRIMARY KEY,
    entry_uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    vehicle_id INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    empty_scan_id INTEGER REFERENCES scans(id) ON DELETE SET NULL,
    filled_scan_id INTEGER REFERENCES scans(id) ON DELETE SET NULL,
    volume_m3 DOUBLE PRECISION,
    status SMALLINT NOT NULL DEFAULT 0 CHECK (status IN (0, 1, 2)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE UNIQUE INDEX idx_entries_vehicle_active_unique 
ON entries (vehicle_id) 
WHERE status < 2;

CREATE INDEX idx_entries_vehicle_id ON entries(vehicle_id);
CREATE INDEX idx_entries_status ON entries(status);
CREATE INDEX idx_entries_empty_scan_id ON entries(empty_scan_id);
CREATE INDEX idx_entries_filled_scan_id ON entries(filled_scan_id);

