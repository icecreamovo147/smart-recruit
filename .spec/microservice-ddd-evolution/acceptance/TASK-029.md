# Acceptance - TASK-029

## TASK Summary

收缩 `smart-recruit-domain-go` 至 commons-ready shared kernel。

## SPEC References

- FR-016
- FR-027
- FR-028
- ARC-007
- ARC-015

## SDD References

- 3.2 Shared Module Target
- 3.6 Final Commons Target
- 13. Implementation Boundaries

## Acceptance Criteria

- `smart-recruit-domain-go` 不再包含 active 业务 `model/repository/service` 实现。
- 剩余内容分类为 shared kernel、platform-adjacent、testutil 或 migration helper。
- 所有服务对旧业务共享实现依赖清零或转为允许保留的 shared kernel 依赖。

## Required Checks

- 受影响 Go module 的 `go test ./...`
- `node scripts/check-mysql-table-ownership.mjs`
- backend boundary checks
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认；确认不修改 module path。

## Out-of-Scope

目录/module/import 重命名，留到 TASK-030。
