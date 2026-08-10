# ADR 0004: Use embedded versioned SQL migrations

**Status**: Accepted
**Date**: 2026-08-10

## Context

Production schema changes were previously driven by GORM AutoMigrate, which provided no auditable version history or reviewed rollback path.

## Decision

Use ordered PostgreSQL and SQLite SQL migration pairs embedded in the Go binary. Record version, name, checksum, and application time. Application startup moves forward; rollback is an explicit operator command. PostgreSQL serializes concurrent migration attempts with a migration-table lock.

## Consequences

- Applied migrations are immutable; corrections use a new migration.
- Every schema change needs both dialects, up/down SQL, and migration tests.
- Rollback remains operationally risky and requires a verified backup.
- GORM AutoMigrate is allowed only inside the legacy-upgrade compatibility test.
