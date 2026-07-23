# Acceptance - TASK-BDME-001

## TASK Summary

Architecture Baseline Inventory: Inventory current backend architecture, workflow ownership, runtime dependencies, and migration risk map before any code movement.

## SPEC References

- SPEC §5 Functional Requirements
- SPEC §11 Acceptance Criteria

## SDD References

- SDD §1 Existing Architecture Summary
- SDD §3.3 Migration Phases

## Acceptance Criteria

- Current gateway, logic service, workers, MySQL, Redis, RabbitMQ, OSS, SMTP, and AI dependencies are mapped.
- Bounded-context candidates are mapped to current packages, models, repositories, handlers, and workers.
- Cross-domain access risks and startup side effects are listed with source locations or explicit not-found notes.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-001
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
