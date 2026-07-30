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
  - .spec/development-agent-knowledge-base/**
  - .spec/knowledge-base-current-state-refresh/**
  - tools/agent-orchestrator/**
source_refs:
  - AGENTS.md
  - .knowledge/README.md
  - .knowledge/INDEX.md
  - .knowledge/manifest.yaml
  - .knowledge/architecture/auth-rbac-security.md
  - .knowledge/architecture/api-contracts-and-gateway.md
  - .knowledge/architecture/frontend-apps.md
  - .knowledge/architecture/persistence-and-migrations.md
  - .knowledge/domains/notification-outbox.md
  - .knowledge/domains/recruitment-lifecycle.md
  - .knowledge/runbooks/local-development.md
  - interviewer-frontend/README.md
  - pnpm-workspace.yaml
  - .spec/development-agent-knowledge-base/development-agent-knowledge-base-SPEC.md
  - .spec/knowledge-base-current-state-refresh/knowledge-base-current-state-refresh-SPEC.md
  - tools/agent-orchestrator/README.md
  - tools/agent-orchestrator/orchestrate_serial.py
  - tools/agent-orchestrator/tasks.yaml
last_verified: 2026-07-30
review_after: 2026-10-21
---

# Knowledge Coverage Audit Runbook

Use this runbook when expanding or reviewing the coding-Agent knowledge layer. The goal is main route coverage, not a page-by-page or function-by-function encyclopedia.

## Coverage Classes

- Covered: a future TASK touching the area routes to active architecture, domain, runbook, or pitfall knowledge with current source references.
- Partially covered: a broad document exists, but important subflows or recurring risks are not routed.
- High-risk uncovered: the area includes auth, authorization, public contracts, schema, sensitive data, AI configuration, MCP tools, or cross-surface business workflows without a focused route.
- Low-risk uncovered: the area is simple, local, or rarely changed, and broad system knowledge is enough for now.

## Current Coverage Map

| Area | Current status | Remaining review focus |
|---|---|---|
| System overview and service boundaries | Covered by active system overview and service-boundary documents | Recheck on new top-level modules or ownership changes |
| Agent runtime, semantic retrieval, Skill, Memory, Embedding fallback | Covered by focused active Agent documents and pitfalls | Recheck on provider, retrieval, memory, or fallback behavior changes |
| Auth, refresh tokens, RBAC, data scopes, security audit | Covered by focused architecture, runbook, and alignment pitfall documents | Keep gateway, Identity, frontend permission metadata, and seed catalog aligned |
| Gin gateway, handlers, middleware, gRPC clients | Covered by gateway/API architecture, service boundaries, and local-development routing | Recheck on route, middleware, metadata, or public contract changes |
| Proto, generated code, migrations, model, repository | Covered by protobuf, persistence/migration architecture, runbook, and drift pitfalls | Recheck generated contracts, baseline schema, ownership, and adapters together |
| Jobs, applications, interviews, offers | Covered by recruitment and lifecycle documents plus debugging/status pitfalls | Recheck cross-context transitions and frontend labels together |
| Collaboration, notification, outbox, SSE, analytics | Notification/outbox and cross-context drift are covered. **Accepted coverage gap (KREM-010 / KNO-010):** Analytics remains broad system/service-boundary coverage only; Inbox candidate `analytics-projection-knowledge` stays draft. Owner: `architecture-and-operations`. Revisit by `2026-10-21` or sooner if projection rebuild/consistency work becomes frequent. | Promote focused Analytics Active knowledge only after that revisit |
| LLM providers/models, prompt templates, Agent config, MCP tools | Covered by AI configuration and MCP governance documents | Recheck secrets, policy, provenance, and runtime binding behavior |
| Resume upload, parsing, structured profile, candidate matching | Covered by resume intelligence, diagnostics, and sensitive-data pitfall documents | Recheck privacy and source ownership on every data-flow change |
| Three active Vue apps (`hr`, `user`, `platform`) and `packages/shared` | Covered by frontend architecture, validation, menu pitfall, and explicit shared-package routing; `interviewer-frontend` is isolated to a legacy Manifest route that does not treat it as an active frontend validation target | Validate every consuming active app for shared-package changes; keep legacy interviewer routes from re-entering the active frontend matrix |
| Deployment and infrastructure | Covered across system overview, local development, service-binary convention, and routes for `docker/`, `deploy/`, and `smart-recruit-deploy/` | Recheck service names, images, health, and configuration together |
| Email delivery | **Accepted partial coverage gap (KREM-010 / KNO-010):** provider/retry/fault handling remains partially covered by notification/outbox knowledge only. Owner: `architecture-and-operations`. Revisit by `2026-10-21` or sooner if email operations expand. | Add a focused delivery runbook if provider/retry operations expand |
| Command-line tools under `cmd/` directories | Migration command is explicitly routed; service binaries are covered by convention knowledge | Add focused routes for other operational commands when they gain independent workflows |
| Historical `tools/agent-orchestrator` | Isolated in place as read-only historical evidence. Only `--dry-run` inventory inspection is allowed; actual execution fails before Git/Agent side effects. Missing `docs/agent-harness/tasks/` references, the old `integration/agent-platform` model, task list, implementation, and logs are retained for provenance, not current routing. | Keep the exact Manifest route pointed at this runbook; do not reactivate or migrate without a new explicit owner decision |

## Audit Procedure

1. Read `AGENTS.md`, `.knowledge/README.md`, `INDEX.md`, and `manifest.yaml`. Read an active feature contract only when the user explicitly named `spec-harness` or instructed the Agent to use the repository Harness workflow.
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

## Accepted Coverage Gaps

Recorded by KREM-010 on 2026-07-23 after explicit owner confirmation:

| Gap | Decision | Owner | Review by |
|---|---|---|---|
| Analytics projection/rebuild knowledge | Accept broad coverage; keep Inbox candidate draft; do not promote yet | architecture-and-operations | 2026-10-21 |
| Email delivery operations runbook | Accept partial notification/outbox coverage; defer dedicated runbook | architecture-and-operations | 2026-10-21 |

## Verification

Run the standard knowledge checks after any coverage update:

```bash
node .knowledge/scripts/validate-knowledge.mjs --root .
node .knowledge/scripts/check-references.mjs --root .
```

When the current work explicitly uses an active Harness feature and has a reliable TASK base tree, also run its task-scope and agent checks. Do not invoke `spec-harness` or create a feature contract solely because this coverage audit is being used.

Verified against the active knowledge catalog, manifest routes, current frontend shared package, `pnpm-workspace.yaml`, `interviewer-frontend/README.md`, migration startup path, deployment roots, accepted Analytics/email coverage-gap decision, the isolated historical Agent Orchestrator CLI/task inventory, and current knowledge-validation workflow on 2026-07-30.
