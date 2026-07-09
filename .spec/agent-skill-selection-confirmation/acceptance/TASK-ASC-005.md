# Acceptance - TASK-ASC-005

## TASK Summary

Run broad validation and complete final feature report.

## SPEC References

- Acceptance Criteria
- Non-Functional Requirements
- Compatibility Requirements

## SDD References

- Section 11 Testing Strategy
- Section 12 Migration Risks

## Acceptance Criteria

- Feature-scoped backend checks pass or failures are documented as pre-existing/out-of-scope.
- HR frontend typecheck passes.
- Scope check passes after unrelated dirty changes are handled.
- Final TASK report summarizes remaining risks and manual verification.

## Required Checks

- `git diff --name-only`
- `bash .spec/agent-skill-selection-confirmation/scripts/check-task-scope.sh TASK-ASC-005`
- `bash .spec/agent-skill-selection-confirmation/scripts/agent-check.sh`
- `cd logic-grpc-service && go test ./...`
- `cd web-gin-service && go test ./...`
- `pnpm --filter hr-frontend typecheck`

## Manual Verification, if needed

Perform one HR chat flow covering selection-required, confirm one, confirm none, and manual Skill bypass.

## Out-of-Scope

- Fixing unrelated failing tests.
- Package/dependency changes.
