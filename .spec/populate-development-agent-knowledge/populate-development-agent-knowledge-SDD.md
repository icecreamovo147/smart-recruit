# Populate Development Agent Knowledge SDD

## 1. Existing Architecture Summary

Smart Recruit contains three Vue 3 applications (`hr-frontend/`, `user-frontend/`, `interviewer-frontend/`), a Gin HTTP gateway (`web-gin-service/`), and a Go gRPC logic service (`logic-grpc-service/`). The logic service owns domain services, repositories, models, migrations, AI orchestration, object storage, message queue integration, and protobuf service implementations. The gateway owns HTTP routing, middleware, role and permission checks, request limits, streaming responses, and gRPC client calls.

The `.knowledge/` layer already exists with governance docs, schema, validation scripts, a manifest, an index, and 12 initial formal documents. Current validation passes. The next phase must expand project knowledge using current source evidence while preserving `.knowledge/README.md` authority rules.

## 2. Problem Analysis

The initial knowledge base covers the skeleton: system overview, service boundaries, Agent runtime, semantic retrieval, recruitment, Agent Skill, memory/context, local development, retrieval debugging, proto synchronization, embedding fallback, and HR admin menu consistency.

Current project complexity exceeds that initial set. Important implementation areas remain uncovered or only broadly routed:

- auth, refresh token, RBAC, data scope, and audit behavior;
- gateway middleware and handler contracts;
- migration/model/repository alignment;
- outbox, notification, SSE, and email flows;
- deeper recruitment lifecycle cross-effects;
- LLM, Prompt, Agent config, MCP, Skill, Agent Skill, and Embedding governance;
- resume parsing, structured profile, and candidate matching;
- three-frontend routing, request, auth, and validation conventions.

Without filling these areas, future Agents will still need to rediscover large parts of the system and may miss cross-module risks.

## 3. Proposed Design

Implement the feature as eight serial TASKs:

1. Coverage audit and route matrix.
2. Auth/RBAC/security knowledge.
3. Gateway/public-contract/persistence knowledge.
4. Recruitment lifecycle knowledge.
5. AI platform and configuration governance knowledge.
6. Resume intelligence and candidate matching knowledge.
7. Frontend application architecture knowledge.
8. Final audit, validators, and follow-up backlog.

Each TASK modifies only `.knowledge/**` documents and this feature's `.spec/**` artifacts. TASKs may update `INDEX.md` and `manifest.yaml` when new documents must become routable. Each TASK must run knowledge validators and report knowledge impact.

## 4. Data Structure Changes

No database or runtime data structure changes.

New knowledge documents must use the existing frontmatter schema:

```yaml
schema_version: 1
id: current-code-verified-id
title: Human readable title
kind: architecture
status: active
owners:
  - stable-owner
tags:
  - relevant-tag
applies_to:
  - repository/path/**
source_refs:
  - repository/path/source.go
last_verified: 2026-07-10
review_after: 2026-10-08
```

When a finding is not ready for active knowledge, use `.knowledge/inbox/` with `kind: candidate` and `status: draft`.

## 5. API and Interface Changes

No HTTP, gRPC, proto, database, frontend, CLI, or CI interface changes are allowed.

The only interface-like changes are `.knowledge/manifest.yaml` route additions and `INDEX.md` navigation updates. These must remain compatible with existing validators and the current YAML subset parser.

## 6. Algorithm or Workflow Changes

### 6.1 TASK Knowledge Workflow

For each implementation TASK:

1. Read `AGENTS.md`, this SPEC, this SDD, `TASKS.md`, `AGENT_RULES.md`, task scope, and acceptance.
2. Read `.knowledge/README.md`, `manifest.yaml`, and `INDEX.md`.
3. Inspect only source files needed to verify the TASK's knowledge claims.
4. Create or update scoped knowledge documents.
5. Update routes and index entries only when needed for deterministic task routing.
6. Run validators and the feature agent check.
7. Produce TASK report and evidence with `knowledgeImpact`.

### 6.2 Coverage Strategy

The feature uses "main route coverage" instead of exhaustive file coverage. A module is covered when a future TASK touching its primary paths can be routed to:

- at least one architecture or domain document;
- relevant runbook or pitfall when the area has known operational risks;
- source references that let the Agent verify critical facts.

