---
schema_version: 1
id: knowledge-coverage-audit
title: Knowledge coverage audit runbook
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - knowledge
  - coverage
  - routing
  - harness
applies_to:
  - .knowledge/**
  - .spec/populate-development-agent-knowledge/**
source_refs:
  - .knowledge/README.md
  - .knowledge/INDEX.md
  - .knowledge/manifest.yaml
  - .spec/populate-development-agent-knowledge/populate-development-agent-knowledge-SPEC.md
  - .spec/populate-development-agent-knowledge/populate-development-agent-knowledge-SDD.md
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Knowledge Coverage Audit Runbook

Use this runbook when expanding or reviewing the coding-Agent knowledge layer. The goal is main route coverage, not a page-by-page or function-by-function encyclopedia.

## Coverage Classes

- Covered: a future TASK touching the area routes to active architecture, domain, runbook, or pitfall knowledge with current source references.
- Partially covered: a broad document exists, but important subflows or recurring risks are not routed.
- High-risk uncovered: the area includes auth, authorization, public contracts, schema, sensitive data, AI configuration, MCP tools, or cross-surface business workflows without a focused route.
- Low-risk uncovered: the area is simple, local, or rarely changed, and broad system knowledge is enough for now.

## First-Round Coverage Map

| Area | Current status | Planned owner TASK |
|---|---|---|
| System overview and service boundaries | Covered by initial architecture documents | Existing knowledge |
| Agent runtime, semantic retrieval, Skill, Memory, Embedding fallback | Partially covered | TASK-005 |
| Auth, refresh tokens, RBAC, data scopes, security audit | High-risk uncovered | TASK-002 |
| Gin gateway, handlers, middleware, gRPC clients | Partially covered by service boundaries | TASK-003 |
| Proto, generated code, migrations, model, repository | Partially covered by proto pitfall and recruitment domain | TASK-003 |
| Jobs, applications, interviews, offers | Partially covered by recruitment domain | TASK-004 |
| Collaboration, notification, outbox, SSE, analytics | Partially covered; Analytics projection has a draft Inbox candidate pending promotion | TASK-004 / TASK-BDME-024 |
| LLM providers/models, prompt templates, Agent config, MCP tools | High-risk uncovered | TASK-005 |
| Resume upload, parsing, structured profile, candidate matching | High-risk uncovered | TASK-006 |
| Three Vue apps, route guards, auth stores, API wrappers, validation | Partially covered by system overview and HR admin pitfall | TASK-007 |
| Deployment and infrastructure | Low-risk partially covered by local-development | Follow-up if deployment work becomes active |
| Email delivery | Low-risk uncovered relative to current request | Follow-up or notification TASK extension |
| Command-line tools under `logic-grpc-service/cmd/` | Low-risk uncovered | Follow-up if tool work begins |

## Audit Procedure

1. Read `AGENTS.md`, the active feature contract, `.knowledge/README.md`, `INDEX.md`, and `manifest.yaml`.
2. List planned changed paths and match them against manifest routes.
3. Classify each touched area as covered, partially covered, high-risk uncovered, or low-risk uncovered.
4. Verify critical knowledge claims against each document's `source_refs`.
5. If a routed active document conflicts with source code, report `CONFLICT` or `STALE`; do not silently rely on it.
6. If a core path has no route, report `coverage_gap` and add an Inbox candidate or TASK follow-up when scope allows.
7. Keep route additions narrow enough that future TASKs load useful context rather than the whole knowledge base.

## Candidate Handling

Use `.knowledge/inbox/` when:

- a useful claim is not yet verified;
- a conclusion would define security, authorization, data lifecycle, or public API policy;
- the current TASK scope does not allow modifying the formal document;
- a coverage gap needs human prioritization before becoming active knowledge.

## Verification

Run the standard knowledge checks after any coverage update:

```bash
node .knowledge/scripts/validate-knowledge.mjs --root .
node .knowledge/scripts/check-references.mjs --root .
```

When a reliable TASK base tree exists, also run the task scope and agent checks from the active feature harness.
