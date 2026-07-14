---
schema_version: 1
id: mcp-policy-audit
title: MCP policy audit pitfall
kind: pitfall
status: active
owners:
  - agent-platform
tags:
  - mcp
  - policy
  - audit
applies_to:
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/infrastructure/mcp/doc.go
  - smart-recruit-gateway/handler/hr/mcp.go
source_refs:
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/infrastructure/mcp/doc.go
  - smart-recruit-gateway/handler/hr/mcp.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# MCP Policy Audit Pitfall

MCP policy changes can expose sensitive arguments, bypass confirmation, permit private-network targets, or lose audit context. Keep policy order, redaction, allowlists, rate limits, and persisted logs aligned.

When adding MCP admin behavior, do not turn unsupported runtime actions into success. Native list and CRUD paths can return persisted data, but live connection tests, tool discovery, and tool execution must stay non-success until a runner enforces private-network, command, policy, confirmation, and redaction rules end to end.

## Verification

Verified against current repository files on 2026-07-14.
