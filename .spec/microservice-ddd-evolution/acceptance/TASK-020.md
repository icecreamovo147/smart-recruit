# Acceptance - TASK-020

## TASK Summary

迁移 Analytics reporting/projection application。

## SPEC References

- FR-013
- FR-020
- ARC-010

## SDD References

- 4. Data Structure Changes
- 8. Compatibility Strategy

## Acceptance Criteria

- Reporting/projection application 本地化。
- 不写回事务业务状态。
- 报表口径兼容，过渡只读债务被记录。

## Required Checks

- Analytics tests
- `go test ./...` in `smart-recruit-analytics-service`
- scope check
- agent-check

## Manual Verification, if needed

核对 dashboard/funnel/time-in-stage/interview-offer 指标口径。

## Out-of-Scope

schema/proto 修改和长期只读例外固化。
