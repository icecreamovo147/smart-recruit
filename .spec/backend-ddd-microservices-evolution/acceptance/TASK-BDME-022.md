# Acceptance - TASK-BDME-022

## TASK Summary

Notification Event Conversion: Convert notification-producing cross-domain writes to domain events and idempotent Notification consumers.

## SPEC References

- SPEC §5 FR-006, FR-011, FR-012
- SPEC §9 Error Handling and Fallback Requirements

## SDD References

- SDD §6.2 Domain Event Flow
- SDD §9 Error Handling and Fallback Design

## Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-022
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
