# Acceptance - TASK-BDME-047

## TASK Summary

Schema Separation Plan: Prepare schema or physical database separation after service boundaries and table ownership are proven.

## SPEC References

- SPEC §5 FR-001..FR-008
- SPEC §11 AC-004..AC-008

## SDD References

- SDD §3.2 Target DDD Package Shape
- SDD §3.4 Target Ownership Matrix

## Acceptance Criteria

- Schema/database separation follows the ownership manifest.
- Expand-contract, rollback, reconciliation, RTO, and RPO considerations are documented.
- Any migration/model/db.sql change is scoped and aligned.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-047
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
