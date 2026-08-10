ALTER TABLE notes ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE note_revisions ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE note_revisions ADD COLUMN version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE note_revisions ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE note_revisions ADD COLUMN content_html TEXT NOT NULL DEFAULT '';
ALTER TABLE note_revisions ADD COLUMN source TEXT NOT NULL DEFAULT 'user';

UPDATE note_revisions
SET user_id = (SELECT notes.user_id FROM notes WHERE notes.id = note_revisions.note_id),
    title = (SELECT notes.title FROM notes WHERE notes.id = note_revisions.note_id),
    content_html = (SELECT notes.content_html FROM notes WHERE notes.id = note_revisions.note_id),
    version = (
        SELECT COUNT(*) FROM note_revisions AS prior
        WHERE prior.note_id = note_revisions.note_id AND prior.id <= note_revisions.id
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_note_revision_version ON note_revisions (note_id, version);
CREATE INDEX IF NOT EXISTS idx_note_revisions_user_id ON note_revisions (user_id);
UPDATE notes
SET version = COALESCE((SELECT MAX(version) FROM note_revisions WHERE note_id = notes.id), 1);

CREATE TABLE IF NOT EXISTS refresh_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    token_id TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME,
    replaced_by_token_id TEXT,
    created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user_id ON refresh_sessions (user_id);

CREATE TABLE IF NOT EXISTS outbox_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    aggregate_type TEXT NOT NULL,
    aggregate_id INTEGER NOT NULL,
    aggregate_version INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    available_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    processed_at DATETIME
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_outbox_aggregate_event
    ON outbox_events (aggregate_type, aggregate_id, aggregate_version, event_type);
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events (status, available_at);

CREATE TABLE IF NOT EXISTS background_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    job_type TEXT NOT NULL,
    deduplication_key TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at DATETIME NOT NULL,
    locked_at DATETIME,
    last_error TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_background_jobs_deduplication ON background_jobs (deduplication_key);
CREATE INDEX IF NOT EXISTS idx_background_jobs_pending ON background_jobs (status, available_at);

CREATE TABLE IF NOT EXISTS ai_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    note_id INTEGER REFERENCES notes(id) ON UPDATE CASCADE ON DELETE CASCADE,
    base_version INTEGER NOT NULL,
    operation TEXT NOT NULL,
    provider TEXT,
    model TEXT,
    prompt_version TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    input_tokens INTEGER,
    output_tokens INTEGER,
    error_code TEXT,
    created_at DATETIME NOT NULL,
    completed_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_ai_runs_note_version ON ai_runs (note_id, base_version);

CREATE TABLE IF NOT EXISTS ai_candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    note_id INTEGER NOT NULL REFERENCES notes(id) ON UPDATE CASCADE ON DELETE CASCADE,
    ai_run_id INTEGER REFERENCES ai_runs(id) ON UPDATE CASCADE ON DELETE SET NULL,
    base_version INTEGER NOT NULL,
    candidate_md TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL,
    decided_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_ai_candidates_note_status ON ai_candidates (note_id, status);
