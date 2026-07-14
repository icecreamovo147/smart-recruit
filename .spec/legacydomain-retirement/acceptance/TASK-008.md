# Acceptance - TASK-008

## TASK Summary

Delete AI Agent `legacydomain` after active runtime no longer depends on it.

## SPEC References

- FR-007, FR-010
- AC-003, AC-004, AC-005, AC-007

## SDD References

- Sections 3, 10, 11, 13

## Acceptance Criteria

- AI Agent non-test code has no `legacydomain` import.
- `smart-recruit-ai-agent-service/internal/legacydomain` is deleted.
- Guardrail scripts no longer whitelist AI Agent legacy paths.
- Knowledge references to AI Agent legacy paths are updated.

## Required Checks

- `go test ./...` in `smart-recruit-ai-agent-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-008`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

Recruitment, Offer, and Interview cleanup.
