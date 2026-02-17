-- Add created_by as UUID referencing users.user_id
ALTER TABLE scans
ADD COLUMN IF NOT EXISTS created_by CHAR(36)
REFERENCES users(user_id) ON DELETE SET NULL;

-- Index for fast lookups
CREATE INDEX IF NOT EXISTS idx_scans_created_by ON scans(created_by);