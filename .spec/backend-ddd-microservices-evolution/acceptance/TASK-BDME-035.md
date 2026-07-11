# Acceptance - TASK-BDME-035

## TASK Summary

Identity Auth RBAC API Extraction: Extract Identity auth, token invalidation, RBAC, scopes, and audit APIs into Identity service runtime.

## SPEC References

- SPEC §10 Security and Safety Requirements

## SDD References

- SDD §5 API and Interface Changes
- SDD §8 Compatibility Strategy

## Acceptance Criteria

- The TASK goal is implemented or documented exactly as scoped.
- Existing behavior remains compatible unless explicitly confirmed in this TASK.
- Report and evidence record scope, SPEC, SDD, acceptance, tests, risks, and knowledge impact.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-035
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

## Manual Verification, if needed

- This TASK requires explicit human confirmation before implementation because it may affect security, database, deployment, traffic routing, public contracts, or production readiness.

## Out-of-Scope

- Changes outside task-scope.json allowedFiles for this TASK.
- Modifications to SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or Harness scripts.
- Frontend application changes.
- Package manifest, lockfile, go.mod, go.sum, dependency, CI/CD, or global configuration changes unless explicitly listed in scope and confirmed.
- Product behavior changes not stated in this TASK acceptance file.
