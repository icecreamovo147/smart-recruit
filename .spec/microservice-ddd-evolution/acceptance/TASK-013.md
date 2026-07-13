# Acceptance - TASK-013

## TASK Summary

迁移 Identity domain/application。

## SPEC References

- FR-002
- FR-003
- FR-011
- SSR-002

## SDD References

- 9. Error Handling and Fallback Design
- 11. Testing Strategy

## Acceptance Criteria

- Identity domain/application 本地化。
- JWT/Refresh/RBAC/data scope/audit 语义兼容。
- Domain 不依赖外层技术。

## Required Checks

- Identity domain/application tests
- `go test ./...` in `smart-recruit-identity-service`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认；安全语义变化必须停止。

## Out-of-Scope

权限模型、token 语义、protobuf/schema 修改。
