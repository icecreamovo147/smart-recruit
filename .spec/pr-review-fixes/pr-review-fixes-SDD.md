# PR Review Fixes SDD

## 1. Existing Architecture Summary

The repository contains Vue frontends, a Go gRPC logic service, and a Go Gin HTTP gateway. The reviewed issues are concentrated in:

- `.github/workflows/ci.yml`
- `logic-grpc-service/service/llm_config_service.go`
- `logic-grpc-service/repository/model_config_repo.go`
- `logic-grpc-service/migrations/`
- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/embedding_service.go`
- `logic-grpc-service/repository/ai_embedding_repo.go`
- `logic-grpc-service/service/embedding_config_service.go`
- management services with list pagination

The migration system is versioned SQL under `logic-grpc-service/migrations/` and has MySQL consistency tests. Agent default uniqueness already uses a generated column plus unique key in migration `000039`, which is a local precedent for enforcing conditional uniqueness.

`ai_embeddings` already has a `status` column. `EmbeddingService.Search` filters candidates with `EmbeddingStatusReady`, making logical invalidation feasible without deleting rows.

## 2. Problem Analysis

- CI is configured for PRs to `main` only, but the team's integration branch is `dev`.
- LLM default model writes clear defaults by provider, but runtime lookup reads the first enabled default globally.
- Agent SKILL status updates bypass embedding event refresh/invalidation.
- Provider extra headers are accepted as raw strings. LLM masking returns the original string on parse failure; embedding provider responses currently omit headers entirely.
- Multiple list services normalize only non-positive `page_size`, allowing very large limits.

## 3. Proposed Design

### CI

Update the workflow pull request branch filter to include both `dev` and `main`.

### Global LLM Default Model

Introduce global default semantics:

- Add a migration that cleans duplicate enabled defaults, keeping the lowest ID or a deterministic single row.
- Add a generated column such as `global_default_key` that is non-null only when `is_default = 1 AND is_enabled = 1`.
- Add a unique key on that generated column.
- Replace provider-scoped default clearing with global clearing.
- Wrap create/update default operations in repository transactions.

### Agent SKILL Embedding Lifecycle

Use logical invalidation:

- Add repository/service methods to mark embeddings inactive by object type and object ID.
- Add status constant, for example `EmbeddingStatusInactive = "inactive"`.
- On SKILL disable: mark existing `agent_skill` embeddings inactive.
- On SKILL enable: publish embedding upsert if the SKILL is enabled and has indexable content.
- On SKILL version activation or indexable metadata update: publish upsert as today.

No physical deletion is allowed.

### Extra Headers

Add shared validation/masking helpers in the logic service:

- Parse JSON into `map[string]string`.
- Reject non-object JSON, non-string values, empty invalid keys, and oversized payloads.
- Canonicalize to compact JSON before persistence.
- Mask all values before responses.
- For invalid legacy stored data, return `{}` or empty string and log a warning without raw content.

Apply the helper to both LLM provider and embedding provider config services.

### Pagination

Add shared pagination normalization helper:

- `page <= 0` becomes `1`.
- `page_size <= 0` becomes `20`.
- `page_size > 100` becomes `100`.

Apply to management list services without changing API fields.

## 4. Data Structure Changes

- Add one new LLM migration for global default uniqueness.
- No schema change is required for SKILL embedding logical invalidation because `ai_embeddings.status` already exists.
- No protobuf field change is required.

## 5. API and Interface Changes

- No route path changes.
- No protobuf message shape changes.
- Provider create/update behavior changes by rejecting invalid `extra_headers_json`.
- Provider response behavior changes by returning masked header JSON instead of raw or empty values where applicable.
- List behavior changes by clamping page size to 100.

## 6. Algorithm or Workflow Changes

- LLM default workflow becomes transactional and global.
- SKILL status workflow coordinates DB state and embedding status state.
- Header workflow becomes validate-on-write and mask-on-read.
- List workflow normalizes pagination via shared helper.

## 7. Configuration Design

- CI branch configuration changes in `.github/workflows/ci.yml`.
- No runtime config additions are required.

## 8. Compatibility Strategy

- Keep existing request/response protobuf fields.
- Keep default model fallback to first enabled model only when no enabled default exists.
- Clamp pagination rather than fail oversized requests.
- Preserve historical embeddings by changing status only.
- Frontend can continue to submit `extra_headers_json`; invalid input now receives a clear validation failure.

## 9. Error Handling and Fallback Design

- Invalid headers return invalid argument errors.
- Default model constraint failures return controlled service errors.
- Embedding invalidation failures should fail the SKILL status update only if the DB update itself fails; best-effort async upsert remains non-blocking.
- Legacy invalid header data is not returned raw.

## 10. Observability and Debug Output Design

- Log default model transaction failures with model ID and provider ID, not secrets.
- Log embedding invalidation counts where available.
- Log invalid legacy header data as a warning without raw values.
- Existing CI output remains the primary signal for workflow verification.

## 11. Testing Strategy

- CI task: no runtime tests required beyond YAML inspection and existing checks.
- LLM task: service/repository tests for global default behavior; migration consistency where MySQL is available.
- SKILL embedding task: service and repository tests for inactive status and search filtering.
- Header task: service tests for validation, canonicalization, masking, invalid legacy data, and provider test behavior.
- Pagination task: service tests for normalization in LLM, embedding, MCP, prompt, skill, and agent config list paths.

## 12. Migration Risks

- Existing databases may already contain multiple enabled default LLM models. The migration must deterministically clean duplicates before adding the unique key.
- Generated-column syntax must be compatible with the repository's MySQL version target.
- `db.sql` must be updated with migration-equivalent schema if migration consistency tests require it.

## 13. Implementation Boundaries

- Each TASK must modify only its allowed files.
- Do not update package manifests or lockfiles.
- Do not change authorization, role, or data-scope behavior.
- Do not change public route paths.
- Do not physically delete embedding rows.

## 14. Alternatives Considered

- Per-provider LLM defaults plus agent-level provider selection: rejected because the user confirmed a globally unique default model.
- Physical deletion of disabled SKILL embeddings: rejected because the user requires logical invalidation.
- Reject oversized `page_size`: rejected in favor of clamping for compatibility.
- Returning raw headers for edit forms: rejected because the user requires masked display.

## 15. Assumptions Requiring Confirmation

None. The user confirmed all required product decisions.

## 16. Open Questions

None for the current scope.
