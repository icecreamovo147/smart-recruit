# Acceptance - TASK-HATO-006

## TASK Summary

Optionally add backward-compatible pagination or lazy retrieval support for long trace histories.

## SPEC References

- 4.10 Pagination and Long History
- CR-001
- CR-002
- CR-006
- CR-007
- AC-012
- Out of Scope constraints for database and unconfirmed API changes

## SDD References

- 5.2 Optional Later API Extension
- 8 Compatibility Strategy
- 12 Migration Risks
- 13 Implementation Boundaries

## Acceptance Criteria

- Explicit user confirmation is recorded before implementation starts.
- Existing clients without pagination parameters retain compatible behavior.
- New pagination or cursor parameters are backward-compatible and covered by tests.
- Frontend can request additional trace pages or indicate truncation/long-history state.
- Backend tests cover default and paginated behavior.
- No database schema or migration changes are made.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh TASK-HATO-006`
- `bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh TASK-HATO-006`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`
- `cd logic-grpc-service && go test ./service ./repository -count=1`
- `cd web-gin-service && go test ./handler/... ./router/... -count=1`

## Manual Verification, if needed

- Verify old unpaginated calls still load traces.
- Verify paginated calls fetch additional data or show truncation state as designed.

## Out-of-Scope

- Starting without explicit confirmation.
- Database migrations.
- Auth, permission, or quota changes.
- Breaking existing trace API response behavior.
