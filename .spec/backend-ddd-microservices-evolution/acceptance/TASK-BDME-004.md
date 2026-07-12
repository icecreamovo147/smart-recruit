# Acceptance - TASK-BDME-004

## TASK Summary

Core Workflow Regression Baseline: Protect auth, recruitment, interview, offer, notification, and AI workflows with focused regression tests.

## SPEC References

- SPEC §5 Functional Requirements
- SPEC §11 Acceptance Criteria

## SDD References

- SDD §1 Existing Architecture Summary
- SDD §3.3 Migration Phases

## Acceptance Criteria

- Core workflows are mapped to tests or explicit gaps.
- New tests assert current behavior, not future behavior.
- Backend test suites pass or skipped checks have approved reasons.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-004
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- cd web-gin-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

## Manual Verification, if needed

- Manual verification is required only when automated checks cannot cover an acceptance criterion; skipped checks must be recorded with reasons.

## Out-of-Scope

- Changes outside task-scope.json allowedFiles for this TASK.
- Modifications to SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or Harness scripts.
- Frontend application changes.
- Package manifest, lockfile, go.mod, go.sum, dependency, CI/CD, or global configuration changes unless explicitly listed in scope and confirmed.
- Product behavior changes not stated in this TASK acceptance file.
