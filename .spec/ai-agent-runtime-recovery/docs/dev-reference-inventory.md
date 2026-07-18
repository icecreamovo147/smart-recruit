# Dev Reference Inventory - ai-agent-runtime-recovery

## Purpose

`origin/dev` is the functional behavior reference for AI Agent runtime recovery. The current branch must keep the microservice architecture: `logic-grpc-service` and `web-gin-service` are reference sources only, not runtime dependencies.

## Reference Use Rules

- Use `git show origin/dev:<path>` to inspect legacy behavior.
- Port behavior into `smart-recruit-ai-agent-service`, `smart-recruit-gateway`, and existing shared `smart-recruit-commons/ai` boundaries.
- Do not import or depend on old `logic-grpc-service/...` or `web-gin-service/...` packages.
- Prefer current application/domain/port/persistence layering over copying monolith service wiring.

## Inventory

| Area | `origin/dev` reference paths | Current target areas | Expected behavior mapping | Notes |
|---|---|---|---|---|
| HR AI | `logic-grpc-service/service/ai_service.go`; `logic-grpc-service/service/agent_context.go`; `logic-grpc-service/ai/hr_adk_tools.go`; `logic-grpc-service/ai/tool_executor.go`; `logic-grpc-service/repository/chat_repo.go`; `logic-grpc-service/repository/tool_trace_repo.go`; `web-gin-service/handler/hr/ai.go` | `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`; `smart-recruit-ai-agent-service/internal/application/service/**`; `smart-recruit-ai-agent-service/internal/application/port/**`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go`; `smart-recruit-commons/ai/**`; `smart-recruit-gateway/handler/hr/ai.go` | Restore Chat/ChatStream shared runtime, HR agent config, recruitment context, tool execution, context usage, tool traces, Agent Skill selection, fallback replies, and application analysis sessions. | Do not copy monolith repositories directly; use AI Agent persistence and owner-service clients/ports where data belongs to other services. |
| Candidate AI | `logic-grpc-service/service/candidate_ai_service.go`; `logic-grpc-service/ai/candidate_adk_tools.go`; `logic-grpc-service/ai/candidate_tool_executor.go`; `logic-grpc-service/ai/candidate_tools.go`; `web-gin-service/handler/candidate/ai.go` | `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`; `smart-recruit-ai-agent-service/internal/application/service/**`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`; `smart-recruit-commons/ai/**`; `smart-recruit-gateway/handler/candidate/ai.go`; `smart-recruit-gateway/router/router.go`; `user-frontend/src/api/ai.ts` | Restore candidate-scoped tools, prompt/config lookup, streaming deltas/status, persisted messages, tool traces, usage audit/auth context, suggested-question extraction/fallback, and non-streaming `/candidate/ai/chat` compatibility. | Preserve candidate ownership isolation; avoid protobuf changes unless later confirmed. |
| Agent Run | `logic-grpc-service/service/agent_run_service.go`; `logic-grpc-service/service/agent_run_worker.go`; `logic-grpc-service/service/agent_run_state.go`; `logic-grpc-service/service/agent_run_recorder.go`; `logic-grpc-service/service/agent_run_events.go`; `logic-grpc-service/service/agent_run_hub.go`; `logic-grpc-service/repository/agent_run_repo.go`; `logic-grpc-service/repository/agent_run_event_repo.go` | `smart-recruit-ai-agent-service/internal/domain/model/agent.go`; `smart-recruit-ai-agent-service/internal/domain/policy/agent.go`; `smart-recruit-ai-agent-service/internal/application/service/agent_service.go`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`; `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`; `smart-recruit-gateway/handler/hr/ai.go` | Restore idempotent create, durable payload, queued/running/waiting-confirmation/succeeded/failed/canceled transitions, replayable events, cancellation, confirmation continuation, and execution through HR runtime. | Current direct `complete()` execution should become a durable wrapper around the recovered HR runtime. |
| Recruiting Intelligence | `logic-grpc-service/service/recruiting_intelligence_service.go`; `logic-grpc-service/repository/resume_profile_repo.go`; `logic-grpc-service/repository/candidate_match_repo.go`; related parser and matcher services | `smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go`; `smart-recruit-ai-agent-service/internal/application/service/**`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go`; `smart-recruit-gateway/handler/hr/recruiting_intelligence.go`; Recruitment owner-client ports if needed | Restore `ParseResumeProfile` and `EvaluateCandidateMatch` generation, snapshot persistence/versioning, `agent_run_id` association, and existing read/compare behavior. | Do not cross-write recruitment-owned data without an explicit service-owned adapter/port. |
| MCP | `logic-grpc-service/service/mcp_service.go`; `logic-grpc-service/service/mcp_policy.go`; `logic-grpc-service/service/mcp_policy_api.go`; `logic-grpc-service/service/mcp_log_api.go`; `logic-grpc-service/repository/mcp_repo.go`; `logic-grpc-service/server/mcp*`; `web-gin-service/handler/hr/mcp.go` | `smart-recruit-ai-agent-service/internal/infrastructure/mcp/**`; `smart-recruit-ai-agent-service/internal/application/service/capability_service.go`; `smart-recruit-ai-agent-service/internal/domain/policy/capability.go`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go`; `smart-recruit-gateway/handler/hr/mcp.go` | Restore live connection test, tool discovery, tool execution, policy evaluation, confirmation/deny/rate-limit behavior, redaction, audit logs, and HR Agent MCP tool binding. | Keep `NativeStore` as persistence; live transport belongs in an injected runner. |
| Agent Skill | `logic-grpc-service/service/agent_skill_service.go`; `logic-grpc-service/service/agent_skill_selector.go`; `logic-grpc-service/service/agent_skill_parser.go`; `logic-grpc-service/service/agent_skill_eligibility.go`; `logic-grpc-service/repository/agent_skill_repo.go`; `logic-grpc-service/repository/skill_repo.go` | `smart-recruit-ai-agent-service/internal/application/service/capability_service.go`; `smart-recruit-ai-agent-service/internal/domain/policy/capability.go`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go`; `smart-recruit-gateway/handler/hr/agent_skill.go`; `hr-frontend/src/api/agentSkill.ts` | Restore skill CRUD/version/status behavior, capability validation, manual and semantic selection, rule fallback, confirmation metadata, preview, and semantic debug fields. | Skill instructions are prompt/runtime inputs, not permission bypasses. |
| Embedding | `logic-grpc-service/service/embedding_service.go`; `logic-grpc-service/service/embedding_config_service.go`; `logic-grpc-service/service/embedding_backfill_service.go`; `logic-grpc-service/service/embedding_event_publisher.go`; `logic-grpc-service/service/embedding_provider_bailian.go`; `logic-grpc-service/service/embedding_text_builder.go`; `logic-grpc-service/repository/ai_embedding_repo.go`; `logic-grpc-service/repository/embedding_provider_repo.go`; `logic-grpc-service/repository/embedding_model_repo.go` | `smart-recruit-ai-agent-service/internal/application/service/capability_service.go`; `smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go`; `smart-recruit-ai-agent-service/internal/infrastructure/provider/**`; `smart-recruit-gateway/handler/hr/embedding_config.go`; `hr-frontend/src/api/embedding.ts` | Restore embedding model test, provider runtime resolution, Agent Skill embedding upsert/invalidation/backfill, vector search, semantic retrieval debug metadata, and fallback reasons. | Current unsupported test/backfill paths should be replaced with a service-owned runtime implementation. |

## Current Architecture Targets

- AI runtime ownership: `smart-recruit-ai-agent-service`.
- HTTP transport and auth/rate/risk middleware: `smart-recruit-gateway`.
- Shared AI primitives: `smart-recruit-commons/ai`.
- Canonical contracts: `smart-recruit-proto/proto/recruitment.proto`.
- Frontend compatibility callers: `hr-frontend/src/api/ai.ts`, `hr-frontend/src/api/agentRun.ts`, `hr-frontend/src/api/agentSkill.ts`, `hr-frontend/src/api/embedding.ts`, `hr-frontend/src/api/mcp.ts`, and `user-frontend/src/api/ai.ts`.

## Reference Commands

```bash
git ls-tree -r --name-only origin/dev logic-grpc-service/service logic-grpc-service/ai logic-grpc-service/repository logic-grpc-service/server web-gin-service/handler
git show origin/dev:logic-grpc-service/service/ai_service.go
git show origin/dev:logic-grpc-service/service/candidate_ai_service.go
git show origin/dev:logic-grpc-service/service/recruiting_intelligence_service.go
git show origin/dev:logic-grpc-service/service/agent_run_service.go
git show origin/dev:logic-grpc-service/service/agent_run_worker.go
git show origin/dev:logic-grpc-service/service/agent_run_state.go
git show origin/dev:logic-grpc-service/service/agent_skill_service.go
git show origin/dev:logic-grpc-service/service/agent_skill_selector.go
git show origin/dev:logic-grpc-service/service/mcp_service.go
git show origin/dev:logic-grpc-service/service/embedding_service.go
git show origin/dev:logic-grpc-service/service/embedding_config_service.go
```
