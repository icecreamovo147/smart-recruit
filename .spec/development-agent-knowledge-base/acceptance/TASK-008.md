# Acceptance - TASK-008

## TASK Summary

只读审计完整知识库、Harness 集成、跨平台约束和代表性路径影响检测，并生成最终证据。

## SPEC References

- 全部功能需求
- AC-001 至 AC-015

## SDD References

- Section 11, Testing Strategy
- Section 12, Migration Risks
- Section 13, TASK-008

## Acceptance Criteria

- Feature 被共享 validator 分类为 `current`。
- SPEC、SDD、TASKS、scope、acceptance、实际文件和报告一致。
- 所有知识工具测试、结构校验、引用检查和 Harness 回归通过。
- 代表性 Agent Skill、Memory、Proto 和 HR 管理页路径命中预期知识。
- 无绝对路径、敏感数据、Provider 专属知识副本和未记录 coverage gap。
- macOS/Linux 命令可运行，Windows 路径归一化有自动化覆盖。
- CI 已通过或存在用户批准、证据完整的延期例外。
- 报告明确剩余风险与 feature 是否真正完成。

## Required Checks

- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `node .agents/skills/spec-harness/scripts/validator.test.mjs`
- `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/development-agent-knowledge-base`
- `git diff --check`
- Harness scope 与 agent check。

## Manual Verification, if needed

- Reviewer 复核真实 TASK 样本和任何 approved exception。

## Out-of-Scope

- 修复审计发现的问题。
- 修改 `.knowledge`、AGENTS、Harness、CI 或业务文件。
