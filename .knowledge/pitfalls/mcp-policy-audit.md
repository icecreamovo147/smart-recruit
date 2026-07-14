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
  - smart-recruit-ai-agent-service/internal/domain/policy/capability.go
  - smart-recruit-ai-agent-service/internal/infrastructure/mcp/doc.go
  - smart-recruit-gateway/handler/hr/mcp.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# MCP Policy Audit Pitfall

MCP policy changes can expose sensitive arguments, bypass confirmation, permit private-network targets, or lose audit context. Keep policy order, redaction, allowlists, rate limits, and persisted logs aligned.

## Verification

Verified against current repository files on 2026-07-14.
