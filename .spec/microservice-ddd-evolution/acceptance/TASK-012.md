# Acceptance - TASK-012

## TASK Summary

创建 Identity DDD 骨架并完成安全契约盘点。

## SPEC References

- FR-001 至 FR-007
- FR-011
- SSR-002

## SDD References

- 3.4 Per-Service Migration Pattern
- 12. Migration Risks

## Acceptance Criteria

- Identity DDD 骨架存在。
- auth/refresh/principal/RBAC/data scope/invite/audit 契约盘点完成。
- 未改变认证授权行为。

## Required Checks

- `go test ./...` in `smart-recruit-identity-service`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认后执行。

## Out-of-Scope

任何 auth/authz 行为变化。
