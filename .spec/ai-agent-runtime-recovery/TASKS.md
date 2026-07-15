# TASKS - ai-agent-runtime-recovery

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | Validation Baseline and Dev Reference Inventory | pending | AI Agent/Gateway `go.sum`, feature docs | `.spec/ai-agent-runtime-recovery/acceptance/TASK-001.md` |
| TASK-002 | Restore HR AI Chat Runtime | pending | AI Agent runtime, provider, persistence, gRPC | `.spec/ai-agent-runtime-recovery/acceptance/TASK-002.md` |
| TASK-003 | Restore Durable HR Agent Runs | pending | AI Agent run state, events, worker, recorder | `.spec/ai-agent-runtime-recovery/acceptance/TASK-003.md` |
| TASK-004 | Restore Candidate AI Runtime and Compatibility Route | pending | Candidate AI service, gateway candidate AI route | `.spec/ai-agent-runtime-recovery/acceptance/TASK-004.md` |
| TASK-005 | Restore Recruiting Intelligence Generation | pending | AI Agent Recruiting Intelligence generation | `.spec/ai-agent-runtime-recovery/acceptance/TASK-005.md` |
| TASK-006 | Connect Agent, Prompt, Skill, and Capability Governance to Runtime | pending | AI Agent configuration/runtime integration | `.spec/ai-agent-runtime-recovery/acceptance/TASK-006.md` |
| TASK-007 | Restore MCP Runtime Execution | pending | AI Agent MCP runner, policy, logs, gateway MCP mapping | `.spec/ai-agent-runtime-recovery/acceptance/TASK-007.md` |
| TASK-008 | Restore Embedding and Semantic Retrieval Runtime | pending | AI Agent embedding, semantic debug, Agent Skill selection | `.spec/ai-agent-runtime-recovery/acceptance/TASK-008.md` |
| TASK-009 | Verify Frontend Contract Compatibility | pending | HR/User frontend AI contracts and focused tests | `.spec/ai-agent-runtime-recovery/acceptance/TASK-009.md` |
| TASK-010 | Regression Hardening and Final Evidence | pending | Cross-module tests, smoke checklist, feature evidence | `.spec/ai-agent-runtime-recovery/acceptance/TASK-010.md` |

## TASK-001 - Validation Baseline and Dev Reference Inventory

### Goal

Unblock reliable Go validation and create a feature-owned inventory of the `origin/dev` files that will serve as behavior references for later TASKs.

### Scope

- Repair missing Go checksum entries that currently prevent `GOWORK=off go test ./...` from starting in `smart-recruit-ai-agent-service` and `smart-recruit-gateway`.
- Create `.spec/ai-agent-runtime-recovery/docs/dev-reference-inventory.md` with dev-branch reference files, current-branch target files, and expected behavior mapping.
- Do not implement runtime behavior in this TASK.

### Allowed Files

- `smart-recruit-ai-agent-service/go.sum`
- `smart-recruit-gateway/go.sum`
- `.spec/ai-agent-runtime-recovery/docs/dev-reference-inventory.md`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Business source files.
- `go.mod` files.
- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- Existing SPEC and SDD.
- `origin/dev` must be available locally.

### Acceptance Criteria

- Missing `go.sum` entry errors no longer block AI Agent service and Gateway test startup.
- The dev reference inventory maps at least HR AI, Candidate AI, Agent Run, Recruiting Intelligence, MCP, Agent Skill, and Embedding references.
- No runtime/business behavior is changed.

### Required Tests

- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway`.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Additional compile or test failures may be revealed after checksum repair; record them without broadening this TASK.

### Notes

- This TASK is deliberately small so later TASKs can rely on a clean validation baseline.

## TASK-002 - Restore HR AI Chat Runtime

### Goal

Restore HR AI Chat and ChatStream to dev-compatible tool-calling behavior inside the current AI Agent microservice boundary.

### Scope

- Build or migrate an HR Agent runtime orchestrator under `smart-recruit-ai-agent-service`.
- Restore context assembly, runtime config lookup, model resolution, tool execution, tool traces, context usage, fallback replies, and stream events.
- Keep gateway behavior compatible with existing `/api/v1/hr/ai/chat` and `/api/v1/hr/ai/chat/stream` contracts.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/provider/**`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/**`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-ai-agent-service/cmd/ai-agent-service/**`
- `smart-recruit-commons/ai/**`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Gateway routes and frontend code.
- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-001.

### Acceptance Criteria

- HR Chat uses real recruitment-aware tools and does not simply call provider `complete()` on raw prompt.
- ChatStream emits deltas, status events, context usage, tool events, fallback events, and done/error events compatible with HR frontend types.
- Tool traces are persisted and available through existing trace retrieval.
- Dev behavior references are cited in the TASK report.

### Required Tests

- Targeted AI Agent service tests for HR Chat and ChatStream.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- This TASK may need shared `smart-recruit-commons/ai` changes; human confirmation is required before editing because it is a shared module.

### Notes

- If proto, schema, auth, or public contract changes appear necessary, stop and request confirmation.

## TASK-003 - Restore Durable HR Agent Runs

### Goal

Restore durable Agent Run execution, event replay, cancellation, confirmation continuation, and status transitions using the recovered HR runtime.

### Scope

- Persist full run request payloads needed for resumable execution.
- Execute runs through the HR Agent runtime from TASK-002.
- Restore replayable event streams, run recorder behavior, terminal states, cancel behavior, and confirmation continuation.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/**`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/**`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Frontend code.
- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-001.
- TASK-002.

### Acceptance Criteria

- `CreateAgentRun` creates idempotent queued runs with enough durable payload to execute after request completion.
- Runs transition legally through queued/running/waiting_confirmation/succeeded/failed/canceled states.
- `SubscribeAgentRunEvents` replays and streams meaningful runtime events by sequence.
- `ConfirmAgentRun` continues execution after Skill confirmation.
- `CancelAgentRun` terminates queued/running work and persists canceled state.

### Required Tests

- Targeted AI Agent service tests for run create, execute, replay, confirm, cancel, idempotency, and invalid transitions.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Stuck active runs and race conditions are the main risk; tests must cover cancellation and terminal-state idempotency.

### Notes

- If missing schema fields prevent durable payload storage, stop and request schema-change confirmation.

## TASK-004 - Restore Candidate AI Runtime and Compatibility Route

### Goal

Restore Candidate AI assistant business-aware behavior and fix the missing non-streaming compatibility route.

### Scope

- Restore candidate-scoped tool execution, candidate prompt/config use, tool traces, suggested questions, fallback, and usage audit.
- Add or restore `POST /api/v1/candidate/ai/chat` compatibility through gateway aggregation or existing service behavior without changing proto by default.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/provider/**`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/**`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-gateway/handler/candidate/**`
- `smart-recruit-gateway/router/**`
- `smart-recruit-gateway/docs/**`
- `user-frontend/src/api/ai.ts`
- `user-frontend/src/types/**`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Protobuf files unless explicitly confirmed.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-001.

### Acceptance Criteria

- Candidate ChatStream uses candidate-owned data tools for applications, resumes, jobs, interviews, and offers.
- Candidate suggested questions are returned from parsed model output or deterministic fallback.
- Candidate usage audit/auth context behavior is restored where dev branch had it.
- `POST /api/v1/candidate/ai/chat` no longer returns 404 for the current frontend contract.

### Required Tests

- Targeted AI Agent service tests for candidate runtime, suggested questions, fallback, and audit writes.
- Gateway route/handler test for `POST /api/v1/candidate/ai/chat`.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway`.
- `pnpm --filter user-frontend typecheck` if user frontend is touched.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Candidate data isolation must be preserved; any auth/ownership change requires confirmation.

### Notes

- Prefer gateway aggregation over protobuf changes for non-stream compatibility unless a later confirmation approves public contract changes.

## TASK-005 - Restore Recruiting Intelligence Generation

### Goal

Restore generation behavior for resume profile parsing and candidate match evaluation.

### Scope

- Make `ParseResumeProfile` generate and persist profile snapshots for fresh data.
- Make `EvaluateCandidateMatch` generate and persist match evaluations and evidence for fresh data.
- Preserve current read APIs for profile/evaluation lookup and candidate comparison.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/provider/**`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/**`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-gateway/handler/hr/recruiting_intelligence.go`
- `smart-recruit-gateway/handler/hr/*recruiting*test.go`
- `hr-frontend/src/api/recruitingIntelligence.ts`
- `hr-frontend/src/types/recruitingIntelligence.ts`
- `hr-frontend/src/views/hr/ApplicationIntelligenceView.vue`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-001.
- TASK-002 for shared provider/runtime behavior where needed.

### Acceptance Criteria

- Resume profile parse calls a real parser flow and persists generated profile snapshot data.
- Candidate match evaluation calls a real matcher flow and persists evaluation/evidence data.
- `agent_run_id` association is preserved where provided.
- Authorization for staff, application, job, resume, and HR AI permission remains intact.
- Candidate comparison uses generated evaluations and reports missing entries correctly.

### Required Tests

- Targeted AI Agent service tests for parse/evaluate generation and auth failures.
- Gateway handler tests if HTTP mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway` if gateway is touched.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Resume and match data may be sensitive; logs and evidence must not include raw candidate data.

### Notes

- If required source data is not available through current service-owned adapters or existing clients, stop and request confirmation before changing proto or schema.

## TASK-006 - Connect Agent, Prompt, Skill, and Capability Governance to Runtime

### Goal

Ensure admin-managed Agent, Prompt, Skill, Agent Skill, and capability configuration affects HR/Candidate runtime behavior.

### Scope

- Apply AgentConfig, prompt templates, instructions, max iterations, temperature overrides, tool/capability bindings, and selected capability keys during runtime.
- Restore Agent Skill manual selection, semantic/rule fallback selection, confirmation requirements, and metadata persistence.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/**`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/**`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-gateway/handler/hr/agent_config.go`
- `smart-recruit-gateway/handler/hr/prompt.go`
- `smart-recruit-gateway/handler/hr/skill.go`
- `smart-recruit-gateway/handler/hr/agent_skill.go`
- `hr-frontend/src/api/agent.ts`
- `hr-frontend/src/api/prompt.ts`
- `hr-frontend/src/api/skill.ts`
- `hr-frontend/src/api/agentSkill.ts`
- `hr-frontend/src/types/agent.ts`
- `hr-frontend/src/types/agentSkill.ts`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-002.
- TASK-003 for durable run confirmation integration.

### Acceptance Criteria

- Default enabled AgentConfig for `hr_recruiting_agent` affects subsequent HR runtime requests.
- Prompt template and instruction changes are reflected in new runtime requests.
- Capability bindings and request `skill_capability_keys` restrict available tools/skills.
- Agent Skill selection, confirmation, and selected skill metadata behave like the dev reference.

### Required Tests

- Targeted AI Agent service tests for runtime config lookup, prompt application, capability filtering, and Agent Skill selection/confirmation.
- Gateway or frontend tests if contract mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Configuration can accidentally broaden tool access; tests must cover disabled and filtered capabilities.

### Notes

- Do not change auth, RBAC, schema, or proto contracts without confirmation.

## TASK-007 - Restore MCP Runtime Execution

### Goal

Restore live MCP connection validation, tool discovery, tool execution, policy enforcement, audit logging, and Agent tool integration.

### Scope

- Add an MCP runner outside `NativeStore`.
- Connect `TestMCPConnection`, `ListMCPTools`, and `CallMCPTool` to live runtime behavior.
- Enforce policy, confirmation, redaction, timeout, and audit logging.
- Make approved MCP tools available to HR Agent runtime through config/Skill/capability bindings.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/policy/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/mcp/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-gateway/handler/hr/mcp.go`
- `smart-recruit-gateway/handler/hr/*mcp*test.go`
- `hr-frontend/src/api/mcp.ts`
- `hr-frontend/src/views/hr/admin/McpManageView.vue`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-002.
- TASK-006.

### Acceptance Criteria

- MCP connection test returns real live status and details.
- MCP tool discovery returns real tool schemas.
- MCP tool execution enforces policies and records redacted logs.
- Unsupported or unsafe runtime conditions remain explicit non-success responses.
- HR Agent runtime can call approved MCP tools when configured.

### Required Tests

- Targeted AI Agent service tests for MCP connection, discovery, execution, policy denial, redaction, and logs.
- Gateway handler tests if mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway` if gateway is touched.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- MCP execution is security-sensitive; this TASK requires human confirmation before implementation.

### Notes

- Do not report MCP live actions as success until a runner enforces policy, redaction, confirmation, and transport constraints.

## TASK-008 - Restore Embedding and Semantic Retrieval Runtime

### Goal

Restore embedding model testing, Agent Skill embedding backfill, semantic retrieval debug, and semantic Agent Skill selection.

### Scope

- Implement or connect embedding provider/model runtime calls.
- Implement Agent Skill embedding upsert/backfill/invalidation using existing schema.
- Restore Semantic Retrieval Debug response fields and fallback behavior.
- Feed semantic scores into Agent Skill selection with explicit rule fallback.

### Allowed Files

- `smart-recruit-ai-agent-service/internal/application/**`
- `smart-recruit-ai-agent-service/internal/domain/**`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/provider/**`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/config_services.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go`
- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`
- `smart-recruit-ai-agent-service/internal/runtime/**`
- `smart-recruit-gateway/handler/hr/embedding_config.go`
- `smart-recruit-gateway/handler/hr/agent_skill.go`
- `hr-frontend/src/api/embedding.ts`
- `hr-frontend/src/api/agentSkill.ts`
- `hr-frontend/src/views/hr/EmbeddingConfigView.vue`
- `hr-frontend/src/views/hr/admin/AgentSkillManageView.vue`
- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`
- `hr-frontend/src/types/embedding.ts`
- `hr-frontend/src/types/agentSkill.ts`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-006.

### Acceptance Criteria

- Embedding model test calls the configured provider/model and returns real status.
- Agent Skill embedding backfill supports at least `agent_skill`.
- Semantic Retrieval Debug returns embedding metadata, score breakdowns, candidates, and fallback reason.
- Agent Skill selection uses semantic scores when available and explicit fallback otherwise.

### Required Tests

- Targeted AI Agent service tests for embedding model test, backfill, semantic debug, and fallback.
- Gateway or frontend tests if mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway` if gateway is touched.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Embedding calls may require credentials and can be slow; tests should use fakes and runtime should expose fallback details.

### Notes

- Adding or upgrading dependencies requires confirmation.

## TASK-009 - Verify Frontend Contract Compatibility

### Goal

Ensure HR and Candidate frontends remain compatible with restored backend behavior and expose runtime metadata correctly.

### Scope

- Adjust frontend API/types/views only where restored backend behavior requires compatibility fixes.
- Preserve existing UI flows for HR AI Chat, Agent Run, Application Intelligence, MCP, Embedding, Agent Skill, and Candidate AI.
- Add focused tests for contract parsing where touched.

### Allowed Files

- `hr-frontend/src/api/**`
- `hr-frontend/src/components/hr/ai/**`
- `hr-frontend/src/types/**`
- `hr-frontend/src/views/hr/AIChatView.vue`
- `hr-frontend/src/views/hr/ApplicationIntelligenceView.vue`
- `hr-frontend/src/views/hr/EmbeddingConfigView.vue`
- `hr-frontend/src/views/hr/admin/**`
- `user-frontend/src/api/ai.ts`
- `user-frontend/src/types/**`
- `user-frontend/src/views/**`
- `.spec/ai-agent-runtime-recovery/reports/**`

### Forbidden Files

- Backend source files.
- `package.json`.
- Lockfiles.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-002 through TASK-008 as applicable.

### Acceptance Criteria

- HR Chat UI handles restored status events, context usage, Agent Skill selection, candidate options, and durable run metadata.
- Application Intelligence UI handles generated profile/evaluation responses.
- MCP/Embedding/Semantic Debug admin views handle real runtime success/failure responses.
- Candidate AI UI handles streaming and non-streaming compatibility behavior.
- No broad UI redesign is introduced.

### Required Tests

- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- `pnpm --filter user-frontend typecheck` if user frontend is touched.
- Focused Vitest for touched parsing/composable behavior where existing test patterns are present.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Backend fixes should not be masked by frontend-only fallbacks; frontend changes must reflect actual API behavior.

### Notes

- Do not add package dependencies or change global frontend config.

## TASK-010 - Regression Hardening and Final Evidence

### Goal

Close the feature with focused regression tests, manual smoke checklist, and final evidence across AI Agent, Gateway, and touched frontend contracts.

### Scope

- Add or strengthen tests that cover restored behavior across completed TASKs.
- Create a feature-owned manual smoke checklist under `.spec/ai-agent-runtime-recovery/docs/`.
- Update final reports/evidence only inside `.spec/ai-agent-runtime-recovery/reports/`.

### Allowed Files

- `smart-recruit-ai-agent-service/**/*_test.go`
- `smart-recruit-gateway/**/*_test.go`
- `hr-frontend/src/**/*.test.ts`
- `user-frontend/src/**/*.test.ts`
- `.spec/ai-agent-runtime-recovery/docs/**`
- `.spec/ai-agent-runtime-recovery/reports/**`
- `.spec/ai-agent-runtime-recovery/scripts/**`

### Forbidden Files

- Production business source files.
- Protobuf files.
- Database migrations.
- K8s manifests.
- Repository-root `docs/**`.

### Dependencies

- TASK-001 through TASK-009.

### Acceptance Criteria

- AI Agent service test suite passes.
- Gateway test suite passes.
- Touched frontend type checks pass.
- Manual smoke checklist covers HR AI Chat, durable Agent Run, Candidate AI, Recruiting Intelligence parse/evaluate, MCP, Embedding, and Semantic Debug.
- Final report identifies remaining risks without claiming unrun checks passed.

### Required Tests

- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway`.
- `pnpm --filter hr-frontend typecheck` if HR frontend was touched in this feature.
- `pnpm --filter user-frontend typecheck` if user frontend was touched in this feature.
- Focused Vitest for touched frontend tests.
- Harness checks required by `AGENT_RULES.md`.

### Risks

- Live AI/MCP/Embedding smoke may require local credentials or services; skipped live checks must be documented honestly.

### Notes

- This TASK must not implement new runtime behavior. It is for tests, checklist, and evidence hardening only.
