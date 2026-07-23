# Acceptance - TASK-007

## TASK Summary

迁移 Interview domain/application。

## SPEC References

- FR-002
- FR-003
- FR-009

## SDD References

- 6. Algorithm or Workflow Changes
- 9. Error Handling and Fallback Design

## Acceptance Criteria

- Interview domain/application 本地化。
- 保持 schedule/update/cancel/batch cancel/feedback 规则。
- Domain 不依赖外层技术。

## Required Checks

- Interview domain/application tests
- `go test ./...` in `smart-recruit-interview-service`
- scope check
- agent-check

## Manual Verification, if needed

若通知事件契约需要改变，停止确认。

## Out-of-Scope

Runtime 完整切换、proto/schema 修改。
