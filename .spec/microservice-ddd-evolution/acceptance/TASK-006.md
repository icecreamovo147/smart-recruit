# Acceptance - TASK-006

## TASK Summary

创建 Interview DDD 骨架并完成契约盘点。

## SPEC References

- FR-001 至 FR-007
- FR-009

## SDD References

- 3.3 Required Migration Order
- 3.4 Per-Service Migration Pattern

## Acceptance Criteria

- Interview DDD 骨架存在。
- schedule/update/cancel/batch cancel/feedback/listing 依赖被记录。
- 未改变 protobuf 或用户行为。

## Required Checks

- `go test ./...` in `smart-recruit-interview-service`
- scope check
- agent-check

## Manual Verification, if needed

确认 Offer 迁移成果未回退。

## Out-of-Scope

Interview 业务迁移、shared cleanup、schema/proto 修改。
