# Acceptance - TASK-021

## TASK Summary

完成 Analytics infrastructure/interfaces/runtime/tests 收敛。

## SPEC References

- FR-004 至 FR-007
- FR-013
- FR-016

## SDD References

- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- Analytics runtime 使用本地 implementation。
- Reporting API 兼容。
- 对共享 analytics implementation 的依赖清除或记录债务。

## Required Checks

- `go test ./...` in `smart-recruit-analytics-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

核对 AdminService analytics 子集注册。

## Out-of-Scope

修改报表 public contract。
