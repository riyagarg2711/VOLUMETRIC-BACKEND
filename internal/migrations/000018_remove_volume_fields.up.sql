DROP INDEX IF EXISTS idx_scans_volume_calculated_at;
DROP INDEX IF EXISTS idx_scans_vehicle_filled_created;


ALTER TABLE scans DROP COLUMN IF EXISTS volume_m3;
ALTER TABLE scans DROP COLUMN IF EXISTS volume_calculated_at;