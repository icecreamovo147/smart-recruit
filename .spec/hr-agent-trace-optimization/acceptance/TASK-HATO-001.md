# Acceptance - TASK-HATO-001

## TASK Summary

Create pure HR frontend trace view-model helpers and tests.

## SPEC References

- FR-002 through FR-006
- FR-015
- NFR-005 through NFR-008
- AC-011

## SDD References

- 3.2 View Model
- 6.2 Derivation Workflow
- 11.1 Unit Tests
- 13 Implementation Boundaries

## Acceptance Criteria

- Overview metrics are derived from `AgentRunItem[]` and `ToolTraceItem[]`.
- Failed, warning, success, active, evidence, tool, and legacy categories are classified.
- MCP policy decisions and risk flags are parsed into issue summaries.
- Searchable text is generated without mutating source data.
- Valid JSON is formatted and invalid JSON is preserved as raw text.
- Legacy-only data produces a useful view model.
- Tests cover the above behavior.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh TASK-HATO-001`
- `bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh TASK-HATO-001`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

## Manual Verification, if needed

None required.

## Out-of-Scope

- Visible panel redesign.
- Real-time subscription.
- Backend API changes.
- Package or dependency changes.
