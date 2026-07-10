# Acceptance - TASK-007

## TASK Summary

在用户确认 CI 平台与最终 scope 后，将知识校验接入 PR/分支流水线。

## SPEC References

- FR-014
- AC-008、AC-012、AC-013、AC-014

## SDD References

- Section 7.3, CI Configuration
- Section 11.4, Cross-Platform
- Section 13, TASK-007

## Acceptance Criteria

- CI 平台和 scope 已由用户明确确认并记录。
- 若采用 GitHub Actions，只创建指定知识校验工作流。
- 工作流运行知识工具测试、结构校验、引用检查和 feature validator。
- 工作流不需要 Secret，不运行部署或业务环境。
- Node 版本明确，触发范围避免无关高成本执行。
- `.knowledge/README.md` 记录本地等价命令。
- 不修改 package manifest、lockfile、部署或业务配置。

## Required Checks

- 本地执行 CI 中的全部命令。
- CI YAML 语法检查；若无仓库内可用解析器，记录人工 Review。
- `git diff --check`
- Harness scope 与 agent check。

## Manual Verification, if needed

- 在目标 CI 平台验证一次 PR/分支运行结果。

## Out-of-Scope

- 部署流水线。
- Secret 配置。
- 业务测试矩阵扩展。
- 未确认平台的工作流文件。
