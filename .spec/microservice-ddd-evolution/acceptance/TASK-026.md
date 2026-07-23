# Acceptance - TASK-026

## TASK Summary

创建 Worker workload 骨架并盘点边界。

## SPEC References

- FR-001 至 FR-007
- FR-015
- NFR-007

## SDD References

- 3.3 Required Migration Order
- 3.4 Per-Service Migration Pattern

## Acceptance Criteria

- Worker workload profile/toggle 骨架存在。
- Outbox/Inbox/DLQ/notification/email/resume/embedding/agent-run/analytics owner contract 被记录。
- 默认 workload 行为不变。

## Required Checks

- `go test ./...` in `smart-recruit-worker-service`
- scope check
- agent-check

## Manual Verification, if needed

确认不拆分 worker binary。

## Out-of-Scope

新增 workload、shared cleanup。
