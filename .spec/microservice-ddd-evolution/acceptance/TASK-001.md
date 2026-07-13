# Acceptance - TASK-001

## TASK Summary

建立迁移总则与边界检查基线。

## SPEC References

- FR-001 至 FR-007
- FR-024
- AC-006 至 AC-008

## SDD References

- 3.1 Target Service Shape
- 11. Testing Strategy
- 13. Implementation Boundaries

## Acceptance Criteria

- DDD 分层、迁移顺序、shared kernel、Hard Stop 条件被文档化。
- 边界检查规则覆盖 domain 禁止依赖外层技术。
- 未修改业务代码。

## Required Checks

- `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-001`
- `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh`
- `node scripts/check-backend-boundaries.mjs`，如不可用需记录原因。

## Manual Verification, if needed

确认规则没有把后续已知必要迁移误判为永久禁止。

## Out-of-Scope

业务迁移、protobuf、schema、auth、安全和部署变更。
