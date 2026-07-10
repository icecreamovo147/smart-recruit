---
schema_version: 1
id: embedding-fallback
title: Embedding fallback pitfall
kind: pitfall
status: active
owners:
  - agent-platform
tags:
  - embedding
  - fallback
  - retrieval
  - debug
applies_to:
  - logic-grpc-service/service/embedding_service.go
  - logic-grpc-service/service/embedding_provider_factory.go
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/service/agent_context.go
source_refs:
  - logic-grpc-service/service/embedding_service.go
  - logic-grpc-service/service/embedding_provider_factory.go
  - logic-grpc-service/service/agent_skill_selector.go
  - logic-grpc-service/service/agent_skill_service.go
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Embedding Fallback Pitfall

Embedding unavailable is a valid runtime state, not a successful semantic retrieval. The code has explicit unavailable provider behavior, status values, and debug metadata. Treating fallback as a normal vector-backed result can mislead ranking review and product debugging.

## Trigger Conditions

- Provider configuration is missing, invalid, or cannot be built.
- Provider call fails during embedding creation or search.
- Vector data is empty, invalid, or has mismatched dimensions.
- Debug endpoints reuse stale or overwritten search metadata.

## Risk

Agent Skill ranking or memory recall may still produce results, but the results are not evidence that embedding search worked. Debug UIs can look healthy if they show candidates without also surfacing provider availability and fallback reason.

## Prevention

- Preserve explicit unavailable status and fallback reason.
- Keep provider/model/dimension/candidate count/latency metadata accurate for the current request.
- Test fallback paths separately from vector-backed paths.
- Do not use semantic score alone as proof of relevance.

## Verification

This pitfall was verified from `EmbeddingService`, provider factory, Skill selector, and semantic debug service code on 2026-07-10.
