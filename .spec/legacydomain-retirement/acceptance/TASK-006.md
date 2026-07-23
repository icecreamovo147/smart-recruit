# Acceptance - TASK-006

## TASK Summary

Delete Recruitment `legacydomain` after active runtime no longer depends on it.

## SPEC References

- FR-006, FR-010
- AC-003, AC-004, AC-005

## SDD References

- Sections 3, 10, 11, 13

## Acceptance Criteria

- Recruitment non-test code has no `legacydomain` import.
- `smart-recruit-recruitment-service/internal/legacydomain` is deleted.
- Guardrail scripts no longer whitelist Recruitment legacy paths.
- Knowledge references to Recruitment legacy paths are updated or removed.

## Required Checks

- `go test ./...` in `smart-recruit-recruitment-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-006`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

AI Agent legacydomain deletion.
