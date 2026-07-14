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
  - governance
  - model
applies_to:
  - smart-recruit-ai-agent-service/**
  - smart-recruit-gateway/handler/hr/*config*.go
  - hr-frontend/src/views/hr/*ConfigView.vue
  - hr-frontend/src/views/hr/PromptManageView.vue
source_refs:
  - smart-recruit-ai-agent-service/internal/domain/model/agent.go
  - smart-recruit-ai-agent-service/internal/domain/policy/agent.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/config_services.go
  - smart-recruit-ai-agent-service/internal/application/service/agent_service.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/config_store.go
  - smart-recruit-gateway/handler/hr/llm_config.go
  - smart-recruit-gateway/handler/hr/embedding_config.go
  - smart-recruit-gateway/handler/hr/prompt.go
  - smart-recruit-gateway/handler/hr/agent_config.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# AI Configuration Governance

AI configuration covers LLM providers/models, embedding providers/models, prompt templates and versions, agent configs, capability bindings, runtime policy, MCP policy, and Agent Skills. Preserve auditability, credential redaction, default uniqueness, prompt version history, rollback semantics, and admin permission checks.

The native AI Agent gRPC runtime now backs LLM, embedding, prompt, and agent configuration surfaces with `NativeStore` persistence instead of inherited unimplemented stubs. Live provider and embedding model tests validate persisted configuration and return explicit non-success unsupported/configuration responses when no provider client or worker is bound; secrets returned through configuration reads remain masked or redacted.

## Verification

Verified against current repository files on 2026-07-14.
