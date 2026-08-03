---
schema_version: 1
id: api-contracts-and-gateway
title: API contracts and gateway architecture
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - gateway
  - api
  - grpc
  - contract
applies_to:
  - smart-recruit-gateway/config/**
  - smart-recruit-gateway/router/**
  - smart-recruit-gateway/handler/**
  - smart-recruit-gateway/middleware/**
  - smart-recruit-gateway/rpc/**
  - smart-recruit-proto/**
source_refs:
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/rpc/client.go
  - smart-recruit-gateway/config/config.go
  - smart-recruit-gateway/middleware/body_limit.go
  - smart-recruit-gateway/middleware/ratelimit.go
  - smart-recruit-gateway/middleware/observability.go
  - smart-recruit-gateway/handler/hr/ai.go
  - smart-recruit-gateway/handler/hr/agent_skill.go
  - smart-recruit-gateway/router/contract_baseline_test.go
  - smart-recruit-gateway/cmd/gateway/main.go
  - smart-recruit-platform-go/businessclock/clock.go
  - smart-recruit-platform-go/i18n/i18n.go
  - smart-recruit-platform-go/i18n/catalogs/zh-CN.json
  - smart-recruit-platform-go/i18n/catalogs/en-US.json
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - hr-frontend/src/api/ai.ts
  - hr-frontend/src/types/ai.ts
  - hr-frontend/src/utils/hrAgentRunReducer.ts
  - packages/shared/src/types/agentRun.ts
  - packages/shared/src/types/agentSkill.ts
  - scripts/check-agent-skill-v2-cutover.mjs
last_verified: 2026-07-28
review_after: 2026-10-21
---

# API Contracts and Gateway Architecture

The gateway exposes `/api/v1`, applies timeout/body/rate/quota/risk/auth middleware, and calls generated gRPC clients. `smart-recruit-gateway/rpc/client.go` defaults route modes to independent service targets and forwards internal auth, request, and trace metadata.

Runtime system messages use the deployment-wide `APP_LOCALE` (`zh-CN` by
default, or `en-US`). The Gateway is the only HTTP localization boundary:
responses retain `code/msg/data/request_id`, add a stable `message_key`, and
render `msg` from the canonical platform catalog. The unauthenticated
`GET /api/v1/public/runtime-config` endpoint exposes only the active `locale` so
the three product frontends can initialize their shared catalog and Element Plus
locale without a rebuild. gRPC `msg` values and SSE system messages are keys;
user content and model output are not translated. Raw upstream failures stay in
structured server-side `cause` fields and must never be forwarded as `msg`.

AI daily and burst quota accounting distinguishes admission from an attempted HTTP request. Candidate streaming handlers mark the quota consumed only after the AI service emits its first runtime event; capability, configuration, and billing failures before that event refund the provisional daily count and do not contribute to the burst risk block. Once runtime admission occurs, later provider/stream failures remain counted because upstream work may already have happened.

The HR durable Agent Run endpoint requires a positive session ID and a non-empty trimmed message. It rejects blank input before invoking the AI gRPC client, while the AI service repeats the validation as a defense-in-depth boundary. Application-analysis session responses use the existing repeated `messages` field; no wire-shape change is needed to return the canonical seeded analysis message.

Agent Skill Package v2 is a direct-cutover contract:

- `ChatRequest`, `CreateAgentRunRequest`, and the HR HTTP payload use `agent_skill_version_ids` for exact immutable Package versions. Old `agent_skill_ids` and boolean/message Skill-confirmation fields are reserved in Proto and rejected as inbound JSON.
- Per-message data/Tool selection is `capability_keys`. The active discovery endpoint is `GET /api/v1/hr/ai/capabilities`; the old `skill_capability_keys` field and `/api/v1/hr/ai/skill-capabilities` route have no alias.
- Agent Skill approval uses dedicated confirmation ID/decision/exact selected version IDs and message identity/expiry fields. `confirmation_payload_json` remains the independent opaque MCP Tool channel.
- Agent Skill admin and preview payloads contain schema-v2 manifest/Core/sections. Retired `skill_md`, `flow_json`, `frontmatter_json`, and `body_markdown` names are reserved and never emitted as active Package data.

The route baseline and repository cutover scanner enforce absence of retired routes/symbols while allowing reserved descriptors, destructive migration/down SQL, archives, and explicit negative tests.

HR AI chat JSON and Agent Run event JSON now forward two additive contract fields that must stay aligned across Proto, AI Agent, Gateway, and HR frontend:

- `suggested_questions` (`ChatResponse` field 13, `ChatStreamResponse` field 14, `AgentRunResultMetadata` field 8): optional repeated strings. Gateway chat handlers copy `resp.GetSuggestedQuestions()` into the HTTP body; Agent Run result metadata includes the same key. Treat absence or empty arrays as “no suggestions,” never as an error.
- `snapshot_text` (`AgentRunEvent` field 7): optional full replacement text for assistant or process snapshots. Gateway `agentRunEventPayload` exposes it beside `payload_json`. Prefer the first-class field over re-parsing nested JSON when present; clients may still fall back to payload JSON for older events.

These fields are additive and privacy-sensitive. Do not log suggestion text or snapshot bodies into ordinary diagnostics, and do not document or persist hidden prompt/provider payloads through knowledge.

Changing HTTP routes normally requires route registration, handler mapping, permission metadata, frontend API/types, and a matching protobuf or service contract. Changing protobuf wire shape is rooted in `smart-recruit-proto/proto/recruitment.proto`; public-facing service extensions can force gateway and handler test clients to implement new methods, so internal owner contracts should prefer separate internal gRPC services when they are not part of frontend/gateway behavior.

The legacydomain retirement contract adds internal `ApplicationOwnerService` and `AuthService.AuthorizeInternal` without adding new HTTP routes or frontend entry points. `rpc.Clients` may expose generated internal clients, but gateway route behavior remains unchanged unless handlers are explicitly modified.

The platform business timezone is fixed at `Asia/Shanghai`. RFC3339 response strings are normalized to `+08:00`; RFC3339 inputs with `Z` or another legal offset retain their instant and are converted at the business boundary. Unix seconds/milliseconds, JWT claims, TTLs, and durations remain absolute and are never adjusted by eight hours. Daily quota keys and reset timestamps use Beijing civil-day boundaries.

## Verification

Verified against current `recruitment.proto` Package v2/reserved fields,
Gateway HR AI and Agent Skill handlers, registered/removed route baselines,
shared/HR exact-version types and payloads, cutover scanner, ChatResponse and
AgentRunEvent fields, runtime-config handlers, canonical i18n catalogs, and
frontend runtime initialization on 2026-07-28.
