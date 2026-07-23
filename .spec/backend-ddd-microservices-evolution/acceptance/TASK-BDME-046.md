# Acceptance - TASK-BDME-046

## TASK Summary

Table Ownership Manifest: Create and enforce a table ownership manifest for every existing table and target service context.

## SPEC References

- SPEC §5 FR-001..FR-008
- SPEC §11 AC-004..AC-008

## SDD References

- SDD §3.2 Target DDD Package Shape
- SDD §3.4 Target Ownership Matrix

## Acceptance Criteria

- Every current table is assigned an owner or justified shared platform owner.
- Allowed readers/writers are explicit per table.
- Every transitional shared DB access has owner, reason, risk, and removal plan.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-046
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

## Manual Verification, if needed

- Manual verification is required only when automated checks cannot cover an acceptance criterion; skipped checks must be recorded with reasons.

## Out-of-Scope

- Changes outside task-scope.json allowedFiles for this TASK.
- Modifications to SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or Harness scripts.
- Frontend application changes.
- Package manifest, lockfile, go.mod, go.sum, dependency, CI/CD, or global configuration changes unless explicitly listed in scope and confirmed.
- Product behavior changes not stated in this TASK acceptance file.
