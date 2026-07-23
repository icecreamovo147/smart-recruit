# AI Agent Runtime Recovery SDD

## 1. Existing Architecture Summary

The current branch has already moved the AI Agent bounded context into `smart-recruit-ai-agent-service`. The service registers AI, LLM config, Prompt, Agent Config, MCP, Skill, Agent Skill, Recruiting Intelligence, and Embedding Config gRPC services through `smart-recruit-ai-agent-service/internal/runtime/runtime.go` and `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`.

The gateway remains the HTTP transport boundary:

- `smart-recruit-gateway/router/router.go` registers HR AI, Candidate AI, Recruiting Intelligence, MCP, Agent Skill, and Embedding admin HTTP routes.
- `smart-recruit-gateway/handler/hr/*.go` and `smart-recruit-gateway/handler/candidate/*.go` map HTTP payloads to protobuf requests.
- `smart-recruit-gateway/rpc/client.go` creates AI Agent service clients when `AI_AGENT_ROUTE_MODE=ai-agent`.

The current AI Agent service persistence layer is centered on:

- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go`
- `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go`

The repository also contains shared AI primitives in `smart-recruit-commons/ai`, including Eino/ADK clients, fallback builders, planner support, tool metadata, HR tools, and candidate tools.

The dev branch reference implementation lived primarily under `logic-grpc-service`, with the following relevant behavior:

- `service/ai_service.go`: HR chat, stream, tool calling, runtime config, context usage, Agent Skill selection, fallback, application analysis, tool traces.
- `service/candidate_ai_service.go`: candidate chat, candidate tools, ADK/legacy fallback, suggested questions, usage audit, tool traces.
- `service/recruiting_intelligence_service.go`: resume profile parse and candidate match evaluation generation.
- `service/agent_run_*`: durable run state, worker, recorder, event replay, cancellation, confirmation.
- `service/agent_skill_*`: semantic Agent Skill selection, metadata validation, embedding event publishing, debug semantic retrieval.
- `repository/*`: persistence behavior for AI chat, tool traces, runs, configs, memories, MCP, embeddings, resume profiles, and candidate matches.
- `ai/*`: HR/Candidate tool executors and AI client integration.

The recovery design must migrate behavior into the new service-owned structure rather than reintroducing old runtime roots.

## 2. Problem Analysis

The architecture refactor preserved many API surfaces but lost runtime semantics:

1. HR Chat currently appends user/assistant messages and calls `complete()` directly. It does not build recruitment context, use tools, read AgentConfig, select Agent Skills, record process content, or emit rich stream events.
2. HR ChatStream currently wraps non-streaming Chat and sends one done chunk. The frontend expects status events, context usage, skill selection, candidate options, and tool progress.
3. Durable Agent Run currently creates a fallback run and asynchronously calls `complete()` once. It does not execute the Agent runtime or continue after confirmation.
4. Candidate ChatStream currently renders prompt/history and calls `complete()`. It does not use candidate business tools, tool traces, suggested-question parsing, usage audit, or deterministic fallback from tool traces.
5. Candidate non-streaming chat route is missing while the frontend still calls it.
6. Recruiting Intelligence parse/evaluate endpoints only read existing snapshots and report unsupported for missing snapshots.
7. Agent, Prompt, MCP, Skill, Agent Skill, and Embedding admin data is mostly persisted but not fully connected to runtime behavior.
8. MCP live validation, discovery, and execution return unsupported.
9. Embedding live validation, backfill, and semantic debug return unsupported.
10. Go module tests are blocked by missing `go.sum` entries.

The central design challenge is to restore dev-equivalent runtime behavior while respecting the new service boundaries. Old code often used direct repositories from the monolith. New code must use service-owned persistence adapters or explicit gRPC owner clients where the current architecture requires cross-service access.

## 3. Proposed Design

### 3.1 Runtime service decomposition

Introduce focused runtime components inside `smart-recruit-ai-agent-service/internal` rather than expanding `native_servers.go` indefinitely.

Recommended package layout:

- `internal/application/service` for orchestration services that own HR runtime, candidate runtime, recruiting intelligence generation, Agent Run execution, semantic retrieval, and embedding orchestration.
- `internal/application/port` for interfaces to persistence, provider clients, tool executors, MCP runner, embedding runner, and owner service clients.
- `internal/infrastructure/persistence` for DB-backed implementations that already exist or need focused extension.
- `internal/infrastructure/provider` for LLM and embedding client adapters.
- `internal/infrastructure/mcp` for MCP runner implementation and transport adapters.
- `internal/interfaces/grpc` for protobuf mapping and thin endpoint adapters.

`native_servers.go` may remain as the registration facade, but complex runtime logic should move into testable service objects.

### 3.2 HR Agent runtime

Create an HR Agent runtime orchestrator equivalent to dev `AIService.runToolCallingChatWithUsage`.

Responsibilities:

1. Validate request and session ownership.
2. Resolve model/client using existing LLM config behavior.
3. Resolve `hr_recruiting_agent` runtime config from AgentConfig, Prompt, tool bindings, capability bindings, and selected request keys.
4. Build an AgentContext using current service boundaries:
   - service-owned AI chat/session/memory persistence from AI Agent DB tables;
   - Recruitment service clients for application/job/candidate/resume snapshots where cross-service ownership requires gRPC;
   - Interview and Offer clients only if current gateway/proto runtime exposes needed ownership snapshots; otherwise use already approved read models in AI Agent persistence.
5. Select Agent Skills using manual IDs, semantic retrieval, and rule fallback.
6. Emit Agent Skill selection event and stop with a non-terminal waiting state when confirmation is required.
7. Assemble tool-capable messages and instructions.
8. Execute ADK or legacy tool calling according to dev behavior and available `smart-recruit-commons/ai` primitives.
9. Persist tool traces asynchronously or transactionally as appropriate.
10. Build and emit context usage snapshots.
11. Persist assistant response, process content, selected skills, model metadata, and context usage.
12. Use deterministic fallback from tool traces when provider output fails after useful tool execution.

The gRPC `Chat` method should call the same runtime with a collector for deltas/events. The gRPC `ChatStream` method should call the same runtime with stream senders.

### 3.3 Durable Agent Run runtime

Rebuild durable runs around the same HR Agent runtime.

Design:

- `CreateAgentRun` persists a run request payload that includes session, HR user, message, model ID, application ID, capability keys, Agent Skill IDs, confirmation flags, and client request ID.
- A run dispatcher/worker loads the durable payload and invokes the HR runtime with an Agent Run recorder.
- Recorder writes events for run created, status changed, model selected, planning, tool calling, tool done, process delta, assistant delta, skill selection, fallback, error, canceled, and completed.
- `SubscribeAgentRunEvents` keeps existing replay by `after_seq`, but events must now reflect actual runtime steps.
- `ConfirmAgentRun` must persist confirmation metadata and dispatch/continue execution.
- `CancelAgentRun` must cancel in-memory execution when present and persist cancel requested/canceled state. If a run is not currently in memory, worker/replay logic must still observe cancel requested before continuing.

The run state machine should be restored from dev `agent_run_state.go` behavior where possible and covered by tests.

### 3.4 Candidate AI runtime

Create a Candidate AI runtime orchestrator equivalent to dev `CandidateAIService.StreamChatGRPC`.

Responsibilities:

1. Validate authenticated candidate and session ownership.
2. Append user message.
3. Resolve candidate prompt and candidate agent config/tool allowlist.
4. Build candidate context from user-owned applications, resumes, jobs, interviews, and offers through service-owned clients/adapters.
5. Execute ADK candidate tools or legacy candidate tools using `smart-recruit-commons/ai`.
6. Emit stream deltas and status events.
7. Persist tool traces.
8. Build deterministic fallback from tool traces when needed.
9. Extract suggested questions from model output; fallback to deterministic suggested questions when malformed or absent.
10. Persist assistant message and usage audit/auth context.

For `POST /api/v1/candidate/ai/chat`, implement a gateway handler that either calls a non-streaming gRPC method if available or aggregates `CandidateChatStream` chunks into a response matching `user-frontend/src/api/ai.ts`.

### 3.5 Recruiting Intelligence generation

Split Recruiting Intelligence into read model methods and generation methods.

Read methods can continue using current `recruitingReadStore`:

- `GetResumeProfile`
- `GetCandidateMatchEvaluation`
- `CompareCandidatesForJob`

Generation methods must be restored:

- `ParseResumeProfile` resolves resume ID and calls a profile parser service equivalent to dev `profileSvc.ParseResume`.
- `EvaluateCandidateMatch` authorizes application access and calls a matcher service equivalent to dev `matchSvc.EvaluateApplication`.

The parser and matcher should live in AI Agent application services and use current persistence adapters for:

- resume text and metadata;
- application/job/candidate context;
- generated profile rows and snapshots;
- match evaluation rows and evidence.

If the current service does not yet have direct access to some required source data, introduce narrow gRPC owner-client ports instead of cross-service table writes.

### 3.6 Governance runtime integration

Keep existing DB-backed config CRUD, but connect it to runtime:

- AgentConfig lookup by `agent_type` and default/enabled status.
- Prompt template lookup and rendering.
- Agent capability bindings and request selected capability keys.
- Skill registry and bound tools.
- Agent Skill metadata, version content, required capabilities, semantic tags, and status.

Runtime config should be built once per request/run and passed to HR/Candidate orchestrators. Tests should verify that disabling or selecting capabilities changes tool availability.

### 3.7 MCP runtime

Move live MCP behavior out of `NativeStore` into an injected MCP runner.

Design:

- `NativeStore` remains responsible for persisted MCP servers, policies, and logs.
- `internal/infrastructure/mcp` owns connection validation, discovery, and tool call execution.
- `nativeMCPService` coordinates store + runner.
- MCP runner enforces transport policy, command/url constraints, private network restrictions where dev behavior had them, timeout, redaction, and policy decision.
- Tool call logs persist request/response metadata and redacted args.
- HR tool executor can include approved MCP tools through runtime config and Skill/capability bindings.

Unsupported runner absence should still produce explicit non-success responses; success requires actual runner execution.

### 3.8 Embedding and semantic retrieval runtime

Introduce an embedding runtime port and implementation:

- resolve default embedding provider/model from config tables;
- test models by calling the provider;
- generate embeddings for Agent Skill text;
- upsert/invalidate embeddings through existing `ai_embeddings` tables;
- run vector search and return search metadata;
- use semantic scores in Agent Skill selection and debug views.

Agent Skill changes should publish embedding upsert events where dev behavior did so. If no worker is available yet, TASK scope should choose either synchronous backfill for admin calls or a bounded service-local worker, but the behavior must be explicit and observable.

## 4. Data Structure Changes

Default design is no schema change.

Existing migrations appear to cover required data structures:

- prompt templates;
- agent configs and capability bindings;
- AI chat history process content and selected Agent Skills;
- MCP servers, tool policies, and logs;
- Agent Skills and versions;
- AI embeddings;
- embedding provider config;
- resumable agent runs;
- recruiting intelligence prompt/data tables.

Implementation TASKs must verify current table coverage before editing business code. If any required field is missing, the TASK must stop as a Hard Stop and request confirmation for a schema migration.

Go module checksum changes may be needed to unblock tests. Such changes must be explicitly scoped in the relevant TASK and limited to required `go.sum` entries.

## 5. API and Interface Changes

Default design is API-compatible restoration.

Expected API behavior:

- Keep existing HR AI routes under `/api/v1/hr/ai/*`.
- Keep existing Candidate AI stream route `/api/v1/candidate/ai/chat/stream`.
- Restore compatibility for `POST /api/v1/candidate/ai/chat`.
- Keep Recruiting Intelligence routes under `/api/v1/hr/resume-profiles*` and `/api/v1/hr/applications/:application_id/match-*`.
- Keep existing MCP, Skill, Agent Skill, Embedding admin routes.
- Keep protobuf service and field compatibility by default.

If implementation discovers that a protobuf method is missing for a required behavior, prefer implementing behavior in gateway aggregation or service-owned existing methods before changing proto. Proto changes are a Hard Stop unless explicitly confirmed.

## 6. Algorithm or Workflow Changes

### 6.1 HR Chat workflow

1. Receive Chat or ChatStream request.
2. Validate HR actor and session.
3. Resolve model and runtime config.
4. Append user message.
5. Select Agent Skills.
6. If confirmation is needed, emit selection event and persist state.
7. Build context and messages.
8. Execute tool-calling loop with max iterations and configured tool set.
9. Persist tool traces and status/process output.
10. Produce fallback if needed.
11. Persist assistant message and context usage.
12. Return or stream final response.

### 6.2 Agent Run workflow

1. Create idempotent queued run with durable request payload.
2. Dispatch worker.
3. Transition to running.
4. Execute HR runtime with recorder.
5. Persist all events with monotonic sequence.
6. If confirmation is needed, transition to waiting_confirmation and stop.
7. On confirmation, continue or redispatch from durable payload.
8. On cancellation, persist canceled terminal state.
9. On success/failure, persist terminal state and final metadata.

### 6.3 Candidate Chat workflow

1. Receive stream or non-stream request.
2. Validate candidate ownership.
3. Append user message.
4. Resolve candidate prompt/config/tools.
5. Execute candidate tool-calling runtime.
6. Persist traces and audit.
7. Extract or generate suggested questions.
8. Persist assistant response.
9. Return stream chunks or aggregated response.

### 6.4 Recruiting Intelligence generation workflow

1. Validate request identifiers.
2. Authorize staff and application/job/resume scope.
3. Resolve source resume/application/job context.
4. Invoke profile parser or matcher.
5. Persist generated rows, snapshots, evidence, and versioning.
6. Return generated snapshot in existing response shape.

### 6.5 MCP workflow

1. Load server and policy.
2. Validate transport and target.
3. Connect or discover/call with timeout.
4. Redact and persist log.
5. Return live result.

### 6.6 Embedding workflow

1. Resolve default embedding model.
2. Generate query/object embeddings.
3. Store or search vectors.
4. Return semantic metadata and fallback details.

## 7. Configuration Design

Configuration sources:

- LLM provider/model config tables and current environment fallback.
- Prompt templates and active prompt role.
- AgentConfig default enabled row per agent type.
- Agent capability bindings and request selected keys.
- MCP server and policy tables.
- Embedding provider/model config tables.
- Runtime environment values already used by AI provider clients, timeouts, concurrency, and circuit breaker settings.

Configuration should be loaded per request or cached with clear invalidation. Admin changes should affect subsequent requests without service restart when feasible.

No new global configuration should be introduced without a later TASK that explicitly scopes it.

## 8. Compatibility Strategy

1. Treat `origin/dev` as the functional compatibility baseline.
2. Keep current microservice ownership boundaries as the architectural baseline.
3. Move old behavior into new service-local packages rather than importing old monolith packages.
4. Use `smart-recruit-commons/ai` as the shared AI/tooling source where available.
5. Keep gateway handlers thin and compatible with current frontend payloads.
6. Preserve existing table names and protobuf fields.
7. Implement missing Candidate non-stream compatibility in the least disruptive way.
8. Add regression tests around behavior that disappeared during the refactor.

## 9. Error Handling and Fallback Design

Error handling should mirror dev behavior where possible:

- validation failures return bad request business codes;
- authorization failures return forbidden business codes;
- provider/model unavailable returns explicit configuration errors;
- tool errors produce status events and tool trace error fields;
- LLM timeout after partial tool work may return deterministic fallback;
- embedding unavailable returns rule fallback metadata;
- MCP runner unavailable returns unsupported, not success;
- Agent Run cancellation always reaches a terminal status;
- invalid Agent Run transitions return an explicit business error without mutating state.

Fallbacks must be visible in stream events, logs, and persisted metadata.

## 10. Observability and Debug Output Design

Add or restore observability at these points:

- request start/end logs for HR Chat, Candidate Chat, Agent Run, Recruiting Intelligence generation, MCP operations, and Embedding operations;
- run event persistence for durable Agent Run;
- tool trace persistence for HR/Candidate/MCP tools;
- context usage persistence and stream events;
- usage audit for HR/Candidate AI operations where dev behavior did so;
- semantic retrieval debug fields for embedding availability, provider, model, dimension, latency, score breakdown, and fallback reason;
- test evidence in Harness reports once TASKS are generated.

Logs must avoid secret leakage and use redacted payloads for tool/MCP arguments.

## 11. Testing Strategy

Testing must map to the SPEC acceptance criteria.

### Go service tests

AI Agent service:

- HR runtime unit tests with fake provider, fake tools, fake store, and fake owner clients.
- ChatStream event tests for context usage, tool calling, fallback, and skill selection.
- Durable Agent Run state tests for create, replay, confirm, cancel, success, failure, and idempotency.
- Candidate runtime tests for tool use, suggested questions, fallback, and audit writes.
- Recruiting Intelligence tests for parse/evaluate generation and authorization.
- MCP runner/service tests for connection, discovery, call, policy denial, redaction, and logs.
- Embedding tests for model test, backfill, semantic debug, and fallback.

Gateway:

- route baseline tests including `POST /api/v1/candidate/ai/chat`;
- handler contract tests for AI Chat/Run/Recruiting Intelligence/MCP/Embedding payload mapping.

### Frontend tests

Run type checks for touched apps:

- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter user-frontend typecheck`

Add focused Vitest only where frontend behavior changes:

- Candidate AI non-stream/stream compatibility if touched;
- HR AIChatView Agent Skill selection and context usage if touched;
- ApplicationIntelligenceView parse/evaluate if touched;
- admin MCP/Embedding/Semantic Debug behavior if touched.

### Validation commands

At minimum after relevant TASKs:

- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service`
- `GOWORK=off go test ./...` in `smart-recruit-gateway`
- targeted Go tests in other touched service modules
- frontend typecheck for touched apps

## 12. Migration Risks

1. Old dev code may depend on monolith repositories that cannot be directly reused under new service ownership.
2. Some data access may need new owner-service read ports; changing protobuf contracts would be a Hard Stop.
3. Reusing old code mechanically may violate current service boundaries.
4. MCP execution can introduce security risk if policy, redaction, and transport restrictions are incomplete.
5. Embedding backfill can be expensive or slow if implemented synchronously without bounds.
6. Agent Run confirmation/cancellation bugs can leave active runs stuck.
7. Tool traces and logs can leak sensitive data if redaction is incomplete.
8. Missing `go.sum` entries block validation until explicitly repaired in a scoped TASK.

## 13. Implementation Boundaries

Allowed implementation areas for future TASK generation are expected to include:

- `smart-recruit-ai-agent-service/**`
- `smart-recruit-gateway/**`
- `hr-frontend/src/api/**`, `hr-frontend/src/views/hr/**`, `hr-frontend/src/components/hr/**`, `hr-frontend/src/types/**` only when frontend compatibility requires it
- `user-frontend/src/api/**`, `user-frontend/src/views/**`, `user-frontend/src/types/**` only when candidate compatibility requires it
- `smart-recruit-commons/ai/**` only if shared AI/tool behavior must be restored and the TASK explicitly allows shared module changes
- `smart-recruit-commons/migrations/**` only after a Hard Stop confirmation for schema changes
- `smart-recruit-proto/**` only after a Hard Stop confirmation for public contract changes

Forbidden by default:

- `deploy/k8s/**`
- repository-root `docs/**`
- unrelated frontend apps
- unrelated service modules
- package manifests and lockfiles, except narrowly scoped `go.sum` checksum repair when a TASK explicitly allows it
- authentication/authorization policy changes not required for dev-compatible restoration

### 13.1 Goal mode and subagent execution model

Future Harness generation should assume the user may execute the work through Codex Goal mode. TASK prompts and Agent rules should therefore be written so the controlling agent can keep the canonical feature state, scope checks, reports, and evidence clean while delegating focused implementation or review work to subagents when available.

Subagent usage should follow these boundaries:

- The controlling agent must read `AGENTS.md`, `spec-harness`, active SPEC/SDD/TASK/acceptance files, and required knowledge instructions itself.
- Subagents may inspect code, compare against `origin/dev`, implement a scoped TASK, perform read-only review, or investigate failed checks when their prompt includes the exact TASK scope and required files.
- Subagents must receive narrow context: one TASK, allowed files, forbidden files, acceptance criteria, and relevant dev-branch reference paths.
- Subagents must not redefine requirements, expand scope, skip Harness checks, or make decisions that require Hard Stop confirmation.
- Independent read-only subagents should be preferred for `self-review` when available, matching the review independence requirement in `spec-harness`.
- The controlling agent must consolidate subagent output, run required checks, update reports/evidence, and stop for user confirmation before the next TASK unless an explicit pipeline mode is invoked.

## 14. Alternatives Considered

### Alternative A: Revert AI Agent service to dev monolith code

Rejected. This would conflict with the current microservice architecture and reintroduce retired runtime roots.

### Alternative B: Keep current complete-only runtime and adjust frontend expectations

Rejected. The user explicitly needs broken AI Agent features restored, and dev behavior already existed.

### Alternative C: Implement only route-level fixes

Rejected. Missing routes are only one symptom. Most failures are semantic regressions behind existing routes.

### Alternative D: Redesign AI Agent from scratch

Rejected. Dev branch already contains working behavior, and the recovery should minimize product semantics risk by using it as the reference.

### Alternative E: Restore runtime in a single large TASK

Rejected for Harness execution. The blast radius spans HR runtime, candidate runtime, Recruiting Intelligence, MCP, Embedding, gateway, and frontend contracts. TASKs should be split after SPEC/SDD review.

## 15. Assumptions Requiring Confirmation

1. `origin/dev` remains available locally and can be used by implementation TASKs through `git show origin/dev:<path>`.
2. The desired outcome is behavior parity with dev within the current microservice architecture, not byte-for-byte code restoration.
3. K8s deployment work must remain out of scope for this feature.
4. Existing DB schema is sufficient unless implementation proves otherwise.
5. Candidate non-streaming chat compatibility should be restored rather than removing the frontend call.
6. The implementation may update `go.sum` entries in scoped TASKs to make tests run.
7. The user intends to execute later TASKs in Codex Goal mode and prefers subagent delegation for context isolation; Harness artifacts should support that execution style.

## 16. Open Questions

1. Should the first implementation TASK focus on HR Chat runtime, or should it first unblock `go test` by repairing `go.sum`?
2. Should MCP runner support every transport present in stored configs immediately, or should unsupported transports remain explicit non-success until a later TASK?
3. Should embedding backfill be implemented synchronously for admin calls first, or through a service-local worker/outbox path?
4. Should Recruiting Intelligence generation reuse dev prompts exactly, including Chinese prompt migrations, or regenerate prompt text from current active Prompt templates?
5. Should Agent Run confirmation resume from the exact interrupted planning state, or is redispatch from durable payload acceptable if persisted events remain coherent?
