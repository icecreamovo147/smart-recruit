# Acceptance - TASK-002

## TASK Summary

Make single Agent Skill embedding regeneration produce deterministic non-success behavior for no-op results.

## SPEC References

- FR-002 Embedding Regeneration Result Semantics
- AC-003
- AC-004

## SDD References

- Section 3, TASK-002 Deterministic Embedding Regeneration
- Section 9, Error Handling and Fallback Design

## Acceptance Criteria

- When `object_id > 0` and no eligible Agent Skill is found, backend result is not all-zero success.
- Frontend only shows success when `success_count > 0`.
- Frontend treats skipped or all-zero results as warning/non-success.
- Batch backfill behavior remains compatible.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-review-fixes/scripts/check-task-scope.sh TASK-002`
- `bash .spec/agent-skill-review-fixes/scripts/agent-check.sh`
- `cd logic-grpc-service && go test ./...`
- `pnpm --filter hr-frontend typecheck`

## Manual Verification, if needed

- Attempt to regenerate embedding for an unavailable/ineligible Skill row and confirm no success toast is shown.

## Out-of-Scope

- Protobuf response field additions unless explicitly confirmed.
- Database migrations.
- Permission changes.
