# AI Agent Dependency Inventory

TASK-022 establishes the AI Agent DDD skeleton and records the current complex
dependency surface. It does not change runtime wiring, protobuf contracts,
database schema, AI provider behavior, MCP policy, security behavior, or public
API semantics.

## Current Runtime Boundary

- Service root: `smart-recruit-ai-agent-service`.
- Active runtime entrypoint: `cmd/ai-agent-service/main.go`.
- Runtime registration: `internal/runtime/runtime.go` registers AI-owned gRPC
  services:
  - `AIService`
  - `LlmConfigService`
  - `PromptService`
  - `AgentConfigService`
  - `MCPService`
  - `SkillService`
  - `AgentSkillService`
  - `RecruitingIntelligenceService`
  - `EmbeddingConfigService`
- Current implementation source: service-local `internal/legacydomain/service`,
  `internal/legacydomain/repository`, `internal/legacydomain/model`, and
  `internal/legacydomain/ai` compatibility packages. These were localized by
  TASK-029 so active AI Agent runtime no longer imports shared
  `smart-recruit-commons/model`, `repository`, or `service`.
- Runtime platform behavior retained: Nacos discovery/config, gRPC internal
  auth, optional TLS, health, metrics, trace, MySQL, Redis, RabbitMQ,
  structured logging, provider timeout/retry/circuit-breaker settings, and
  graceful shutdown.

## Capability Inventory

| Capability | Current responsibility | Current dependency surface |
| --- | --- | --- |
| HR chat and streaming | Chat, chat stream, history, sessions, application analysis, tool traces. | Shared `AIService`, chat/session/tool trace repositories, Redis/cache, AI client, prompt/agent context services, gRPC streaming. |
| Candidate assistant | Candidate chat stream, sessions, session messages, updates, deletes. | Shared `CandidateAIService`, user/candidate data, chat/session repositories, AI provider client, stream handling. |
| Durable agent runs | Create/list/get/active/subscribe/cancel/confirm agent runs. | Shared agent run repository, RabbitMQ agent-run queue, SSE/gRPC stream events, tool trace and audit data. |
| Prompt/config | LLM providers/models, prompt templates, agent configs, embedding configs. | Provider/model repositories, encryption key, config validation, RBAC permissions, optional embedding backfill service. |
| MCP governance | MCP server/tool config, policy, logs, allowlist and tool execution controls. | MCP repository, config, private-network/command policy, audit logs, external MCP SDK/runtime. |
| Skill and agent skill | Skill catalog, agent-skill assignment, semantic retrieval support. | Skill repositories, agent config repository, embedding/retrieval dependencies. |
| Embedding runtime | Embedding provider factory, embedding service, embedding backfill, embedding queue consumer. | Provider credentials, encryption key, embedding provider SDKs, RabbitMQ embedding queue, Inbox idempotency, vector rows. |
| Recruiting intelligence | Resume profile extraction, candidate matching, compare candidates. | Recruitment/resume/application/job data, OSS/resume parser/LLM matchers, AI provider, usage audit. |
| Provider fallback | AI and embedding provider retries, circuit breaker, timeout budgets, model selection. | `smart-recruit-commons/ai`, provider config tables, encrypted credentials, runtime config. |
| Usage and audit | AI usage logs, tool traces, MCP logs, authorization/audit context. | Usage log repository, tool trace repository, authz context, request/trace metadata. |

## Target DDD Ownership

- `domain/model`: chat session, chat message, agent run, tool trace, prompt,
  agent config, MCP server/tool/policy, skill, agent skill, embedding provider,
  embedding model, recruiting intelligence request/result, usage audit value
  objects.
- `domain/policy`: prompt selection, provider failover eligibility, tool
  allowlist, stream timeout, retry budget, sensitive-data redaction, MCP command
  safety, embedding backfill eligibility.
- `application/service`: chat orchestration, stream event orchestration, durable
  agent-run lifecycle, prompt/config workflows, MCP governance workflows, skill
  assignment, embedding operations, recruiting intelligence, usage audit.
- `infrastructure/provider`: AI and embedding provider SDK adapters, credential
  decryption, timeout/retry/circuit-breaker wiring.
- `infrastructure/mcp`: MCP client/tool adapters with policy and audit hooks.
- `infrastructure/mq`: embedding and agent-run consumers/publishers with Inbox
  idempotency and DLQ/retry support.
- `interfaces/grpc`: protobuf mapping for all AI-owned service surfaces and
  stream responses.

## Local Chat, Agent Run, And Prompt Application Boundary

TASK-023 localizes the first AI Agent domain/application semantics without
switching active runtime wiring.

- `domain/model/agent.go` defines transport-neutral chat sessions/messages,
  durable agent runs/events, prompt templates/versions, model/provider
  candidates, and audit events.
- `domain/policy/agent.go` owns the durable agent-run state machine,
  stale/duplicate event sequence detection, create-run request validation,
  prompt validation/versioning, and provider fallback selection.
- `domain/repository/agent.go` defines local ports for chat sessions, agent
  runs, prompts, and audit events.
