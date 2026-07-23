# Acceptance - TASK-009

## TASK Summary

创建 Notification DDD 骨架并完成契约盘点。

## SPEC References

- FR-001 至 FR-007
- FR-010

## SDD References

- 3.4 Per-Service Migration Pattern
- 10. Observability and Debug Output Design

## Acceptance Criteria

- Notification DDD 骨架存在。
- notification/email/Outbox/Inbox/SSE 依赖盘点完成。
- 不改变通知 API 行为。

## Required Checks

- `go test ./...` in `smart-recruit-notification-service`
- scope check
- agent-check

## Manual Verification, if needed

确认邮件测试不会触发真实发送。

## Out-of-Scope

通知业务迁移、消息 schema 修改。
