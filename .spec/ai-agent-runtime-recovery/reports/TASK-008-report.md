# TASK Report - TASK-008

## 1. TASK ID

TASK-008 - Restore Embedding and Semantic Retrieval Runtime

## 2. Status

Completed.

Independent self-review round 1 returned `不通过` for stale `agent_skill` embeddings remaining `ready` after content/version changes. The implementation now invalidates previous `agent_skill` embedding rows before upserting the current text hash, and tests cover duplicate/stale row prevention.

Independent self-review round 2 returned `不通过` because this report and the machine-readable evidence file had not yet been created. This report and `TASK-008-evidence.json` were then generated.

Independent self-review round 3 returned `通过`.

Human confirmation was not required for TASK-008.

## 3. Modified File List

- `.spec/ai-agent-runtime-recovery/pipeline-state.json`
- `.spec/ai-agent-runtime-recovery/reports/TASK-008-report.md`
- `.spec/ai-agent-runtime-recovery/reports/TASK-008-evidence.json`
- `smart-recruit-ai-agent-service/internal/domain/policy/capability_test.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/provider/embedding_test.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/config_services.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`

`git diff --name-only` does not list untracked files before staging; the new embedding provider files are included from `git status --short` and verified by the TASK scope checker.

## 4. Change Summary by File

- `internal/infrastructure/provider/embedding.go`: adds an embedding runtime runner and service for OpenAI-compatible HTTP embedding calls, model validation, `agent_skill` backfill, upsert/invalidation, semantic debug retrieval, cosine ranking, fallback reporting, and semantic score extraction for Agent Skill selection.
- `internal/infrastructure/provider/embedding_test.go`: verifies configured endpoint/model calls, backfill, debug metadata, semantic scores, explicit fallback behavior, and stale duplicate invalidation for updated Agent Skill embeddings.
- `internal/infrastructure/persistence/config_store.go`: adds `ai_embeddings` persistence records plus native store methods to resolve embedding provider/model config, update model test status, build Agent Skill embedding documents, upsert/invalidate/list embedding rows, and decrypt configured provider credentials for runtime use.
- `internal/interfaces/grpc/config_services.go`: routes `TestEmbeddingModel` and `BackfillEmbeddings` through the native embedding service when bound.
- `internal/interfaces/grpc/mcp_skill_services.go`: triggers synchronous Agent Skill embedding upsert or invalidation after create, update, activate, status change, and create-version-with-activate flows; routes Semantic Retrieval Debug through the native embedding service when bound.
- `internal/interfaces/grpc/native_servers.go`: wires `RuntimeDeps.EmbeddingRunner` into the native runtime, shares the embedding service with config, Agent Skill, and HR AI runtime services, and injects semantic scores into HR Agent Skill selection when available.
- `internal/domain/policy/capability_test.go`: adds coverage that semantic scores can outrank rule priority during automatic Agent Skill selection.
- `pipeline-state.json`: records TASK-008 base tree and non-required human confirmation.

## 5. Scope Check Result

Passed.

Command:

```bash
TASK_BASE_TREE=d037ab7dc3d115ff168264a629917e41fc080ee6 bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-008
```

Result:

```text
scope_result: PASS (TASK-008)
```

## 6. SPEC Comparison Result

Passed.

The implementation restores runtime embedding validation, `agent_skill` embedding backfill, Agent Skill upsert/invalidation triggers, Semantic Retrieval Debug metadata and scoring fields, semantic score use in HR Agent Skill selection, and explicit fallback behavior without proto, schema, dependency, auth/RBAC, global config, root docs, or K8s changes.

## 7. SDD Comparison Result

Passed.

Embedding runtime behavior is kept inside the AI Agent microservice through an injected runner/service under `internal/infrastructure/provider`. `NativeStore` remains a persistence adapter and does not import or depend on the old monolith runtime.

## 8. Acceptance Comparison Result

Passed.

- Embedding model test calls the configured provider/model through the runner or returns explicit non-success runtime details.
- `BackfillEmbeddings` supports `object_type=agent_skill` and writes current Agent Skill embeddings.
- Agent Skill create, update, activation, status change, and create-version-with-activate flows trigger embedding upsert or invalidation.
- Semantic Retrieval Debug returns provider, model, dimension, query latency, candidate count, vector/lexical/metadata/final scores, confidence, skills, and fallback reason.
- HR Agent Skill selection passes semantic scores into the domain selector when available and falls back to rule scoring when embeddings are unavailable.

## 9. Test Commands and Results

