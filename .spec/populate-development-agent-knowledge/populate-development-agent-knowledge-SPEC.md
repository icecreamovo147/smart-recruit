# Populate Development Agent Knowledge SPEC

## 1. Background

Smart Recruit already has a canonical Agent development control plane through `AGENTS.md`, `spec-harness`, and feature-specific `.spec/<feature-name>/` contracts. The previous `development-agent-knowledge-base` feature created the `.knowledge/` governance layer, validation tools, route manifest, initial architecture/domain/runbook/pitfall documents, and CI validation.

The current request is the next phase: use the current project code to fill the knowledge layer so coding Agents can work with accurate project cognition. The goal is not to create a code encyclopedia. The goal is to cover the main implementation routes, service boundaries, cross-module risks, validation commands, and maintenance mechanisms that let future TASKs identify what knowledge must be read, updated, marked stale, or proposed as candidate knowledge.

## 2. Goals

1. Fill `.knowledge/` with project-specific, code-verified knowledge for high-risk and high-frequency coding areas.
2. Expand `manifest.yaml` and `INDEX.md` so major project paths route to relevant active knowledge.
3. Cover authentication/RBAC, HTTP gateway contracts, persistence and migrations, recruitment lifecycle, notification/outbox/SSE, AI configuration governance, MCP, resume intelligence, and frontend app architecture.
4. Preserve the existing `.knowledge` authority model: code, proto, database schema, tests, and active SPECs remain higher authority than ordinary knowledge.
5. Provide explicit follow-up mechanisms for uncovered areas through `coverage_gap`, Inbox candidates, stale detection, and final coverage audit reporting.
6. Keep each TASK independently reviewable and scoped to knowledge artifacts and this feature contract.
7. Avoid business behavior changes, dependency changes, schema changes, public API changes, and CI changes unless a later explicit feature authorizes them.

## 3. Non-Goals

1. Do not modify Go, Vue, Proto, database, deployment, package manifest, lockfile, or runtime configuration files.
2. Do not introduce new knowledge tooling dependencies.
3. Do not copy complete source files, full proto definitions, full database schema, secrets, production logs, or personal data into knowledge documents.
4. Do not promote unverified claims from historical specs, `.ai-guides/`, comments, or prior Agent reasoning into active knowledge.
5. Do not create a product-facing help center, runtime RAG source, or user-facing UI.
6. Do not promise 100% file-level documentation coverage.
7. Do not automatically decide new L3 policy for security, authorization, public API compatibility, or data lifecycle. Use Inbox candidates or ADRs when policy decisions are needed.

## 4. User-Facing Behavior

The direct users are coding Agents, developers, and reviewers.

1. After this feature, a TASK that touches major project areas should be routed to a small set of relevant `.knowledge` documents through `manifest.yaml`.
2. Agents should be able to understand module ownership, common cross-module impacts, key source references, and appropriate validation commands before editing.
3. If a touched area has no suitable active knowledge route, the TASK report should identify a `coverage_gap`.
4. If current code contradicts an active knowledge document, the TASK report should mark `STALE` or `CONFLICT` instead of silently relying on it.
5. If useful knowledge is discovered outside the current TASK scope or requires human approval, it should be recorded as an Inbox candidate or follow-up item.

## 5. Functional Requirements

### FR-001: Coverage Audit and Routing Plan

Create a current-code coverage audit that maps major project areas to existing and missing knowledge documents. The audit must distinguish:

- covered modules;
- partially covered modules;
- high-risk uncovered modules;
- low-risk uncovered modules;
- proposed document IDs and manifest route changes;
- items that require Inbox candidate or ADR treatment.

### FR-002: Authentication, Authorization, and Security Knowledge

Add active knowledge for the current auth and RBAC implementation, including JWT principal handling, refresh tokens, role and permission catalogs, route-level permission enforcement, data scopes, audit logging, three-frontend cookie separation, and common alignment pitfalls.

### FR-003: HTTP Gateway, Public Contract, and Persistence Knowledge

Add active knowledge for the Gin gateway, gRPC client boundary, middleware responsibilities, request limits, streaming endpoints, protobuf synchronization, database migration practice, model/repository ownership, and schema drift risks.

### FR-004: Recruitment Lifecycle Knowledge

Deepen recruitment-domain knowledge for jobs, applications, status transitions, interviews, offers, collaboration artifacts, notifications, outbox/SSE, analytics, and cross-surface impacts across HR, candidate, and interviewer flows.

### FR-005: AI Platform and Configuration Governance Knowledge

