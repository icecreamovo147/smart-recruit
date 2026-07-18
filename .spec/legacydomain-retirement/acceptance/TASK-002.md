# Acceptance - TASK-002

## TASK Summary

Add confirmed internal owner contracts for application lifecycle/snapshot and identity authorization/principal checks.

## SPEC References

- FR-008, FR-009
- CR-001, CR-002, CR-003
- SSR-001

## SDD References

- Sections 3, 5, 8, 9, 13

## Acceptance Criteria

- Recruitment contract supports Offer/Interview application snapshot and lifecycle needs.
- Identity contract supports service authorizer principal/scope needs.
- Public gateway/frontend behavior is unchanged.
- Protobuf changes are additive and documented.
- Human confirmation is recorded before implementation.

## Required Checks

- `go test ./...` in `smart-recruit-proto`
- `go test ./...` in `smart-recruit-recruitment-service`
- `go test ./...` in `smart-recruit-identity-service`
- `go test ./...` in `smart-recruit-gateway`
- `node scripts/check-backend-boundaries.mjs`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-002`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

Confirm the internal contract wire shape before editing.

## Out-of-Scope

Frontend API changes, database schema changes, and deleting legacydomain files.
