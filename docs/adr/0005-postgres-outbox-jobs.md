# ADR 0005: Start asynchronous work with PostgreSQL outbox and jobs

**Status**: Accepted
**Date**: 2026-08-10

## Context

Indexing and AI generation will outlive HTTP requests and need retry, cancellation, and idempotency. An external broker would add operational complexity before workload characteristics are known.

## Decision

Use transactional `outbox_events` and PostgreSQL-backed `background_jobs` as the initial durable boundary. Keep job payloads provider-neutral and require deduplication keys. Do not implement a separate message cluster yet.

## Consequences

- Note changes cannot be lost between the database and a broker publish.
- Workers must claim rows safely, use bounded retries, and expose failures.
- Retention, dead-letter handling, and worker scaling remain future work.
- A broker may be introduced later without moving transaction rules out of noteapp.
