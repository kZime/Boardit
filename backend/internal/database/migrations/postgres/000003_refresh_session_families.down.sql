DROP INDEX IF EXISTS idx_refresh_sessions_family;
ALTER TABLE refresh_sessions DROP COLUMN IF EXISTS family_id;
