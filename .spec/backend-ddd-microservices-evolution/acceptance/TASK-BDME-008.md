# Acceptance - TASK-BDME-008

## TASK Summary

Bounded Context Architecture Contract: Create the enforceable bounded-context contract for target domains and workers.

## SPEC References

- SPEC §5 FR-001..FR-008
- SPEC §11 AC-004..AC-008

## SDD References

- SDD §3.2 Target DDD Package Shape
- SDD §3.4 Target Ownership Matrix

## Acceptance Criteria

- Each context has explicit owned aggregates, data, interfaces, and forbidden ownership.
- Recruitment owns candidate/resume/application profile data; AI Agent owns AI-derived intelligence.
- Analytics final source is domain-event projections/read models, not transitional service read APIs.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-008
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
