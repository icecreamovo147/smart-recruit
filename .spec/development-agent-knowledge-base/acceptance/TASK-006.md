# Acceptance - TASK-006

## TASK Summary

向共享 spec-harness 增加向后兼容的 knowledge scope、报告和 evidence 语义。

## SPEC References

- FR-006、FR-007、FR-012
- AC-005、AC-006、AC-010、AC-014

## SDD References

- Section 4.3, Knowledge Impact Schema
- Section 5.2, Harness Interface
- Section 8, Compatibility Strategy
- Section 11.2, Harness Regression
- Section 13, TASK-006

## Acceptance Criteria

- 实施前已记录明确人工确认。
- 新 feature 能声明知识 review/modify/candidate 范围和 required knowledgeImpact。
- TASK report/evidence 能记录结果、trigger、逐文档 verdict、coverage gap 和校验退出码。
- 当前和 legacy-compatible 历史 feature 不因缺少新字段变为 unsupported。
- Scope 校验仍要求可靠 TASK baseline。
- 失败、跳过、STALE、CONFLICT 和缺少确认不能被描述为无条件完成。
- 不修改 harness-pipeline 或 pipeline-state 权威语义。

## Required Checks

- `node .agents/skills/spec-harness/scripts/validator.test.mjs`
- 知识工具测试、结构校验和引用检查。
- `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/development-agent-knowledge-base`
- 验证一个 current 和一个 legacy-compatible 历史 feature。
- `git diff --check`
- Harness scope 与 agent check。

## Manual Verification, if needed

- Reviewer 检查新字段为向后兼容扩展，没有改变 pipeline 状态机。

## Out-of-Scope

- harness-pipeline 修改。
- 历史 feature 批量迁移。
- AGENTS、CI 和业务代码。
