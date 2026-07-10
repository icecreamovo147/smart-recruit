---
schema_version: 1
id: debug-ai-configuration
title: Debug AI configuration
kind: runbook
status: active
owners:
  - agent-platform
tags:
  - ai
  - configuration
  - runtime
  - mcp
  - embedding
applies_to:
  - logic-grpc-service/service/llm_config_service.go
  - logic-grpc-service/service/embedding_config_service.go
  - logic-grpc-service/service/prompt_service.go
  - logic-grpc-service/service/agent_service.go
  - logic-grpc-service/service/mcp_service.go
  - web-gin-service/router/router.go
  - hr-frontend/src/views/hr/LlmConfigView.vue
  - hr-frontend/src/views/hr/EmbeddingConfigView.vue
  - hr-frontend/src/views/hr/admin/AgentManageView.vue
  - hr-frontend/src/views/hr/admin/McpManageView.vue
source_refs:
  - logic-grpc-service/service/llm_config_service.go
  - logic-grpc-service/service/embedding_config_service.go
  - logic-grpc-service/service/prompt_service.go
  - logic-grpc-service/service/agent_service.go
  - logic-grpc-service/service/mcp_service.go
  - logic-grpc-service/service/agent_runtime_policy.go
  - web-gin-service/router/router.go
  - hr-frontend/src/api/llm.ts
  - hr-frontend/src/api/embedding.ts
  - hr-frontend/src/api/agent.ts
  - hr-frontend/src/api/mcp.ts
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Debug AI Configuration

Use this runbook when AI chat, semantic retrieval, Agent Skill selection, provider tests, MCP capabilities, or admin AI configuration behave unexpectedly.

## 1. Identify the Surface

- Runtime chat failure: inspect HR AI route permission, selected model, context window, prompt tokens, selected skills, and stream errors.
- Provider/model failure: inspect LLM or embedding provider/model admin state and test endpoint behavior.
- Prompt/agent behavior drift: inspect prompt template version, active state, agent config, prompt binding, instruction, default/enabled state, and capability bindings.
- MCP capability failure: inspect server enabled state, tool list, policy decision, and tool logs.
- Semantic retrieval failure: inspect embedding provider/model availability, vector dimension, fallback reason, and backfill status.

## 2. Check Permissions and Routes

Use `web-gin-service/router/router.go` as the route authority. Config routes generally require system or AI admin permissions; runtime HR AI use requires AI HR use permission. If a frontend page is visible but the API fails with authorization errors, compare frontend route meta, menu permissions, and gateway route permission.

## 3. Check Configuration State

- LLM: provider enabled, model enabled, default model, context window tokens, max tokens, concurrency, timeout, provider type, base URL shape.
- Embedding: provider enabled, model enabled, default model, embedding dimension, token limit, batch size, retry/timeout, provider rebuild after changes.
- Prompt: template active state, agent type, prompt role, current version, version history, rollback record.
- Agent config: active config for agent type, prompt binding, max iterations, temperature override, enabled/default state, capability bindings.
- Runtime policy: feature gate values and timeout defaults from `AgentRuntimePolicy`.

Do not print or paste API keys, extra headers, MCP env vars, or other secrets into reports.

## 4. Check MCP Governance

For MCP issues, compare server config, tool list, policy list, call result, and tool logs. If a call is denied or requires confirmation, record policy decision and reason. If there is no policy and the call is allowed, record that as current-code behavior and decide separately whether a policy should be added.

## 5. Check Embedding and Backfill

Semantic retrieval depends on provider availability and stored vectors. If embeddings are unavailable, debug output should show fallback state rather than fake semantic success. After changing default embedding model or regenerating Agent Skill embeddings, verify backfill counts and debug retrieval metadata.

## 6. Suggested Verification

- `go test ./...` from `logic-grpc-service/` when changing services, repositories, policy, embedding, or runtime code.
- `go test ./...` from `web-gin-service/` when changing handlers, route permissions, or response shape.
- `pnpm --filter hr-frontend typecheck` when changing HR admin API/types/views.
- Targeted frontend tests if added or touched by the TASK.

## Evidence to Record

Record route, permission, provider/model IDs or names, enabled/default booleans, prompt template ID/version, agent type, capability source/key, policy decision/reason, embedding provider/model/dimension, and test command results. Exclude secrets and raw sensitive business content.
