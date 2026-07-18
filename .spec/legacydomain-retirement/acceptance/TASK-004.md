# Acceptance - TASK-004

## TASK Summary

Retire Interview service `legacydomain` by replacing active dependencies with local infrastructure and owner adapters.

## SPEC References

- FR-005, FR-008, FR-009
- CR-001, CR-003, CR-005
- SSR-003

## SDD References

- Sections 3, 4, 5, 8, 11, 13

## Acceptance Criteria

- Interview non-test code has no `legacydomain` import.
- `smart-recruit-interview-service/internal/legacydomain` is deleted.
- Interview owner persistence uses local infrastructure records/adapters.
- Candidate-facing filtering and feedback behavior remain compatible.
- Existing Interview gRPC behavior remains compatible.

## Required Checks

- `go test ./...` in `smart-recruit-interview-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-004`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

Offer, Recruitment, AI Agent, protobuf, and schema changes.
