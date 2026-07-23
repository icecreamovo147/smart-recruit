# Acceptance - TASK-009

## TASK Summary

Make legacydomain absence a hard rule and converge final documentation and knowledge.

## SPEC References

- FR-010
- AC-003, AC-004, AC-005, AC-006, AC-007

## SDD References

- Sections 3, 10, 11, 13

## Acceptance Criteria

- No target service `internal/legacydomain` directory remains.
- No non-test `legacydomain` import remains.
- Boundary/table scripts fail future legacydomain reintroduction.
- Knowledge validation and reference checks pass.
- Final report records retired roots and any remaining non-legacy debt.

## Required Checks

- `find smart-recruit-*-service -path '*/internal/legacydomain' -type d -print`
- `rg -n 'legacydomain' smart-recruit-*-service scripts .knowledge -g '*.go' -g '*.md' -g '*.mjs'`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-009`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

Any business behavior change.
