# Acceptance - TASK-004

## TASK Summary

迁移 Offer domain/application。

## SPEC References

- FR-002
- FR-003
- FR-008
- EFR-001 至 EFR-005

## SDD References

- 3.4 Per-Service Migration Pattern
- 6. Algorithm or Workflow Changes
- 9. Error Handling and Fallback Design

## Acceptance Criteria

- Offer domain model/policy/event/repository port 本地化。
- Application command/query 保持现有 Offer 语义。
- Domain 不依赖 GORM/proto/gRPC。

## Required Checks

- Offer domain/application tests
- `go test ./...` in `smart-recruit-offer-service`
- scope check
- agent-check

## Manual Verification, if needed

若需要新增跨服务 API 或事件 schema，停止确认。

## Out-of-Scope

Runtime 切换、删除共享旧实现、proto/schema 修改。
