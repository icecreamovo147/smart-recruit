# AI Agent Runtime Recovery SPEC

## 1. Background

The current branch `codex/microservice-runtime-implementation` refactored the monolithic `logic-grpc-service` and `web-gin-service` architecture into independent services, including `smart-recruit-ai-agent-service` and `smart-recruit-gateway`. After that architecture refactor, several AI Agent related user workflows still expose HTTP/gRPC APIs but no longer perform the behavior that existed and worked on `origin/dev`.

The user confirmed that the `dev` branch functionality was implemented and working before the refactor. Therefore, `origin/dev` is the functional behavior reference for this recovery. The implementation must preserve the current branch's microservice boundaries while restoring behavior equivalent to the old working implementation.

Current evidence from repository inspection:

- `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go` currently implements HR chat, candidate chat, application analysis, and durable agent runs mostly through a direct provider `complete()` call.
- HR AI HTTP routes and frontend payloads still include agent skills, capability keys, context usage, tool traces, and application analysis fields.
- Recruiting Intelligence parse/evaluate endpoints exist but only read existing snapshots and return unsupported when generation is needed.
- MCP, Semantic Retrieval, and Embedding admin APIs exist but live runtime operations return explicit unsupported responses.
- `user-frontend/src/api/ai.ts` still calls `POST /api/v1/candidate/ai/chat`, while the gateway only registers `POST /api/v1/candidate/ai/chat/stream`.
- `go test ./...` in the AI Agent service and Gateway currently fails before full validation because several `go.sum` entries are missing.

Reference behavior exists in `origin/dev`, especially:

- `logic-grpc-service/service/ai_service.go`
- `logic-grpc-service/service/candidate_ai_service.go`
- `logic-grpc-service/service/recruiting_intelligence_service.go`
- `logic-grpc-service/service/agent_run_*`
- `logic-grpc-service/service/agent_skill_*`
- `logic-grpc-service/repository/*`
- `logic-grpc-service/ai/*`
- `logic-grpc-service/server/mcp*`

## 2. Goals

1. Restore HR AI Chat and ChatStream behavior to the working dev-branch level inside `smart-recruit-ai-agent-service`.
2. Restore durable HR Agent Run execution, event streaming, cancellation, confirmation, tool trace, and resumability semantics.
3. Restore Candidate AI assistant business-aware behavior, including candidate tools, suggested questions, non-streaming compatibility, and usage audit.
4. Restore Recruiting Intelligence generation flows for resume profile parsing and candidate match evaluation.
5. Make Agent, Prompt, Skill, MCP, and Embedding governance data affect runtime behavior where dev branch behavior did so.
6. Restore MCP connection validation, tool discovery, tool execution, policy/audit behavior, and Agent tool integration.
7. Restore embedding model runtime validation, embedding backfill, semantic retrieval debug, and Agent Skill semantic selection behavior.
8. Preserve current microservice architecture, service-owned runtime wiring, and gateway transport boundary.
9. Add focused regression tests and validation so recovered behavior does not silently regress again.

## 3. Non-Goals

1. Do not reintroduce `logic-grpc-service` or `web-gin-service` as runtime dependencies.
2. Do not collapse independent services back into the previous monolith.
3. Do not perform Kubernetes manifest work in this feature. K8s deployment repair is explicitly out of scope for now.
4. Do not redesign product UX, page layout, or navigation except for compatibility fixes required by restored APIs.
5. Do not create new AI product features beyond restoring dev-branch behavior and current frontend contract compatibility.
6. Do not modify database schema or protobuf contracts unless a later TASK explicitly identifies an unavoidable gap and receives confirmation.
7. Do not change authentication, authorization, quota, or risk-control policy semantics except to restore dev-compatible behavior behind existing permissions.

## 4. User-Facing Behavior

### HR AI Chat

Recruiters and recruiting admins must be able to use HR AI chat to ask recruitment-aware questions about candidates, jobs, applications, interviews, offers, and next-step recommendations. The assistant must use real business data through approved tools rather than responding only from the raw prompt.

The HR chat UI must continue to receive:

- incremental stream deltas;
- status events such as planning, tool calling, tool done, fallback, and done;
- context usage snapshots;
- tool traces for the session;
- Agent Skill selection prompts when confirmation is required;
- action metadata and candidate options where applicable.

### Durable Agent Runs

The HR AI durable run flow must execute real Agent runtime work and remain resumable after page refresh. Users must see meaningful run events, tool execution progress, terminal status, cancellation behavior, and confirmation continuation.

