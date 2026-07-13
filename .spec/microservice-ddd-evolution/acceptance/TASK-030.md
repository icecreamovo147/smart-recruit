# Acceptance - TASK-030

## TASK Summary

最终重命名 `smart-recruit-domain-go` 为 `smart-recruit-commons`。

## SPEC References

- FR-027
- FR-028
- CR-011
- AC-012

## SDD References

- 3.2 Shared Module Target
- 3.6 Final Commons Target
- 13. Implementation Boundaries

## Acceptance Criteria

- 目录与 Go module path 重命名为 `smart-recruit-commons`。
- 旧 `smart-recruit-domain-go` import/path 在业务代码、测试、构建、部署和文档中清零，除非报告中列出批准的历史引用。
- `smart-recruit-commons` 只包含 shared kernel 和通用技术能力。
- 全仓库验证通过或记录不可运行原因。

## Required Checks

- 所有 Go module 或 workspace-equivalent `go test ./...`
- `node scripts/check-mysql-table-ownership.mjs`
- backend boundary checks
- `rg "smart-recruit-domain-go"`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认，是本功能点最后一个迁移任务。

## Out-of-Scope

新增业务功能、protobuf 行为变化、schema 行为变化、auth/security 行为变化。
