# Acceptance - TASK-011

## TASK Summary

完成 Notification infrastructure/interfaces/runtime/tests 收敛。

## SPEC References

- FR-004 至 FR-010
- FR-016

## SDD References

- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- Notification runtime 使用本地 implementation。
- Outbox/Inbox retry/DLQ/idempotency 兼容。
- 对共享 notification implementation 的依赖清除或记录债务。

## Required Checks

- `go test ./...` in `smart-recruit-notification-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

核对 consumer lifecycle 和 readiness。

## Out-of-Scope

修改消息 schema、删除 shared 旧实现。
