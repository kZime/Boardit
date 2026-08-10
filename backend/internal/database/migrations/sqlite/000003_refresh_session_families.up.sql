ALTER TABLE refresh_sessions ADD COLUMN family_id TEXT NOT NULL DEFAULT '';

WITH RECURSIVE session_families(token_id, family_id) AS (
    SELECT session.token_id, session.token_id
    FROM refresh_sessions AS session
    WHERE NOT EXISTS (
        SELECT 1
        FROM refresh_sessions AS predecessor
        WHERE predecessor.replaced_by_token_id = session.token_id
    )
    UNION ALL
    SELECT child.token_id, parent.family_id
    FROM session_families AS parent
    JOIN refresh_sessions AS current_session ON current_session.token_id = parent.token_id
    JOIN refresh_sessions AS child ON child.token_id = current_session.replaced_by_token_id
)
UPDATE refresh_sessions
SET family_id = COALESCE(
    (SELECT family_id FROM session_families WHERE session_families.token_id = refresh_sessions.token_id),
    token_id
);

CREATE INDEX IF NOT EXISTS idx_refresh_sessions_family ON refresh_sessions (user_id, family_id);
