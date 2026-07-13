# Acceptance - TASK-017

## TASK Summary

迁移 Recruitment application/collaboration/taxonomy。

## SPEC References

- FR-002
- FR-003
- FR-012
- FR-019

## SDD References

- 6. Algorithm or Workflow Changes
- 8. Compatibility Strategy

## Acceptance Criteria

- Application 状态机、collaboration、taxonomy、usage stats 本地化。
- 保留 transitions、outbox、权限和分页语义。
- 覆盖 apply/update status/collaboration/taxonomy 测试。

## Required Checks

- Recruitment targeted tests
- `go test ./...` in `smart-recruit-recruitment-service`
- scope check
- agent-check

## Manual Verification, if needed

新增 snapshot API 或事件 schema 必须停止确认。

## Out-of-Scope

Runtime 最终切换、proto/schema 修改。
