# Acceptance - TASK-HATO-002

## TASK Summary

Add overview, abnormal issue summary, and responsive drawer shell to the execution trace panel.

## SPEC References

- 4.1 Panel Entry
- 4.2 Top Overview
- 4.7 Failure, Risk, and Policy Emphasis
- 4.8 Layout Responsiveness
- FR-001 through FR-004
- FR-007
- FR-009
- AC-001
- AC-002
- AC-006
- AC-010

## SDD References

- 3.1 Component Decomposition
- 3.3 Layout
- 3.7 Abnormal Summary
- 8 Compatibility Strategy
- 13 Implementation Boundaries

## Acceptance Criteria

- Trace panel preserves existing load behavior for durable runs and legacy traces.
- Overview appears before detailed timeline when trace data exists.
- Abnormal summary surfaces failed runs/steps, partial/fallback states, risk flags, and MCP policy issues.
- Final answer markdown remains sanitized.
- Legacy-only sessions remain usable.
- Drawer is wider on desktop and responsive on mobile without obvious clipping or overlap.

## Required Checks

- `git diff --name-only`
- `bash .spec/hr-agent-trace-optimization/scripts/check-task-scope.sh TASK-HATO-002`
- `bash .spec/hr-agent-trace-optimization/scripts/agent-check.sh TASK-HATO-002`
- `pnpm --filter hr-frontend typecheck`
- `pnpm --filter hr-frontend test`

## Manual Verification, if needed

- Open the HR AI chat trace drawer at desktop and mobile widths and verify the overview, issue summary, and legacy-only path render cleanly.

## Out-of-Scope

- Search and filtering controls.
- JSON viewer replacement.
- Real-time active-run subscription.
- Backend API changes.
