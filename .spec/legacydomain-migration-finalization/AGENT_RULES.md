# AGENT RULES - legacydomain-migration-finalization

## Authority

Follow, in order:

1. `AGENTS.md`
2. `.agents/skills/spec-harness/SKILL.md`
3. `.spec/legacydomain-migration-finalization/*`
4. Current source code, tests, schema, protobuf, and active `.knowledge` documents

## Feature-Specific Rules

- Execute exactly one TASK at a time.
- Do not implement behavior that is not described in SPEC, SDD, TASKS, and the TASK acceptance file.
- Do not reintroduce `internal/legacydomain`.
- Do not move GORM record structs into `internal/domain`.
- Do not change database schema or public protobuf contracts.
- Do not change Gateway route paths, frontend API shapes, or public RBAC names.
- Do not return `Code: 0` / `success` for unsupported, unimplemented, or unconfigured runtime behavior.
- Do not hide provider, MCP, embedding, or persistence errors; return safe explicit errors.
- Redact secrets in all responses, logs, reports, and tests.
- Update or report `.knowledge` impact for every TASK.

## Scope Boundaries

- Recruitment work is limited to `smart-recruit-recruitment-service/**`, feature docs, guard scripts, and routed knowledge.
- AI Agent work is limited to `smart-recruit-ai-agent-service/**`, feature docs, guard scripts, and routed knowledge.
- Shared scripts may be changed only for boundary and ownership checks explicitly listed in the TASK.
- Shared modules, protobuf, migrations, root docs, package manifests, and lockfiles are forbidden unless a TASK records a Hard Stop and the user explicitly confirms widening scope.

## Required Checks

Each implementation TASK must run or explain why it cannot run:

- `git diff --name-only`
- `bash .spec/legacydomain-migration-finalization/scripts/check-task-scope.sh <TASK-ID>`
- `bash .spec/legacydomain-migration-finalization/scripts/agent-check.sh`
- TASK-specific `go test ./...` commands
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`

## Report Requirements

Every TASK report must include:

- TASK ID
- modified file list
- change summary by file
- scope check result
- SPEC/SDD/acceptance comparison
- test commands and results
- `knowledge_impact`
- risks
- whether the next TASK can start

## Hard Stop Conditions

Stop and ask for confirmation if implementation requires:

- modifying `smart-recruit-proto/**`;
- modifying migrations, `db.sql`, or table ownership for schema changes;
- modifying frontend behavior or Gateway public route shapes;
- adding dependencies;
- changing auth/RBAC semantics;
- expanding beyond Recruitment, AI Agent, guard scripts, feature docs, and routed knowledge;
- accepting an empty-success runtime stub as final behavior.
