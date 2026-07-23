# Acceptance - TASK-BDME-027

## TASK Summary

Notification Service Skeleton: Create a compile-safe Notification service binary skeleton without production traffic cutover.

## SPEC References

- SPEC §5 FR-002, FR-021..FR-025
- SPEC §11 AC-009..AC-014

## SDD References

- SDD §3.1 Target Service Architecture
- SDD §6.4 Shadow and Cutover

## Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-027
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

## Manual Verification, if needed

- Manual verification is required only when automated checks cannot cover an acceptance criterion; skipped checks must be recorded with reasons.

## Out-of-Scope

- Changes outside task-scope.json allowedFiles for this TASK.
- Modifications to SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or Harness scripts.
- Frontend application changes.
- Package manifest, lockfile, go.mod, go.sum, dependency, CI/CD, or global configuration changes unless explicitly listed in scope and confirmed.
- Product behavior changes not stated in this TASK acceptance file.
