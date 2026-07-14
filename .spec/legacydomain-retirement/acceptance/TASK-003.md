# Acceptance - TASK-003

## TASK Summary

Retire Offer service `legacydomain` by replacing active dependencies with local infrastructure and owner adapters.

## SPEC References

- FR-004, FR-008, FR-009
- CR-001, CR-003, CR-005

## SDD References

- Sections 3, 4, 5, 8, 11, 13

## Acceptance Criteria

- Offer non-test code has no `legacydomain` import.
- `smart-recruit-offer-service/internal/legacydomain` is deleted.
- Offer owner persistence uses local infrastructure records/adapters.
- Outbox, lifecycle, snapshot, and authz dependencies use explicit adapters.
- Existing Offer gRPC behavior remains compatible.

## Required Checks

- `go test ./...` in `smart-recruit-offer-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-003`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

Interview, Recruitment, AI Agent, protobuf, and schema changes.