Add active knowledge for LLM providers/models, prompt templates, agent configuration, MCP servers/tool policies/logs, Skill/Agent Skill relationships, runtime capability binding, auditability, and configuration change risks.

### FR-006: Resume Intelligence and Candidate Matching Knowledge

Add active knowledge for resume upload, object storage, parse runs, structured resume profiles, heuristic/LLM extractors, candidate-job matching, evidence indexing, semantic matching, quotas, sensitive data boundaries, and debugging paths.

### FR-007: Frontend Application Architecture Knowledge

Add active knowledge for the three Vue apps, route guards, auth stores, API request wrappers, type organization, notification components, HR admin-console conventions, and per-app validation commands.

### FR-008: Final Coverage Audit and Maintenance Loop

Run the knowledge validators and produce a final audit report that records:

- final document list;
- route coverage by project area;
- unresolved coverage gaps;
- stale or candidate-required items;
- validation commands and results;
- recommended follow-up TASKs.

## 6. Non-Functional Requirements

1. Knowledge must be concise enough for Agent context use.
2. Every active knowledge document must include valid frontmatter and repository-relative `source_refs`.
3. Claims must be grounded in current repository files, tests, active specs, or generated contracts.
4. Routes must be deterministic and avoid routing broad paths to too many documents without reason.
5. Documents must use repository-relative POSIX paths only.
6. Markdown must avoid hidden instructions, secrets, personal data, and local machine paths.
7. The feature must remain compatible with current `.knowledge` validators and `spec-harness` validation.

## 7. Compatibility Requirements

1. Existing `.knowledge` documents and routes must remain valid unless a TASK explicitly updates them.
2. Existing `.spec/development-agent-knowledge-base` artifacts must not be rewritten.
3. No business code, public API, database schema, protobuf contract, CI workflow, package manifest, or lockfile may be changed.
4. Existing validation commands must continue to pass after each TASK, or failures must be recorded truthfully.
5. Historical features that do not use knowledge impact fields remain compatible.

## 8. Observability and Debug Requirements

1. Every TASK report must include a Knowledge Impact section.
2. Machine-readable evidence must include `knowledgeImpact` when the TASK changes or reviews `.knowledge`.
3. Final audit output must make unresolved gaps visible rather than hiding them in prose.
4. Validation failures must include the command, exit code, and relevant output summary.

## 9. Error Handling and Fallback Requirements

1. If code inspection shows a proposed knowledge claim is ambiguous, write the ambiguity as an assumption or candidate, not active fact.
2. If a TASK needs to modify a file outside scope, stop and request confirmation.
3. If security, authorization, public contract, or data lifecycle policy conclusions go beyond describing current code, create an Inbox candidate or proposed ADR instead of active policy.
4. If validators fail, do not mark the TASK complete without an approved exception.
5. If an area cannot be covered in the current TASK set, record it as a follow-up coverage gap in the final audit.

## 10. Security and Safety Requirements

1. Do not store secrets, credentials, keys, live tokens, personal data, raw production logs, or database snapshots.
2. Use placeholders for examples.
3. Do not include hidden prompts or Agent private reasoning.
4. Treat auth, authorization, AI provider, MCP, resume, and candidate matching knowledge as sensitive-data-adjacent and verify carefully.
5. Keep knowledge descriptive; do not create new binding policy without review.

## 11. Acceptance Criteria

1. The feature contains a valid SPEC, SDD, TASKS, AGENT_RULES, task-scope, acceptance files, prompts, scripts, and reports directory.
2. The TASK plan contains 6-8 independently executable TASKs.
3. Each TASK has explicit allowed and forbidden files.
4. The generated Harness validates with `validate-feature.mjs`.
5. The generated agent check runs existing knowledge validators.
6. The plan includes a follow-up mechanism for uncovered knowledge through coverage audit, Inbox candidates, stale detection, and final reporting.

## 12. Out of Scope

1. Implementing the knowledge documents during init.
2. Running TASK implementation, self-review, or fix modes.
3. Editing existing business code or configuration.
4. Creating a branch, commit, PR, deployment, or CI change.

## 13. Assumptions Requiring Confirmation

1. The feature name is `populate-development-agent-knowledge`.
2. First-round completion means main route coverage and explicit unresolved gaps, not exhaustive documentation of every source file.
3. New active knowledge can describe current security and authorization behavior, but new policy decisions require Inbox or ADR review.
4. The first implementation TASK should start with coverage audit before creating additional active documents.

## 14. Open Questions

1. Which human owner labels should be preferred for newly added domains if the existing owner taxonomy is too coarse?
2. Should the final audit require human sign-off before later TASKs rely on the expanded routes?
