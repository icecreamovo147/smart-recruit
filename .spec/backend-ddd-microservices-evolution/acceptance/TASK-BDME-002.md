# Acceptance - TASK-BDME-002

## TASK Summary

Secret And Config Safety Baseline: Establish committed-config safety and production fail-fast rules for secrets and internal auth.

## SPEC References

- SPEC §10 Security and Safety Requirements

## SDD References

- SDD §7 Configuration Design
- SDD §11 Security tests

## Acceptance Criteria

- Tracked config examples contain placeholders only.
- Production profiles fail fast when required secrets and internal auth are missing.
- Local examples remain usable and no live local env file is added or modified.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-002
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
