# Agent Skill Review Fixes SPEC

## 1. Background

Recent code review of local commits on `feature/intelligent-recruiting-agent-upgrade` identified three correctness gaps in the Agent Skill confirmation, embedding regeneration, and semantic debug flows:

- HR AI chat retry does not render the Agent Skill selection confirmation card when a retried stream emits `agent_skill_selection_required`.
- Single Agent Skill embedding regeneration can return `0 success / 0 failed / 0 skipped` and the frontend treats that as success.
- Semantic retrieval debug metadata is read from `EmbeddingService.LastSearchMeta()` after memory retrieval, so provider/model/dim/candidate_count/latency may describe memory search rather than skill search.

This feature packages those review findings into a controlled SPEC + SDD + Harness workflow.

## 2. Goals

- Ensure HR AI chat retry handles Agent Skill confirmation events consistently with normal submit and confirmed submit flows.
- Ensure single Agent Skill embedding regeneration has deterministic user-facing and service-level results.
- Ensure semantic debug embedding metadata corresponds to the intended Skill retrieval phase and is not overwritten by later Memory retrieval.
- Add focused regression tests for each fix.
- Keep changes scoped to the affected frontend and Go service files.

## 3. Non-Goals

- Do not redesign Agent Skill selection ranking.
- Do not change database schema.
- Do not introduce new dependencies.
- Do not redesign the SSE protocol beyond handling the existing `agent_skill_selection_required` event correctly.
- Do not change authentication, authorization, or permissions.
- Do not alter unrelated Agent Skill management UI behavior.

## 4. User-Facing Behavior

- When retrying a failed HR AI assistant response, if the backend asks for Agent Skill confirmation, the retry response area must show the same confirmation card used by normal submission.
- The retry flow must not refresh session messages over the pending confirmation card.
- When regenerating an Agent Skill embedding, the UI must show success only if at least one embedding was actually regenerated.
- If the target Skill is missing, disabled, unpublished, or has no embeddable content, the UI must show a warning or error rather than success.
- Semantic retrieval debug metadata must accurately represent the Skill retrieval phase shown in the Skill debug result area.

## 5. Functional Requirements

### FR-001 Retry Skill Confirmation

- `retry()` in `hr-frontend/src/views/hr/AIChatView.vue` must handle `agent_skill_selection_required` events.
- On that event, retry must call the same confirmation message creation path used by `submit()`.
- Retry must track that Skill confirmation is pending and skip post-stream fallback refresh that would replace the confirmation card.
- The implementation should reduce duplicated stream status handling where practical.

### FR-002 Embedding Regeneration Result Semantics

- Single-object embedding regeneration for `agent_skill` must never report an all-zero result as success.
- When `object_id` is specified and no eligible Skill is found, backend result semantics must indicate skipped/not found/failed clearly.
- Frontend success toast must require `success_count > 0`.
- Frontend must surface skipped or all-zero results as warning/non-success.

### FR-003 Semantic Debug Metadata Accuracy

- Semantic debug Skill metadata must be captured from the Skill embedding search before Memory debug search can overwrite any shared search state.
- `DebugSemanticRetrieval` must use request-local metadata for the Skill search response fields.
- The fix must avoid relying on mutable service-level "last search" state for the response when multiple retrieval phases occur.

## 6. Non-Functional Requirements

- Keep implementation small and reviewable.
- Preserve existing API shapes unless explicitly required by the TASK and confirmed by the user.
- Avoid broad refactors.
- Preserve existing logging conventions.
- Avoid casual `any` in TypeScript.
- Preserve current user-facing Chinese copy style.

## 7. Compatibility Requirements

- Existing HR AI chat streaming behavior must continue to work for normal submit, confirmed Skill selection, retry, stop, and timeout paths.
- Existing `BackfillEmbeddings` RPC and HTTP gateway behavior for batch operations must remain compatible.
- Existing semantic debug response fields must remain populated; their meaning must become more accurate.
- Existing generated protobuf files must not be modified unless a TASK explicitly requires public API changes and user confirmation.

## 8. Observability and Debug Requirements

- Retry confirmation handling should preserve existing debug status and context usage updates.
- Embedding regeneration should keep existing backend error logging and frontend debug logs.
- Semantic debug metadata should remain visible via existing response fields: `embedding_provider`, `embedding_model`, `embedding_dim`, `candidate_count`, and `query_embedding_latency_ms`.

## 9. Error Handling and Fallback Requirements

- Retry stream errors must continue to mark the assistant message failed.
- Retry Skill confirmation must not be treated as a stream failure.
- Embedding regeneration must distinguish actual success from skipped/no-op results.
- Semantic debug must continue to return fallback Skill/Memory rankings if embedding retrieval is unavailable.

## 10. Security and Safety Requirements

- Do not change permissions or route authorization.
- Do not expose secrets in embedding error details.
- Do not loosen HR session ownership checks.
- Do not add endpoints or public API fields without explicit confirmation.

## 11. Acceptance Criteria

- AC-001: A retried failed assistant message renders the Agent Skill confirmation card when the stream emits `agent_skill_selection_required`.
- AC-002: Retry does not call the post-stream message refresh path when Skill confirmation is pending.
- AC-003: Frontend embedding regeneration shows success only when `success_count > 0`.
- AC-004: Backend single-object Agent Skill backfill reports a skipped/non-success result when no eligible row exists.
- AC-005: Semantic debug response metadata for Skill retrieval is not overwritten by Memory retrieval.
- AC-006: Focused frontend and Go tests cover the changed behavior where feasible.

## 12. Out of Scope

- Ranking algorithm changes.
- New database migrations.
- UI redesign of chat bubbles or Agent Skill management page.
- New embedding provider configuration features.
- Protobuf schema changes unless a later TASK receives explicit confirmation.

## 13. Assumptions Requiring Confirmation

- All fixes should preserve current public request/response schemas if possible.
- Returning `skipped_count = 1` for an ineligible single-object backfill is acceptable if no new response fields are added.
- Frontend unit tests may be added if existing test setup supports the touched components.

## 14. Open Questions

- Should a missing Agent Skill ID during regeneration be represented as `skipped_count = 1` or a hard HTTP/RPC error?
- Should semantic debug eventually expose separate Skill and Memory metadata fields, or keep the existing generic embedding metadata fields scoped to Skill retrieval?
