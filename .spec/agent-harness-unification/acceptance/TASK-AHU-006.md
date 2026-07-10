# Acceptance - TASK-AHU-006

## TASK Summary

为旧 Harness 和 `.ai-guides` 添加 Legacy/Reference 入口，并生成不改变历史状态的迁移清单。

## SPEC References

- FR-004、FR-005、FR-006、FR-012
- AC-004、AC-005、AC-013
- Security Requirements 5、6

## SDD References

- 3.4 Legacy 标识设计
- 3.5 遗留迁移清单设计
- 8.3 历史文档
- 12 Migration Risks 1、2、9

## Acceptance Criteria

- [ ] `docs/agent-harness` 和 `.ai-guides` 有明确 Legacy/Reference 入口。
- [ ] Legacy 入口说明冻结不等于取消，pending 需按需迁移。
- [ ] 旧任务、Review、ADR、delivery、acceptance 和 Agent memory 未删除或批量修改。
- [ ] Inventory 包含旧日志中 11 个 pending 任务。
- [ ] Inventory 覆盖含 constitution/spec/plan/tasks 的 `.ai-guides` phase。
- [ ] Inventory 标记 phase 是否已有 delivery/acceptance。
- [ ] Inventory 包含 `semantic-retrieval-score-fixes`，decision 为 separate-repair。
- [ ] 每条 inventory 记录全部规定字段。
- [ ] 不声称未核对的业务代码已经验证。

## Required Checks

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-006
bash .spec/agent-harness-unification/scripts/agent-check.sh
```

还需通过只读脚本：

- 从 `EXECUTION_LOG.md` 统计 pending 数量并与 inventory 对齐；
- 枚举 `.ai-guides` phase contract 目录并与 inventory 对齐；
- 检查不允许的历史文件没有进入 diff。

## Manual Verification, if needed

- 人工抽查至少一个 completed 历史任务、一个 pending 任务和一个 phase delivery，确认原内容未变。

## Out-of-Scope

- 批量迁移 11 个 pending 业务任务；
- 修改旧任务状态；
- 分析或实现 `.ai-guides` 业务方案；
- 修复 semantic retrieval 业务或 Harness。