### Candidate AI Assistant

Candidates must be able to ask about their applications, resumes, job recommendations, interviews, and offers. The assistant must use candidate-scoped tools and return suggested follow-up questions. Existing streaming behavior and non-streaming frontend compatibility must both work.

### Recruiting Intelligence

HR users must be able to trigger:

- resume profile parsing from an application or resume;
- candidate match evaluation for an application;
- reading generated profile and evaluation snapshots;
- comparing candidates for a job using available evaluations.

For fresh data, parse/evaluate actions must generate snapshots instead of returning unsupported messages.

### Admin and Governance Pages

AI admin pages for Agent, Prompt, Skill, MCP, Agent Skill, Embedding, and Semantic Debug must not be cosmetic-only. Saved configuration must be used by runtime flows where applicable. Live test, discovery, execution, backfill, and debug actions must return real success/failure states with useful details.

## 5. Functional Requirements

### FR-001 HR AI runtime restoration

`/api/v1/hr/ai/chat` and `/api/v1/hr/ai/chat/stream` must use an HR Agent runtime equivalent to `origin/dev` behavior:

- load the effective HR agent runtime config for `hr_recruiting_agent`;
- apply selected `skill_capability_keys`;
- resolve model/provider configuration;
- build recruitment context from current service boundaries;
- call tools through an approved tool executor;
- persist user and assistant messages;
- persist process content and context usage;
- record tool traces;
- emit stream events compatible with the existing HR frontend;
- return fallback replies from tool traces when the LLM fails after useful tool results.

### FR-002 HR application analysis restoration

`/api/v1/hr/ai/analyze-application` and `/api/v1/hr/ai/application-analysis-sessions` must restore application-aware behavior:

- load application, candidate, job, interview, offer, and status context through service-owned clients or persistence adapters;
- return expected fields such as candidate name, job title, status, and round number where available;
- create an analysis session with initial messages or analysis content matching frontend expectations;
- avoid empty sessions unless no analysis can be produced and a clear error is returned.

### FR-003 Durable Agent Run restoration

`/api/v1/hr/ai/runs` and related run endpoints must restore dev-compatible durable run semantics:

- create idempotent queued runs with durable request payloads;
- execute through the same HR Agent runtime used by ChatStream;
- persist run events, status, assistant text, process text, model info, selected skills, tool traces, errors, and terminal timestamps;
- stream replayable events using `after_seq`;
- support cancellation while queued/running;
- support Skill confirmation and continue execution after confirmation;
- maintain legal state transitions and reject invalid transitions.

### FR-004 Candidate AI restoration

Candidate AI must restore dev-compatible behavior:

- streaming chat must use candidate-scoped tools or ADK agent tools;
- tools must be scoped to the authenticated candidate;
- assistant responses must be persisted;
- suggested questions must be extracted from model output or generated by fallback;
- usage audit and auth context records must be written;
- tool traces must be persisted where existing schema supports them;
- non-streaming candidate chat compatibility must be restored for `POST /api/v1/candidate/ai/chat` or the frontend must be changed under an explicitly approved compatibility plan.

### FR-005 Recruiting Intelligence generation restoration

Recruiting Intelligence must restore generation behavior:

- `ParseResumeProfile` must call a real profile parser equivalent to `origin/dev` `profileSvc.ParseResume`.
- `EvaluateCandidateMatch` must call a real matcher equivalent to `origin/dev` `matchSvc.EvaluateApplication`.
- Generated resume profile snapshots and match evaluation snapshots must be persisted and versioned according to existing schema.
- `agent_run_id` association must be preserved for match evaluation where provided.
- Authorization must preserve application/job scope and HR AI permission checks.

### FR-006 Agent, Prompt, Skill, and Capability runtime integration

Runtime flows must consume governance data:

- HR runtime must read default enabled AgentConfig for `hr_recruiting_agent`.
- Candidate runtime must continue to read active candidate prompts and should use Candidate AgentConfig where supported by dev behavior.
- Prompt templates, agent instructions, max iterations, temperature override, tool/capability bindings, and selected capability keys must affect runtime behavior.
- Agent Skill selection must support manual IDs, semantic recommendation, rule fallback, and confirmation requirements.
- Selected skill metadata must be persisted on user messages or run metadata where existing schema supports it.

### FR-007 MCP runtime restoration

MCP governance operations must be backed by a real runtime runner:

