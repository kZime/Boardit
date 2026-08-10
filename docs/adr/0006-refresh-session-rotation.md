# ADR 0006: Use server-side rotating refresh sessions

**Status**: Accepted
**Date**: 2026-08-10

## Context

Stateless refresh JWTs could not revoke one session or detect replay. Access and refresh tokens also needed explicit usage boundaries.

## Decision

Use short-lived access JWTs and persisted refresh sessions keyed by a random `jti`. A successful refresh atomically revokes the old session and creates a replacement. Reuse of the old token fails. Logout revokes the supplied session.

## Consequences

- Refresh requires a database lookup/update but gains rotation, replay rejection, and per-session revocation.
- Concurrent refresh attempts allow only one winner.
- Expired/revoked session cleanup needs a retention job.
- Browser refresh storage remains in localStorage temporarily; moving it to an HttpOnly cookie requires a separate CSRF-aware decision.
