# ADR 0003: Version notes and atomically record revisions/events

**Status**: Accepted
**Date**: 2026-08-10

## Context

Timestamp-only concurrency was fragile, NoteRevision was disconnected, and future AI/indexing work needed a reliable change identity.

## Decision

Give every note a monotonically increasing integer `version`. An accepted create/update conditionally writes the expected version, an immutable revision, and an outbox event in one transaction. Stale updates return `VERSION_CONFLICT` with the current snapshot.

AI output is stored as a candidate against a base version. Acceptance must use the ordinary note use-case.

## Consequences

- `(user_id, note_id, version)` is the stable identity for workers, AI runs, and stale-work detection.
- A revision or outbox failure rolls back the note update.
- Tree moves advance the version because they mutate note state.
- Deletion emits an event so future indexes can remove stale content.
- Revision growth and retention will need an explicit policy as usage grows.
