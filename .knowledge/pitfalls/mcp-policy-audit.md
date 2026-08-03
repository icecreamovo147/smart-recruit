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
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_agent_skill_confirmation.go
  - smart-recruit-ai-agent-service/internal/infrastructure/mcp/doc.go
  - smart-recruit-gateway/handler/hr/mcp.go
source_refs:
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/mcp_skill_services.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/mcp_skill_store.go
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/infrastructure/mcp/doc.go
  - smart-recruit-gateway/handler/hr/mcp.go
  - smart-recruit-gateway/handler/hr/ai.go
  - smart-recruit-gateway/router/contract_baseline_test.go
  - smart-recruit-proto/proto/recruitment.proto
last_verified: 2026-07-28
review_after: 2026-10-14
---

# MCP Policy Audit Pitfall

MCP policy changes can expose sensitive arguments, bypass confirmation, permit private-network targets, or lose audit context. Keep policy order, redaction, allowlists, rate limits, and persisted logs aligned.

When adding MCP admin behavior, do not turn unavailable or policy-blocked runtime actions into success. Native list and CRUD paths can return persisted data. Live connection tests, discovery, and execution may succeed only when a bound runner enforces private-network/transport and command constraints, policy, confirmation, redaction, and audit rules end to end.

Never interpret an absent data/Tool capability selection as permission to run every Agent-bound MCP Tool. HR runtime requires explicit `capability_keys` and intersects them with enabled Agent bindings. Empty selection is zero calls; rejected, malformed, unavailable, or failed calls remain error traces and cannot become useful model facts. Do not restore the retired `skill_capability_keys` name or `/api/v1/hr/ai/skill-capabilities` route as an alias.

Do not merge MCP and Agent Skill confirmation. MCP approval is bound to capability and arguments through the opaque MCP payload. Agent Skill Package approval is bound separately to actor, Run, exact user message, capability release snapshot, exact version IDs/compiled hashes, risk, expiry, and a durable CAS/lease state machine. The old boolean Skill confirmation is neither a compatibility input nor authority for either path.

## Verification

Verified against cumulative policy runtime, `capability_keys` gateway/Proto contracts, separate MCP and Agent Skill durable confirmation tests, HR Tool error traces, removed-route regression tests, and AI Agent service tests on 2026-07-28.
