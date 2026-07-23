# Acceptance - TASK-008

## TASK Summary

完成 Interview infrastructure/interfaces/runtime/tests 收敛。

## SPEC References

- FR-004 至 FR-009
- FR-016

## SDD References

- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- Interview runtime 使用本地 implementation 注册 `InterviewService`。
- 对共享 `service.InterviewService` 的直接依赖消除或记录债务。
- gRPC 语义兼容。

## Required Checks

- `go test ./...` in `smart-recruit-interview-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

核对列表查询和权限范围未漂移。

## Out-of-Scope

删除 shared 旧实现、修改 Recruitment/Offer 行为。
