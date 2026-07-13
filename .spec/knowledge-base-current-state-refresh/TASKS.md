# TASKS - knowledge-base-current-state-refresh

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-KBR-001 | Refresh active knowledge to current microservice state | pending | `.knowledge/**`, feature evidence | `.spec/knowledge-base-current-state-refresh/acceptance/TASK-KBR-001.md` |

## TASK-KBR-001 - Refresh active knowledge to current microservice state

### Goal

Update active knowledge routes, source references, and prose so they match the current microservice codebase.

### Scope

- Active formal knowledge under `.knowledge/architecture/`, `.knowledge/domains/`, `.knowledge/runbooks/`, and `.knowledge/pitfalls/`.
- `.knowledge/manifest.yaml`, `.knowledge/INDEX.md`, and validation script fixtures needed for current-path validation.
- Feature report and evidence files under `.spec/knowledge-base-current-state-refresh/`.

### Allowed Files

- `.knowledge/**`
- `.spec/knowledge-base-current-state-refresh/**`

### Forbidden Files

- Business code, frontend code, generated protobufs, database schemas, root docs, package manifests, lockfiles, and global runtime config.
- `.knowledge/inbox/**` and `.knowledge/archive/**` content updates, except read-only reporting.

### Dependencies

- Current source tree inspection.
- Existing `.knowledge` schema and scripts.

### Acceptance Criteria

- Active formal knowledge has no deleted `web-gin-service` or `logic-grpc-service` paths.
- Active formal `source_refs` point to existing files.
- Knowledge validation and reference checks pass.
- TASK scope and feature agent checks pass.

### Required Tests

- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `rg -n "web-gin-service|logic-grpc-service" .knowledge --glob '!archive/**' --glob '!inbox/**'`
- `git diff --name-only`
- `bash .spec/knowledge-base-current-state-refresh/scripts/check-task-scope.sh TASK-KBR-001`
- `bash .spec/knowledge-base-current-state-refresh/scripts/agent-check.sh`

### Risks

- Stale claims may survive if only paths are updated.
- Draft inbox and archive documents may still contain legacy text by design.

### Notes

Do not continue to any other TASK; this feature intentionally has exactly one TASK.
