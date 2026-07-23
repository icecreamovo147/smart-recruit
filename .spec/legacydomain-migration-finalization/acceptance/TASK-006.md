# Acceptance - TASK-006

## TASK Summary

Run final verification and converge `.knowledge` on the finalized post-legacydomain architecture.

## SPEC References

- SPEC 5.8
- SPEC 5.9
- SPEC 5.10
- SPEC 11

## SDD References

- SDD 3 Guardrails
- SDD 10
- SDD 11

## Acceptance Criteria

- No targeted service contains `internal/legacydomain`.
- Non-test Go code has no `legacydomain` import.
- Guardrails pass.
- Active `.knowledge` documents reflect final Recruitment and AI Agent architecture.
- Final report states whether the feature is ready for pipeline completion.

## Required Checks

- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- common Harness checks from AGENT_RULES.

## Manual Verification, if needed

Review final knowledge diffs for stale native-adapter language.

## Out-of-Scope

Business code changes.
