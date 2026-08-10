# AI feature delivery playbook

Use this checklist for every AI capability. The first intended application is candidate-based editing; the same rules apply to summarization, metadata generation, retrieval, and publishing assistance.

## 1. Define the user contract

- State the exact user action and expected benefit.
- Define input scope: full note, selected text, retrieved sources, or metadata.
- Define cancellation, timeout, retry, quota, and offline/error behavior.
- Decide whether the result is advisory, a candidate, or an approved write. AI must not publish automatically by default.

## 2. Preserve data safety

- Capture `user_id`, `note_id`, and base `version` before work starts.
- Store generated content separately as an `ai_candidate`.
- On acceptance, call the note use-case with the base version.
- Return a conflict when the note changed while generation was running.
- Use an idempotency/deduplication key for retries.
- Never place model credentials or private prompt payloads in browser code or logs.

## 3. Make execution observable

Record at least:

- AI run ID, operation, provider, model, and prompt version.
- Start/end time, status, cancellation, and normalized error code.
- Input/output token counts and estimated cost.
- Base note version and resulting candidate or accepted revision.
- User feedback without storing unnecessary sensitive content.

## 4. Test and evaluate

- Unit-test prompt/input construction and structured-output validation.
- Test timeout, cancellation, provider failure, retry limits, and stale-version rejection.
- Add cross-user isolation and prompt-injection cases when retrieval or tools are involved.
- Create a representative eval dataset with an explicit metric and threshold.
- Include regression examples for every material production failure.

## 5. Review experience

- Show source context and what the model changed.
- Present a diff before acceptance.
- Allow partial acceptance or rejection when the operation supports it.
- Keep the original note and accepted revisions recoverable.
- Make latency, failure, and cancellation states understandable rather than leaving indefinite loading.

## Definition of done

An AI feature is not complete with a successful model call alone. It requires safe candidate storage, version-aware acceptance, failure UX, observability, representative evals, security tests, and the ordinary project quality gates.
