# Acceptance - TASK-005

## TASK Summary

Cut Recruitment runtime over from legacy service graph to local DDD application and infrastructure adapters.

## SPEC References

- FR-006, FR-008, FR-009
- CR-001, CR-003, CR-005

## SDD References

- Sections 3, 5, 6, 8, 11, 12, 13

## Acceptance Criteria

- Active Recruitment runtime no longer constructs legacy service graph.
- Job, taxonomy/admin, candidate, application, collaboration, and usage surfaces are backed by local adapters.
- Existing public Recruitment protobuf behavior remains compatible.
- Recruitment `legacydomain` is not deleted in this TASK.

## Required Checks

- `go test ./...` in `smart-recruit-recruitment-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-005`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

Deleting Recruitment legacydomain, AI Agent changes, protobuf, and schema changes.
