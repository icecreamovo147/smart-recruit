# Acceptance - TASK-BDME-005

## TASK Summary

Harness And Knowledge Impact Baseline: Verify this feature Harness, knowledge impact routing, and reporting conventions before implementation tasks begin.

## SPEC References

- SPEC §5 Functional Requirements
- SPEC §11 Acceptance Criteria

## SDD References

- SDD §1 Existing Architecture Summary
- SDD §3.3 Migration Phases

## Acceptance Criteria

- Feature validates as schemaVersion 1 current Harness.
- Knowledge routes relevant to backend architecture, service boundaries, persistence, auth, API contracts, and operations are identified.
- TASK report expectations include knowledge impact result and coverage gaps.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-005
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed
- node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/backend-ddd-microservices-evolution --json

## Manual Verification, if needed

- Manual verification is required only when automated checks cannot cover an acceptance criterion; skipped checks must be recorded with reasons.

## Out-of-Scope

- Changes outside task-scope.json allowedFiles for this TASK.
- Modifications to SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or Harness scripts.
- Frontend application changes.
- Package manifest, lockfile, go.mod, go.sum, dependency, CI/CD, or global configuration changes unless explicitly listed in scope and confirmed.
- Product behavior changes not stated in this TASK acceptance file.