- test connection must attempt the configured transport and return live status;
- tool discovery must return real tool schemas;
- tool execution must enforce policy, confirmation, allowlists, argument redaction, error handling, and audit logging;
- HR Agent tools must include approved MCP tools when bound through Skill or capability configuration;
- unsupported runtime conditions must remain non-success and explicit.

### FR-008 Embedding and semantic retrieval restoration

Embedding and semantic retrieval runtime must be restored:

- embedding model test must call the configured embedding provider/model;
- embedding backfill must support at least `agent_skill` objects;
- Agent Skill create/update/activate/status changes must publish or trigger embedding upsert/invalidations consistent with dev behavior;
- Semantic Retrieval Debug must return embedding availability, provider/model/dim, latency, vector score, rule score, final score, fallback reason, skills, memories, and candidate count where applicable;
- Agent Skill selector must prefer semantic signals when available and use explicit fallback when unavailable.

### FR-009 Validation and dependency hygiene

The feature must fix module validation blockers needed for reliable testing:

- `GOWORK=off go test ./...` in `smart-recruit-ai-agent-service` must no longer fail because of missing `go.sum` entries.
- `GOWORK=off go test ./...` in `smart-recruit-gateway` must no longer fail because of missing `go.sum` entries.
- Frontend type checks for touched apps must pass.
- TASK-specific tests must be added for restored behavior.

### FR-010 Compatibility of public contracts

Existing HTTP routes, frontend payloads, protobuf fields, and persisted data semantics must remain compatible unless a later TASK identifies a Hard Stop requiring explicit approval. The default implementation path is to restore expected behavior behind existing contracts, not redesign contracts.

## 6. Non-Functional Requirements

1. Runtime behavior must be service-owned and maintainable in the current microservice architecture.
2. Gateway handlers must remain transport/policy adapters and must not own AI business logic.
3. AI Agent service code must separate orchestration, provider calls, tool execution, persistence, and gRPC mapping clearly enough for focused tests.
4. Tool execution must respect timeouts, cancellation, quota/risk middleware, and existing request contexts.
5. Agent Run event streaming must avoid data races and must not block run execution on slow subscribers.
6. Fallback behavior must be deterministic and observable, not silent.
7. Sensitive values in MCP/tool/LLM/embedding configuration must not be logged or returned unredacted.
8. Recovered runtime should reuse existing packages and code already present in the repository where feasible, especially `smart-recruit-commons/ai`.
9. Harness execution for this feature should be compatible with Codex Goal mode and should prefer subagent-based TASK execution/review where the environment supports it, so long-running implementation context is isolated from planning, review, and evidence context.

## 7. Compatibility Requirements

1. Preserve current frontend API callers in `hr-frontend/src/api/ai.ts`, `hr-frontend/src/api/agentRun.ts`, `hr-frontend/src/api/recruitingIntelligence.ts`, `hr-frontend/src/api/agentSkill.ts`, `hr-frontend/src/api/embedding.ts`, and `user-frontend/src/api/ai.ts`.
2. Preserve current gateway route paths and permissions for HR AI, Candidate AI, Recruiting Intelligence, MCP, Agent Skill, and Embedding admin routes.
3. Preserve protobuf service names and request/response fields unless explicitly confirmed otherwise.
4. Preserve current database table names and existing migrations unless a later approved TASK permits a schema migration.
5. Preserve existing `smart-recruit-ai-agent-service` independent module and runtime registration.
6. Preserve current route-mode architecture where the gateway routes AI Agent APIs to `smart-recruit-ai-agent-service`.
7. Preserve dev-branch functional behavior as the expected business compatibility baseline.

## 8. Observability and Debug Requirements

1. HR ChatStream must emit visible status events for model selection, planning, tool execution, context usage, fallback, skill selection, and completion.
2. Agent Run event streams must be replayable by sequence and include terminal status events.
3. Tool traces must record tool name, arguments after redaction, result summary/content, duration, error message, run ID, and step ID when available.
4. Recruiting Intelligence generation must log parse/evaluate start, authorization outcome, resolved resume/application IDs, generated snapshot IDs, and failures.
5. MCP runtime operations must persist tool logs and expose connection/test errors without leaking secrets.
6. Embedding debug must expose fallback reason and runtime metadata so admins can distinguish semantic ranking from rule fallback.
7. Tests and TASK reports must record skipped checks and reasons rather than implying success.

## 9. Error Handling and Fallback Requirements