Uncovered modules are not hidden. They are recorded as `coverage_gap`, Inbox candidates, or final follow-up items.

## 7. Configuration Design

No runtime configuration changes.

`manifest.yaml` route additions should group paths by task-relevant ownership, for example:

- auth/RBAC: gateway middleware, authz packages, auth service, user/role repositories, RBAC migrations, HR staff user pages;
- gateway contracts: router, handlers, rpc clients, proto generated paths;
- persistence: migrations, model, repository;
- notification: outbox, mq, notification services and frontend notification components;
- AI configuration: LLM, prompt, agent config, MCP, embedding config services and HR admin pages;
- resume intelligence: OSS, resumeparser, resume profile/match services and candidate resume/intelligence pages;
- frontend architecture: routers, API clients, stores, request wrappers, shared components.

## 8. Compatibility Strategy

1. Preserve all existing document IDs unless a TASK explicitly supersedes a document.
2. Avoid route explosion by attaching new documents to meaningful path groups.
3. Do not change `.knowledge` schema, validation scripts, or CI workflow in this feature.
4. Keep new documents active only when claims are verified against current code.
5. Keep uncertain or policy-level claims as Inbox candidates.

## 9. Error Handling and Fallback Design

If validation fails, the TASK must report failure and either repair within scope or stop. If source references are unavailable, do not create active knowledge from memory. If a route would point to a non-existing document, create the document in the same TASK or postpone the route to the final audit.

If a TASK finds a broader scope need, such as schema changes or business behavior changes, stop and record a Hard Stop. This feature may document current code but must not alter runtime behavior.

## 10. Observability and Debug Output Design

Each TASK report must include:

- modified file list;
- document-by-document change summary;
- source references inspected;
- route/index changes;
- validation commands and exit codes;
- knowledge impact result and verdicts;
- unresolved gaps.

Final audit must summarize route coverage and follow-up candidates.

## 11. Testing Strategy

Every TASK must run or explain why unable:

```bash
git diff --name-only
bash .spec/populate-development-agent-knowledge/scripts/check-task-scope.sh <TASK-ID>
bash .spec/populate-development-agent-knowledge/scripts/agent-check.sh
```

The feature agent check must run:

```bash
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/populate-development-agent-knowledge
node .knowledge/scripts/knowledge-validator.test.mjs
node .knowledge/scripts/validate-knowledge.mjs --root .
node .knowledge/scripts/check-references.mjs --root .
git diff --check
```

TASK-specific checks may include targeted `rg`, route inspection, and knowledge impact detection with a reliable base tree.

## 12. Migration Risks

No runtime migration is performed. Documentation risks include stale claims, overbroad routes, missing source references, and active knowledge that should have been an Inbox candidate.

Mitigation: keep each document concise, source-backed, validator-clean, and scoped. Use final audit for unresolved gaps.

## 13. Implementation Boundaries

Allowed implementation files are limited to:

- `.knowledge/**` documents and manifest/index updates allowed by each TASK;
- `.spec/populate-development-agent-knowledge/**` reports and evidence.

Forbidden implementation files include all business source code, generated proto files, database migrations, package manifests, lockfiles, deployment files, CI workflows, and prior feature contracts unless explicitly listed in a TASK.

## 14. Alternatives Considered

1. Single large knowledge rewrite: rejected because it would be hard to review and easy to make stale.
2. Per-file documentation: rejected because `.knowledge` is a routing and cognition layer, not source-code commentary.
3. Only final audit without active documents: rejected because future TASKs need routable knowledge, not just a gap list.
4. Automatic generation from source: rejected because behavior and risk claims require human/Agent review against evidence.

## 15. Assumptions Requiring Confirmation

1. `populate-development-agent-knowledge` is the accepted feature name.
2. Eight TASKs are an acceptable first-round decomposition.
3. Human confirmation is required only for promoting policy-level decisions, not for source-backed descriptive active knowledge.

## 16. Open Questions

1. Should owner names be standardized beyond the current `engineering-platform`, `agent-platform`, `recruitment-domain`, and `frontend-platform` labels?
2. Should unresolved final coverage gaps be scheduled immediately through a second feature or handled opportunistically by later TASKs?
