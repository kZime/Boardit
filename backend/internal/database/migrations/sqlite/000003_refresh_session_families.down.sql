DROP INDEX IF EXISTS idx_refresh_sessions_family;
ALTER TABLE refresh_sessions DROP COLUMN family_id;
