# PR Review Fixes SPEC

## 1. Background

Code review of PR `icecreamovo147/smart-recruit#1` identified five production-readiness issues:

- CI does not run for pull requests targeting `dev`.
- LLM default model writes are scoped per provider while runtime default lookup is global.
- Agent SKILL status changes do not keep semantic embeddings in sync.
- LLM and embedding provider `extra_headers_json` handling can store invalid JSON and may expose unmasked secrets.
- New management list APIs do not consistently cap `page_size`.

The user confirmed the product and operational decisions:

- All feature PRs target `dev`; `dev` is later merged into `main`.
- LLM default model must be globally unique.
- Disabling a SKILL must logically invalidate historical embeddings, not physically delete them.
- Provider extra headers must be displayed masked, never raw.
- Management list `page_size` must be capped at 100.

## 2. Goals

- Ensure CI runs on PRs targeting `dev` and `main`.
- Make enabled default LLM model globally unique and enforce this in service and database layers.
- Keep Agent SKILL embeddings consistent when a SKILL is enabled, disabled, updated, or has a version activated.
- Validate provider extra headers before persistence and return only masked header values to clients.
- Apply a shared pagination normalization policy to management list APIs with max `page_size = 100`.

## 3. Non-Goals

- Do not redesign the whole AI provider configuration UI.
- Do not change the user-facing route names or protobuf service names unless required by existing generated APIs.
- Do not physically delete existing `ai_embeddings` rows for disabled SKILLs.
- Do not add new external dependencies.
- Do not alter unrelated recruitment, candidate, interview, offer, or notification behavior.

## 4. User-Facing Behavior

- Pull requests to `dev` run the same CI checks as pull requests to `main`.
- When an admin marks an LLM model as default, that model becomes the runtime default across all providers.
- Only one enabled model can be the global default at any time.
- When an admin enables a SKILL, it becomes eligible for semantic embedding generation and semantic retrieval after the embedding worker processes the event.
- When an admin disables a SKILL, existing embedding rows remain for history but no longer participate in semantic retrieval.
- Provider extra headers are accepted only as valid JSON objects and displayed with masked values.
- List views remain functional if a client asks for large page sizes; backend clamps to 100.

## 5. Functional Requirements

- FR-001: CI workflow must trigger for `pull_request` events targeting `dev` and `main`.
- FR-002: LLM default model selection must be globally unique among enabled models.
- FR-003: LLM default model create/update operations must be transactionally safe.
- FR-004: Database constraints must prevent multiple enabled global default LLM models.
- FR-005: Disabling an Agent SKILL must logically invalidate all ready embeddings for that SKILL.
- FR-006: Enabling an Agent SKILL must publish an embedding upsert event when the SKILL has indexable text.
- FR-007: Activating an Agent SKILL version or updating indexable metadata must refresh the SKILL embedding.
- FR-008: Embedding searches must not return logically inactive SKILL embeddings.
- FR-009: `extra_headers_json` must be valid JSON object data with string values before being persisted.
- FR-010: Provider responses must return masked extra header values only.
- FR-011: Invalid legacy extra header data must not be returned raw.
- FR-012: Management list services must normalize pagination to `page >= 1`, default `page_size = 20`, max `page_size = 100`.

## 6. Non-Functional Requirements

- Changes must be small, testable, and task-scoped.
- Security-sensitive values must never be logged or returned raw.
- DB writes for default model changes must be safe under concurrent admin requests.
- Embedding invalidation must be idempotent.
- Existing public route paths and protobuf message shapes should remain compatible.

## 7. Compatibility Requirements

- Existing frontend calls to provider/model/SKILL/list APIs must continue to work.
- Existing `ai_embeddings.status = 'ready'` search behavior must remain valid.
- Existing migration runner and MySQL migration consistency tests must continue to pass.
- Existing fallback behavior for no default LLM model may continue to choose the first enabled model only when no enabled default exists.
- Existing `page_size` callers requesting values under or equal to 100 must observe no behavior change.

## 8. Observability and Debug Requirements

- Embedding invalidation/upsert operations should log object type, object ID, status transition, and result, without raw text or secrets.
- Provider header validation errors should return clear admin-facing messages without exposing secret values.
- Default model conflicts should be visible through tests and database constraints.

## 9. Error Handling and Fallback Requirements

- If extra headers are invalid, create/update/test provider operations must fail with invalid argument or business error.
- If setting a default LLM model violates constraints, return a controlled service error and do not leave partial state.
- If embedding event publishing is unavailable, existing best-effort semantics may remain, but status updates must not fail solely because the async publisher is absent.
- Pagination normalization must clamp oversized values rather than reject requests.

## 10. Security and Safety Requirements

- Never return raw API keys or raw extra header secrets.
- Do not store malformed extra headers that may later be returned raw or silently ignored.
- Do not broaden authorization or permissions.
- Do not change auth, RBAC, or data-scope semantics.
- Do not delete historical embedding data when disabling a SKILL.

## 11. Acceptance Criteria

- AC-001: CI workflow includes `dev` in pull request branch filters.
- AC-002: Tests prove setting a default model in provider B clears provider A's default.
- AC-003: Database migration enforces at most one enabled global default LLM model.
- AC-004: Enabling a previously disabled SKILL publishes an upsert event or calls equivalent indexing logic.
- AC-005: Disabling a SKILL changes existing SKILL embeddings to an inactive status and search excludes them.
- AC-006: Invalid `extra_headers_json` is rejected and never echoed raw.
- AC-007: Masked extra headers are returned for LLM and embedding provider list/detail responses.
- AC-008: Management services clamp `page_size > 100` to 100 in tests.
- AC-009: Existing Go tests for touched services pass.
- AC-010: Harness scope checks pass for each task.

## 12. Out of Scope

- UI redesign beyond any necessary masked header display compatibility.
- Full secret management system replacement.
- Physical purge jobs for old embeddings.
- New CI providers or deployment pipelines.
- New external libraries.

## 13. Assumptions Requiring Confirmation

All previously open product decisions have been confirmed by the user. No additional assumptions are required before TASK-001.

## 14. Open Questions

None for the current scope.
