# Acceptance - TASK-BDME-007

## TASK Summary

Deployment And Readiness Baseline: Align current deployment, probes, local scripts, and readiness assumptions with the staged migration plan.

## SPEC References

- SPEC §6 Non-Functional Requirements
- SPEC §8 Observability and Debug Requirements

## SDD References

- SDD §10 Observability and Debug Output Design
- SDD §11 Testing Strategy

## Acceptance Criteria

- Current gateway, logic, and worker deployment/readiness behavior is documented.
- Hard and soft dependencies are identified.
- Local development workflow remains usable or has equivalent documentation.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-007
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

## Manual Verification, if needed

- This TASK requires explicit human confirmation before implementation because it may affect security, database, deployment, traffic routing, public contracts, or production readiness.

## Out-of-Scope

- Changes outside task-scope.json allowedFiles for this TASK.
- Modifications to SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or Harness scripts.
- Frontend application changes.
- Package manifest, lockfile, go.mod, go.sum, dependency, CI/CD, or global configuration changes unless explicitly listed in scope and confirmed.
- Product behavior changes not stated in this TASK acceptance file.
