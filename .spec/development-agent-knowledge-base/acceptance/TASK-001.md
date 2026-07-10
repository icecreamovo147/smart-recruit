# Acceptance - TASK-001

## TASK Summary

建立 `.knowledge` 的基础治理契约、目录入口、元数据 schema、模板以及目录局部 Git 规则。

## SPEC References

- FR-001 至 FR-004
- FR-007、FR-008
- AC-001、AC-002、AC-006、AC-013、AC-014

## SDD References

- Section 3, Proposed Design
- Section 4, Data Structure Changes
- Section 7, Configuration Design
- Section 13, TASK-001

## Acceptance Criteria

- `.knowledge/README.md` 定义目标、非目标、权威层级、读取/写入、生命周期、冲突和安全协议。
- `INDEX.md` 是稳定路由，不包含动态完整 catalog。
- `manifest.yaml` 只负责全局政策、routes 和 triggers。
- Frontmatter/manifest JSON schema 语法有效并覆盖 SPEC 必需字段。
- Templates 覆盖普通知识、ADR 和 Inbox 候选。
- `.knowledge/.gitignore` 排除 `.local/` 和 `generated/`。
- `.knowledge/.gitattributes` 对知识文本使用 LF。
- 不创建未核验的架构、领域、Runbook 或 Pitfall 内容。

## Required Checks

- `git diff --name-only`
- `git diff --check`
- 使用 Node 读取两个 JSON schema。
- `bash .spec/development-agent-knowledge-base/scripts/check-task-scope.sh TASK-001`
- `bash .spec/development-agent-knowledge-base/scripts/agent-check.sh`

## Manual Verification, if needed

- 确认 README 不把 `.knowledge` 声明为高于 AGENTS、SPEC、代码或测试的来源。
- 确认 Inbox 和 Archive 默认不用于实现。

## Out-of-Scope

- 校验脚本实现。
- 首批正式知识。
- AGENTS、共享 Harness、CI、业务代码和历史材料修改。
