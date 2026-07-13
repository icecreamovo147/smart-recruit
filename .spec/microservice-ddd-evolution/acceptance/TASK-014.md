# Acceptance - TASK-014

## TASK Summary

完成 Identity infrastructure/interfaces/runtime/tests 收敛。

## SPEC References

- FR-004 至 FR-011
- FR-016
- SSR-002

## SDD References

- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- Identity runtime 使用本地 implementation。
- Auth/Admin 安全语义与 audit 行为兼容。
- 对共享 auth/admin implementation 的直接依赖清除或记录债务。

## Required Checks

- `go test ./...` in `smart-recruit-identity-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认。

## Out-of-Scope

修改 public auth/authz 行为、权限表 schema。
