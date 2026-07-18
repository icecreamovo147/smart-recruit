# Acceptance - TASK-005

## TASK Summary

完成 Offer infrastructure/interfaces/runtime/tests 收敛。

## SPEC References

- FR-004 至 FR-008
- FR-016
- CR-001 至 CR-006

## SDD References

- 3.4 Per-Service Migration Pattern
- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- Offer runtime 使用本地 implementation 注册 `OfferService`。
- gRPC 行为兼容。
- 对共享 `service.OfferService` 的直接依赖清除或记录为临时债务。

## Required Checks

- `go test ./...` in `smart-recruit-offer-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

核对 Offer 发送、撤回、接受、拒绝、事件列表语义。

## Out-of-Scope

删除 shared 旧实现、修改 protobuf/schema。
