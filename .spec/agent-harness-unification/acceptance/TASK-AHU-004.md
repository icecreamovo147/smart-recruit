# Acceptance - TASK-AHU-004

## TASK Summary

建立 Claude Code 根入口，并将单 TASK Developer、Reviewer、Fixer 配置收敛为 canonical mode 薄适配器。

## SPEC References

- FR-001、FR-002、FR-003、FR-009
- AC-001、AC-002、AC-003、AC-010、AC-012
- Compatibility Requirements 5、6

## SDD References

- 3.2 根入口设计
- 3.3 Claude Skill 和 Agent 适配
- 5.2 Agent 工作流接口
- 8.2 Claude 旧命令

## Acceptance Criteria

- [ ] 根 `CLAUDE.md` 可发现并复用 `AGENTS.md`。
- [ ] 根入口只包含 Claude-specific 适配，不复制完整 Harness。
- [ ] `.claude/CLAUDE.md` 不再是独立权威规则源。
- [ ] Developer 映射 `implement-task`。
- [ ] Reviewer 只读并映射 `self-review`，使用权威 verdict。
- [ ] Fixer 映射 `fix-check-failures`。
- [ ] 适配器不再使用旧任务目录、EXECUTION_LOG、固定 integration branch 或独立完成定义。
- [ ] 适配器不含绝对用户路径。
- [ ] 无 subagent 时披露 self-review 降级。
- [ ] `.claude/settings.local.json` 和 `.claude/agent-memory` 未修改。

## Required Checks

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-004
bash .spec/agent-harness-unification/scripts/agent-check.sh
rg -n "spec-harness|implement-task|self-review|fix-check-failures|\.spec/" CLAUDE.md .claude/CLAUDE.md .claude/skills .claude/agents/task-*.md
```

还需确认可执行适配器中不存在未豁免的 `docs/agent-harness`、`EXECUTION_LOG.md`、`integration/agent-platform` 或绝对项目路径。

## Manual Verification, if needed

- 如本地 Claude Code 可用，执行只读 discovery/smoke check。
- 不可用时记录未验证项和后续验证命令。

## Out-of-Scope

- Batch coordinator 和 phase 入口；
- Legacy banner；
- 本地权限配置和 Agent memory；
- 业务代码。
