---
schema_version: 1
id: memory-and-context
title: Memory and Agent context domain
kind: domain
status: active
owners:
  - agent-platform
tags:
  - memory
  - context
  - agent
  - prompt
applies_to:
  - logic-grpc-service/service/agent_context.go
  - logic-grpc-service/repository/memory_repo.go
  - logic-grpc-service/repository/session_summary_repo.go
  - logic-grpc-service/model/model.go
source_refs:
  - logic-grpc-service/service/agent_context.go
  - logic-grpc-service/repository/memory_repo.go
  - logic-grpc-service/repository/session_summary_repo.go
  - .spec/skill-memory-ranking/skill-memory-ranking-SPEC.md
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Memory and Agent Context Domain

Agent context combines recent chat messages, session summary, active system prompt template, long-term memories, current message text, and budget metadata. Long-term memories are recalled by scope, ranked, then trimmed by configured count and character budget before prompt assembly.

Memory behavior is a domain concern because it changes what the AI assistant can see. It must remain auditable and bounded by HR/session/application/job context. Do not document memory recall as product runtime RAG for `.knowledge`; this knowledge base is a separate coding-Agent layer.

## Context Layers

- Recent messages: bounded chronological session history.
- Session summary: compact summary if present for the session.
- System prompt template: active HR agent system template from repository.
- Long-term memories: scoped recall candidates ranked for current request.
- Budget metadata: character counts for prompt estimate and trimming.

## Review Triggers

- Scope derivation for memory recall.
- Ranking signals, fallback ordering, or prompt budget trimming.
- Debug fields for memory ranking or pool confidence.
- Persistence schema or repository queries for memories and summaries.

## Verification

This document was verified from `agent_context.go`, memory and summary repositories, and the Skill/Memory ranking SPEC on 2026-07-10.
