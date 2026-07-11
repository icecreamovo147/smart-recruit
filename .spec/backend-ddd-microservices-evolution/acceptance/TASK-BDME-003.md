# Acceptance - TASK-BDME-003

## TASK Summary

HTTP And GRPC Contract Baseline: Capture current public HTTP and gateway-to-backend gRPC behavior before service extraction.

## SPEC References

- SPEC §5 FR-009, FR-010
- SPEC §7 Compatibility Requirements

## SDD References

- SDD §5 API and Interface Changes
- SDD §8 Compatibility Strategy

## Acceptance Criteria

- Core route groups and auth/RBAC requirements are covered by tests or documented gaps.
- Proto source and generated code synchronization is validated or documented.
- No public HTTP behavior or proto contract change is introduced without confirmation.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-003
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