| Command | Result |
|---|---|
| `GOWORK=off go test ./internal/infrastructure/provider -run 'TestHTTPEmbeddingRunnerCallsConfiguredEndpoint|TestEmbeddingServiceBackfillDebugAndSemanticScores|TestEmbeddingServiceExplicitFallbackWhenRunnerFails' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./internal/domain/policy -run 'TestSkillVersionAndSelectionPolicy' -count=1 -v` from `smart-recruit-ai-agent-service` | Passed |
| `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service` | Passed |
| `git diff --name-only` | Passed; command does not list untracked new provider files before staging |
| `git status --short` | Passed; used to record untracked new embedding provider files |
| `TASK_BASE_TREE=d037ab7dc3d115ff168264a629917e41fc080ee6 bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-008` | Passed |
| `TASK_BASE_TREE=d037ab7dc3d115ff168264a629917e41fc080ee6 bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh` | Passed |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree d037ab7dc3d115ff168264a629917e41fc080ee6` | Passed; result `update_required` |

Gateway and HR frontend files were not touched, so gateway-specific mapping changes and `pnpm --filter hr-frontend typecheck` were not required by TASK-008 acceptance. `agent-check.sh` still ran gateway tests due earlier staged gateway changes and they passed.

## 10. Knowledge Impact

Result: `update_required`.

Routed active knowledge was reviewed: `agent-runtime`, `semantic-retrieval`, `agent-skill`, `memory-and-context`, `ai-configuration-governance`, `embedding-fallback`, and `debug-agent-retrieval`. The impact detector also matched broad service, persistence, gateway, protobuf, and migration documents.

No `.knowledge/**` files were edited because they are outside TASK-008 scope. Knowledge debt is recorded for native embedding runtime support, Semantic Retrieval Debug behavior, and Agent Skill semantic selection replacing previous unsupported/fallback-only behavior.

## 11. Human Confirmation

Not required.

## 12. Dev Reference

Read-only dev reference subagent inspected:

- `origin/dev:logic-grpc-service/service/embedding_config_service.go`
- `origin/dev:logic-grpc-service/service/embedding_service.go`
- `origin/dev:logic-grpc-service/service/embedding_backfill_service.go`
- `origin/dev:logic-grpc-service/service/embedding_event_publisher.go`
- `origin/dev:logic-grpc-service/service/embedding_event_consumer.go`
- `origin/dev:logic-grpc-service/service/agent_skill_service.go`
- `origin/dev:logic-grpc-service/service/agent_skill_selector.go`
- `origin/dev:logic-grpc-service/service/skill_memory_ranking.go`
- `origin/dev:logic-grpc-service/repository/ai_embedding_repo.go`

The restored behavior follows dev semantics for provider/model validation, Agent Skill embedding text, backfill, invalidation, debug score fields, and semantic selection while preserving the current microservice boundary.

## 13. Risks

- The default runner uses an OpenAI-compatible HTTP embeddings request shape. Non-compatible provider types may need a separately scoped adapter.
- Agent Skill embedding upsert is synchronous best-effort through the native service. If the provider is unavailable, the response remains successful for the Agent Skill change while the embedding row records failure or the debug endpoint reports explicit fallback.
- Memory retrieval debug remains empty in this TASK because acceptance focuses on Agent Skill runtime recovery and no memory embedding store/runtime was in TASK scope.

## 14. Follow-up Items

- Update `.knowledge/**` semantic retrieval and embedding fallback documents in a future knowledge-maintenance scope.
- Add provider-specific adapters if product requirements need non-OpenAI-compatible embedding APIs.
- Add asynchronous worker/event integration in a separately scoped task if synchronous best-effort upsert is insufficient operationally.

## 15. Repair Summary

### Failed Review Round 1

Independent reviewer Sagan reported:

- Stale Agent Skill embeddings stayed `ready` after text/content/version updates because upsert uniqueness includes `text_hash`.

### Root Cause

The first implementation inserted a new ready row for changed text while leaving the previous hash row ready. Semantic debug and selection listed all ready rows for the model, so one Agent Skill could appear more than once or rank from stale vectors.

### Round 1 Fix Summary

- `EmbeddingService.UpsertAgentSkillDocument` now invalidates existing `agent_skill` rows for the object before writing the current row.
- Provider tests now mutate an Agent Skill and assert Semantic Retrieval Debug returns only one current entry for that skill.

### Failed Review Round 2

Independent reviewer Nash reported:

- Required TASK report and evidence artifacts were missing.

### Round 2 Fix Summary

- Created `.spec/ai-agent-runtime-recovery/reports/TASK-008-report.md`.
- Created `.spec/ai-agent-runtime-recovery/reports/TASK-008-evidence.json`.

## 16. Review Rounds

- Round 1: `不通过`; stale ready Agent Skill embedding rows.
- Round 2: `不通过`; report/evidence artifacts missing. Code findings: none.
- Round 3: `通过`; no blocking findings.

## 17. Whether the Next TASK Can Start

Yes. TASK-008 checks passed and independent self-review returned `通过`; TASK-009 can start.
