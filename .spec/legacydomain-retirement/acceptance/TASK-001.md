# Acceptance - TASK-001

## TASK Summary

Create baseline inventory, staged guardrails, and knowledge-impact routing for legacydomain retirement.

## SPEC References

- FR-001, FR-002, FR-003, FR-010
- AC-001, AC-002

## SDD References

- Sections 1, 2, 3, 10, 11, 13

## Acceptance Criteria

- Inventory records target legacy roots, active imports, runtime entrypoints, and knowledge references.
- Boundary/table scripts can report legacy usage without blocking the current baseline.
- Routed active knowledge is reviewed and updated or explicitly reported as debt.
- No business service files are modified.

## Required Checks

- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-001`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

Runtime cutovers and deleting legacydomain files.
