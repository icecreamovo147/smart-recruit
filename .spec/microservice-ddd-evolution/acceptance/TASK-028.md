# Acceptance - TASK-028

## TASK Summary

完成 Worker infrastructure/tests 与 owner contract 收敛。

## SPEC References

- FR-015
- FR-016
- NFR-007

## SDD References

- 8. Compatibility Strategy
- 11. Testing Strategy

## Acceptance Criteria

- Worker 不再依赖共享业务 service 编排，或剩余依赖记录为 shared cleanup 债务。
- Workload readiness、idempotency、retry/DLQ 测试覆盖。
- 默认 workload 行为兼容。

## Required Checks

- `go test ./...` in `smart-recruit-worker-service`
- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

确认可进入 shared cleanup。

## Out-of-Scope

重命名 shared module。
