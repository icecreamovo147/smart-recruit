# Acceptance - TASK-BDME-017

## TASK Summary

Analytics Projection Boundary: Create the Analytics bounded-context boundary around event-projection read models and reporting APIs.

## SPEC References

- SPEC §5 FR-020
- SPEC §11 AC-014
- SPEC §13 D-014

## SDD References

- SDD §3.2 Target DDD Package Shape
- SDD §3.4 Target Ownership Matrix

## Acceptance Criteria

- Analytics boundary owns reporting read models and query APIs only.
- Analytics does not mutate transactional domain state.
- No transitional service read API dependency is introduced.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-017
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
