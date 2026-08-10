ALTER TABLE notes ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 1;

ALTER TABLE note_revisions ADD COLUMN IF NOT EXISTS user_id BIGINT;
ALTER TABLE note_revisions ADD COLUMN IF NOT EXISTS version BIGINT;
ALTER TABLE note_revisions ADD COLUMN IF NOT EXISTS title VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE note_revisions ADD COLUMN IF NOT EXISTS content_html TEXT NOT NULL DEFAULT '';
ALTER TABLE note_revisions ADD COLUMN IF NOT EXISTS source VARCHAR(40) NOT NULL DEFAULT 'user';

UPDATE note_revisions AS revision
SET user_id = note.user_id,
    title = note.title,
    content_html = note.content_html
FROM notes AS note
WHERE revision.note_id = note.id AND revision.user_id IS NULL;

WITH ranked AS (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY note_id ORDER BY id) AS revision_version
    FROM note_revisions
)
UPDATE note_revisions AS revision
SET version = ranked.revision_version
FROM ranked
WHERE revision.id = ranked.id AND revision.version IS NULL;

ALTER TABLE note_revisions ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE note_revisions ALTER COLUMN version SET NOT NULL;
ALTER TABLE note_revisions ADD CONSTRAINT fk_note_revisions_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_note_revision_version ON note_revisions (note_id, version);
CREATE INDEX IF NOT EXISTS idx_note_revisions_user_id ON note_revisions (user_id);

UPDATE notes AS note
SET version = latest.version
FROM (
    SELECT note_id, MAX(version) AS version FROM note_revisions GROUP BY note_id
) AS latest
WHERE note.id = latest.note_id AND note.version < latest.version;

CREATE TABLE IF NOT EXISTS refresh_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    token_id VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    replaced_by_token_id VARCHAR(64),
    created_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user_id ON refresh_sessions (user_id);

CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    aggregate_type VARCHAR(80) NOT NULL,
    aggregate_id BIGINT NOT NULL,
    aggregate_version BIGINT NOT NULL,
    event_type VARCHAR(120) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    available_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_outbox_aggregate_event
    ON outbox_events (aggregate_type, aggregate_id, aggregate_version, event_type);
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events (status, available_at);

CREATE TABLE IF NOT EXISTS background_jobs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    job_type VARCHAR(120) NOT NULL,
    deduplication_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMP NOT NULL,
    locked_at TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_background_jobs_deduplication ON background_jobs (deduplication_key);
CREATE INDEX IF NOT EXISTS idx_background_jobs_pending ON background_jobs (status, available_at);

CREATE TABLE IF NOT EXISTS ai_runs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    note_id BIGINT REFERENCES notes(id) ON UPDATE CASCADE ON DELETE CASCADE,
    base_version BIGINT NOT NULL,
    operation VARCHAR(80) NOT NULL,
    provider VARCHAR(80),
    model VARCHAR(120),
    prompt_version VARCHAR(80),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    input_tokens BIGINT,
    output_tokens BIGINT,
    error_code VARCHAR(80),
    created_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ai_runs_note_version ON ai_runs (note_id, base_version);

CREATE TABLE IF NOT EXISTS ai_candidates (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    note_id BIGINT NOT NULL REFERENCES notes(id) ON UPDATE CASCADE ON DELETE CASCADE,
    ai_run_id BIGINT REFERENCES ai_runs(id) ON UPDATE CASCADE ON DELETE SET NULL,
    base_version BIGINT NOT NULL,
    candidate_md TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL,
    decided_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ai_candidates_note_status ON ai_candidates (note_id, status);
