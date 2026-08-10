# Known debt and open decisions

This list contains intentional gaps after R0-R4. It prevents unfinished work from being mistaken for an accidental omission.

## P0 before a public AI demo

| Item | Why it matters | Exit condition |
|---|---|---|
| Refresh token remains in localStorage | XSS can expose a long-lived credential | Decide and implement Secure, HttpOnly, SameSite cookie flow with CSRF protection |
| No AI runtime or eval gate | AI tables are only boundaries, not a product feature | First AI operation ships with candidate review, cancellation, representative evals, and CI threshold |
| No worker/outbox consumer | Pending outbox rows are transactionally durable, but no delivery or processing guarantee exists yet | Idempotent worker claims events/jobs, records retry/dead-letter outcomes, and validates note user/version |
| No AI observability | Cost, latency, quality, and failures cannot be demonstrated | Persist run metrics and expose P95/error/cost/eval reporting |

## Product decisions

- Should a published note keep a stable slug when its title changes?
- Are description and tags real persisted product fields or should the placeholder UI be removed?
- Is “private knowledge to trustworthy publication” the single primary product narrative?
- Should AI acceptance create a dedicated author/source attribution beyond revision `source`?

## Engineering debt

- Authentication handlers still access `database.DB`; a future auth application service should own session issuance and revocation.
- Root API documentation exists as both Markdown and OpenAPI YAML. YAML is authoritative; consider generating human-readable API docs from it and retiring duplicated endpoint prose.
- MDXEditor produces an approximately 748 KB lazy chunk. It does not affect the public initial route, but editor load performance should be measured and optimized.
- PostgreSQL migration SQL is exercised in CI; local development defaults to SQLite unless a PostgreSQL test DSN is configured.
- Refresh-session cleanup and retention policies are not implemented.
- Outbox/job retry, dead-letter, retention, and operational dashboards are not implemented.

## Deferred by design

- Microservices and a separate message broker.
- Kubernetes and service mesh.
- Multi-agent orchestration and dynamic multi-model routing.
- Vector search before a tested full-text/RAG baseline exists.

Update this document when a debt item is accepted, reprioritized, or completed. Link the implementing ADR or commit instead of silently deleting historical context.
