ALTER TABLE scans DROP COLUMN IF EXISTS created_by;
DROP INDEX IF EXISTS idx_scans_created_by;