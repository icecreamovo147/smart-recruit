# Acceptance - TASK-002

## TASK Summary

使用 Node.js 标准库实现知识结构校验、引用检查、Git 影响检测、catalog 生成和自动化测试。

## SPEC References

- FR-009 至 FR-011
- AC-004、AC-007、AC-008、AC-013、AC-014

## SDD References

- Section 5.1, CLI Interfaces
- Section 6, Algorithm or Workflow Changes
- Section 9, Error Handling
- Section 11.1, Unit Tests
- Section 13, TASK-002

## Acceptance Criteria

- 四个 CLI 支持 SDD 规定的参数、退出码和可选 JSON 输出。
- 所有路径归一化为仓库相对 POSIX 路径。
- 无可靠 baseline 时 `detect-impact` 明确失败。
- tracked 和 untracked 变化均纳入影响检测。
- 校验器覆盖必需元数据、唯一 ID、状态、owner、引用、绝对路径、大小写冲突、Inbox/Archive 和 ADR 关系。
- route 输出稳定排序并能报告 coverage gap。
- 只使用 Node 标准库，不修改 package manifest 或 lockfile。

## Required Checks

- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `node --check` 检查所有 `.knowledge/scripts/*.mjs`
- `git diff --check`
- Harness scope 与 agent check。

## Manual Verification, if needed

- 检查文本和 JSON 输出不包含知识全文或环境敏感信息。

## Out-of-Scope

- 修改 schema、manifest 或治理正文。
- 首批知识内容。
- AGENTS、共享 Harness 和 CI。
