# Acceptance - TASK-002

## TASK Summary

盘点共享依赖、表归属与服务迁移基线。

## SPEC References

- FR-016
- FR-018 至 FR-020
- CR-005

## SDD References

- 1. Existing Architecture Summary
- 2. Problem Analysis
- 12. Migration Risks

## Acceptance Criteria

- 每个服务对 `smart-recruit-domain-go` 的依赖被记录。
- 每个服务 owner 表、过渡只读表、潜在违规写风险被记录。
- 输出可作为后续 TASK baseline。

## Required Checks

- `node scripts/check-mysql-table-ownership.mjs`
- scope check
- agent-check

## Manual Verification, if needed

确认盘点未把历史 `.spec/backend-ddd-microservices-evolution` 当作当前执行合同。

## Out-of-Scope

修复业务依赖、修改表归属、修改代码。
