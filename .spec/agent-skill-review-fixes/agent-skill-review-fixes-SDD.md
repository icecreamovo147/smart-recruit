# Agent Skill Review Fixes SDD

## 1. Existing Architecture Summary

The HR frontend uses Vue 3 and Vite. HR chat lives in `hr-frontend/src/views/hr/AIChatView.vue`, with SSE handled through `hr-frontend/src/api/ai.ts`. Chat messages are rendered by `ChatMessageList.vue`, which emits `confirm-skill-selection` when the user confirms or skips Agent Skill use.

The Go logic service owns Agent Skill selection, embeddings, and semantic debug behavior. HR AI chat streams from `logic-grpc-service/service/ai_service.go`. Embedding backfill lives in `logic-grpc-service/service/embedding_backfill_service.go`. Semantic retrieval debug lives in `logic-grpc-service/service/agent_skill_service.go` and calls `EmbeddingService.Search` / `SearchObjects` from `logic-grpc-service/service/embedding_service.go`.

The web gateway maps HTTP requests to gRPC in `web-gin-service/handler/hr`.

## 2. Problem Analysis

### Retry Confirmation Gap

`submit()` and `submitConfirmedSkillSelection()` handle `agent_skill_selection_required`, but `retry()` only handles context usage, model info, process delta, and process clear. If a retried stream requires Skill confirmation, the backend can send a confirmation event and complete the stream, but the frontend will not render the confirmation card.

### Embedding Regeneration False Success

`backfillSkills()` filters by `object_id` when supplied, but an empty query result returns all counters as zero. The frontend currently treats no failed and no skipped rows as success, even if no embedding was regenerated.

### Semantic Debug Metadata Overwrite

`DebugSemanticRetrieval()` reads `embeddingMetaSnapshot(s.embeddings)` after `semanticDebugMemories()`. Memory debug can trigger a second embedding search and overwrite `LastSearchMeta()`, causing metadata fields in the response to describe memory search instead of skill search.

## 3. Proposed Design

### TASK-001 Retry Confirmation Handling

- Add retry-specific handling for `agent_skill_selection_required`.
- Prefer extracting a helper inside `AIChatView.vue` to centralize status-event handling across submit, confirmed submit, and retry.
- Track `skillSelectionRequired` in retry and return before post-stream refresh when true.
- Reuse `setSkillSelectionMessage()` so the UI card remains consistent.

### TASK-002 Deterministic Embedding Regeneration

- In `EmbeddingBackfillService.backfillSkills`, if `input.ObjectID > 0` and no eligible rows are found, return a non-success result, preferably `SkippedCount = 1` with an error/reason entry.
- In `AgentSkillManageView.vue`, require `result.success_count > 0` before showing success.
- Treat `failed_count > 0`, `skipped_count > 0`, or all-zero results as non-success messages.
- Keep existing API schema unless explicit confirmation is given to add details fields.

### TASK-003 Request-Local Semantic Debug Metadata

- Add a request-local metadata path for Skill embedding search.
- Recommended low-risk design: introduce `SearchWithMeta()` and `SearchObjectsWithMeta()` wrappers in `EmbeddingService` or a helper that returns `SearchMeta` alongside results, while keeping existing `Search()` and `SearchObjects()` signatures.
- Update `semanticDebugSkillScores()` to return `(map[int64]float64, SearchMeta, error)`.
- Use the Skill search meta captured before `semanticDebugMemories()` when building `DebugSemanticRetrievalResponse`.
- Do not add proto fields unless explicitly confirmed.

## 4. Data Structure Changes

- TASK-001: No persistent data structure changes.
- TASK-002: No schema changes. `BackfillResult.Errors` may carry a reason internally.
- TASK-003: Add internal Go return struct or tuple for embedding search metadata. No database or protobuf changes by default.

## 5. API and Interface Changes

- Public HTTP and gRPC API changes are not required by default.
- Internal service method signatures may change within `logic-grpc-service/service`.
- If implementation requires adding response fields to protobufs, stop and request user confirmation.

## 6. Algorithm or Workflow Changes

- Retry stream workflow must mirror submit stream workflow for Skill confirmation events.
- Single-object embedding workflow must distinguish "nothing matched" from "success".
- Semantic debug workflow must capture Skill search metadata immediately and carry it explicitly through the request.

## 7. Configuration Design

No configuration changes.

## 8. Compatibility Strategy

- Keep existing routes, request bodies, and response fields.
- Keep batch embedding backfill behavior compatible.
- Keep semantic debug UI response shape compatible.
- Preserve existing generated protobuf files unless explicitly confirmed.

## 9. Error Handling and Fallback Design

- Retry confirmation is a pending user action, not an error.
- Ineligible single-object embedding regeneration should be warning/skipped or not found, not success.
- Semantic debug should still show fallback rankings when embedding search fails, while metadata should describe the attempted Skill search.

## 10. Observability and Debug Output Design

- Preserve frontend debug logs around embedding regeneration.
- Preserve backend embedding error logs.
- Semantic debug metadata should remain visible in the existing fields and become deterministic for the Skill phase.

## 11. Testing Strategy

- TASK-001: Frontend typecheck and targeted test if an existing test harness covers `AIChatView.vue`; otherwise document manual verification.
- TASK-002: Go unit tests for `EmbeddingBackfillService` single-object no-match behavior and frontend typecheck for UI condition changes.
- TASK-003: Go unit test for semantic debug metadata using distinguishable Skill and Memory search metadata, or focused test on helper behavior if full service setup is heavy.
- Always run harness scripts after each TASK.

## 12. Migration Risks

- Existing dirty commits may contain unrelated changes; TASK scope checks must prevent accidental edits outside the current task.
- Protobuf changes would widen blast radius and require explicit confirmation.
- Frontend component tests may be hard to add if current test setup lacks stable mocks for streaming.

## 13. Implementation Boundaries

- TASK-001 may modify only HR chat frontend files and tests.
- TASK-002 may modify only embedding backfill service, Agent Skill management frontend file/API typing if needed, and focused tests.
- TASK-003 may modify only embedding service, semantic debug service, and focused tests.
- No package manifests, lockfiles, auth, permissions, database migrations, or global configuration.

## 14. Alternatives Considered

- Add new protobuf fields for detailed embedding regeneration errors. Rejected for default plan because it changes public API behavior.
- Use `LastSearchMeta()` as-is and reorder calls. Rejected because it remains fragile with multiple retrieval phases and concurrent calls.
- Duplicate confirmation handling in retry only. Acceptable as a minimal implementation, but a small shared handler is preferred if it stays local to `AIChatView.vue`.

## 15. Assumptions Requiring Confirmation

- It is acceptable to represent no eligible single-object embedding target as skipped without a public API schema change.
- Existing semantic debug generic metadata fields should represent Skill search metadata.

## 16. Open Questions

- Should future UI display separate Skill and Memory embedding metadata?
- Should single-object regeneration eventually get a dedicated endpoint/service method with richer status codes?
