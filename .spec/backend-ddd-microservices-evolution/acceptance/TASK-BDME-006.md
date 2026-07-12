# Acceptance - TASK-BDME-006

## TASK Summary

Observability Baseline Design: Define the telemetry contract for metrics, traces, logs, and cutover diagnostics.

## SPEC References

- SPEC §6 Non-Functional Requirements
- SPEC §8 Observability and Debug Requirements

## SDD References

- SDD §10 Observability and Debug Output Design
- SDD §11 Testing Strategy

## Acceptance Criteria

- Telemetry conventions cover HTTP, gRPC, DB, Redis, RabbitMQ, workers, Outbox/Inbox, and AI paths.
- Trace/request id propagation expectations cover gateway, gRPC, events, and workers.
- Redaction requirements cover secrets, tokens, prompts, and unnecessary candidate data.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-006
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
