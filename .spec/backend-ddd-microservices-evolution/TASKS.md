# TASKS - backend-ddd-microservices-evolution

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-BDME-001 | Architecture Baseline Inventory | pending | Reports and knowledge only; no business code changes. | acceptance/TASK-BDME-001.md |
| TASK-BDME-002 | Secret And Config Safety Baseline | pending | Config examples, secret policy docs, and production fail-fast checks. | acceptance/TASK-BDME-002.md |
| TASK-BDME-003 | HTTP And GRPC Contract Baseline | pending | Gateway route/auth/RBAC/proto contract tests and documentation. | acceptance/TASK-BDME-003.md |
| TASK-BDME-004 | Core Workflow Regression Baseline | pending | Backend regression tests for current core workflows. | acceptance/TASK-BDME-004.md |
| TASK-BDME-005 | Harness And Knowledge Impact Baseline | pending | Harness validation and knowledge impact baseline only. | acceptance/TASK-BDME-005.md |
| TASK-BDME-006 | Observability Baseline Design | pending | Observability design and runbooks. | acceptance/TASK-BDME-006.md |
| TASK-BDME-007 | Deployment And Readiness Baseline | pending | Deployment/readiness docs and minimal manifest baseline. | acceptance/TASK-BDME-007.md |
| TASK-BDME-008 | Bounded Context Architecture Contract | pending | DDD ownership matrix, dependency direction, and exception policy. | acceptance/TASK-BDME-008.md |
| TASK-BDME-009 | DDD Module Skeleton | pending | Internal DDD package skeletons and tests only. | acceptance/TASK-BDME-009.md |
| TASK-BDME-010 | Identity Boundary Modularization | pending | Identity modularization inside current backend. | acceptance/TASK-BDME-010.md |
| TASK-BDME-011 | Recruitment Job Boundary Modularization | pending | Recruitment job modularization. | acceptance/TASK-BDME-011.md |
| TASK-BDME-012 | Recruitment Candidate Resume Application Boundary | pending | Recruitment candidate/resume/application modularization. | acceptance/TASK-BDME-012.md |
| TASK-BDME-013 | Interview Boundary Modularization | pending | Interview modularization. | acceptance/TASK-BDME-013.md |
| TASK-BDME-014 | Offer Boundary Modularization | pending | Offer modularization. | acceptance/TASK-BDME-014.md |
| TASK-BDME-015 | Notification Boundary Modularization | pending | Notification modularization without extraction. | acceptance/TASK-BDME-015.md |
| TASK-BDME-016 | AI Agent Boundary Modularization | pending | AI Agent modularization without extraction. | acceptance/TASK-BDME-016.md |
| TASK-BDME-017 | Analytics Projection Boundary | pending | Analytics boundary with no transitional service read APIs. | acceptance/TASK-BDME-017.md |
| TASK-BDME-018 | Boundary Enforcement Checks | pending | Boundary enforcement scripts/tests. | acceptance/TASK-BDME-018.md |
| TASK-BDME-019 | Domain Event Envelope Contract | pending | Event envelope metadata/types/tests. | acceptance/TASK-BDME-019.md |
| TASK-BDME-020 | Outbox Standardization | pending | Outbox model/repository/publisher/migration standardization. | acceptance/TASK-BDME-020.md |
| TASK-BDME-021 | Inbox Idempotency Foundation | pending | Consumer idempotency storage and tests. | acceptance/TASK-BDME-021.md |
| TASK-BDME-022 | Notification Event Conversion | pending | Notification event producers/consumers. | acceptance/TASK-BDME-022.md |
| TASK-BDME-023 | Interview Offer Recruitment Event Decoupling | pending | Core lifecycle event decoupling. | acceptance/TASK-BDME-023.md |
| TASK-BDME-024 | Analytics Projection Event Ingestion | pending | Analytics projection consumers/read models. | acceptance/TASK-BDME-024.md |
| TASK-BDME-025 | Event Replay And Dead Letter Runbooks | pending | Event replay/DLQ runbooks and dry-run scripts. | acceptance/TASK-BDME-025.md |
| TASK-BDME-026 | Service Binary And Deployment Convention | pending | Service cmd/config/deployment convention without routing traffic. | acceptance/TASK-BDME-026.md |
| TASK-BDME-027 | Notification Service Skeleton | pending | Notification service skeleton. | acceptance/TASK-BDME-027.md |
| TASK-BDME-028 | Notification Service Runtime Extraction | pending | Notification runtime extraction and validation. | acceptance/TASK-BDME-028.md |
| TASK-BDME-029 | Notification Gateway Cutover | pending | Gateway notification route cutover only. | acceptance/TASK-BDME-029.md |
| TASK-BDME-030 | AI Agent Service Skeleton | pending | AI Agent service skeleton. | acceptance/TASK-BDME-030.md |
| TASK-BDME-031 | AI Agent Runtime Extraction | pending | AI runtime extraction. | acceptance/TASK-BDME-031.md |
| TASK-BDME-032 | AI Agent Knowledge Runtime Extraction | pending | AI knowledge/embedding/MCP extraction. | acceptance/TASK-BDME-032.md |
| TASK-BDME-033 | AI Agent Gateway Cutover | pending | Gateway AI route cutover only. | acceptance/TASK-BDME-033.md |
| TASK-BDME-034 | Identity Service Skeleton | pending | Identity service skeleton. | acceptance/TASK-BDME-034.md |
| TASK-BDME-035 | Identity Auth RBAC API Extraction | pending | Identity API extraction without gateway cutover. | acceptance/TASK-BDME-035.md |
| TASK-BDME-036 | Identity Gateway Cutover | pending | Gateway auth/RBAC cutover. | acceptance/TASK-BDME-036.md |
| TASK-BDME-037 | Recruitment Service Skeleton | pending | Recruitment service skeleton. | acceptance/TASK-BDME-037.md |
| TASK-BDME-038 | Recruitment Service API Extraction | pending | Recruitment runtime/API extraction without gateway cutover. | acceptance/TASK-BDME-038.md |
| TASK-BDME-039 | Recruitment Gateway Cutover | pending | Gateway recruitment route cutover. | acceptance/TASK-BDME-039.md |
| TASK-BDME-040 | Interview Service Skeleton And API Extraction | pending | Interview service runtime/API extraction without gateway cutover. | acceptance/TASK-BDME-040.md |
| TASK-BDME-041 | Interview Gateway Cutover | pending | Gateway interview route cutover. | acceptance/TASK-BDME-041.md |
| TASK-BDME-042 | Offer Service Skeleton And API Extraction | pending | Offer service runtime/API extraction without gateway cutover. | acceptance/TASK-BDME-042.md |
| TASK-BDME-043 | Offer Gateway Cutover | pending | Gateway offer route cutover. | acceptance/TASK-BDME-043.md |
| TASK-BDME-044 | Analytics Service Extraction | pending | Analytics service extraction. | acceptance/TASK-BDME-044.md |
| TASK-BDME-045 | Worker Service Decomposition | pending | Worker runtime decomposition. | acceptance/TASK-BDME-045.md |
| TASK-BDME-046 | Table Ownership Manifest | pending | Table ownership manifest and drift audit. | acceptance/TASK-BDME-046.md |
| TASK-BDME-047 | Schema Separation Plan | pending | Schema separation plan and approved migration scaffolding. | acceptance/TASK-BDME-047.md |
| TASK-BDME-048 | Internal Service Security Hardening | pending | Internal TLS/token hardening and mTLS evaluation. | acceptance/TASK-BDME-048.md |
| TASK-BDME-049 | Metrics And Tracing Implementation | pending | Telemetry implementation across services/events/workers. | acceptance/TASK-BDME-049.md |
| TASK-BDME-050 | Readiness Worker Health And Dependency Degradation | pending | Readiness and soft-dependency degradation implementation. | acceptance/TASK-BDME-050.md |
| TASK-BDME-051 | Load Test Harness And Initial Targets | pending | Load-test harness, scripts, and evidence. | acceptance/TASK-BDME-051.md |
| TASK-BDME-052 | Final Readiness Cleanup And Architecture Review | pending | Final readiness report, exception register, docs, and knowledge updates. | acceptance/TASK-BDME-052.md |

