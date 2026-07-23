# Acceptance - TASK-019

## TASK Summary

创建 Analytics DDD 骨架并盘点投影策略。

## SPEC References

- FR-001 至 FR-007
- FR-013

## SDD References

- 3.3 Required Migration Order
- 8. Compatibility Strategy

## Acceptance Criteria

- Analytics DDD 骨架存在。
- reporting API、projection、只读跨表债务被记录。
- 不改变报表 API 行为。

## Required Checks

- `go test ./...` in `smart-recruit-analytics-service`
- scope check
- agent-check

## Manual Verification, if needed

确认 Analytics 不写事务业务状态。

## Out-of-Scope

创建新 projection schema。
