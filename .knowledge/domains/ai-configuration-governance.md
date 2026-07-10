---
schema_version: 1
id: ai-configuration-governance
title: AI configuration governance
kind: domain
status: active
owners:
  - agent-platform
tags:
  - ai
  - configuration
  - llm
  - prompt
  - embedding
  - agent
applies_to:
  - logic-grpc-service/service/llm_config_service.go
  - logic-grpc-service/service/embedding_config_service.go
  - logic-grpc-service/service/prompt_service.go
  - logic-grpc-service/service/agent_service.go
  - logic-grpc-service/service/agent_runtime_policy.go
  - web-gin-service/handler/hr/llm_config.go
  - web-gin-service/handler/hr/embedding_config.go
  - web-gin-service/handler/hr/prompt.go
  - web-gin-service/handler/hr/agent_config.go
  - hr-frontend/src/views/hr/LlmConfigView.vue
  - hr-frontend/src/views/hr/EmbeddingConfigView.vue
  - hr-frontend/src/views/hr/PromptManageView.vue
  - hr-frontend/src/views/hr/admin/AgentManageView.vue
source_refs:
  - logic-grpc-service/service/llm_config_service.go
  - logic-grpc-service/service/embedding_config_service.go
  - logic-grpc-service/service/prompt_service.go
  - logic-grpc-service/service/agent_service.go
  - logic-grpc-service/service/agent_runtime_policy.go
  - logic-grpc-service/model/model.go
  - web-gin-service/router/router.go
  - hr-frontend/src/views/hr/LlmConfigView.vue
  - hr-frontend/src/views/hr/EmbeddingConfigView.vue
  - hr-frontend/src/views/hr/PromptManageView.vue
  - hr-frontend/src/views/hr/admin/AgentManageView.vue
last_verified: 2026-07-10
review_after: 2026-10-08
---

# AI Configuration Governance

AI configuration is a management surface, not the agent runtime itself. Admin code persists providers, models, prompts, agent configs, capability bindings, embedding providers, and embedding models. Runtime code consumes enabled/default configuration and must handle missing, disabled, or unavailable providers explicitly.

## Configuration Surfaces

- `LlmConfigService` manages LLM providers and models. API keys are encrypted before persistence and decrypted only for runtime/test operations.
- `EmbeddingConfigService` manages embedding providers and models, rebuilds the provider after provider changes, supports default model selection, tests models, and can trigger embedding backfill.
- `PromptService` manages prompt templates and version history. Content changes create new version records, and rollback creates another version snapshot from historical content.
- `AgentConfigService` manages agent configs, prompt binding, iteration limits, temperature overrides, default/enabled state, and capability bindings.
- `AgentRuntimePolicy` represents feature gates and timeouts for planner, resume parsing, candidate match, semantic retrieval, MCP policy, Skill governance, and fallbacks.

## Access Boundary

The gateway routes LLM, embedding, MCP server, SKILL registry, and most system configuration endpoints through `SYSTEM_CONFIG_MANAGE`. Prompt management uses `AI_PROMPT_MANAGE`; agent config uses `AI_AGENT_MANAGE`; Agent Skill management uses `AI_AGENT_SKILL_MANAGE`; runtime HR AI use routes use `AI_HR_USE`.

When adding or moving an AI admin route, update both the router permission and the HR frontend route/menu permission. Treat permission drift as a security and operations risk.

## Data Handling

- Do not document or copy real API keys, provider endpoints, extra headers, MCP environment variables, or runtime secrets into active knowledge.
- Provider credentials are represented by encrypted columns and request payloads. Active knowledge should describe fields and flows, not values.
- Prompt content, Skill instructions, tool outputs, and MCP logs can contain business-sensitive data. Quote only minimal structural examples if a future TASK explicitly permits it.

## Runtime Relationship

Runtime request assembly can depend on active model, context window, max output tokens, prompt template, agent instruction, capability bindings, selected Agent Skills, and embedding availability. Configuration changes can alter runtime behavior without code changes, so regression reports should capture the active configuration shape and permission used to change it.

## Review Triggers

- Provider/model fields, encrypted credential handling, default model selection, connection test behavior, or backfill behavior.
- Prompt versioning, rollback, active state, or agent type/role semantics.
- Agent config capability binding or runtime policy defaults.
- Gateway route permission changes for any AI admin or runtime endpoint.
- Frontend admin pages that create, update, delete, test, activate, or backfill AI configuration.

## Verification

Verified against current LLM, embedding, prompt, agent config, runtime policy, gateway route, model, and HR admin view sources on 2026-07-10.
