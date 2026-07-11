# Acceptance - TASK-HATO-003

## TASK Summary

Add search, filters, issue-only focus, and layered navigation to the trace panel.

## SPEC References

- 4.3 Information Layers
- 4.4 Search and Filtering
- FR-005
- FR-006
- EFR-006
- AC-003
- AC-004
- AC-005

## SDD References

- 3.3 Layout
- 3.4 Filtering
- 6.3 Filter Workflow
- 11 Testing Strategy

## Acceptance Criteria

- Keyword search filters visible run/step/trace content using view-model searchable fields.
- Status filter supports all, failed, warning, succeeded, and active states.
- Type filter supports plan, tool, evidence, memory, prompt/skill, model, fallback/recovery, and legacy where present.
- Issue-only mode focuses failures, risks, and policy issues.
- Filter reset restores all data.
- Filter-empty state is distinct from no-trace empty state.
- Content is organized into overview/plan, steps, raw data, and legacy trace layers.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh TASK-HATO-003`
- `bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh TASK-HATO-003`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

## Manual Verification, if needed

- Verify filters with a mixed trace fixture containing success, failed, evidence, tool, and legacy records.

## Out-of-Scope

- Replacing raw JSON display.
- SSE active-run integration.
- Backend pagination.