1. Provider unavailable or misconfigured errors must return explicit non-success business codes and user-safe messages.
2. Tool execution errors must be reflected in status events and tool traces.
3. If LLM generation fails after tools returned useful data, the runtime must use dev-compatible deterministic fallback replies.
4. Candidate AI suggested questions must fall back to deterministic generation when model output cannot be parsed.
5. MCP runtime failures must not be reported as success and must update connection status where applicable.
6. Embedding runtime failures must degrade to explicit rule fallback, not empty silent selection.
7. Invalid Agent Run state transitions must be rejected with a clear error and must not corrupt persisted state.
8. Cancellation must complete runs as canceled and must not leave active runs stuck indefinitely.

## 10. Security and Safety Requirements

1. HR AI, Recruiting Intelligence, MCP, Agent Skill, and Embedding admin actions must continue to require existing permissions and roles.
2. Candidate AI tools must only access data owned by the authenticated candidate.
3. HR tools must enforce job/application scope through existing Identity and Recruitment ownership contracts.
4. MCP execution must enforce transport restrictions, private-network protections if present in dev behavior, policy allowlists, confirmation rules, argument redaction, and audit logging.
5. Tool traces and logs must not expose secrets, tokens, or credentials.
6. AI usage audit records must include actor, operation, status, timing, and auth context where dev behavior did so.
7. No task may weaken auth, permission, quota, or risk middleware without explicit user confirmation.

## 11. Acceptance Criteria

1. HR users can ask a recruitment-aware question in `/api/v1/hr/ai/chat/stream`; the response uses real tools and produces tool traces.
2. HR non-streaming chat returns a business-aware answer and context usage metadata where applicable.
3. Agent Run create, stream, confirm, cancel, and resume flows work with persisted events and legal status transitions.
4. Candidate streaming chat can answer candidate-owned application/interview/offer/resume/job questions and returns suggested questions.
5. `POST /api/v1/candidate/ai/chat` no longer returns 404 for the current candidate frontend contract.
6. Application analysis returns application-aware fields and creates a useful analysis session.
7. Resume profile parse generates and persists a profile snapshot for a fresh application or resume.
8. Candidate match evaluation generates and persists an evaluation snapshot for a fresh application.
9. Candidate comparison shows generated evaluations and missing entries correctly.
10. Agent/Prompt config changes affect subsequent HR or Candidate runtime behavior as specified.
11. MCP test connection, tool discovery, and tool call return real runtime results and persist audit logs.
12. Embedding model test, Agent Skill embedding backfill, and Semantic Retrieval Debug return real runtime or explicit fallback data.
13. `GOWORK=off go test ./...` passes in `smart-recruit-ai-agent-service`.
14. `GOWORK=off go test ./...` passes in `smart-recruit-gateway`.
15. Type checks pass for touched frontend apps.
16. No K8s manifests are modified by this feature.

## 12. Out of Scope

1. Kubernetes deployment manifests and service objects.
2. New product UX or new AI capabilities that did not exist in dev or current frontend contracts.
3. Database splitting or table ownership redesign.
4. Protobuf redesign.
5. Authentication/authorization policy redesign.
6. Production migration runbooks outside `.spec/ai-agent-runtime-recovery/`.
7. Repository-root `docs/` changes.

## 13. Assumptions Requiring Confirmation

1. `origin/dev` is the authoritative behavior reference for all AI Agent functionality listed above.
2. Current microservice boundaries must be preserved even when dev code lived under `logic-grpc-service`.
3. Existing migrations in `smart-recruit-commons/migrations` already contain the tables needed for restored behavior.
4. Restoring `POST /api/v1/candidate/ai/chat` is acceptable as compatibility restoration because the current frontend still calls it.
5. Changes to `go.sum` are acceptable when a specific TASK explicitly scopes dependency checksum repair.
6. Live MCP and Embedding runtime should reuse existing dependencies where possible; adding or upgrading dependencies requires explicit confirmation.
7. Future Harness prompts may delegate implementation, review, or repair work to subagents, but the controlling agent remains responsible for reading canonical skill instructions, enforcing task scope, validating evidence, and reporting completion.

## 14. Open Questions

1. Should the restored runtime prefer ADK tool calling or legacy tool calling when both are available, or exactly mirror the current dev branch selection logic?
2. Should Candidate AI expose non-streaming behavior by implementing a true non-stream RPC path or by gateway aggregation over stream chunks?
3. Which MCP transports are required for this recovery: stdio, SSE, streamable HTTP, or only the transports currently present in dev?
4. Should embedding backfill execute synchronously for admin calls or enqueue work through the existing worker/outbox pattern?
5. Which AI smoke scenarios should become mandatory manual verification before moving from implementation to self-review?