## TASK-BDME-001 - Architecture Baseline Inventory

### Goal

Inventory current backend architecture, workflow ownership, runtime dependencies, and migration risk map before any code movement.

### Scope

Reports and knowledge only; no business code changes.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-001*
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Current gateway, logic service, workers, MySQL, Redis, RabbitMQ, OSS, SMTP, and AI dependencies are mapped.
- Bounded-context candidates are mapped to current packages, models, repositories, handlers, and workers.
- Cross-domain access risks and startup side effects are listed with source locations or explicit not-found notes.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-001
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-002 - Secret And Config Safety Baseline

### Goal

Establish committed-config safety and production fail-fast rules for secrets and internal auth.

### Scope

Config examples, secret policy docs, and production fail-fast checks.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-002*
- .gitignore
- logic-grpc-service/config/**
- web-gin-service/config/**
- deploy/**
- docker/.env.example
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Tracked config examples contain placeholders only.
- Production profiles fail fast when required secrets and internal auth are missing.
- Local examples remain usable and no live local env file is added or modified.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-002
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-003 - HTTP And GRPC Contract Baseline

### Goal

Capture current public HTTP and gateway-to-backend gRPC behavior before service extraction.

### Scope

Gateway route/auth/RBAC/proto contract tests and documentation.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-003*
- web-gin-service/**
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Core route groups and auth/RBAC requirements are covered by tests or documented gaps.
- Proto source and generated code synchronization is validated or documented.
- No public HTTP behavior or proto contract change is introduced without confirmation.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-003
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-004 - Core Workflow Regression Baseline

### Goal

Protect auth, recruitment, interview, offer, notification, and AI workflows with focused regression tests.

### Scope

Backend regression tests for current core workflows.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-004*
- logic-grpc-service/**
- web-gin-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Core workflows are mapped to tests or explicit gaps.
- New tests assert current behavior, not future behavior.
- Backend test suites pass or skipped checks have approved reasons.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-004
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-005 - Harness And Knowledge Impact Baseline

### Goal

Verify this feature Harness, knowledge impact routing, and reporting conventions before implementation tasks begin.

### Scope

Harness validation and knowledge impact baseline only.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-005*
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Feature validates as schemaVersion 1 current Harness.
- Knowledge routes relevant to backend architecture, service boundaries, persistence, auth, API contracts, and operations are identified.
- TASK report expectations include knowledge impact result and coverage gaps.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-005
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed
- node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/backend-ddd-microservices-evolution --json

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-006 - Observability Baseline Design

### Goal

Define the telemetry contract for metrics, traces, logs, and cutover diagnostics.

### Scope

Observability design and runbooks.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-006*
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Telemetry conventions cover HTTP, gRPC, DB, Redis, RabbitMQ, workers, Outbox/Inbox, and AI paths.
- Trace/request id propagation expectations cover gateway, gRPC, events, and workers.
- Redaction requirements cover secrets, tokens, prompts, and unnecessary candidate data.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-006
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-007 - Deployment And Readiness Baseline

### Goal

Align current deployment, probes, local scripts, and readiness assumptions with the staged migration plan.

### Scope

Deployment/readiness docs and minimal manifest baseline.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-007*
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Current gateway, logic, and worker deployment/readiness behavior is documented.
- Hard and soft dependencies are identified.
- Local development workflow remains usable or has equivalent documentation.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-007
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-008 - Bounded Context Architecture Contract

### Goal

Create the enforceable bounded-context contract for target domains and workers.

### Scope

DDD ownership matrix, dependency direction, and exception policy.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-008*
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Each context has explicit owned aggregates, data, interfaces, and forbidden ownership.
- Recruitment owns candidate/resume/application profile data; AI Agent owns AI-derived intelligence.
- Analytics final source is domain-event projections/read models, not transitional service read APIs.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-008
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-009 - DDD Module Skeleton

### Goal

Introduce compile-safe bounded-context module skeletons without moving behavior.

### Scope

Internal DDD package skeletons and tests only.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-009*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- None

### Acceptance Criteria

- Skeletons exist for identity, recruitment, interview, offer, notification, aiagent, analytics, and platform/workers where appropriate.
- Code compiles with no runtime behavior changes.
- Each context documents domain, application, infrastructure, and interfaces responsibilities.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-009
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-010 - Identity Boundary Modularization

### Goal

Move auth, users, tokens, RBAC, scopes, and audit logic behind Identity-owned boundaries.

### Scope

Identity modularization inside current backend.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-010*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- Identity-owned use cases and adapters are separated from non-Identity domain logic.
- Auth/RBAC regression tests remain compatible.
- Other contexts do not own token invalidation, scopes, or permission decisions.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-010
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-011 - Recruitment Job Boundary Modularization

### Goal

Move job and HR recruitment use cases behind Recruitment-owned boundaries.

### Scope

Recruitment job modularization.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-011*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-011
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-012 - Recruitment Candidate Resume Application Boundary

### Goal

Move candidate profile, resume, and application lifecycle ownership into Recruitment.

### Scope

Recruitment candidate/resume/application modularization.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-012*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- Candidate recruitment profile, resumes, and applications are Recruitment-owned.
- AI-derived profiles, embeddings, matching artifacts, memory, and intelligence remain AI Agent-owned.
- Application lifecycle tests preserve current visible behavior.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-012
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-013 - Interview Boundary Modularization

### Goal

Move interview schedules, tasks, feedback, and lifecycle state into Interview boundaries.

### Scope

Interview modularization.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-013*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-013
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-014 - Offer Boundary Modularization

### Goal

Move offer lifecycle, offer events, and candidate offer decisions into Offer boundaries.

### Scope

Offer modularization.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-014*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-014
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-015 - Notification Boundary Modularization

### Goal

Move notification persistence, unread counts, email coordination, and realtime delivery behind Notification boundaries.

### Scope

Notification modularization without extraction.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-015*
- logic-grpc-service/**
- web-gin-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-015
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-016 - AI Agent Boundary Modularization

### Goal

Move chat sessions, agent runs, prompt/skill/memory, embeddings, MCP governance, provider fallback, and AI audit into AI Agent boundaries.

### Scope

AI Agent modularization without extraction.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-016*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-016
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-017 - Analytics Projection Boundary

### Goal

Create the Analytics bounded-context boundary around event-projection read models and reporting APIs.

### Scope

Analytics boundary with no transitional service read APIs.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-017*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- **/analytics/*service_read*
- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-009

### Acceptance Criteria

- Analytics boundary owns reporting read models and query APIs only.
- Analytics does not mutate transactional domain state.
- No transitional service read API dependency is introduced.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-017
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-018 - Boundary Enforcement Checks

### Goal

Add automated or scripted checks for forbidden cross-domain imports, repository access, and ownership drift.

### Scope

Boundary enforcement scripts/tests.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-018*
- scripts/**
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-008
- TASK-BDME-009

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-018
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-019 - Domain Event Envelope Contract

### Goal

Define and implement the standard domain-event envelope shared by Outbox, Inbox, consumers, and Analytics projections.

### Scope

Event envelope metadata/types/tests.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-019*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-008

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-019
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-020 - Outbox Standardization

### Goal

Standardize Outbox writes, publish states, retry metadata, retention, and observability.

### Scope

Outbox model/repository/publisher/migration standardization.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-020*
- logic-grpc-service/**
- logic-grpc-service/migrations/**
- logic-grpc-service/migration/**
- logic-grpc-service/model/**
- logic-grpc-service/repository/**
- db.sql
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-019

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-020
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-021 - Inbox Idempotency Foundation

### Goal

Introduce Inbox or equivalent idempotency controls for event consumers.

### Scope

Consumer idempotency storage and tests.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-021*
- logic-grpc-service/**
- logic-grpc-service/migrations/**
- logic-grpc-service/migration/**
- logic-grpc-service/model/**
- logic-grpc-service/repository/**
- db.sql
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-019
- TASK-BDME-020

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-021
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-022 - Notification Event Conversion

### Goal

Convert notification-producing cross-domain writes to domain events and idempotent Notification consumers.

### Scope

Notification event producers/consumers.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-022*
- logic-grpc-service/**
- web-gin-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-015
- TASK-BDME-019
- TASK-BDME-020
- TASK-BDME-021

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-022
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-023 - Interview Offer Recruitment Event Decoupling

### Goal

Replace hidden cross-domain state writes among Recruitment, Interview, and Offer with events, adapters, or process-manager logic.

### Scope

Core lifecycle event decoupling.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-023*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-011
- TASK-BDME-012
- TASK-BDME-013
- TASK-BDME-014
- TASK-BDME-019
- TASK-BDME-020
- TASK-BDME-021

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-023
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-024 - Analytics Projection Event Ingestion

### Goal

Build Analytics ingestion directly on domain-event projections/read models.

### Scope

Analytics projection consumers/read models.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-024*
- logic-grpc-service/**
- logic-grpc-service/migrations/**
- logic-grpc-service/migration/**
- logic-grpc-service/model/**
- logic-grpc-service/repository/**
- db.sql
- docs/**
- .knowledge/**

### Forbidden Files

- **/analytics/*service_read*
- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-017
- TASK-BDME-019
- TASK-BDME-020
- TASK-BDME-021

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-024
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-025 - Event Replay And Dead Letter Runbooks

### Goal

Document and script safe replay, repair, dead-letter inspection, and retention workflows.

### Scope

Event replay/DLQ runbooks and dry-run scripts.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-025*
- scripts/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-019
- TASK-BDME-020
- TASK-BDME-021

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-025
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-026 - Service Binary And Deployment Convention

### Goal

Establish the convention for independently deployable service binaries inside the monorepo.

### Scope

Service cmd/config/deployment convention without routing traffic.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-026*
- logic-grpc-service/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-007
- TASK-BDME-008

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-026
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-027 - Notification Service Skeleton

### Goal

Create a compile-safe Notification service binary skeleton without production traffic cutover.

### Scope

Notification service skeleton.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-027*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-015
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-027
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-028 - Notification Service Runtime Extraction

### Goal

Run Notification persistence, unread counts, email coordination, realtime delivery, and consumers from the Notification service runtime.

### Scope

Notification runtime extraction and validation.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-028*
- logic-grpc-service/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-022
- TASK-BDME-027

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-028
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-029 - Notification Gateway Cutover

### Goal

Route gateway notification APIs and realtime delivery to the extracted Notification service with rollback controls.

### Scope

Gateway notification route cutover only.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-029*
- web-gin-service/**
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-028

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-029
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-030 - AI Agent Service Skeleton

### Goal

Create a compile-safe AI Agent service binary skeleton without production traffic cutover.

### Scope

AI Agent service skeleton.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-030*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-016
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-030
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-031 - AI Agent Runtime Extraction

### Goal

Move AI chat, agent runs, provider fallback, and async AI workload execution into AI Agent service runtime.

### Scope

AI runtime extraction.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-031*
- logic-grpc-service/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-030

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-031
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-032 - AI Agent Knowledge Runtime Extraction

### Goal

Extract AI prompt, skill, memory, embedding, MCP governance, and AI-derived intelligence ownership into AI Agent runtime boundaries.

### Scope

AI knowledge/embedding/MCP extraction.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-032*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-031

### Acceptance Criteria

- AI-derived profiles, matching artifacts, embeddings, memory, and intelligence are AI Agent-owned.
- Recruitment-owned source data is accessed through explicit APIs/events/adapters.
- Sensitive resume or prompt data is redacted in logs/events/traces.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-032
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-033 - AI Agent Gateway Cutover

### Goal

Route gateway AI endpoints to the extracted AI Agent service with controlled rollout and rollback evidence.

### Scope

Gateway AI route cutover only.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-033*
- web-gin-service/**
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-031
- TASK-BDME-032

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-033
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-034 - Identity Service Skeleton

### Goal

Create a compile-safe Identity service binary skeleton without production auth traffic cutover.

### Scope

Identity service skeleton.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-034*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-010
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-034
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-035 - Identity Auth RBAC API Extraction

### Goal

Extract Identity auth, token invalidation, RBAC, scopes, and audit APIs into Identity service runtime.

### Scope

Identity API extraction without gateway cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-035*
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-034

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-035
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-036 - Identity Gateway Cutover

### Goal

Route gateway authentication and authorization integration to Identity service with rollback controls.

### Scope

Gateway auth/RBAC cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-036*
- web-gin-service/**
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-035

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-036
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-037 - Recruitment Service Skeleton

### Goal

Create a compile-safe Recruitment service binary skeleton without production recruitment traffic cutover.

### Scope

Recruitment service skeleton.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-037*
- logic-grpc-service/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-011
- TASK-BDME-012
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-037
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-038 - Recruitment Service API Extraction

### Goal

Extract Recruitment jobs, candidate recruitment profile, resumes, applications, and lifecycle APIs into Recruitment service runtime.

### Scope

Recruitment runtime/API extraction without gateway cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-038*
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-023
- TASK-BDME-037

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-038
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-039 - Recruitment Gateway Cutover

### Goal

Route gateway recruitment APIs to Recruitment service with controlled rollout and rollback evidence.

### Scope

Gateway recruitment route cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-039*
- web-gin-service/**
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-038

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-039
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-040 - Interview Service Skeleton And API Extraction

### Goal

Create and extract Interview service runtime for schedules, interviewer tasks, feedback, and lifecycle APIs.

### Scope

Interview service runtime/API extraction without gateway cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-040*
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-013
- TASK-BDME-023
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-040
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-041 - Interview Gateway Cutover

### Goal

Route gateway interview APIs to Interview service with controlled rollout and rollback evidence.

### Scope

Gateway interview route cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-041*
- web-gin-service/**
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-040

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-041
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-042 - Offer Service Skeleton And API Extraction

### Goal

Create and extract Offer service runtime for offer lifecycle, offer events, and candidate offer decisions.

### Scope

Offer service runtime/API extraction without gateway cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-042*
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-014
- TASK-BDME-023
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-042
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-043 - Offer Gateway Cutover

### Goal

Route gateway offer APIs to Offer service with controlled rollout and rollback evidence.

### Scope

Gateway offer route cutover.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-043*
- web-gin-service/**
- logic-grpc-service/**
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-042

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-043
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-044 - Analytics Service Extraction

### Goal

Extract Analytics service runtime backed directly by event projections/read models, not transitional transactional service read APIs.

### Scope

Analytics service extraction.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-044*
- logic-grpc-service/**
- web-gin-service/**
- logic-grpc-service/migrations/**
- logic-grpc-service/migration/**
- logic-grpc-service/model/**
- logic-grpc-service/repository/**
- db.sql
- logic-grpc-service/proto/**
- logic-grpc-service/recruitment/pb/**
- web-gin-service/proto/**
- web-gin-service/recruitment/pb/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- **/analytics/*service_read*
- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-017
- TASK-BDME-024
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-044
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-045 - Worker Service Decomposition

### Goal

Separate async worker workloads from request-serving workloads with independently scalable worker service binaries.

### Scope

Worker runtime decomposition.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-045*
- logic-grpc-service/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-026
- TASK-BDME-028
- TASK-BDME-031
- TASK-BDME-044

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-045
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-046 - Table Ownership Manifest

### Goal

Create and enforce a table ownership manifest for every existing table and target service context.

### Scope

Table ownership manifest and drift audit.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-046*
- scripts/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-018

### Acceptance Criteria

- Every current table is assigned an owner or justified shared platform owner.
- Allowed readers/writers are explicit per table.
- Every transitional shared DB access has owner, reason, risk, and removal plan.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-046
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-047 - Schema Separation Plan

### Goal

Prepare schema or physical database separation after service boundaries and table ownership are proven.

### Scope

Schema separation plan and approved migration scaffolding.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-047*
- logic-grpc-service/**
- logic-grpc-service/migrations/**
- logic-grpc-service/migration/**
- logic-grpc-service/model/**
- logic-grpc-service/repository/**
- db.sql
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-046

### Acceptance Criteria

- Schema/database separation follows the ownership manifest.
- Expand-contract, rollback, reconciliation, RTO, and RPO considerations are documented.
- Any migration/model/db.sql change is scoped and aligned.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-047
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-048 - Internal Service Security Hardening

### Goal

Harden service-to-service authentication and transport with internal TLS plus required GRPC_INTERNAL_TOKEN.

### Scope

Internal TLS/token hardening and mTLS evaluation.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-048*
- logic-grpc-service/**
- web-gin-service/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-002
- TASK-BDME-026

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-048
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-049 - Metrics And Tracing Implementation

### Goal

Implement Prometheus-compatible metrics, OpenTelemetry-compatible tracing, and structured log propagation.

### Scope

Telemetry implementation across services/events/workers.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-049*
- logic-grpc-service/**
- web-gin-service/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-006
- TASK-BDME-019

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-049
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: false

## TASK-BDME-050 - Readiness Worker Health And Dependency Degradation

### Goal

Implement true liveness/readiness and dependency degradation behavior for services and workers.

### Scope

Readiness and soft-dependency degradation implementation.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-050*
- logic-grpc-service/**
- web-gin-service/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-045
- TASK-BDME-049

### Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-050
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-051 - Load Test Harness And Initial Targets

### Goal

Create and run load-test evidence for the confirmed initial concurrency, latency, and async AI targets.

### Scope

Load-test harness, scripts, and evidence.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-051*
- scripts/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-049
- TASK-BDME-050

### Acceptance Criteria

- Harness can exercise 200 QPS gateway APIs, 50 QPS core writes, and 10-20 concurrent AI/Embedding tasks or document environment limits.
- Ordinary API P95 below 300 ms and complex query P95 below 1 s are measured where possible.
- AI work is validated as asynchronous and not blocking main transactions.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-051
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true

## TASK-BDME-052 - Final Readiness Cleanup And Architecture Review

### Goal

Complete final architecture readiness review and remove or formally approve remaining transitional adapters, direct access, and operational gaps.

### Scope

Final readiness report, exception register, docs, and knowledge updates.

### Allowed Files

- .spec/backend-ddd-microservices-evolution/pipeline-state.json
- .spec/backend-ddd-microservices-evolution/reports/TASK-BDME-052*
- scripts/**
- deploy/**
- docker/**
- docs/**
- .knowledge/**

### Forbidden Files

- package.json
- pnpm-lock.yaml
- pnpm-workspace.yaml
- **/package.json
- **/go.mod
- **/go.sum
- **/.env
- **/.env.local
- **/.env.development.local
- **/.env.production.local
- hr-frontend/**
- user-frontend/**
- interviewer-frontend/**
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md
- .spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md
- .spec/backend-ddd-microservices-evolution/TASKS.md
- .spec/backend-ddd-microservices-evolution/AGENT_RULES.md
- .spec/backend-ddd-microservices-evolution/task-scope.json
- .spec/backend-ddd-microservices-evolution/acceptance/**
- .spec/backend-ddd-microservices-evolution/prompts/**
- .spec/backend-ddd-microservices-evolution/scripts/**

### Dependencies

- TASK-BDME-001
- TASK-BDME-002
- TASK-BDME-003
- TASK-BDME-004
- TASK-BDME-005
- TASK-BDME-006
- TASK-BDME-007
- TASK-BDME-008
- TASK-BDME-009
- TASK-BDME-010
- TASK-BDME-011
- TASK-BDME-012
- TASK-BDME-013
- TASK-BDME-014
- TASK-BDME-015
- TASK-BDME-016
- TASK-BDME-017
- TASK-BDME-018
- TASK-BDME-019
- TASK-BDME-020
- TASK-BDME-021
- TASK-BDME-022
- TASK-BDME-023
- TASK-BDME-024
- TASK-BDME-025
- TASK-BDME-026
- TASK-BDME-027
- TASK-BDME-028
- TASK-BDME-029
- TASK-BDME-030
- TASK-BDME-031
- TASK-BDME-032
- TASK-BDME-033
- TASK-BDME-034
- TASK-BDME-035
- TASK-BDME-036
- TASK-BDME-037
- TASK-BDME-038
- TASK-BDME-039
- TASK-BDME-040
- TASK-BDME-041
- TASK-BDME-042
- TASK-BDME-043
- TASK-BDME-044
- TASK-BDME-045
- TASK-BDME-046
- TASK-BDME-047
- TASK-BDME-048
- TASK-BDME-049
- TASK-BDME-050
- TASK-BDME-051

### Acceptance Criteria

- Final review proves target service boundaries and DDD ownership are implemented or exceptions are approved with removal plans.
- No forbidden cross-domain repository imports, table writes, or state ownership violations remain outside approved exceptions.
- Security, observability, readiness, load, HA, rollback, and knowledge artifacts are current.

### Required Tests

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-052
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

### Risks

- Architecture migration risk must be recorded with rollback or residual-risk notes.

### Notes

- requiresHumanConfirmation: true