- `application/service/agent_service.go` orchestrates idempotent durable run
  create/dispatch, cancel, confirm, prompt create/update, and audit intents
  through local ports.

Streaming, cancel, confirm, audit, and provider fallback semantics are pinned by
local unit tests. The active gRPC runtime still delegates to the shared AI
implementation until later TASKs add local infrastructure/interfaces adapters
and cut over safely.

## Local MCP, Skill, Embedding, And Intelligence Boundary

TASK-024 localizes the second AI Agent domain/application surface without
changing active runtime wiring, protobuf contracts, schema, provider behavior,
or security policy.

- `domain/model/capability.go` defines local MCP server/policy, Skill manifest
  and version, Agent Skill selection, embedding provider/model/runtime state,
  and candidate match aggregation value objects.
- `domain/policy/capability.go` owns MCP transport safety validation,
  private-network denial, command allowlist checks, MCP tool policy evaluation,
  Skill manifest validation/versioning, Agent Skill selection gates, embedding
  runtime availability/fallback, and candidate match score aggregation.
- `domain/repository/capability.go` defines local ports for MCP policy lookup,
  Skill/Agent Skill records, embedding default config lookup, and candidate
  match persistence.
- `application/service/capability_service.go` orchestrates MCP policy
  evaluation, Skill version creation/activation, Agent Skill selection,
  embedding runtime resolution, and candidate match aggregation through local
  ports.

Compatibility tests pin no-policy allow, deny/role/scope/args/rate/confirmation
MCP decisions, private-network and stdio allowlist restrictions, Skill version
creation, embedding unavailable fallback, and candidate knockout scoring. Active
gRPC runtime still delegates to shared AI/MCP/embedding/intelligence
implementations until TASK-025 adds local infrastructure/interfaces adapters and
cuts over safely.

## Local Runtime And Interface Convergence

TASK-025 moves AI Agent runtime assembly closer to service-local ownership while
preserving public protobuf behavior and provider credential handling.

- `internal/runtime` no longer imports shared `smart-recruit-commons/service`
  types. It registers all AI-owned protobuf services from local `Deps` and
  validates local `LongTaskControls`.
- `internal/interfaces/grpc/legacy_servers.go` contains temporary forwarding
  adapters for local legacy AI chat/agent-run and recruiting intelligence
  protobuf services.
- `cmd/ai-agent-service` converts the currently active local legacy
  `AIAgentRuntime` into local runtime dependencies and explicitly passes
  embedding/agent-run worker controls.

Remaining shared implementation debt is now isolated to the command bootstrap
and local legacy gRPC bridge: shared `service.NewServices`, repositories,
provider factory, MCP/embedding workers, and intelligence implementation still
serve active traffic until follow-up work replaces those adapters with native
AI Agent infrastructure.

## Security And Safety Dependencies

- Provider credentials and encryption keys must never be logged or committed.
- MCP private-network restrictions, command allowlists, tool policy, and audit
  behavior must remain compatible unless a later TASK explicitly confirms a
  security change.
- Chat and recruiting intelligence outputs can contain candidate-sensitive data;
  DTOs and logs must avoid unnecessary PII exposure.
- Streaming and long task paths must keep request cancellation, deadlines, and
  retry/timeout budgets visible.
- Agent-run and embedding workers must remain RabbitMQ-gated and idempotent.
- True provider calls in tests must stay fake/env-gated.

## Transitional Shared Dependencies

The active runtime still uses service-local legacy compatibility packages for
AI chat, agent-run, MCP, skill, embedding, and recruiting intelligence behavior.
This is allowed only as owner-local migration debt while the AI Agent service
moves through native adapter cutover. Shared `smart-recruit-commons/ai`
remaining usage is limited to generic AI provider/fallback metadata; RabbitMQ
remains `smart-recruit-commons/mq` infrastructure bridge until the commons
rename.

Known shared dependencies to remove in later AI Agent TASKs:

- Service-local legacy business services: `AIService`, `CandidateAIService`,
  `AIAgentRuntime`, `LlmConfigService`, `PromptService`, `AgentConfigService`,
  `MCPService`, `SkillService`, `AgentSkillService`,
  `RecruitingIntelligenceService`, `EmbeddingConfigService`,
  `EmbeddingService`, `EmbeddingBackfillService`, `EmbeddingConsumer`, and
  `AgentRunConsumer`.
- Service-local legacy repositories: chat, session summary, tool trace, agent run, memory,
  provider/model config, MCP, skill, agent skill, embedding provider/model,
  usage log, recruitment/resume/application/job snapshot dependencies.
- Shared infrastructure bridges: RabbitMQ connection/config and selected generic
  AI provider/fallback helpers remain commons candidates.

## Migration Notes For Later TASKs

- Migrate chat/agent-run/prompt domain/application first while preserving all
  gRPC and streaming behavior.
- Migrate provider/MCP/embedding infrastructure only after local ports express
  credential, timeout, retry, circuit-breaker, allowlist, and audit contracts.
- Keep RabbitMQ long-task workers explicit and idempotent; do not run provider
  calls in tests without fake/env-gated adapters.
- Record every remaining shared dependency as compatibility debt until removed
  or reclassified as true shared kernel.
