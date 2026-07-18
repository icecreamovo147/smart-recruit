# Acceptance - TASK-018

## TASK Summary

完成 Recruitment infrastructure/interfaces/runtime/tests 收敛。

## SPEC References

- FR-004 至 FR-007
- FR-012
- FR-016

## SDD References

- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- Recruitment runtime 使用本地 implementation。
- Job/Candidate/Application/Collaboration/Admin 子集兼容。
- 对共享 recruitment implementation 的依赖清除或记录债务。

## Required Checks

- `go test ./...` in `smart-recruit-recruitment-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

核对 AdminService 切分兼容。

## Out-of-Scope

修改 protobuf AdminService 契约。
