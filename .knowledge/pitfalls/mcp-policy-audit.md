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
last_verified: 2026-07-16
review_after: 2026-10-14
---

# MCP Policy Audit Pitfall

MCP policy changes can expose sensitive arguments, bypass confirmation, permit private-network targets, or lose audit context. Keep policy order, redaction, allowlists, rate limits, and persisted logs aligned.

When adding MCP admin behavior, do not turn unavailable or policy-blocked runtime actions into success. Native list and CRUD paths can return persisted data. Live connection tests, discovery, and execution may succeed only when a bound runner enforces private-network/transport and command constraints, policy, confirmation, redaction, and audit rules end to end.

Never interpret an absent user/Skill capability selection as permission to run every Agent-bound MCP tool. HR runtime requires an explicit selected key and intersects it with enabled Agent bindings. Empty selection is zero calls; rejected, malformed, unavailable, or failed calls remain error traces and cannot become useful model facts.

## Verification

Verified against cumulative policy runtime, HR MCP selection, Tool error trace, and AI Agent service tests on 2026-07-16.
