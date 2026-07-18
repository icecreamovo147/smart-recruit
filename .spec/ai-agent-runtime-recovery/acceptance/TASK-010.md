# Acceptance - TASK-010

## TASK Summary

Harden regression coverage and produce final feature-owned smoke/evidence documentation.

## SPEC References

- FR-009 Validation and dependency hygiene
- Observability and Debug Requirements 7
- Acceptance Criteria 13, 14, 15, 16

## SDD References

- Testing Strategy
- Migration Risks
- Implementation Boundaries

## Acceptance Criteria

- AI Agent service tests pass.
- Gateway tests pass.
- Touched frontend type checks pass.
- Focused frontend Vitest coverage exists for touched compatibility parsing where applicable.
- `.spec/ai-agent-runtime-recovery/docs/manual-smoke-checklist.md` covers HR AI Chat, durable Agent Run, Candidate AI, Recruiting Intelligence parse/evaluate, MCP, Embedding, and Semantic Debug.
- Final report/evidence honestly records any skipped live checks and remaining risks.
- No production runtime behavior is implemented in this TASK.

## Required Checks

- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway`.
- `pnpm --filter hr-frontend typecheck` if HR frontend was touched by this feature.
- `pnpm --filter user-frontend typecheck` if user frontend was touched by this feature.
- Focused Vitest for touched frontend tests.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-010`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

Manual smoke checklist must state which live checks were run, skipped, or blocked.

## Out-of-Scope

- Production source implementation.
- K8s work.
- Proto, schema, or auth changes.
