# ADR 0002: OpenAPI-first generated frontend client

**Status**: Accepted
**Date**: 2026-08-10

## Context

Handwritten frontend DTOs and requests drifted from backend responses. Public and authenticated pages also used different request styles.

## Decision

Use `backend/docs/api/api-contract-v1.yaml` as the API source of truth. Orval generates TypeScript models and React Query hooks through the shared Axios requester. CI regenerates the client and rejects drift.

## Consequences

- `frontend/src/api/gen` must never be edited manually.
- Contract changes include YAML, backend behavior/tests, regenerated files, and relevant mocks.
- Human-readable API prose is secondary; it must not contradict the YAML.
- Authentication may use a small explicit service where interceptor/session behavior makes generated hooks inappropriate, but token storage remains outside React context.
