# AI and asynchronous data boundaries

R4 establishes storage boundaries without selecting a model provider or queue product.

| Boundary | Stable identity | Purpose |
|---|---|---|
| `notes` | `id`, `user_id`, `version` | Current accepted document state |
| `note_revisions` | `note_id`, `user_id`, `version` | Immutable accepted snapshots; written atomically with note updates |
| `outbox_events` | aggregate type/id/version/event | Durable note-change handoff for future indexing and jobs |
| `background_jobs` | `user_id`, type, deduplication key | Provider-neutral retry/claim boundary for long-running work |
| `ai_runs` | `user_id`, optional note, base version | Provider/model/prompt/status/cost trace boundary |
| `ai_candidates` | note, base version, run, status | Proposed Markdown kept separate until the user accepts it |

## Required invariants

- Every note create or content/metadata update increments `notes.version`, creates one revision and creates one outbox event in the same transaction.
- Revision and API queries include `user_id`; an inaccessible note is reported as not found.
- Workers must discard or recompute work when `(note_id, user_id, version)` no longer matches the current note.
- A candidate never updates `notes.content_md` directly. Acceptance must call the note application service with the candidate's `base_version`, so stale candidates conflict instead of overwriting newer work.
- Outbox consumers and background jobs must be idempotent using their unique aggregate event or deduplication key.

The current phase intentionally does not implement workers, provider SDKs, prompt execution or automatic candidate acceptance.
