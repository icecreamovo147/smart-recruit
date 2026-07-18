# Acceptance - TASK-010

## TASK Summary

迁移 Notification domain/application。

## SPEC References

- FR-002
- FR-003
- FR-010
- EFR-003

## SDD References

- 6. Algorithm or Workflow Changes
- 9. Error Handling and Fallback Design

## Acceptance Criteria

- Notification domain/application 本地化。
- unread/summary/mark read/email coordination/idempotency 语义兼容。
- Domain 不依赖 GORM/RabbitMQ/gRPC。

## Required Checks

- Notification domain/application tests
- `go test ./...` in `smart-recruit-notification-service`
- scope check
- agent-check

## Manual Verification, if needed

确认 unread count 和 summary 边界。

## Out-of-Scope

Runtime 切换、真实 SMTP、消息 schema 修改。
