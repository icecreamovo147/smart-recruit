# TASKS - populate-development-agent-knowledge

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | Coverage audit and routing plan | pending | Coverage report, index/manifest planning, optional navigation/runbook doc | acceptance/TASK-001.md |
| TASK-002 | Auth, RBAC, and security knowledge | pending | Auth/RBAC architecture, pitfall, runbook, route updates | acceptance/TASK-002.md |
| TASK-003 | Gateway, contracts, and persistence knowledge | pending | Gateway/API contract, migration/model/repository knowledge, route updates | acceptance/TASK-003.md |
| TASK-004 | Recruitment lifecycle knowledge | pending | Recruitment flow, collaboration, notification/outbox knowledge, route updates | acceptance/TASK-004.md |
| TASK-005 | AI platform governance knowledge | pending | LLM, prompt, agent config, MCP, Skill governance knowledge, route updates | acceptance/TASK-005.md |
| TASK-006 | Resume intelligence and matching knowledge | pending | Resume parsing, profile, matching, storage and sensitive data knowledge | acceptance/TASK-006.md |
| TASK-007 | Frontend app architecture knowledge | pending | Three frontend apps, router/auth/API/UI validation knowledge | acceptance/TASK-007.md |
| TASK-008 | Final coverage audit and validation | pending | Final audit report, route review, follow-up backlog | acceptance/TASK-008.md |

## TASK-001 - Coverage audit and routing plan

### Goal

Create a code-grounded coverage matrix for the project knowledge layer and identify the exact first-round document and route plan.

### Scope

Inspect current project structure, existing `.knowledge` documents, routes, and high-risk modules. Produce a report and, if useful, a scoped knowledge runbook/navigation document for future coverage audits.

### Allowed Files

- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.knowledge/runbooks/knowledge-coverage-audit.md`
- `.knowledge/inbox/*.md`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

None.

### Acceptance Criteria

- Coverage matrix lists covered, partially covered, high-risk uncovered, and low-risk uncovered modules.
- Proposed documents and route changes are traceable to source paths.
- Any policy-level or uncertain claims are placed in Inbox or report follow-up, not active knowledge.
- Validators pass.

### Required Tests

- `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/populate-development-agent-knowledge`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `git diff --check`
- Harness scope and agent check.

### Risks

- Overbroad route changes could make future TASKs load too much context.
- Coverage matrix may confuse "uncovered" with "must document immediately"; record priority explicitly.

### Notes

This TASK should not create most domain documents. It prepares the map.

## TASK-002 - Auth, RBAC, and security knowledge

### Goal

Add current-code verified knowledge for authentication, authorization, RBAC, data scopes, cookies, token lifecycle, audit logging, and permission alignment risks.

### Scope

Create or update auth/RBAC/security knowledge documents and route relevant auth, middleware, repository, migration, and frontend admin paths.

### Allowed Files

- `.knowledge/architecture/auth-rbac-security.md`
- `.knowledge/runbooks/debug-auth-permissions.md`
- `.knowledge/pitfalls/auth-permission-alignment.md`
- `.knowledge/inbox/*.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 completed or explicitly accepted to proceed without the coverage runbook.

### Acceptance Criteria

- Auth/RBAC knowledge cites current gateway middleware/router, logic auth service, authz packages, repositories, migrations, and relevant frontend auth files.
- Route changes cover auth and RBAC-related paths.
- Sensitive data and security policy statements remain descriptive or candidate-only.
- Validators pass.

### Required Tests

- Knowledge validators and Harness checks.
- Targeted `rg` evidence for auth, RBAC, permissions, and cookie paths recorded in the TASK report.

### Risks

- Active knowledge may accidentally become a security policy. Use Inbox for policy decisions.

### Notes

Do not change authentication, authorization, permission, or security behavior.

## TASK-003 - Gateway, contracts, and persistence knowledge

### Goal

Add knowledge for HTTP gateway contracts, gRPC clients, middleware responsibilities, protobuf synchronization, migrations, models, repositories, and schema drift risks.

### Scope

Create or update gateway/API/persistence documents and route gateway, proto, generated code, migration, model, and repository paths.

### Allowed Files

- `.knowledge/architecture/api-contracts-and-gateway.md`
- `.knowledge/architecture/persistence-and-migrations.md`
- `.knowledge/runbooks/protobuf-and-migration-change.md`
- `.knowledge/pitfalls/migration-model-drift.md`
- `.knowledge/pitfalls/protobuf-synchronization.md`
- `.knowledge/inbox/*.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001.

### Acceptance Criteria

- Gateway and persistence boundaries are described with current source references.
- Proto and migration runbooks make generated-code and schema/model consistency risks explicit.
- Routes do not conflict with existing service-boundary and proto pitfall routes.
- Validators pass.

### Required Tests

- Knowledge validators and Harness checks.
- Targeted source inspection evidence for router, rpc clients, proto services, migrations, model, and repository paths.

### Risks

- Route overlap can make impact detection noisy. Keep route additions meaningful.

### Notes

Do not modify proto, generated code, migrations, models, repositories, or gateway source files.

## TASK-004 - Recruitment lifecycle knowledge

### Goal

Deepen recruitment-domain knowledge across jobs, applications, status transitions, interviews, offers, collaboration, notifications, outbox, SSE, analytics, and frontend surfaces.

### Scope

Update existing recruitment knowledge and add focused domain/runbook/pitfall documents for lifecycle and notification risks.

### Allowed Files

- `.knowledge/domains/recruitment.md`
- `.knowledge/domains/recruitment-lifecycle.md`
- `.knowledge/domains/notification-outbox.md`
- `.knowledge/runbooks/debug-recruitment-lifecycle.md`
- `.knowledge/pitfalls/status-notification-drift.md`
- `.knowledge/inbox/*.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001.
- TASK-003 recommended for persistence/proto context.

### Acceptance Criteria

- Lifecycle knowledge identifies cross-surface impacts for HR, candidate, and interviewer workflows.
- Notification/outbox/SSE behavior is described at a boundary level with source references.
- Status, analytics, and collaboration risks are routed to relevant knowledge.
- Validators pass.

### Required Tests

- Knowledge validators and Harness checks.
- Targeted source inspection evidence for job/application/interview/offer/collaboration/notification services and frontend views.

### Risks

- Business state-machine details can drift. Keep source references strong and avoid duplicating full status tables.

### Notes

Do not change recruitment behavior.

## TASK-005 - AI platform governance knowledge

### Goal

Add knowledge for LLM providers/models, prompt templates, agent configs, MCP servers/tool policies/logs, Skill/Agent Skill relationships, runtime capability binding, and AI auditability.

### Scope

Create or update AI platform governance documents and route AI configuration, MCP, prompt, skill, provider, embedding, and HR admin pages.

### Allowed Files

- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/architecture/semantic-retrieval.md`
- `.knowledge/domains/agent-skill.md`
- `.knowledge/domains/ai-configuration-governance.md`
- `.knowledge/domains/mcp-tool-governance.md`
- `.knowledge/runbooks/debug-ai-configuration.md`
- `.knowledge/pitfalls/mcp-policy-audit.md`
- `.knowledge/inbox/*.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001.
- TASK-002 recommended for permission context.

### Acceptance Criteria

- AI governance knowledge distinguishes runtime behavior from admin configuration.
- MCP tool policy and audit concerns are explicit and source-backed.
- Existing Agent runtime, semantic retrieval, and Agent Skill docs remain consistent.
- Validators pass.

### Required Tests

- Knowledge validators and Harness checks.
- Targeted source inspection evidence for LLM, prompt, agent config, MCP, Skill, Agent Skill, embedding config, and HR admin views.

### Risks

- AI provider and MCP details may include sensitive configuration. Use placeholders only.

### Notes

Do not change AI runtime, provider, MCP, prompt, or Skill behavior.

## TASK-006 - Resume intelligence and matching knowledge

### Goal

Add knowledge for resume upload/storage, parse runs, structured resume profiles, extractors, candidate matching, evidence indexing, semantic match, quota, and sensitive data boundaries.

### Scope

Create domain/runbook/pitfall documents and route resume, OSS, parser, matching, recruiting intelligence, candidate resume, and HR intelligence paths.

### Allowed Files

- `.knowledge/domains/resume-intelligence.md`
- `.knowledge/runbooks/debug-resume-intelligence.md`
- `.knowledge/pitfalls/resume-sensitive-data.md`
- `.knowledge/inbox/*.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001.
- TASK-005 recommended for AI/embedding context.

### Acceptance Criteria

- Resume intelligence knowledge covers upload, parsing, profile, match, evidence, and quota boundaries.
- Sensitive data handling is descriptive and avoids examples containing personal data.
- Routes cover relevant backend and frontend paths.
- Validators pass.

### Required Tests

- Knowledge validators and Harness checks.
- Targeted source inspection evidence for OSS, resumeparser, resume profile, candidate match, recruiting intelligence, and related frontend views.

### Risks

- Resume examples can accidentally include personal data. Use generic placeholders.

### Notes

Do not change resume, storage, parsing, or matching behavior.

## TASK-007 - Frontend app architecture knowledge

### Goal

Add knowledge for the three Vue apps, route guards, auth stores, request wrappers, API/type organization, notification components, admin-console layout, and validation commands.

### Scope

Create frontend architecture and validation knowledge, update HR admin menu pitfall if needed, and route frontend paths more precisely.

### Allowed Files

- `.knowledge/architecture/frontend-apps.md`
- `.knowledge/runbooks/frontend-validation.md`
- `.knowledge/pitfalls/frontend-menu-consistency.md`
- `.knowledge/inbox/*.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001.
- TASK-002 recommended for auth route guard context.

### Acceptance Criteria

- Frontend knowledge covers all three apps without turning into a page-by-page inventory.
- Route guard, auth, request, notification, and validation patterns are source-backed.
- HR admin menu consistency pitfall remains aligned with `AGENTS.md`.
- Validators pass.

### Required Tests

- Knowledge validators and Harness checks.
- Targeted source inspection evidence for frontend routers, API clients, stores, request wrappers, and major view groups.

### Risks

- Frontend docs can become stale if they list every page. Prefer patterns and representative anchors.

### Notes

Do not change frontend source code.

## TASK-008 - Final coverage audit and validation

### Goal

Run final validation, inspect the complete knowledge route coverage, and produce a follow-up backlog for uncovered or candidate-required areas.

### Scope

Do a read-only review of project and knowledge artifacts. Update this feature's reports and optionally perform final narrow `INDEX.md`/`manifest.yaml` corrections if validation exposes a mechanical issue.

### Allowed Files

- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.knowledge/inbox/*.md`
- `.spec/populate-development-agent-knowledge/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.spec/development-agent-knowledge-base/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 through TASK-007.

### Acceptance Criteria

- Final audit lists all active documents added or updated by this feature.
- Final audit maps major project areas to routed knowledge.
- Final audit records unresolved coverage gaps and candidate-required items.
- Knowledge validation, reference checks, Harness validation, and `git diff --check` pass or failures are truthfully reported.

### Required Tests

- `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/populate-development-agent-knowledge`
- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `git diff --check`
- Harness scope and agent check.

### Risks

- Final audit may reveal gaps too large for this feature. Record follow-up instead of expanding scope.

### Notes

Do not treat final audit as permission to create additional broad documents outside earlier scopes.
