# Acceptance - TASK-025

## TASK Summary

完成 AI Agent infrastructure/interfaces/runtime/tests 收敛。

## SPEC References

- FR-004 至 FR-007
- FR-014
- FR-016

## SDD References

- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- AI Agent runtime 使用本地 implementation 注册全部 AI 相关服务。
- Embedding/agent-run worker control 兼容。
- 对共享 AI implementation 的依赖清除或记录债务。

## Required Checks

- `go test ./...` in `smart-recruit-ai-agent-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认；检查 credential 不泄露。

## Out-of-Scope

修改 provider credential 存储或 public API。
